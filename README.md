# A Kubernetes Operator for Synapse

The Synapse
[operator](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)
offers a convenient way to deploy and manage a
[Synapse](https://github.com/matrix-org/synapse/) server. It was built with
[operator-sdk](https://sdk.operatorframework.io/).

## Deploying the Synapse Operator

Each of the following sections present one way to deploy the Synapse operator.

### Deploy the controller using the existing manifest file

The easiest way to deploy the Synapse operator is to use the manifest file
present in the `install` directory:

```shell
$ kubectl apply -f install/synapse-operator.yaml
namespace/synapse-operator-system created
customresourcedefinition.apiextensions.k8s.io/synapses.synapse.opdev.io configured
serviceaccount/synapse-operator-controller-manager created
role.rbac.authorization.k8s.io/synapse-operator-leader-election-role created
clusterrole.rbac.authorization.k8s.io/synapse-operator-manager-role created
clusterrole.rbac.authorization.k8s.io/synapse-operator-metrics-reader created
clusterrole.rbac.authorization.k8s.io/synapse-operator-proxy-role created
rolebinding.rbac.authorization.k8s.io/synapse-operator-leader-election-rolebinding created
clusterrolebinding.rbac.authorization.k8s.io/synapse-operator-manager-rolebinding created
clusterrolebinding.rbac.authorization.k8s.io/synapse-operator-proxy-rolebinding created
configmap/synapse-operator-manager-config created
service/synapse-operator-controller-manager-metrics-service created
deployment.apps/synapse-operator-controller-manager created
```

### Run the controller locally with `make run`

Install the `Synapse` CRD with:

```shell
$ make install
/home/mgoerens/dev/github.com/opdev/synapse-operator/bin/controller-gen "crd:trivialVersions=true,preserveUnknownFields=false" rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
/home/mgoerens/dev/github.com/opdev/synapse-operator/bin/kustomize build config/crd | kubectl apply -f -
customresourcedefinition.apiextensions.k8s.io/synapses.synapse.opdev.io configured
```

Run the controller locally with:

```shell
$ make run
/home/mgoerens/dev/github.com/opdev/synapse-operator/bin/controller-gen "crd:trivialVersions=true,preserveUnknownFields=false" rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
/home/mgoerens/dev/github.com/opdev/synapse-operator/bin/controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./..."
go fmt ./...
go vet ./...
go run ./main.go
I0329 13:14:32.483664   26252 request.go:665] Waited for 1.039947837s due to client-side throttling, not priority and fairness, request: GET:https://api.crc.testing:6443/apis/operators.coreos.com/v1?timeout=32s
1.6485524737848275e+09	INFO	controller-runtime.metrics	Metrics server is starting to listen	{"addr": ":8080"}
1.648552473785254e+09	INFO	setup	starting manager
1.6485524737853944e+09	INFO	Starting server	{"kind": "health probe", "addr": "[::]:8081"}
1.6485524737853956e+09	INFO	Starting server	{"path": "/metrics", "kind": "metrics", "addr": "[::]:8080"}
1.6485524737854843e+09	INFO	controller.synapse	Starting EventSource	{"reconciler group": "synapse.opdev.io", "reconciler kind": "Synapse", "source": "kind source: *v1alpha1.Synapse"}
1.648552473785519e+09	INFO	controller.synapse	Starting EventSource	{"reconciler group": "synapse.opdev.io", "reconciler kind": "Synapse", "source": "kind source: *v1.Service"}
1.6485524737855291e+09	INFO	controller.synapse	Starting EventSource	{"reconciler group": "synapse.opdev.io", "reconciler kind": "Synapse", "source": "kind source: *v1.Deployment"}
1.6485524737855399e+09	INFO	controller.synapse	Starting EventSource	{"reconciler group": "synapse.opdev.io", "reconciler kind": "Synapse", "source": "kind source: *v1.PersistentVolumeClaim"}
1.6485524737855482e+09	INFO	controller.synapse	Starting Controller	{"reconciler group": "synapse.opdev.io", "reconciler kind": "Synapse"}
1.6485524738865452e+09	INFO	controller.synapse	Starting workers	{"reconciler group": "synapse.opdev.io", "reconciler kind": "Synapse", "worker count": 1}

```

This runs the controller until you hit `Ctrl` + `C`.

To uninstall the `Synapse `CRD:

```shell
$ make uninstall
/home/mgoerens/dev/github.com/opdev/synapse-operator/bin/controller-gen "crd:trivialVersions=true,preserveUnknownFields=false" rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
/home/mgoerens/dev/github.com/opdev/synapse-operator/bin/kustomize build config/crd | kubectl delete -f -
customresourcedefinition.apiextensions.k8s.io "synapses.synapse.opdev.io" deleted
```

### Deploy the controller in the Kubernetes cluster with `make deploy`

Deploy the controller with:

```shell
$ make deploy
/home/mgoerens/dev/github.com/opdev/synapse-operator/bin/controller-gen "crd:trivialVersions=true,preserveUnknownFields=false" rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
cd config/manager && /home/mgoerens/dev/github.com/opdev/synapse-operator/bin/kustomize edit set image controller=controller:latest
/home/mgoerens/dev/github.com/opdev/synapse-operator/bin/kustomize build config/default | kubectl apply -f -
namespace/synapse-operator-system created
customresourcedefinition.apiextensions.k8s.io/synapses.synapse.opdev.io configured
serviceaccount/synapse-operator-controller-manager created
role.rbac.authorization.k8s.io/synapse-operator-leader-election-role created
clusterrole.rbac.authorization.k8s.io/synapse-operator-manager-role created
clusterrole.rbac.authorization.k8s.io/synapse-operator-metrics-reader created
clusterrole.rbac.authorization.k8s.io/synapse-operator-proxy-role created
rolebinding.rbac.authorization.k8s.io/synapse-operator-leader-election-rolebinding created
clusterrolebinding.rbac.authorization.k8s.io/synapse-operator-manager-rolebinding created
clusterrolebinding.rbac.authorization.k8s.io/synapse-operator-proxy-rolebinding created
configmap/synapse-operator-manager-config created
service/synapse-operator-controller-manager-metrics-service created
deployment.apps/synapse-operator-controller-manager created
```

This creates a dedicated namespace `synapse-operator-system` and all required
resources (including the CRD) for the controller to run.

To cleanup all resources:

```shell
$ make undeploy
/home/mgoerens/dev/github.com/opdev/synapse-operator/bin/kustomize build config/default | kubectl delete -f -
namespace "synapse-operator-system" deleted
customresourcedefinition.apiextensions.k8s.io "synapses.synapse.opdev.io" deleted
serviceaccount "synapse-operator-controller-manager" deleted
role.rbac.authorization.k8s.io "synapse-operator-leader-election-role" deleted
clusterrole.rbac.authorization.k8s.io "synapse-operator-manager-role" deleted
clusterrole.rbac.authorization.k8s.io "synapse-operator-metrics-reader" deleted
clusterrole.rbac.authorization.k8s.io "synapse-operator-proxy-role" deleted
rolebinding.rbac.authorization.k8s.io "synapse-operator-leader-election-rolebinding" deleted
clusterrolebinding.rbac.authorization.k8s.io "synapse-operator-manager-rolebinding" deleted
clusterrolebinding.rbac.authorization.k8s.io "synapse-operator-proxy-rolebinding" deleted
configmap "synapse-operator-manager-config" deleted
service "synapse-operator-controller-manager-metrics-service" deleted
deployment.apps "synapse-operator-controller-manager" deleted
```

## Deploying a Synapse instance

A set of example how to use the Synapse operator to deploy a Synapse server is
provided under the
[examples](https://github.com/opdev/synapse-operator/tree/master/examples)
directory.

## Notes and pre-requisites

- The [postgres-operator](https://github.com/CrunchyData/postgres-operator)
  needs to be installed. Specifically the `PostgresCluster` CRD is required.
  This is only required if you intend to deploy a PostgreSQL instance alongside
  Synapse.
- Tested on OpenShift 4.9.0

# Related links

- [Matrix project homepage](https://matrix.org/)
- [Synape Repository](https://github.com/matrix-org/synapse/)
- [postgres-operator](https://github.com/CrunchyData/postgres-operator)
- [Heisenbridge](https://github.com/hifi/heisenbridge)
