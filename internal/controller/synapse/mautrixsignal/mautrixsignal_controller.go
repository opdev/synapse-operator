/*
Copyright 2025.

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

package mautrixsignal

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/opdev/subreconciler"
	synapsev1alpha1 "github.com/opdev/synapse-operator/api/synapse/v1alpha1"
	"github.com/opdev/synapse-operator/helpers/utils"
)

// MautrixSignalReconciler reconciles a MautrixSignal object
type MautrixSignalReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=synapse.opdev.io,resources=mautrixsignals,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=synapse.opdev.io,resources=mautrixsignals/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=synapse.opdev.io,resources=mautrixsignals/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MautrixSignal object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.4/pkg/reconcile
func (r *MautrixSignalReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var ms synapsev1alpha1.MautrixSignal // The mautrix-signal object being reconciled
	if r, err := utils.GetResource(ctx, r.Client, req, &ms); subreconciler.ShouldHaltOrRequeue(r, err) {
		return subreconciler.Evaluate(r, err)
	}

	// The list of subreconcilers for mautrix-signal.
	var subreconcilersForMautrixSignal []subreconciler.FnWithRequest

	// We need to trigger a Synapse reconciliation so that it becomes aware of
	// the MautrixSignal. We also need to complete the MautrixSignal Status.
	subreconcilersForMautrixSignal = []subreconciler.FnWithRequest{
		utils.HandleDelete(r.Client, &synapsev1alpha1.MautrixSignal{}),
		utils.AddFinalizer(r.Client, &synapsev1alpha1.MautrixSignal{}),
		utils.TriggerSynapseReconciliation(r.Client, &synapsev1alpha1.MautrixSignal{}),
		r.buildMautrixSignalStatus,
	}

	// The user may specify a ConfigMap, containing the config.yaml config
	// file, under Spec.Bridges.MautrixSignal.ConfigMap
	if ms.Spec.ConfigMap.Name != "" {
		// If the user provided a custom mautrix-signal configuration via a
		// ConfigMap, we need to validate that the ConfigMap exists, and
		// create a copy. We also need to edit the mautrix-signal
		// configuration.
		subreconcilersForMautrixSignal = append(
			subreconcilersForMautrixSignal,
			utils.CopyInputConfigMap(r.Client, r.Scheme, &synapsev1alpha1.MautrixSignal{}),
			r.configureMautrixSignalConfigMap,
		)
	} else {
		// If the user hasn't provided a ConfigMap with a custom
		// config.yaml, we create a new ConfigMap with a default
		// config.yaml.
		subreconcilersForMautrixSignal = append(
			subreconcilersForMautrixSignal,
			r.reconcileMautrixSignalConfigMap,
		)
	}

	// SA and RB are only necessary if we're running on OpenShift
	if ms.Status.IsOpenshift {
		subreconcilersForMautrixSignal = append(
			subreconcilersForMautrixSignal,
			r.reconcileMautrixSignalServiceAccount,
			r.reconcileMautrixSignalRoleBinding,
		)
	}

	// Reconcile mautrix-signal resources: Service, PVC and Deployment
	subreconcilersForMautrixSignal = append(
		subreconcilersForMautrixSignal,
		r.reconcileMautrixSignalService,
		r.reconcileMautrixSignalPVC,
		r.reconcileMautrixSignalDeployment,
	)

	// Run all subreconcilers sequentially
	for _, f := range subreconcilersForMautrixSignal {
		if r, err := f(ctx, req); subreconciler.ShouldHaltOrRequeue(r, err) {
			return subreconciler.Evaluate(r, err)
		}
	}

	return subreconciler.Evaluate(subreconciler.DoNotRequeue())
}

func (r *MautrixSignalReconciler) buildMautrixSignalStatus(
	ctx context.Context,
	req ctrl.Request,
) (*ctrl.Result, error) {
	log := logf.FromContext(ctx)

	ms := &synapsev1alpha1.MautrixSignal{}
	if r, err := utils.GetResource(ctx, r.Client, req, ms); subreconciler.ShouldHaltOrRequeue(r, err) {
		return r, err
	}

	s := synapsev1alpha1.Synapse{}
	if err := utils.FetchSynapseInstance(ctx, r.Client, ms, &s); err != nil {
		log.Error(err, "Error fetching Synapse instance")
		return subreconciler.RequeueWithError(err)
	}

	// Get Synapse ServerName
	serverName, err := utils.GetSynapseServerName(s)
	if err != nil {
		log.Error(
			err,
			"Error getting Synapse ServerName",
			"Synapse Name", ms.Spec.Synapse.Name,
			"Synapse Namespace", utils.ComputeNamespace(ms.Namespace, ms.Spec.Synapse.Namespace),
		)
		return subreconciler.RequeueWithError(err)
	}
	ms.Status.Synapse.ServerName = serverName

	ms.Status.IsOpenshift = s.Spec.IsOpenshift

	err = utils.UpdateResourceStatus(ctx, r.Client, ms, &synapsev1alpha1.MautrixSignal{})
	if err != nil {
		log.Error(err, "Error updating mautrix-signal Status")
		return subreconciler.RequeueWithError(err)
	}

	return subreconciler.ContinueReconciling()
}

// SetupWithManager sets up the controller with the Manager.
func (r *MautrixSignalReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&synapsev1alpha1.MautrixSignal{}).
		Named("synapse-mautrixsignal").
		Complete(r)
}
