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

	reconc "github.com/opdev/synapse-operator/helpers/reconcileresults"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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

func (r *SynapseReconciler) GetPostgresClusterResourceName(synapse synapsev1alpha1.Synapse) string {
	return strings.Join([]string{synapse.Name, "pgsql"}, "-")
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

// subreconcilerFuncs are functions that are called by Reconcile() functions
// in an ordered fashion. Returning a ctrl.Result with a value of nil
// indicates that the Reconcile() function should continue reconciling.
// Any other returned ctrl.Result indicates to the Reconcile() function
// that reconciliation should halt.
type subreconcilerFuncs func(*synapsev1alpha1.Synapse, context.Context) (*ctrl.Result, error)

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

	// The list of subreconcilers to be run to reconciliate the Synapse
	// ConfigMap. This is temporary, and will be merged with
	// subreconcilersForSynapse in a later commit.
	var subreconcilersForSynapseConfigMap []subreconcilerFuncs

	// Synapse should either have a Spec.Homeserver.ConfigMap or Spec.Homeserver.Values
	if synapse.Spec.Homeserver.ConfigMap != nil {
		// If the user provided a ConfigMap for the Homeserver config file:
		// * We ensure that it exists and is a valid yaml file
		// * We populate the Status.HomeserverConfiguration with the values defined in the input ConfigMap
		// * We create a copy of the user-provided ConfigMap.
		subreconcilersForSynapseConfigMap = []subreconcilerFuncs{
			r.parseInputSynapseConfigMap,
			r.copyInputSynapseConfigMap,
		}
	} else {
		// If the user hasn't provided a ConfigMap with a custom
		// homeserver.yaml, we create a new ConfigMap. The default
		// homeserver.yaml is configured with values defined in
		// Spec.Homeserver.Values
		synapse.Status.HomeserverConfiguration.ServerName = synapse.Spec.Homeserver.Values.ServerName
		synapse.Status.HomeserverConfiguration.ReportStats = synapse.Spec.Homeserver.Values.ReportStats

		subreconcilersForSynapseConfigMap = []subreconcilerFuncs{
			r.reconcileSynapseConfigMap,
		}
	}

	for _, f := range subreconcilersForSynapseConfigMap {
		if r, err := f(&synapse, ctx); reconc.ShouldHaltOrRequeue(r, err) {
			return reconc.Evaluate(r, err)
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

			err := errors.New("cannot create PostgreSQL instance for synapse. Potsgres-operator is not installed")
			log.Error(err, "Cannot create PostgreSQL instance for synapse. Potsgres-operator is not installed.")
			return ctrl.Result{}, nil
		}

		// Reconcile the PostgresCluster CR and ConfigMap.
		// Also update the Synapse Status and ConfigMap with database
		// connection information.
		subreconcilersForPostgresCluster := []subreconcilerFuncs{
			r.reconcilePostgresClusterConfigMap,
			r.reconcilePostgresClusterCR,
			r.updateSynapseStatusWithPostgreSQLInfos,
			r.updateSynapseConfigMapForPostgresCluster,
		}

		for _, f := range subreconcilersForPostgresCluster {
			if r, err := f(&synapse, ctx); reconc.ShouldHaltOrRequeue(r, err) {
				return reconc.Evaluate(r, err)
			}
		}
	}

	// The list of subreconcilers for Synapse. The list is populated depending
	// on which bridges are enabled.
	var subreconcilersForSynapse []subreconcilerFuncs

	if synapse.Spec.Bridges.Heisenbridge.Enabled {
		log.Info("Heisenbridge is enabled - deploying Heisenbridge")

		// The list of subreconcilers for Heisenbridge will be built next.
		// Heisenbridge is composed of a ConfigMap, a Service and a Deployment.
		var subreconcilersForHeisenbridge []subreconcilerFuncs

		// The user may specify a ConfigMap, containing the heisenbridge.yaml
		// config file, under Spec.Bridges.Heisenbridge.ConfigMap
		if synapse.Spec.Bridges.Heisenbridge.ConfigMap.Name != "" {
			// If the user provided a custom Heisenbridge configuration via a
			// ConfigMap, we need to validate that the ConfigMap exists, and
			// create a copy. We also need to edit the heisenbridge
			// configuration.
			subreconcilersForHeisenbridge = []subreconcilerFuncs{
				r.copyInputHeisenbridgeConfigMap,
				r.configureHeisenbridgeConfigMap,
			}
		} else {
			// If the user hasn't provided a ConfigMap with a custom
			// heisenbridge.yaml, we create a new ConfigMap with a default
			// heisenbridge.yaml.
			subreconcilersForHeisenbridge = []subreconcilerFuncs{
				r.reconcileHeisenbridgeConfigMap,
			}
		}

		// Reconcile Heisenbridge resources: Service and Deployment
		subreconcilersForHeisenbridge = append(
			subreconcilersForHeisenbridge,
			r.reconcileHeisenbridgeService,
			r.reconcileHeisenbridgeDeployment,
		)

		for _, f := range subreconcilersForHeisenbridge {
			if r, err := f(&synapse, ctx); reconc.ShouldHaltOrRequeue(r, err) {
				return reconc.Evaluate(r, err)
			}
		}

		// Add the update of the Synapse ConfigMap to the Synapse
		// subreconciler list. This is to prepare for future work. When using
		// a multi API approach, we forsee this task to be performed by the
		// Synapse controller (as opposed to the Heisenbridge controller,
		// performing all task listed in subreconcilersForHeisenbridge).
		subreconcilersForSynapse = append(subreconcilersForSynapse, r.updateSynapseConfigMapForHeisenbridge)
	}

	if synapse.Spec.Bridges.MautrixSignal.Enabled {
		log.Info("mautrix-signal is enabled - deploying mautrix-signal")

		// The list of subreconcilers for mautrix-signal will be built next.
		// mautrix-signal is composed of a ConfigMap, a Service, a SA, a RB,
		// a PVC and a Deployment.
		// In addition, a Deployment and a PVC are needed for signald.
		var subreconcilersForMautrixSignal []subreconcilerFuncs

		// The user may specify a ConfigMap, containing the config.yaml config
		// file, under Spec.Bridges.MautrixSignal.ConfigMap
		if synapse.Spec.Bridges.MautrixSignal.ConfigMap.Name != "" {
			// If the user provided a custom mautrix-signal configuration via a
			// ConfigMap, we need to validate that the ConfigMap exists, and
			// create a copy. We also need to edit the mautrix-signal
			// configuration.
			subreconcilersForMautrixSignal = []subreconcilerFuncs{
				r.copyInputMautrixSignalConfigMap,
				r.configureMautrixSignalConfigMap,
			}

		} else {
			// If the user hasn't provided a ConfigMap with a custom
			// config.yaml, we create a new ConfigMap with a default
			// config.yaml.
			subreconcilersForMautrixSignal = []subreconcilerFuncs{
				r.reconcileMautrixSignalConfigMap,
			}
		}

		// Reconcile signald resources: PVC and Deployment
		// Reconcile mautrix-signal resources: Service, SA, RB, PVC and Deployment
		subreconcilersForMautrixSignal = append(
			subreconcilersForMautrixSignal,
			r.reconcileSignaldPVC,
			r.reconcileSignaldDeployment,
			r.reconcileMautrixSignalService,
			r.reconcileMautrixSignalServiceAccount,
			r.reconcileMautrixSignalRoleBinding,
			r.reconcileMautrixSignalPVC,
			r.reconcileMautrixSignalDeployment,
		)

		for _, f := range subreconcilersForMautrixSignal {
			if r, err := f(&synapse, ctx); reconc.ShouldHaltOrRequeue(r, err) {
				return reconc.Evaluate(r, err)
			}
		}

		// Add the update of the Synapse ConfigMap to the Synapse
		// subreconciler list. This is to prepare for future work. When using
		// a multi API approach, we forsee this task to be performed by the
		// Synapse controller (as opposed to the mautrix-signal controller,
		// performing all task listed in subreconcilersForMautrixSignal).
		subreconcilersForSynapse = append(subreconcilersForSynapse, r.updateSynapseConfigMapForMautrixSignal)
	}

	// Reconcile Synapse resources: Service, SA, RB, PVC, Deployment
	subreconcilersForSynapse = append(
		subreconcilersForSynapse,
		r.reconcileSynapseService,
		r.reconcileSynapseServiceAccount,
		r.reconcileSynapseRoleBinding,
		r.reconcileSynapsePVC,
		r.reconcileSynapseDeployment,
	)

	for _, f := range subreconcilersForSynapse {
		if r, err := f(&synapse, ctx); reconc.ShouldHaltOrRequeue(r, err) {
			return reconc.Evaluate(r, err)
		}
	}

	// Update the Synapse status if needed
	if synapse.Status.State != "RUNNING" {
		synapse.Status.State = "RUNNING"
		synapse.Status.Reason = ""
		if err := r.Status().Update(ctx, &synapse); err != nil {
			log.Error(err, "Failed to update Synapse status")
			return ctrl.Result{}, err
		}
	}

	return reconc.Evaluate(reconc.DoNotRequeue())
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

func (r *SynapseReconciler) updateSynapseStatusDatabaseState(ctx context.Context, synapse *synapsev1alpha1.Synapse, state string) error {
	synapse.Status.DatabaseConnectionInfo.State = state
	return r.updateSynapseStatus(ctx, synapse)
}

// updateSynapseStatusWithPostgreSQLInfos is a function of type
// subreconcilerFuncs, to be called in the main reconciliation loop.
//
// It parses the PostgresCluster Secret and updates the Synapse status with the
// database connection information.
func (r *SynapseReconciler) updateSynapseStatusWithPostgreSQLInfos(synapse *synapsev1alpha1.Synapse, ctx context.Context) (*ctrl.Result, error) {
	var postgresSecret corev1.Secret

	keyForPostgresClusterSecret := types.NamespacedName{
		Name:      r.GetPostgresClusterResourceName(*synapse) + "-pguser-synapse",
		Namespace: synapse.Namespace,
	}

	// Get PostgresCluster Secret containing information for the synapse user
	if err := r.Get(ctx, keyForPostgresClusterSecret, &postgresSecret); err != nil {
		return reconc.RequeueWithError(err)
	}

	// Locally updates the Synapse Status
	if err := r.updateSynapseStatusDatabase(synapse, postgresSecret); err != nil {
		return reconc.RequeueWithError(err)
	}

	// Actually sends an API request to update the Status
	if err := r.updateSynapseStatus(ctx, synapse); err != nil {
		return reconc.RequeueWithError(err)
	}

	return reconc.ContinueReconciling()
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
