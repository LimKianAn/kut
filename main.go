package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

const defaultKubeconfig = "~/.kube/config"

var (
	rootCmd = &cobra.Command{
		Use:          "kut",
		Short:        "Cut out a self-contained kubeconfig",
		Long:         "Cut out a self-contained kubeconfig of a kind cluster and replace the endpoint of api-server with docker container IP and default port",
		RunE:         rootCmdRunE,
		SilenceUsage: true,
	}

	docker *client.Client
)

func init() {
	rootCmd.Flags().StringP("kubeconfig", "k", "", "path to input kubeconfig (first the flag, then env KUBECONFIG, at last "+defaultKubeconfig)
	rootCmd.Flags().StringP("context", "c", "", "target context")

	if err := rootCmd.MarkFlagRequired("context"); err != nil {
		log.Fatal(err)
	}

	if err := viper.BindPFlags(rootCmd.Flags()); err != nil {
		log.Fatal(err)
	}

	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Fatal(err)
	}
	docker = dockerClient
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func rootCmdRunE(cmd *cobra.Command, args []string) error {
	path, err := kubeconfigPath()
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig path: %w", err)
	}

	config, err := clientcmd.LoadFromFile(path)
	if err != nil {
		return fmt.Errorf("failed to load the kubeconfig: %w", err)
	}

	if !containsCtx(config) {
		return errors.New("no such context in kubeconfig")
	}
	selectContext(config)
	useDockerConainerIPAndDefaultAPIServerPort(config)

	bytes, err := clientcmd.Write(*config)
	if err != nil {
		return fmt.Errorf("failed to serialize the kubeconfig to yaml: %w", err)
	}
	fmt.Print(string(bytes))

	return nil
}

func kubeconfigPath() (string, error) {
	path := viper.GetString("kubeconfig")
	if path == "" {
		path = os.Getenv("KUBECONFIG")
		if path == "" {
			path = defaultKubeconfig
		}
	}
	if strings.Contains(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home dir: %w", err)
		}
		path = strings.ReplaceAll(path, "~", home)
	}
	return path, nil
}

func containsCtx(config *api.Config) bool {
	configCtx := viper.GetString("context")
	_, ok := config.Contexts[configCtx]
	return ok
}

func selectContext(config *api.Config) {
	configCtx := viper.GetString("context")
	config.Contexts = map[string]*api.Context{configCtx: config.Contexts[configCtx]}
	config.Clusters = map[string]*api.Cluster{configCtx: config.Clusters[configCtx]}
	config.AuthInfos = map[string]*api.AuthInfo{configCtx: config.AuthInfos[configCtx]}
	config.CurrentContext = configCtx
}

func useDockerConainerIPAndDefaultAPIServerPort(config *api.Config) {
	configCtx := viper.GetString("context")
	containers, _ := docker.ContainerList(
		context.Background(),
		types.ContainerListOptions{
			Filters: filters.NewArgs(
				filters.Arg("name", strings.TrimPrefix(configCtx, "kind-")+"-control-plane"), // e.g. `kind-kind2` to `kind2-control-plane`
			),
		},
	)
	config.Clusters[configCtx].Server = fmt.Sprintf("https://%s:6443", containers[0].NetworkSettings.Networks["kind"].IPAddress)
}
