# Debugging

> What is `/hack` folder doing?

- a convention in Kubernetes communities, where various configuration/scripts related to devops are collected.

> how to make a pod having access to k8s cluster API, i.e. can create/delete/update oher pods?

a few options:
- inject `kubeconfig` as secret, then initiate `kubernetes.Clientset` from `kubeconfig`.
- same as above, but inject via configmap instead.
- use `role/rolebinding/service-acount` resources to allow k8s control plane.

Ended up with last option.

> how to enable `LoadBalancer` kind Service?

Tried:
- A: MetalLB 1.12. [installation script here](https://github.com/luksa/kubernetes-in-action-2nd-edition/blob/a511b9ede8b34f25b1b0f748e742be770f091f74/Chapter11/install-metallb-kind.sh#L4). Failed because it used deprecated k8s features.
- B: MetalLB 1.14. same installation script as above, just different version. Failed because `podman network inspect kind | jq '.[0].IPAM'` returns empy dict.
    - [potential explanation here](https://github.com/kubernetes-sigs/kind/issues/3560): metalLB does not work out-of-box on Mac/Windows.
- C: cloud-provider-kind@latest. [installation guide here](https://kind.sigs.k8s.io/docs/user/loadbalancer/).
    - example service worked (using `default` namespace).
    - does not work with `faas-gateway` yet.


```shell
# run the health check a few times
bash hack/faas-health-check.sh
# logs from the service
2025/10/06 07:03:23 "GET http://10.89.0.7:8080/health HTTP/1.1" from 10.244.0.1:1071 - 404 19B in 2.123919ms
2025/10/06 07:15:44 "GET http://10.89.0.7:8080/health HTTP/1.1" from 10.244.0.1:8010 - 404 19B in 419.293µs
2025/10/06 07:32:57 "GET http://10.89.0.7:8080/health HTTP/1.1" from 10.244.0.1:6069 - 404 19B in 107.001µs
```

> What is Container image "localhost/faas-gateway-app:latest" is not present error?

- This is because kind cluster does not have access to `docker image`.
- update docker build script from

```shell
podman build -t faas-gateway-app:latest -f {{justfile_directory()}}/cmd/faas/Dockerfile {{justfile_directory()}}
```
to 

```shell
podman build -t faas-gateway-app:latest -f {{justfile_directory()}}/cmd/faas/Dockerfile {{justfile_directory()}}
# load image in
# see https://kind.sigs.k8s.io/docs/user/quick-start/#loading-an-image-into-your-cluster
kind load docker-image localhost/faas-gateway-app:latest --name $(kind get clusters)
```