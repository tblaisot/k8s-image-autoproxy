# k8s-image-autoproxy

## build

```
make build
```

## test

```
make test
```

## ssl/tls

the `ssl/` dir will contain informations used to generate self signed certificates to deploy

## docker

to create a docker image ..

```
make build-image
```

### images

[`tblaisot/k8s-image-autoproxy`](https://cloud.docker.com/repository/docker/tblaisot/k8s-image-autoproxy)

## Running in kind

```bash
# spawn a Kind cluster
make kind

# generate certificates
make gen-deploy-certs

# build image
make build-image

# deploy
make deploy

# make sure it's running ...
kubectl get pods -n k8s-image-autoproxy
kubectl logs k8s-image-autoproxy-<POD> --follow -n k8s-image-autoproxy

# create example pod to see it working
kubectl apply -f integration-tests/pod.yaml
kubectl get pod c7m -o yaml | grep image: # should be prefixed
kubectl delete -f integration-tests/pod.yaml
```

## kudos

- [Writing a very basic kubernetes mutating admission webhook](https://medium.com/ovni/writing-a-very-basic-kubernetes-mutating-admission-webhook-398dbbcb63ec)
- [https://github.com/alexleonhardt/k8s-mutate-webhook](https://github.com/alexleonhardt/k8s-mutate-webhook)
