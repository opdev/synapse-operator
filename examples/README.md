# Examples

Here is a collection of example resources that illustrates the main features of
the Synapse operator.

## Pre-requisites

The Synapse operator should be running in your cluster. See the
[deployment intructions](https://github.com/opdev/synapse-operator#deploying-the-synapse-operator).

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
      {"apiVersion":"synapse.opdev.io/v1alpha1","kind":"Synapse","metadata":{"annotations":{},"name":"your-first-synapse-deployment","namespace":"default"},"spec":{"homeserver":{"values":{"reportStats":true,"serverName":"example.com"}}}}
  creationTimestamp: "2022-11-29T14:27:21Z"
  generation: 1
  name: your-first-synapse-deployment
  namespace: default
  resourceVersion: "98298"
  uid: 086652b1-9ba9-40ed-9223-1336db12681d
spec:
  createNewPostgreSQL: false
  homeserver:
    values:
      reportStats: true
      serverName: example.com
status:
  homeserverConfiguration:
    reportStats: true
    serverName: example.com
  needsReconcile: false
  state: RUNNING
```

If everything looks fine, you can access the Synapse server on your Service
Cluster IP and on port 8008.

To delete the Synapse resources:

```shell
$ kubectl delete synapse my-first-synapse-deployment
synapse.synapse.opdevio "my-first-synapse-deployment" deleted
```

## Using an existing `homeserver.yaml`

If you already possess a custom `homeserver.yaml` or if you are limited by the
customisation options available to you via the `Synapse` resource, you can
create your own `ConfigMap`, containing your custom `homeserver.yaml`, and
configure the `Synapse` resource to use it.

An example is available under the `02-using-existing-configmap` directory. To run
it:

```shell
$ kubectl create configmap my-custom-homeserver --from-file=examples/02-using-existing-configmap/homeserver.yaml 
configmap/my-custom-homeserver created
$ kubectl apply -f examples/02-using-existing-configmap/synapse.yaml
synapse.synapse.opdev.io/using-existing-configmap created
```

To delete the resources:
```shell
$ kubectl delete synapse using-existing-configmap
synapse.synapse..opdevio "using-existing-configmap" deleted
$ oc delete configmap my-custom-homeserver
configmap "my-custom-homeserver" deleted
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

The synapse Operator supports the deployment of:
* [Heisenbridge](https://github.com/hifi/heisenbridge): a bouncer-style IRC
  bridge.
* [mautrix-signal](https://github.com/mautrix/signal): a double-pupetting
  bridge for Signal.

The bridges don't have any dependency to one another. You can choose to deploy
one, several, or none of them.

Bridges are deployed by creating a coresponding Kubernetes resource:
* The `Heisenbridge` resource is used to deploy and manage Heisenbridge.
* The `MautrixSignal` resource is used to deploy and manage the mautrix-signal bridge.

For both bridges, you can choose between using the default configuration file
and providing your custom configuration file.

### Using the default configuration file

In this case, the Synapse operator provides default configuration values.

An example of a `Heisenbridge` resource using the default Heisenbridge
configuration is available under the
`04-deploying-heisenbridge/A-default-configuration` directory:

First create the Synapse resource:

```shell
$ kubectl apply -f examples/04-deploying-heisenbridge/A-default-configuration/synapse.yaml
synapse.synapse.opdev.io/synapse-with-heisenbridge created
```

Then create the Heisenbridge resource:

```shell
$ kubectl apply -f examples/04-deploying-heisenbridge/A-default-configuration/heisenbridge.yaml
heisenbridge.synapse.opdev.io/heisenbridge-for-synapse created
$ $ kubectl get pods,replicaset,deployment,pods,service,configmap
NAME                                             READY   STATUS    RESTARTS      AGE
pod/heisenbridge-for-synapse-7b47fc66f4-t8vz7    1/1     Running   2 (48s ago)   54s
pod/synapse-with-heisenbridge-685c97478f-zltsc   1/1     Running   0             54s

NAME                                                   DESIRED   CURRENT   READY   AGE
replicaset.apps/heisenbridge-for-synapse-7b47fc66f4    1         1         1       54s
replicaset.apps/synapse-with-heisenbridge-6445b859d7   0         0         0       85s
replicaset.apps/synapse-with-heisenbridge-685c97478f   1         1         1       54s

NAME                                        READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/heisenbridge-for-synapse    1/1     1            1           54s
deployment.apps/synapse-with-heisenbridge   1/1     1            1           85s

NAME                                TYPE           CLUSTER-IP     EXTERNAL-IP                            PORT(S)    AGE
service/heisenbridge-for-synapse    ClusterIP      10.217.5.7     <none>                                 9898/TCP   54s
service/kubernetes                  ClusterIP      10.217.4.1     <none>                                 443/TCP    57d
service/openshift                   ExternalName   <none>         kubernetes.default.svc.cluster.local   <none>     57d
service/synapse-with-heisenbridge   ClusterIP      10.217.5.127   <none>                                 8008/TCP   85s

NAME                                  DATA   AGE
configmap/heisenbridge-for-synapse    1      54s
configmap/kube-root-ca.crt            1      57d
configmap/openshift-service-ca.crt    1      57d
configmap/synapse-with-heisenbridge   1      86s
```

A similar example for mautrix-signal is available under the
`05-deploying-mautrixsignal/A-default-configuration` directory.

### Providing a custom configuration file

If the default configuration file doesn't answer your needs, you can use a
custom configuration file. You first have to add your custom config file
(for instance `heisenbridge.yaml`) to a `ConfigMap` and configure the
corresponding section of the Bridge resource (for instance
`Heisenbridge.Spec.ConfigMap`) to reference the `ConfigMap`.

For heisenbridge, this is illustrated in the
`04-deploying-heisenbridge/B-using-existing-configmap` directory:

```shell
$ kubectl apply -f examples/04-deploying-heisenbridge/B-using-existing-configmap/synapse.yaml
synapse.synapse.opdev.io/synapse-with-heisenbridge created
$ kubectl create configmap my-custom-heisenbridge --from-file=heisenbridge.yaml=examples/04-deploying-heisenbridge/B-using-existing-configmap/heisenbridge_config.yaml 
configmap/my-custom-heisenbridge created
$ oc apply -f examples/04-deploying-heisenbridge/B-using-existing-configmap/heisenbridge.yaml
heisenbridge.synapse.opdev.io/heisenbridge-for-synapse created

$ kubectl get pods,replicaset,deployment,pods,service,configmap
NAME                                             READY   STATUS    RESTARTS      AGE
pod/heisenbridge-for-synapse-7b47fc66f4-8jq6z    1/1     Running   2 (52s ago)   57s
pod/synapse-with-heisenbridge-685c97478f-bhjsg   1/1     Running   0             57s

NAME                                                   DESIRED   CURRENT   READY   AGE
replicaset.apps/heisenbridge-for-synapse-7b47fc66f4    1         1         1       57s
replicaset.apps/synapse-with-heisenbridge-6445b859d7   0         0         0       81s
replicaset.apps/synapse-with-heisenbridge-685c97478f   1         1         1       57s

NAME                                        READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/heisenbridge-for-synapse    1/1     1            1           57s
deployment.apps/synapse-with-heisenbridge   1/1     1            1           81s

NAME                                TYPE           CLUSTER-IP     EXTERNAL-IP                            PORT(S)    AGE
service/heisenbridge-for-synapse    ClusterIP      10.217.4.208   <none>                                 9898/TCP   57s
service/kubernetes                  ClusterIP      10.217.4.1     <none>                                 443/TCP    57d
service/openshift                   ExternalName   <none>         kubernetes.default.svc.cluster.local   <none>     57d
service/synapse-with-heisenbridge   ClusterIP      10.217.5.252   <none>                                 8008/TCP   81s

NAME                                  DATA   AGE
configmap/heisenbridge-for-synapse    1      57s
configmap/kube-root-ca.crt            1      57d
configmap/my-custom-heisenbridge      1      67s
configmap/openshift-service-ca.crt    1      57d
configmap/synapse-with-heisenbridge   1      81s
```

A similar example for mautrix-signal is available under the
`05-deploying-mautrixsignal/B-using-existing-configmap` directory.