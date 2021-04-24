# kut

Cut out a dedicated kubeconfig for a kind cluster from a mixed kubeconfig and replace the endpoint of api-server with docker container IP and default port `6443` so that the kubeconfig can be used in a pod in another kind-cluster.

## Install

```bash
go install github.com/LimKianAn/kut
```

## Demo

```bash
kind create cluster
kind create cluster --name kind2
k ctx # kubectx
```

Two contexts are observed.

```console
kind-kind
kind-kind2
```

Use `kut` to get a dedicated kubeconfig which can be used in a pod in another kind-cluster.

```bash
kut -c kind-kind
```
