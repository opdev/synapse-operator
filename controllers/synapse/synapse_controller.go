/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package synapse

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	pgov1beta1 "github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
	synapsev1alpha1 "github.com/opdev/synapse-operator/apis/synapse/v1alpha1"
)

// SynapseReconciler reconciles a Synapse object
type SynapseReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

type HomeserverPgsqlDatabase struct {
	Name     string `yaml:"name"`
	TxnLimit int64  `yaml:"txn_limit"`
	Args     struct {
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		Database string `yaml:"database"`
		Host     string `yaml:"host"`
		Port     int64  `yaml:"port"`
		CpMin    int64  `yaml:"cp_min"`
		CpMax    int64  `yaml:"cp_max"`
	}
}

//+kubebuilder:rbac:groups=synapse.opdev.io,resources=synapses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=synapse.opdev.io,resources=synapses/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=synapse.opdev.io,resources=synapses/finalizers,verbs=update

func (r *SynapseReconciler) GetHeisenbridgeResourceName(synapse synapsev1alpha1.Synapse) string {
	return strings.Join([]string{synapse.Name, "heisenbridge"}, "-")
}

func (r *SynapseReconciler) GetSignaldResourceName(synapse synapsev1alpha1.Synapse) string {
	return strings.Join([]string{synapse.Name, "signald"}, "-")
}

func (r *SynapseReconciler) GetMautrixSignalResourceName(synapse synapsev1alpha1.Synapse) string {
	return strings.Join([]string{synapse.Name, "mautrixsignal"}, "-")
}

func (r *SynapseReconciler) GetSynapseServiceFQDN(synapse synapsev1alpha1.Synapse) string {
	return strings.Join([]string{synapse.Name, synapse.Namespace, "svc", "cluster", "local"}, ".")
}

func (r *SynapseReconciler) GetHeisenbridgeServiceFQDN(synapse synapsev1alpha1.Synapse) string {
	return strings.Join([]string{r.GetHeisenbridgeResourceName(synapse), synapse.Namespace, "svc", "cluster", "local"}, ".")
}

func (r *SynapseReconciler) GetMautrixSignalServiceFQDN(synapse synapsev1alpha1.Synapse) string {
	return strings.Join([]string{r.GetMautrixSignalResourceName(synapse), synapse.Namespace, "svc", "cluster", "local"}, ".")
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *SynapseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)

	var synapse synapsev1alpha1.Synapse // The Synapse object being reconciled

	// Load the Synapse by name
	if err := r.Get(ctx, req.NamespacedName, &synapse); err != nil {
		if k8serrors.IsNotFound(err) {
			// we'll ignore not-found errors, since they can't be fixed by an immediate
			// requeue (we'll need to wait for a new notification), and we can get them
			// on deleted requests.
			log.Error(
				err,
				"Cannot find Synapse - has it been deleted ?",
				"Synapse Name", synapse.Name,
				"Synapse Namespace", synapse.Namespace,
			)
			return ctrl.Result{}, nil
		}
		log.Error(
			err,
			"Error fetching Synapse",
			"Synapse Name", synapse.Name,
			"Synapse Namespace", synapse.Namespace,
		)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	objectMetaForSynapse := setObjectMeta(synapse.Name, synapse.Namespace, map[string]string{})

	// The ConfigMap for Synapse, containing the homeserver.yaml config file.
	// It's either a copy of a user-provided ConfigMap, if defined in
	// Spec.Homeserver.ConfigMap, or a new ConfigMap containing a default
	// homeserver.yaml.
	var createdConfigMap corev1.ConfigMap

	// Synapse should either have a Spec.Homeserver.ConfigMap or Spec.Homeserver.Values
	if synapse.Spec.Homeserver.ConfigMap != nil {
		// If the user provided a ConfigMap for the Homeserver config file:
		// * We ensure that it exists and is a valid yaml file
		// * We populate the Status.HomeserverConfiguration with the values defined in the input ConfigMap
		// * We create a copy of the user-provided ConfigMap in createdConfigMap.

		var inputConfigMap corev1.ConfigMap // the user-provided ConfigMap. It should contain a valid homeserver.yaml
		// Get and validate the inputConfigMap
		ConfigMapName := synapse.Spec.Homeserver.ConfigMap.Name
		ConfigMapNamespace := r.getConfigMapNamespace(synapse, synapse.Spec.Homeserver.ConfigMap.Namespace)
		if err := r.Get(
			ctx,
			types.NamespacedName{Name: ConfigMapName, Namespace: ConfigMapNamespace},
			&inputConfigMap,
		); err != nil {
			reason := "ConfigMap " + ConfigMapName + " does not exist in namespace " + ConfigMapNamespace
			if err := r.setFailedState(ctx, &synapse, reason); err != nil {
				log.Error(err, "Error updating Synapse State")
			}

			log.Error(
				err,
				"Failed to get ConfigMap",
				"ConfigMap.Namespace",
				ConfigMapNamespace,
				"ConfigMap.Name",
				ConfigMapName,
			)
			return ctrl.Result{RequeueAfter: time.Duration(30)}, err
		}

		if err := r.ParseHomeserverConfigMap(ctx, &synapse, inputConfigMap); err != nil {
			return ctrl.Result{RequeueAfter: time.Duration(30)}, err
		}

		// Create a copy of the inputConfigMap defined in Spec.Homeserver.ConfigMap
		// Here we use the configMapForSynapseCopy function as createResourceFunc
		if err := r.reconcileResource(
			ctx,
			r.configMapForSynapseCopy,
			&synapse,
			&createdConfigMap,
			objectMetaForSynapse,
		); err != nil {
			return ctrl.Result{}, nil
		}
	} else {
		// If the user hasn't provided a ConfigMap with a custom
		// homeserver.yaml, we create a new ConfigMap. The default
		// homeserver.yaml is configured with values defined in
		// Spec.Homeserver.Values
		synapse.Status.HomeserverConfiguration.ServerName = synapse.Spec.Homeserver.Values.ServerName
		synapse.Status.HomeserverConfiguration.ReportStats = synapse.Spec.Homeserver.Values.ReportStats

		// Create a new ConfigMap for Synapse
		// Here we use the configMapForSynapse function as createResourceFunc
		if err := r.reconcileResource(
			ctx,
			r.configMapForSynapse,
			&synapse,
			&createdConfigMap,
			objectMetaForSynapse,
		); err != nil {
			return ctrl.Result{}, nil
		}
	}

	if err := r.updateSynapseStatus(ctx, &synapse); err != nil {
		log.Error(err, "Error updating Synapse Status")
		return ctrl.Result{}, err
	}

	if synapse.Spec.CreateNewPostgreSQL {
		if !r.isPostgresOperatorInstalled(ctx) {
			reason := "Cannot create PostgreSQL instance for synapse. Postgres-operator is not installed."
			if err := r.setFailedState(ctx, &synapse, reason); err != nil {
				log.Error(err, "Error updating Synapse State")
			}

			err := errors.New("cannot create PostgreSQL instance for synapse. Potsres-operator is not installed")
			log.Error(err, "Cannot create PostgreSQL instance for synapse. Potsres-operator is not installed.")
			return ctrl.Result{}, nil
		}
		if result, err := r.createPostgresClusterForSynapse(ctx, synapse, createdConfigMap); err != nil {
			return result, err
		}
	}

	// We first need to create the Synapse Service as its IP address is potentially
	// needed by the Bridges
	createdService := &corev1.Service{}
	if err := r.reconcileResource(
		ctx,
		r.serviceForSynapse,
		&synapse,
		createdService,
		objectMetaForSynapse,
	); err != nil {
		return ctrl.Result{}, err
	}

	if synapse.Spec.Bridges.Heisenbridge.Enabled {
		log.Info("Heisenbridge is enabled - deploying Heisenbridge")
		// Heisenbridge is composed of a ConfigMap, a Service and a Deployment.
		// Resources associated to the Heisenbridge are append with "-heisenbridge"
		createdHeisenbridgeService := &corev1.Service{}
		objectMetaHeisenbridge := setObjectMeta(r.GetHeisenbridgeResourceName(synapse), synapse.Namespace, map[string]string{})

		// First create the service as we need its IP address for the
		// heisenbridge.yaml configuration file
		if err := r.reconcileResource(
			ctx,
			r.serviceForHeisenbridge,
			&synapse,
			createdHeisenbridgeService,
			objectMetaHeisenbridge,
		); err != nil {
			return ctrl.Result{}, err
		}

		// The ConfigMap for Heisenbridge, containing the heisenbridge.yaml
		// config file. It's either a copy of a user-provided ConfigMap, if
		// defined in Spec.Bridges.Heisenbridge.ConfigMap, or a new ConfigMap
		// containing a default heisenbridge.yaml.
		createdHeisenbridgeConfigMap := &corev1.ConfigMap{}

		// The user may specify a ConfigMap, containing the heisenbridge.yaml
		// config file, under Spec.Bridges.Heisenbridge.ConfigMap
		inputConfigMapName := synapse.Spec.Bridges.Heisenbridge.ConfigMap.Name
		setConfigMapNamespace := synapse.Spec.Bridges.Heisenbridge.ConfigMap.Namespace
		inputConfigMapNamespace := r.getConfigMapNamespace(synapse, setConfigMapNamespace)
		if inputConfigMapName != "" {
			// If the user provided a custom Heisenbridge configuration via a
			// ConfigMap, we need to validate that the ConfigMap exists, and
			// create a copy in createdHeisenbridgeConfigMap

			// Get and check the input ConfigMap for Heisenbridge
			var inputHeisenbridgeConfigMap = &corev1.ConfigMap{}
			keyForConfigMap := types.NamespacedName{
				Name:      inputConfigMapName,
				Namespace: inputConfigMapNamespace,
			}

			if err := r.Get(ctx, keyForConfigMap, inputHeisenbridgeConfigMap); err != nil {
				reason := "ConfigMap " + inputConfigMapName + " does not exist in namespace " + inputConfigMapNamespace
				if err := r.setFailedState(ctx, &synapse, reason); err != nil {
					log.Error(err, "Error updating Synapse State")
				}

				log.Error(
					err,
					"Failed to get ConfigMap",
					"ConfigMap.Namespace",
					inputConfigMapNamespace,
					"ConfigMap.Name",
					inputConfigMapName,
				)
				return ctrl.Result{RequeueAfter: time.Duration(30)}, err
			}

			// Create a copy of the inputHeisenbridgeConfigMap defined in Spec.Bridges.Heisenbridge.ConfigMap
			// Here we use the configMapForHeisenbridgeCopy function as createResourceFunc
			if err := r.reconcileResource(
				ctx,
				r.configMapForHeisenbridgeCopy,
				&synapse,
				createdHeisenbridgeConfigMap,
				objectMetaHeisenbridge,
			); err != nil {
				return ctrl.Result{}, err
			}

			// Configure correct URL in Heisenbridge ConfigMap
			if err := r.updateConfigMap(
				ctx,
				createdHeisenbridgeConfigMap,
				synapse,
				r.updateHeisenbridgeWithURL,
				"heisenbridge.yaml",
			); err != nil {
				return ctrl.Result{}, err
			}
		} else {
			// If the user hasn't provided a ConfigMap with a custom
			// config.yaml, we create a new ConfigMap with a default
			// config.yaml.

			// Here we use configMapForHeisenbridge as createResourceFunc
			if err := r.reconcileResource(
				ctx,
				r.configMapForHeisenbridge,
				&synapse,
				&corev1.ConfigMap{},
				objectMetaHeisenbridge,
			); err != nil {
				return ctrl.Result{}, err
			}
		}

		// Create Deployment for Heisenbridge
		if err := r.reconcileResource(
			ctx,
			r.deploymentForHeisenbridge,
			&synapse,
			&appsv1.Deployment{},
			objectMetaHeisenbridge,
		); err != nil {
			return ctrl.Result{}, err
		}

		// Update the Synapse ConfigMap to enable Heisenbridge
		if err := r.updateConfigMap(
			ctx,
			&createdConfigMap,
			synapse,
			r.updateHomeserverWithHeisenbridgeInfos,
			"homeserver.yaml",
		); err != nil {
			return ctrl.Result{}, err
		}
	}

	if synapse.Spec.Bridges.MautrixSignal.Enabled {
		log.Info("mautrix-signal is enabled - deploying mautrix-signal")
		// mautrix-signal is composed of a ConfigMap, a Service and 1 Deployment.
		// Resources associated to the mautrix-signal are append with "-mautrixsignal"
		// In addition, a second deployment is needed to run signald. This is append with "-signald"
		createdMautrixSignalService := &corev1.Service{}
		objectMetaMautrixSignal := setObjectMeta(r.GetMautrixSignalResourceName(synapse), synapse.Namespace, map[string]string{})
		objectMetaSignald := setObjectMeta(r.GetSignaldResourceName(synapse), synapse.Namespace, map[string]string{})

		// First create the service as we need its IP address for the
		// config.yaml configuration file
		if err := r.reconcileResource(
			ctx,
			r.serviceForMautrixSignal,
			&synapse,
			createdMautrixSignalService,
			objectMetaMautrixSignal,
		); err != nil {
			return ctrl.Result{}, err
		}

		// The ConfigMap for mautrix-signal, containing the config.yaml config
		// file. It's either a copy of a user-provided ConfigMap, if defined in
		// Spec.Bridges.MautrixSignal.ConfigMap, or a new ConfigMap containing
		// a default config.yaml.
		createdMautrixSignalConfigMap := &corev1.ConfigMap{}

		// The user may specify a ConfigMap, containing the config.yaml config
		// file, under Spec.Bridges.MautrixSignal.ConfigMap
		inputConfigMapName := synapse.Spec.Bridges.MautrixSignal.ConfigMap.Name
		setConfigMapNamespace := synapse.Spec.Bridges.MautrixSignal.ConfigMap.Namespace
		inputConfigMapNamespace := r.getConfigMapNamespace(synapse, setConfigMapNamespace)
		if inputConfigMapName != "" {
			// If the user provided a custom mautrix-signal configuration via a
			// ConfigMap, we need to validate that the ConfigMap exists, and
			// create a copy in createdMautrixSignalConfigMap

			// Get and check the input ConfigMap for mautrix-signal
			var inputMautrixSignalConfigMap = &corev1.ConfigMap{}
			keyForConfigMap := types.NamespacedName{
				Name:      inputConfigMapName,
				Namespace: inputConfigMapNamespace,
			}

			if err := r.Get(ctx, keyForConfigMap, inputMautrixSignalConfigMap); err != nil {
				reason := "ConfigMap " + inputConfigMapName + " does not exist in namespace " + inputConfigMapNamespace
				if err := r.setFailedState(ctx, &synapse, reason); err != nil {
					log.Error(err, "Error updating Synapse State")
				}

				log.Error(
					err,
					"Failed to get ConfigMap",
					"ConfigMap.Namespace",
					inputConfigMapNamespace,
					"ConfigMap.Name",
					inputConfigMapName,
				)
				return ctrl.Result{RequeueAfter: time.Duration(30)}, err
			}

			// Create a copy of the inputMautrixSignalConfigMap defined in Spec.Bridges.MautrixSignal.ConfigMap
			// Here we use the createdMautrixSignalConfigMap function as createResourceFunc
			if err := r.reconcileResource(
				ctx,
				r.configMapForMautrixSignalCopy,
				&synapse,
				createdMautrixSignalConfigMap,
				objectMetaMautrixSignal,
			); err != nil {
				return ctrl.Result{}, err
			}

			// Correct data in mautrix-signal ConfigMap
			if err := r.updateConfigMap(
				ctx,
				createdMautrixSignalConfigMap,
				synapse,
				r.updateMautrixSignalData,
				"config.yaml",
			); err != nil {
				return ctrl.Result{}, err
			}
		} else {
			// If the user hasn't provided a ConfigMap with a custom
			// config.yaml, we create a new ConfigMap with a default
			// config.yaml.

			// Here we use configMapForMautrixSignal as createResourceFunc
			if err := r.reconcileResource(
				ctx,
				r.configMapForMautrixSignal,
				&synapse,
				&corev1.ConfigMap{},
				objectMetaMautrixSignal,
			); err != nil {
				return ctrl.Result{}, err
			}
		}

		// Create a PVC for signald
		if err := r.reconcileResource(
			ctx,
			r.persistentVolumeClaimForSignald,
			&synapse,
			&corev1.PersistentVolumeClaim{},
			objectMetaSignald,
		); err != nil {
			return ctrl.Result{}, err
		}

		// Create Deployment for signald
		if err := r.reconcileResource(
			ctx,
			r.deploymentForSignald,
			&synapse,
			&appsv1.Deployment{},
			objectMetaSignald,
		); err != nil {
			return ctrl.Result{}, err
		}

		// Create the SA for mautrix-signal
		if err := r.reconcileResource(
			ctx,
			r.serviceAccountForMautrixSignal,
			&synapse,
			&corev1.ServiceAccount{},
			objectMetaMautrixSignal,
		); err != nil {
			return ctrl.Result{}, err
		}

		// Create the RoleBinding for mautrix-signal
		if err := r.reconcileResource(
			ctx,
			r.roleBindingForMautrixSignal,
			&synapse,
			&rbacv1.RoleBinding{},
			objectMetaMautrixSignal,
		); err != nil {
			return ctrl.Result{}, err
		}

		// Create a PVC for mautrix-signal
		if err := r.reconcileResource(
			ctx,
			r.persistentVolumeClaimForMautrixSignal,
			&synapse,
			&corev1.PersistentVolumeClaim{},
			objectMetaMautrixSignal,
		); err != nil {
			return ctrl.Result{}, err
		}

		// Create Deployment for mautrix-signal
		if err := r.reconcileResource(
			ctx,
			r.deploymentForMautrixSignal,
			&synapse,
			&appsv1.Deployment{},
			objectMetaMautrixSignal,
		); err != nil {
			return ctrl.Result{}, err
		}

		// Create Deployment for mautrix-signal
		if err := r.reconcileResource(
			ctx,
			r.deploymentForSignald,
			&synapse,
			&appsv1.Deployment{},
			objectMetaSignald,
		); err != nil {
			return ctrl.Result{}, err
		}

		// Update the Synapse ConfigMap to enable mautrix-signal
		if err := r.updateConfigMap(
			ctx,
			&createdConfigMap,
			synapse,
			r.updateHomeserverWithMautrixSignalInfos,
			"homeserver.yaml",
		); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Reconcile Synapse resources: PVC, Deployment and Service
	if err := r.reconcileResource(
		ctx,
		r.serviceAccountForSynapse,
		&synapse,
		&corev1.ServiceAccount{},
		objectMetaForSynapse,
	); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.reconcileResource(
		ctx,
		r.roleBindingForSynapse,
		&synapse,
		&rbacv1.RoleBinding{},
		objectMetaForSynapse,
	); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.reconcileResource(
		ctx,
		r.persistentVolumeClaimForSynapse,
		&synapse,
		&corev1.PersistentVolumeClaim{},
		objectMetaForSynapse,
	); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.reconcileResource(
		ctx,
		r.deploymentForSynapse,
		&synapse,
		&appsv1.Deployment{},
		objectMetaForSynapse,
	); err != nil {
		return ctrl.Result{}, err
	}
	// TODO: If a deployment is found, check that its Spec are correct.

	// Update the Synapse status if needed
	if synapse.Status.State != "RUNNING" {
		synapse.Status.State = "RUNNING"
		synapse.Status.Reason = ""
		if err := r.Status().Update(ctx, &synapse); err != nil {
			log.Error(err, "Failed to update Synapse status")
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

// labelsForSynapse returns the labels for selecting the resources
// belonging to the given synapse CR name.
func labelsForSynapse(name string) map[string]string {
	return map[string]string{"app": "synapse", "synapse_cr": name}
}

// ConfigMap that are created by the user could be living in a different
// namespace as Synapse. getConfigMapNamespace provides a way to default to the
// Synapse namespace if none is provided.
func (r *SynapseReconciler) getConfigMapNamespace(
	synapse synapsev1alpha1.Synapse,
	setNamespace string,
) string {
	if setNamespace != "" {
		return setNamespace
	}
	return synapse.Namespace
}

func (r *SynapseReconciler) setFailedState(ctx context.Context, synapse *synapsev1alpha1.Synapse, reason string) error {

	synapse.Status.State = "FAILED"
	synapse.Status.Reason = reason

	return r.updateSynapseStatus(ctx, synapse)
}

func (r *SynapseReconciler) updateSynapseStatus(ctx context.Context, synapse *synapsev1alpha1.Synapse) error {
	current := &synapsev1alpha1.Synapse{}
	if err := r.Get(
		ctx,
		types.NamespacedName{Name: synapse.Name, Namespace: synapse.Namespace},
		current,
	); err != nil {
		return err
	}

	if !reflect.DeepEqual(synapse.Status, current.Status) {
		if err := r.Status().Patch(ctx, synapse, client.MergeFrom(current)); err != nil {
			return err
		}
	}

	return nil
}

func (r *SynapseReconciler) isPostgresOperatorInstalled(ctx context.Context) bool {
	err := r.Client.List(ctx, &pgov1beta1.PostgresClusterList{})
	return err == nil
}

func (r *SynapseReconciler) createPostgresClusterForSynapse(
	ctx context.Context,
	synapse synapsev1alpha1.Synapse,
	cm corev1.ConfigMap,
) (ctrl.Result, error) {
	var objectMeta metav1.ObjectMeta
	createdPostgresCluster := pgov1beta1.PostgresCluster{}

	// Create ConfigMap for PostgresCluster
	objectMeta = setObjectMeta(synapse.Name+"-pgsql", synapse.Namespace, map[string]string{})
	if err := r.reconcileResource(ctx, r.configMapForPostgresCluster, &synapse, &corev1.ConfigMap{}, objectMeta); err != nil {
		return ctrl.Result{}, err
	}

	// Create PostgresCluster for Synapse
	if err := r.reconcileResource(ctx, r.postgresClusterForSynapse, &synapse, &createdPostgresCluster, objectMeta); err != nil {
		return ctrl.Result{}, err
	}

	// Wait for PostgresCluster to be up
	if err := r.Get(ctx, types.NamespacedName{Name: createdPostgresCluster.Name, Namespace: createdPostgresCluster.Namespace}, &createdPostgresCluster); err != nil {
		return ctrl.Result{}, err
	}
	if !r.isPostgresClusterReady(createdPostgresCluster) {
		r.updateSynapseStatusDatabaseState(ctx, &synapse, "NOT READY")
		err := errors.New("postgreSQL Database not ready yet")
		return ctrl.Result{RequeueAfter: time.Duration(5)}, err
	}

	// Update Synapse Status with PostgreSQL DB information
	if err := r.updateSynapseStatusWithPostgreSQLInfos(ctx, &synapse, createdPostgresCluster); err != nil {
		return ctrl.Result{}, err
	}

	// Update configMap data with PostgreSQL DB information
	if err := r.updateConfigMap(
		ctx,
		&cm,
		synapse,
		r.updateHomeserverWithPostgreSQLInfos,
		"homeserver.yaml",
	); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *SynapseReconciler) isPostgresClusterReady(p pgov1beta1.PostgresCluster) bool {
	var status_found bool

	// Going through instance Specs
	for _, instance_spec := range p.Spec.InstanceSets {
		status_found = false
		for _, instance_status := range p.Status.InstanceSets {
			if instance_status.Name == instance_spec.Name {
				desired_replicas := *instance_spec.Replicas
				if instance_status.Replicas != desired_replicas ||
					instance_status.ReadyReplicas != desired_replicas ||
					instance_status.UpdatedReplicas != desired_replicas {
					return false
				}
				// Found instance in Status, breaking out of for loop
				status_found = true
				break
			}
		}

		// Instance found in spec, but not in status
		if !status_found {
			return false
		}
	}

	// All instances have the correct number of replicas
	return true
}

func (r *SynapseReconciler) updateSynapseStatusDatabaseState(ctx context.Context, synapse *synapsev1alpha1.Synapse, state string) error {
	synapse.Status.DatabaseConnectionInfo.State = state
	return r.updateSynapseStatus(ctx, synapse)
}

func (r *SynapseReconciler) updateSynapseStatusWithPostgreSQLInfos(
	ctx context.Context,
	s *synapsev1alpha1.Synapse,
	createdPostgresCluster pgov1beta1.PostgresCluster,
) error {
	var postgresSecret corev1.Secret

	// Get PostgreSQL secret related, containing information for the synapse user
	if err := r.Get(
		ctx,
		types.NamespacedName{
			Name:      createdPostgresCluster.Name + "-pguser-synapse",
			Namespace: createdPostgresCluster.Namespace,
		},
		&postgresSecret,
	); err != nil {
		return err
	}

	if err := r.updateSynapseStatusDatabase(s, postgresSecret); err != nil {
		return err
	}

	return r.updateSynapseStatus(ctx, s)
}

func (r *SynapseReconciler) updateSynapseStatusDatabase(
	s *synapsev1alpha1.Synapse,
	postgresSecret corev1.Secret,
) error {
	var postgresSecretData map[string][]byte = postgresSecret.Data

	host, ok := postgresSecretData["host"]
	if !ok {
		err := errors.New("missing host in PostgreSQL Secret")
		// log.Error(err, "Missing host in PostgreSQL Secret")
		return err
	}

	port, ok := postgresSecretData["port"]
	if !ok {
		err := errors.New("missing port in PostgreSQL Secret")
		// log.Error(err, "Missing port in PostgreSQL Secret")
		return err
	}

	// See https://github.com/opdev/synapse-operator/issues/12
	// databaseName, ok := postgresSecretData["dbname"]
	_, ok = postgresSecretData["dbname"]
	if !ok {
		err := errors.New("missing dbname in PostgreSQL Secret")
		// log.Error(err, "Missing dbname in PostgreSQL Secret")
		return err
	}

	user, ok := postgresSecretData["user"]
	if !ok {
		err := errors.New("missing user in PostgreSQL Secret")
		// log.Error(err, "Missing user in PostgreSQL Secret")
		return err
	}

	password, ok := postgresSecretData["password"]
	if !ok {
		err := errors.New("missing password in PostgreSQL Secret")
		// log.Error(err, "Missing password in PostgreSQL Secret")
		return err
	}

	s.Status.DatabaseConnectionInfo.ConnectionURL = string(host) + ":" + string(port)
	// s.Status.DatabaseConnectionInfo.DatabaseName = string(databaseName) // See https://github.com/opdev/synapse-operator/issues/12
	s.Status.DatabaseConnectionInfo.DatabaseName = "synapse"
	s.Status.DatabaseConnectionInfo.User = string(user)
	s.Status.DatabaseConnectionInfo.Password = string(base64encode(string(password)))
	s.Status.DatabaseConnectionInfo.State = "READY"

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SynapseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&synapsev1alpha1.Synapse{}).
		Owns(&corev1.Service{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Complete(r)
}
