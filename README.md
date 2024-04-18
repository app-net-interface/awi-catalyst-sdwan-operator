# kube-awi
Cisco k8s operator and controller implementation for Application WAN Interface (AWI)

The kube-awi allows using k8s custom resources to interact with AWI project.

The project contains helm chart for deploying k8s operator along with catalyst
sdwan controller.

# Overview

With kube-awi installed on the k8s cluster, the following actions can be done:

* creating requests to the awi controller with `kubectl apply`
* getting information about instances, network domains etc. with `kubectl get`

Installation of kube-awi on the k8s cluster involves creating Custom Resource
Definitions, namely:

* instances.awi.app-net-interface.io
* internetworkdomainappconnections.awi.app-net-interface.io
* internetworkdomainconnections.awi.app-net-interface.io
* networkdomains.awi.app-net-interface.io
* sites.awi.app-net-interface.io
* subnets.awi.app-net-interface.io
* vpcs.awi.app-net-interface.io
* vpns.awi.app-net-interface.io

They can be grouped into two use cases

1. Interacting with the awi

    Resources `internetworkdomainconnections` and `internetworkdomainappconnections` are
    a way of creating requests to AWI system.

    For instance, applying the following `internetworkdomainconnection.yaml` manifest

    ```yaml
    apiVersion: awi.app-net-interface.io/v1alpha1
    kind: InterNetworkDomainConnection
    metadata:
    name: my-internetworkdomainconnection
    spec:
    metadata:
        name: example-name
    spec:
        destination:
            metadata:
                name: machine-learning-training
                description: "Description of the destination"
            networkDomain:
                selector:
                matchId:
                    id: vpc-097e8ed349c13c004
        source:
            metadata:
                name: machine-learning-dataset
                description: "Description of the source"
            networkDomain:
                selector:
                matchId:
                    id: vpc-04a1eaad3aa81310f
    ```

    will create a request to create Network Domain Connection
    between Source VPC `vpc-097e8ed349c13c004` and Destination
    VPC ` vpc-04a1eaad3aa81310f`.

1. Checking existing VPCs, Instances, Subnets etc.

    Resources `instance`, `network_domain`, `site`, `subnet`, `vpc`
    and `vpn` represent resources that can be inspected using either
    AWI UI or AWI CLI.

    To check the list of instances, simply run

    ```
    // Get all instances
    kubectl get instance -A 
    ```

    and the kubectl will return a list of obtained instances from
    supported providers (AWS and GCP) just like getting list of
    `pods`, `deployments` etc.

    To get details of a certain instance run

    ```
    kubectl get instance INSTANCE_ID -n awi-system -o yaml
    ```

    to see the details of the resource.

## Under the hood

Both use cases described above are available thanks to the k8s
operator.

![Image of Kube AWI](docs/kube-awi.png)

Installation of the kube-awi on the k8s creates a special deployment
called `kube-awi-controller-manager` (kube-awi operator in the graph) which acts as a special process
that will:

* watch for updates of `internetworkdomainconnections` and
    `internetworkdomainappconnections` custom resources and triggers
    actions defined in `controllers/RESOUCE_controller.go`

* synchronizes other resources by periodically obtaining lists of
    subnets, instances etc. from `awi-grpc-catalyst-sdwan` and
    creating custom resources inside the cluster

The Kube-AWI operator consists of so called controllers, that define
methods to be triggered for certain events, syncer which is a simple
goroutine and awi client which implements necessary interfaces and
specifies an address of the actual server from which the information
will be received and where connection requests will be forwarded.

The Kube-AWI operator also includes standard k8s operator manager
responsible for health checks and other useful resources.

### Controllers

Watching for updates and triggering certain actions is accomplished
with so called controllers. To generate a controller go to
[Adding new object](#adding-new-object).

A Controller specifies `Reconcile` method will is triggered whenever
there is an update of the certain Custom Resource. This method will
be called whenever a resource is created/updated/deleted.

Currently, kube-awi defines both resources in the following manner:

* removing Custom Resource using `kubectl delete` triggers deletion
    of VPC Connection or App Connection

* other events trigger Connection Creation attempt.

#### K8s data

Custom Resources specify two important sections:

1. Spec - the desired configuration for a resource

    Spec section is a user's input space. It accepts information about
    desired settings and the Reconciler's job is to attempt accomplishing
    them.

1. Status - the actual state of the object

    Status is a read-only information for the user about actual state of
    the resource. While state specifies the desired state, which may not
    be accomplished yet, impossible to accomplish or be a high-level
    definition of certain settings, the status section is updated by the
    reconciler and it is supposed to provide information about the present
    state and underlying low-level information that may be necessary for
    the user.

Currently, we do not make much use of the status field, but it could be
used for storing information about whether connection succeeded or not,
what was the error etc.

### Synchronizers

Kube-awi operator runs a syncing goroutine which periodically calls
awi-grpc-catalyst-sdwan to obtain resources from the AWI. Later, it
creates or updates Custom Resources associated with these resources.

Since these are read-only resources, they have no Controllers assigned
to them, as the operator does not care about user's changes there.

Since these resources are updated by the periodic sync operation, they
are eventually consistent.

# Development

The kube-awi uses kubebuilder framework for automatic creation of:
* Custom Resource Definitions
* Operator's code for UPDATE actions

The input for both is a go file `api/GROUP/VERSION/NAME_types.go`.

It specifies the structure of the resource and relevant fields.
The resouce can be marked with additional kubebuilder options to
customize the structure and its interactions (for example additional
columns present when running `kubectl get instances -A`)

## Adding new object

To create a new kind such as `instance` or `network_domain`, run kubebuilder
command to initiate a new object

```
kubebuilder create api --group awi --version v1alpha1 --kind <NewObjectName>
```

This will generate a new file in the directory `api/awi/v1alpha1/KIND_types.go`
with placeholder golang structures for both structure itself and a list
wrapper.

The CLI will ask if it should generate a controller for the resource. Replying
`yes` will add a note in the `PROJECT` file `controller: true`. This can be
changed manually later on.

To generate CRDs and operator code follow steps below.

## Updating object

To generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects run

```
make manifests
```

To generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations:

```
make generate
```

Make sure to use up-to-date version of awi-grpc repository.

## Installing objects
Run:
- `make install` to install CRDs into the K8s cluster specified in ~/.kube/config.

## Running controller
Run:
- `make docker-build` to build controller image,
- (if you use kind cluster) change image pull policy for `app-net-interface.io/awi-controller:<TAG>` image to
`imagePullPolicy: IfNotPresent` to avoid pulling issues (TODO: automate this), required one time only,
- (if you use kind cluster) `CLUSTER_NAME=<your-cluster-name> make kind-load`,
- (if you use remote cluster) `make docker-push`,
- `make deploy` to update image in cluster controller deployment.

# Extending Kube-AWI

Currently, the kube-awi project gathers the entire logic in the
`main.go` file which is an entry point for the k8s operator. This
file instantiates k8s operator manager, initializes kube awi client,
registers existing reconcilers and runs syncing goroutine.

To make kube-awi more opened for other possible controllers, the
main file requires some design decisions over how different
controllers should be implemented.

1. The kube-awi embeds all 3 GRPC interfaces (connection, app connection and cloud clients)
    into single client with a configurable address. It means that current implementation
    prevents user from specifying a different address for cloud interface responsible for
    obtaining information about existing subnets etc. and for connection/app connection
    clients.

    It means that a new controller need to implement the entire logic - if we want to
    write a new controller which defines new logic for connections or app connections
    but we want to remain cloud connectivity from old controller, we would have to
    make our new controller forward cloud requests to the old one.

1. Connection and App Connection interfaces are explicitely loaded in the `main.go`
    file. If a new controller requires a different reconciling action, a new provider
    should be specified. Considering a scenario where all controller versions are
    specified inside kube-awi repository, the `main.go` file should be changed to
    dynamically load desired controllers based on the provided configuration.

1. Syncer goroutine is quite specific to awi project and the user may wish to use
    different implementation, use different CRDs for that or to not use syncer at
    all. This topic leads to a further design discussion around making kube-awi
    a library.

The project graph above shows the existing dependencies and potential points of
providing an abstraction over pieces of code that can be turned into customizable
modules.

# Helm chart

The kube-awi repository contains two charts within `chart` directory:

1. catalyst sdwan chart - the chart containing manifests for AWI GRPC Catalyst Sdwan controller

1. operator chart - the second chart responsible for kube-awi k8s operator that allows
    spawning operator and necessary CRDs

The catalyst sdwan chart is the main chart which sets a dependency to an operator
chart. The separation comes from the fact that both charts have different building flow.

## Deploying AWI Catalyst SDWAN with operator

Here is the instruction how to test Kube AWI with locally created
cluster with Minikube.

### Prerequisites

To deploy AWI Catalyst SDWAN with operator, following things are needed:

1. K8s cluster (we will use minikube)
1. Catalyst SDWAN Address and Credentials
1. Helm - for deploying helm charts with operator and controller

### Cluster preparation

Create minikube cluster
```
minikube start
```

The start operation will use set of default options such as CPU and
RAM that will be assigned to the cluster, k8s version etc.

If you need to modify these options, check the minikube help.

Confirm that cluster is running properly by running
```
kubectl get pods -A
```

All pods should be running:
```
NAMESPACE     NAME                                                   READY   STATUS    RESTARTS      AGE
kube-system   coredns-5dd5756b68-nm7hv                               1/1     Running   0             21h
kube-system   etcd-minikube                                          1/1     Running   0             21h
kube-system   kube-apiserver-minikube                                1/1     Running   0             21h
kube-system   kube-controller-manager-minikube                       1/1     Running   0             21h
kube-system   kube-proxy-btvwp                                       1/1     Running   0             21h
kube-system   kube-scheduler-minikube                                1/1     Running   0             21h
kube-system   storage-provisioner                                    1/1     Running   1 (21h ago)   21h
```

### Preparing cluster for the application

Create your desired namespace, for instance `awi-system`.
```
kubectl create ns awi-system
```

Next, create secrets.

**Secrets needs to be created anyway - if you don't use**
**GCP, for instance, leave the values empty.**

#### Catalyst SDWAN Controller credentials

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: catalyst-sdwan-credentials
type: Opaque
data:
  username: "{CATALYST_SDWAN_USERNAME}"
  password: "{CATALYST_SDWAN_PASSWORD}"
```

Remember to base64 encode values.

#### Provider specific credentials

Currently, k8s operator supports listing resources for AWS and
GCP providers.

The AWS secret currently expects base64 encoded `credentials` file
such as `$HOME/.aws/credentials`:

```ini
[default]
aws_access_key_id = KEY
aws_secret_access_key = VALUE
```

and such base64 encoded file should be placed inside a following secret:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: aws-credentials
type: Opaque
data:
  credentials: "{FILE_ENCODED}"
```

Similarly, GCP credentials also require base64 encoded file, which can be
found under `$HOME/.config/gcloud`. The example file content:

**Service Account is required.**

```json
{
  "client_email": "CLIENT_EMAIL",
  "client_id": "CLIENT_ID",
  "private_key": "PRIVATE_KEY",
  "private_key_id": "PRIVATE_KEY_ID",
  "token_uri": "TOKEN_URI",
  "type": "service_account"
}
```

And such base64 encoded file should be put in following secret:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: gcp-credentials
type: Opaque
data:
  gcp-key.json: "{FILE_ENCODED}"
```

#### Cluster Context

If the administrator wants App Net Interface to be able to interact with
k8s cluster (discovery process or creating connections to pods) the kubeconfig
file needs to be provided as a secret (base64 encoded):

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: kube-config
type: Opaque
data:
  config: "{FILE_ENCODED}"
```

### Deploy chart

Before running helm install, prepare your options.

Now, you can create a new yaml file or modify local `values.yaml`
to modify default values.

In `values.yaml`, the `config` section contains configuration
for AWI Catalyst SDWAN controller and `awi-catalyst-sdwan-k8s-operator`
specifies the configuration OVERRIDING `values.yaml` from the
operator chart.

Install chart using helm in the proper namespace (awi-system as example).
Choose the name of the helm project (awi as example).
```
helm install awi chart/ -n awi-system
```

If you want to pass a file overriding values from `values.yaml` use
`-f FILEPATH` parameter. If you want to override only a few fields, you
can also use `--set` option.

You should see
```
NAME: awi
LAST DEPLOYED: Thu Apr 18 12:05:09 2024
NAMESPACE: awi-system
STATUS: deployed
REVISION: 1
TEST SUITE: None
```

It will spawn two pods:

```
NAMESPACE     NAME                                                   READY   STATUS    RESTARTS      AGE
awi-system    awi-grpc-catalyst-sdwan-5c99b764b5-w8fjx               1/1     Running   0             28s
awi-system    awi-k8s-operator-controller-manager-7645c5b477-8zcl2   2/2     Running   0             28s
```

The first one is the actual AWI Catalyst SDWAN controller and the second
one is k8s operator acting as a proxy between k8s environment and the
mentioned controller. Both should be in fully READY state.

They may take a while to warm-up as they need to pull images.

Apart from deployments, the k8s operator will add custom CRDs in the cluster

```
> kubectl get crds -A
---
NAME                                             CREATED AT
instances.awi.app-net-interface.io                          2024-02-09T04:09:55Z
internetworkdomainappconnections.awi.app-net-interface.io   2024-02-09T04:09:55Z
internetworkdomainconnections.awi.app-net-interface.io      2024-02-09T04:09:55Z
networkdomains.awi.app-net-interface.io                     2024-02-09T04:09:55Z
sites.awi.app-net-interface.io                              2024-02-09T04:09:55Z
subnets.awi.app-net-interface.io                            2024-02-09T04:09:55Z
vpcs.awi.app-net-interface.io                               2024-02-09T04:09:55Z
vpns.awi.app-net-interface.io                               2024-02-09T04:09:55Z
```

That's it :)

### Test connection

To create sample connection apply the following file
```
kubectl apply -f samples/awi/v1alpha/internetworkdomainconnection/vpc-to-vpc.yaml
```

Of course, the file needs to be modified to match your desired VPCs.

You can now see created connection:
```
> kubectl get internetworkdomainconnections
---
NAME                              AGE
my-internetworkdomainconnection   42s
```

This connection will most likely exist even if the actual connection was not created.
Check your AWI GRPC Catalyst SDWAN logs to see how the creation went.

To destroy the connection, simply remove the CR:
```
kubectl delete internetworkdomainconnections my-internetworkdomainconnection
```

After your minikube cluster is no longer needed, you can remove it by running:
```
minikube delete
```

## Building/updating chart

Creating a new `catalyst chart` simply requires updating templates, `Chart.yaml`
and `values.yaml` according to your needs, however `operator chart` involves a
few different steps.

### Operator Chart

The `operator chart` is built automatically using `helmify` tool. The helmify
tool allows automatic chart creation based on internal structures provided by
the kubebuilder.

If the kube-awi repository did not change, there should be no need in rebuilding
operator chart.

If the operator chart needs to be refreshed:

1. Ensure kube-awi is recent

1. Make sure kube-awi is kustomized accodringly to the project needs.

    The project's production kustomize configuration should be commited so this step
    is mostly for building custom charts.

1. Generate chart

    ```
    make build-operator-graph
    ```

    It will build a new chart based on `config/` directory, place it in
    `chart/awi-catalyst-sdwan-k8s-operator` and will update and build `chart/`
    dependencies.

1. Update `catalyst chart` Chart.yaml with a new dependency version of your operator chart
    and `values.yaml` to match newer versions

1. Build new `kube-awi` image and release it.

## Contributing

Thank you for interest in contributing! Please refer to our
[contributing guide](CONTRIBUTING.md).

## License

kube-awi is released under the Apache 2.0 license. See
[LICENSE](./LICENSE).

kube-awi is also made possible thanks to
[third party open source projects](NOTICE).
