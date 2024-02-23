# kube-awi
Cisco k8s operator and controller implementation for Application WAN Interface (AWI)

## Development

### Adding new object
- run `kubebuilder create api --group awi --version v1 --kind <NewObjectName>` to create base structure,
- update object spec and controller logic,
- execute steps from [Updating object](#Updating object) section.

### Updating object
Run:
- `go get github.com/app-net-interface/awi-grpc@develop` to pull latest changes in specifications,
- `make manifests` to generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects,
- `make generate` to generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.

### Installing objects
Run:
- `make install` to install CRDs into the K8s cluster specified in ~/.kube/config.

### Running controller
Run:
- `make docker-build` to build controller image,
- (if you use kind cluster) change image pull policy for `app-net-interface.io/awi-controller:<TAG>` image to
`imagePullPolicy: IfNotPresent` to avoid pulling issues (TODO: automate this), required one time only,
- (if you use kind cluster) `CLUSTER_NAME=<your-cluster-name> make kind-load`,
- (if you use remote cluster) `make docker-push`,
- `make deploy` to update image in cluster controller deployment.

## Contributing

Thank you for interest in contributing! Please refer to our
[contributing guide](CONTRIBUTING.md).

## License

kube-awi is released under the Apache 2.0 license. See
[LICENSE](./LICENSE).

kube-awi is also made possible thanks to
[third party open source projects](NOTICE).

## Running with minikube

Here is the instruction how to test Kube AWI with locally created
cluster with Minikube.

Create minikube cluster
```
minikube start
```

Create `awi-system` namespace.
```
kubectl create ns awi-system
```

To avoid pushing images to registries, simply set docker registry
context to use minikube's one:
```
eval $(minikube docker-env)
```

Now you can build images directly to minikube docker registry:
```
docker build --build-arg SSH_PRIVATE_KEY="$(cat PRIVATE_SSH_PATH)" -t IMAGE_NAME:IMAGE_TAG .
```

Replace PRIVATE_SSH_PATH, IMAGE_NAME and IMAGE_TAG with your values.

The private SSH key is necessary only for internal github access.
After opensourcing the project, the Dockerfile will be adjusted properly
and no private key will be longer needed.
The Dockerfile uses 2 stages for creating the image, the final stage
doesn't use the private SSH key so there is no need to worry about
exposing your private SSH key.

Now generate CRDs and deploy Kube AWI:
```
IMG=IMAGE_NAME:IMAGE_TAG make install
IMG=IMAGE_NAME:IMAGE_TAG make deploy
```

Again, replace IMAGE_NAME and IMAGE_TAG with your own values.

This should create CRDs and Manager pod in the cluster:

```
> kubectl get crds -A
---
NAME                                             CREATED AT
instances.awi.app-net-interface.io                          2024-02-09T04:09:55Z
internetworkdomainappconnections.awi.app-net-interface.io   2024-02-09T04:09:55Z
internetworkdomains.awi.app-net-interface.io                2024-02-09T04:09:55Z
networkdomains.awi.app-net-interface.io                     2024-02-09T04:09:55Z
sites.awi.app-net-interface.io                              2024-02-09T04:09:55Z
subnets.awi.app-net-interface.io                            2024-02-09T04:09:55Z
vpcs.awi.app-net-interface.io                               2024-02-09T04:09:55Z
vpns.awi.app-net-interface.io                               2024-02-09T04:09:55Z
```

```
> kubectl get pods -A
---
NAMESPACE     NAME                                          READY   STATUS    RESTARTS      AGE
awi-system    kube-awi-controller-manager-9d4697db6-h9qmn   2/2     Running   0             24m
kube-system   coredns-5dd5756b68-kz7zq                      1/1     Running   0             20h
kube-system   etcd-minikube                                 1/1     Running   0             20h
kube-system   kube-apiserver-minikube                       1/1     Running   0             20h
kube-system   kube-controller-manager-minikube              1/1     Running   0             20h
kube-system   kube-proxy-l6z4w                              1/1     Running   0             20h
kube-system   kube-scheduler-minikube                       1/1     Running   0             20h
kube-system   storage-provisioner                           1/1     Running   1 (20h ago)   20h
```

In order to make `kube-awi-controller-manager` working you need to modify the deployent:

```
kubectl edit deployment  kube-awi-controller-manager -n awi-system
```

Locate args and add `--awi-catalyst-address` pointing at the local process
of awi-grpc-catalyst-sdwan

```
- args:
    - --health-probe-bind-address=:8081
    - --metrics-bind-address=127.0.0.1:8080
    - --awi-catalyst-address=host.minikube.internal:50051
    - --leader-elect
```

The `host.minikube.internal` address points to your host machine.
The AWI GRPC Catalyst SDWAN needs to be started on `0.0.0.0` rather
than `127.0.0.1` - otherwise it won't work.

If, for some reason, this won't be able to reach your host address
(you will know that by the fact that manager logs will show only one
entry: `connecting`), try running `minikube tunnel` in different
terminal.

After doing so, the pod should be restarted and initialized successfully,
which you can inspect by seeing `2/2` containers in `get pods` command and
by seeing normal logs of your manager.

To create a connection, you can try running:

```
kubectl apply -f config/samples/awi_v1_internetworkdomain_vpc_to_vpc.yaml
```

Of course, the file needs to be modified to match your desired VPCs.

You can now see created connection:
```
> kubectl get internetworkdomains
---
NAME                    AGE
my-internetworkdomain   42s
```

This connection will most likely exist even if the actual connection was not created.
Check your AWI GRPC Catalyst SDWAN logs to see how the creation went.

To destroy the connection, simply remove the CR:
```
kubectl delete internetworkdomains my-internetworkdomain
```

That's it :)

To create sample app connection, you can use the following file:
```
kubectl apply -f examples/gcp-vpc-vpc-label-app-connection.yaml
```

After your minikube cluster is no longer needed, you can remove it by running:
```
minikube delete
```
