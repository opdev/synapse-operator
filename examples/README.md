# Examples

Here is a collection of examples of `Synapse` resources that illustrate the
main features of the Synapse operator.

## Pre-requisites

The Synapse operator should be running in your cluster.

## Your first Synapse deployment

A minimal manifest example of a `Synapse` resource can be found under the
`01-my-first-synapse-deployment` folder. To deploy Synapse, simply run:

```shell
$ kubectl apply -f examples/01-my-first-synapse-deployment/synapse.yaml
synapse.synapse.opdev.io/my-first-synapse-deployment created
```

In this configuration, Synapse uses its internal SQLite database. A basic
`homeserver.yaml` is created, its only customization being the `server_name`
and the `report_stats` values (set to `example.com` and `true` respectively in
the example manifest).

The Synapse operator reconciles the `Synapse` resource and creates the
following Kubernetes objects:

- a `Deployment`: is responsible of the `Pod` running Synapse.
- a `Service`: allows the Synapse server to be reachable on a stable IP address
  on port 8008.
- a `ConfigMap`: contains a basic `homeserver.yaml` for the Synapse
  configuration.
- a `ServiceAccount`: runs the Synapse `Pod` with the correct permissions.

You can observe that those resources are successfully created with:

```shell
$ kubectl get pods,replicaset,deployment,service,configmap,serviceaccount
NAME                                                 READY   STATUS    RESTARTS   AGE
pod/my-first-synapse-deployment-5d859d7b57-856r7     1/1     Running   0          73s

NAME                                                       DESIRED   CURRENT   READY   AGE
replicaset.apps/my-first-synapse-deployment-5d859d7b57     1         1         1       73s

NAME                                            READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/my-first-synapse-deployment     1/1     1            1           73s

NAME                                    TYPE        CLUSTER-IP    EXTERNAL-IP   PORT(S)    AGE
service/my-first-synapse-deployment     ClusterIP   10.217.4.72   <none>        8008/TCP   73s

NAME                                      DATA   AGE
configmap/kube-root-ca.crt                1      6d18h
configmap/openshift-service-ca.crt        1      6d18h
configmap/my-first-synapse-deployment     1      73s

NAME                                           SECRETS   AGE
serviceaccount/builder                         2         6d18h
serviceaccount/default                         2         6d18h
serviceaccount/deployer                        2         6d18h
serviceaccount/my-first-synapse-deployment     2         73s
```

You can also observe the state of the `Synapse` resource with:

```shell
$ kubectl get synapse my-first-synapse-deployment -o yaml
apiVersion: synapse.opdev.io/v1alpha1
kind: Synapse
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"synapse.opdev.io/v1alpha1","kind":"Synapse","metadata":{"annotations":{},"name":"my-first-synapse-deployment","namespace":"test-synapse-operator"},"spec":{"homeserver":{"values":{"reportStats":true,"serverName":"example.com"}}}}
  creationTimestamp: "2022-03-25T11:23:31Z"
  generation: 1
  name: my-first-synapse-deployment
  namespace: test-synapse-operator
  resourceVersion: "1808066"
  uid: 862c4adc-3e2e-452b-a355-545010722dca
spec:
  createNewPostgreSQL: false
  homeserver:
    values:
      reportStats: true
      serverName: example.com
status:
  bridgesConfiguration:
    heisenbridge: {}
  databaseConnectionInfo: {}
  homeserverConfigMapName: my-first-synapse-deployment
  homeserverConfiguration:
    reportStats: true
    serverName: example.com
  ip: 10.217.4.72
  state: RUNNING
```

If everything looks fine, you can access the Synapse server on your Service
Cluster IP and on port 8008.

To delete the Synapse resources:

```shell
$ kubectl delete synapse my-first-synapse-deployment
synapse.synapse.opdev.io "my-first-synapse-deployment" deleted
```

## Using an existing `homeserver.yaml`

If you already possess a custom `homeserver.yaml` or if you are limited by the
customisation options available to you via the `Synapse` resource, you can
create your own `ConfigMap`, containing your custom `homeserver.yaml`, and
configure the `Synapse` resource to use it.

An example is available under the `02-using-existing-configmap` directory. To run
it:

```shell
$ kubectl create configmap my-custom-homeserver --from-file=02-using-existing-configmap/homeserver.yaml 
configmap/my-custom-homeserver created
$ kubectl apply -f examples/02-using-existing-configmap/synapse.yaml
synapse.synapse.opdev.io/using-existing-configmap created
```

## Deploying a PostgreSQL instance for Synapse

> *Pre-requisite:* The deployment of a PostgreSQL instance relies on the
>  [postgres-operator](https://github.com/CrunchyData/postgres-operator). Make
>  sure it is running on your cluster if you want to deploy a PostgreSQL
>  instance for Synapse.

The `03-deploying-postgresql` directory provides an example of a `Synapse`
resource requesting for a new PostgreSQL instance to be deployed:


```shell
$ kubectl apply -f examples/03-deploying-postgresql/synapse.yaml
synapse.synapse.opdev.io/synapse-with-postgresql created

$ kubectl get pods,replicaset,deployment,statefulset,jobs,pods,service,configmap
NAME                                               READY   STATUS             RESTARTS   AGE
pod/synapse-with-postgresql-5d859d7b57-wrdl9       1/1     Running            0          77s
pod/synapse-with-postgresql-backup-n29s--1-7c7zh   0/1     Completed          0          87s
pod/synapse-with-postgresql-instance1-b7tj-0       3/3     Running            0          98s
pod/synapse-with-postgresql-repo-host-0            1/1     Running            0          98s

NAME                                                 DESIRED   CURRENT   READY   AGE
replicaset.apps/synapse-with-postgresql-5d859d7b57   1         1         1       78s

NAME                                      READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/synapse-with-postgresql   1/1     1            1           78s

NAME                                                      READY   AGE
statefulset.apps/synapse-with-postgresql-instance1-b7tj   1/1     98s
statefulset.apps/synapse-with-postgresql-repo-host        1/1     98s

NAME                                            COMPLETIONS   DURATION   AGE
job.batch/synapse-with-postgresql-backup-n29s   1/1           27s        87s

NAME                                        TYPE        CLUSTER-IP     EXTERNAL-IP   PORT(S)    AGE
service/synapse-with-postgresql             ClusterIP   10.217.4.222   <none>        8008/TCP   78s
service/synapse-with-postgresql-ha          ClusterIP   10.217.5.159   <none>        5432/TCP   99s
service/synapse-with-postgresql-ha-config   ClusterIP   None           <none>        <none>     98s
service/synapse-with-postgresql-pods        ClusterIP   None           <none>        <none>     99s
service/synapse-with-postgresql-primary     ClusterIP   None           <none>        5432/TCP   99s
service/synapse-with-postgresql-replicas    ClusterIP   10.217.5.24    <none>        5432/TCP   99s

NAME                                                      DATA   AGE
configmap/kube-root-ca.crt                                1      6d19h
configmap/openshift-service-ca.crt                        1      6d19h
configmap/synapse-with-postgresql                         1      99s
configmap/synapse-with-postgresql-config                  1      99s
configmap/synapse-with-postgresql-instance1-b7tj-config   1      98s
configmap/synapse-with-postgresql-pgbackrest-config       3      98s
configmap/synapse-with-postgresql-ssh-config              2      98s
```

## Deploying a bridge

For now, only the deployment of the
[Heisenbridge](https://github.com/hifi/heisenbridge) is supported.

### Using the default Heisenbridge configuration

An example of a `Synapse` resource using the default Heisenbridge configuration
is available under the `04-deploying-heisenbridge/A-default-configuration`
directory:

```shell
$ kubectl apply -f examples/04-deploying-heisenbridge/A-default-configuration/synapse.yaml
synapse.synapse.opdev.io/synapse-with-heisenbridge created
$ kubectl get pods,replicaset,deployment,pods,service,configmap
NAME                                                          READY   STATUS    RESTARTS      AGE
pod/synapse-with-heisenbridge-5474cc57db-k8vlb                1/1     Running   0             21s
pod/synapse-with-heisenbridge-heisenbridge-865d759bff-w4wg4   1/1     Running   1 (15s ago)   22s

NAME                                                                DESIRED   CURRENT   READY   AGE
replicaset.apps/synapse-with-heisenbridge-5474cc57db                1         1         1       22s
replicaset.apps/synapse-with-heisenbridge-heisenbridge-865d759bff   1         1         1       22s

NAME                                                     READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/synapse-with-heisenbridge                1/1     1            1           22s
deployment.apps/synapse-with-heisenbridge-heisenbridge   1/1     1            1           22s

NAME                                             TYPE        CLUSTER-IP     EXTERNAL-IP   PORT(S)    AGE
service/synapse-with-heisenbridge                ClusterIP   10.217.5.178   <none>        8008/TCP   22s
service/synapse-with-heisenbridge-heisenbridge   ClusterIP   10.217.5.121   <none>        9898/TCP   22s

NAME                                               DATA   AGE
configmap/kube-root-ca.crt                         1      6d19h
configmap/openshift-service-ca.crt                 1      6d19h
configmap/synapse-with-heisenbridge                1      22s
configmap/synapse-with-heisenbridge-heisenbridge   1      22s
```

### Using an existing `heisenbridge.yaml` configuration file

If the default `heisenbridge.yaml` doesn't answer your needs, you can use a
custom configuration file. You first have to add your custom
`heisenbridge.yaml` to a `ConfigMap` and configure the `Heisenbridge.ConfigMap`
section of the `Synapse` resource to reference the `ConfigMap`, as illustrated
in the `04-deploying-heisenbridge/B-using-existing-configmap` directory:

```shell
$ kubectl create configmap my-custom-heisenbridge --from-file=04-deploying-heisenbridge/B-using-existing-configmap/heisenbridge.yaml 
configmap/my-custom-heisenbridge created
$ kubectl apply -f examples/04-deploying-heisenbridge/B-using-existing-configmap/synapse.yaml 
synapse.synapse.opdev.io/synapse-with-heisenbridge created
$ kubectl get pods,replicaset,deployment,pods,service,configmap
NAME                                                          READY   STATUS    RESTARTS      AGE
pod/synapse-with-heisenbridge-6546c96f48-f5hv8                1/1     Running   0             78s
pod/synapse-with-heisenbridge-heisenbridge-7b5df49cfd-jqpkk   1/1     Running   1 (70s ago)   78s

NAME                                                                DESIRED   CURRENT   READY   AGE
replicaset.apps/synapse-with-heisenbridge-6546c96f48                1         1         1       78s
replicaset.apps/synapse-with-heisenbridge-heisenbridge-7b5df49cfd   1         1         1       78s

NAME                                                     READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/synapse-with-heisenbridge                1/1     1            1           78s
deployment.apps/synapse-with-heisenbridge-heisenbridge   1/1     1            1           78s

NAME                                             TYPE        CLUSTER-IP     EXTERNAL-IP   PORT(S)    AGE
service/synapse-with-heisenbridge                ClusterIP   10.217.4.127   <none>        8008/TCP   78s
service/synapse-with-heisenbridge-heisenbridge   ClusterIP   10.217.4.124   <none>        9898/TCP   78s

NAME                                  DATA   AGE
configmap/kube-root-ca.crt            1      32d
configmap/my-custom-heisenbridge      1      92s
configmap/openshift-service-ca.crt    1      32d
configmap/synapse-with-heisenbridge   1      78s
```