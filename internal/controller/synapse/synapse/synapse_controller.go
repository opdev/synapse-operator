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

package synapse

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	subreconciler "github.com/opdev/subreconciler"
	synapsev1alpha1 "github.com/opdev/synapse-operator/api/synapse/v1alpha1"
	"github.com/opdev/synapse-operator/helpers/utils"
)

// SynapseReconciler reconciles a Synapse object
type SynapseReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=synapse.opdev.io,resources=synapses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=synapse.opdev.io,resources=synapses/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=synapse.opdev.io,resources=synapses/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=services;persistentvolumeclaims;configmaps;serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=security.openshift.io,resourceNames=anyuid,resources=securitycontextconstraints,verbs=use

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.4/pkg/reconcile
func (r *SynapseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var synapse synapsev1alpha1.Synapse // The Synapse object being reconciled
	if r, err := utils.GetResource(ctx, r.Client, req, &synapse); subreconciler.ShouldHaltOrRequeue(r, err) {
		return subreconciler.Evaluate(r, err)
	}

	// The list of subreconcilers for Synapse.
	var subreconcilersForSynapse []subreconciler.FnWithRequest

	// Synapse should either have a Spec.Homeserver.ConfigMap or Spec.Homeserver.Values
	if synapse.Spec.Homeserver.ConfigMap != nil {
		// If the user provided a ConfigMap for the Homeserver config file:
		// * We ensure that it exists and is a valid yaml file
		// * We populate the Status.HomeserverConfiguration with the values defined in the input ConfigMap
		// * We create a copy of the user-provided ConfigMap.
		subreconcilersForSynapse = []subreconciler.FnWithRequest{
			r.parseInputSynapseConfigMap,
			utils.CopyInputConfigMap(r.Client, r.Scheme, &synapsev1alpha1.Synapse{}),
		}
	} else {
		// If the user hasn't provided a ConfigMap with a custom
		// homeserver.yaml, we create a new ConfigMap. The default
		// homeserver.yaml is configured with values defined in
		// Spec.Homeserver.Values
		subreconcilersForSynapse = []subreconciler.FnWithRequest{
			r.setStatusHomeserverConfiguration,
			r.reconcileSynapseConfigMap,
		}
	}

	// Determine the existence of Bridges referencing this Synapse instance
	subreconcilersForSynapse = append(
		subreconcilersForSynapse,
		r.updateSynapseStatusBridges,
		r.updateSynapseConfigMapForBridges,
	)

	// SA and RB are only necessary if we're running on OpenShift
	if synapse.Spec.IsOpenshift {
		subreconcilersForSynapse = append(
			subreconcilersForSynapse,
			r.reconcileSynapseServiceAccount,
			r.reconcileSynapseRoleBinding,
		)
	}

	// Reconcile Synapse resources: Service, PVC, Deployment
	subreconcilersForSynapse = append(
		subreconcilersForSynapse,
		r.reconcileSynapseService,
		r.reconcileSynapsePVC,
		r.reconcileSynapseDeployment,
		r.setSynapseStatusAsRunning,
	)

	// Run all subreconcilers sequentially
	for _, f := range subreconcilersForSynapse {
		if r, err := f(ctx, req); subreconciler.ShouldHaltOrRequeue(r, err) {
			return subreconciler.Evaluate(r, err)
		}
	}

	return subreconciler.Evaluate(subreconciler.DoNotRequeue())
}

// labelsForSynapse returns the labels for selecting the resources
// belonging to the given synapse CR name.
func labelsForSynapse(name string) map[string]string {
	return map[string]string{"app": "synapse", "synapse_cr": name}
}

func (r *SynapseReconciler) setStatusHomeserverConfiguration(
	ctx context.Context,
	req ctrl.Request,
) (*ctrl.Result, error) {
	log := logf.FromContext(ctx)

	s := &synapsev1alpha1.Synapse{}
	if r, err := utils.GetResource(ctx, r.Client, req, s); subreconciler.ShouldHaltOrRequeue(r, err) {
		return r, err
	}

	s.Status.HomeserverConfiguration.ServerName = s.Spec.Homeserver.Values.ServerName
	s.Status.HomeserverConfiguration.ReportStats = s.Spec.Homeserver.Values.ReportStats

	// Set registration enabled status - default to false if not specified
	registrationEnabled := false
	if s.Spec.Homeserver.Values.EnableRegistration != nil {
		registrationEnabled = *s.Spec.Homeserver.Values.EnableRegistration
	}
	s.Status.HomeserverConfiguration.RegistrationEnabled = registrationEnabled

	err := utils.UpdateResourceStatus(ctx, r.Client, s, &synapsev1alpha1.Synapse{})
	if err != nil {
		log.Error(err, "Error updating Synapse Status")
		return subreconciler.RequeueWithError(err)
	}

	return subreconciler.ContinueReconciling()
}

// setSynapseStatusAsRunning is a function of type FnWithRequest, to be
// called in the main reconciliation loop.
//
// It set the Synapse Status 'State' field to 'RUNNING'.
func (r *SynapseReconciler) setSynapseStatusAsRunning(ctx context.Context, req ctrl.Request) (*ctrl.Result, error) {
	log := logf.FromContext(ctx)

	s := &synapsev1alpha1.Synapse{}
	if r, err := utils.GetResource(ctx, r.Client, req, s); subreconciler.ShouldHaltOrRequeue(r, err) {
		return r, err
	}

	s.Status.NeedsReconcile = false
	s.Status.State = "RUNNING"
	s.Status.Reason = ""

	err := utils.UpdateResourceStatus(ctx, r.Client, s, &synapsev1alpha1.Synapse{})
	if err != nil {
		log.Error(err, "Error updating Synapse Status")
		return subreconciler.RequeueWithError(err)
	}

	return subreconciler.ContinueReconciling()
}

func (r *SynapseReconciler) updateSynapseStatusBridges(ctx context.Context, req ctrl.Request) (*ctrl.Result, error) {
	log := logf.FromContext(ctx)

	s := &synapsev1alpha1.Synapse{}
	if r, err := utils.GetResource(ctx, r.Client, req, s); subreconciler.ShouldHaltOrRequeue(r, err) {
		return r, err
	}

	// Set to default
	s.Status.Bridges.Heisenbridge = synapsev1alpha1.SynapseStatusBridgesHeisenbridge{}
	s.Status.Bridges.MautrixSignal = synapsev1alpha1.SynapseStatusBridgesMautrixSignal{}

	hList := &synapsev1alpha1.HeisenbridgeList{}
	if err := r.List(ctx, hList); err != nil {
		return subreconciler.RequeueWithError(err)
	}
	for _, h := range hList.Items {
		if h.Spec.Synapse.Name == s.Name && h.GetDeletionTimestamp() == nil {
			s.Status.Bridges.Heisenbridge.Enabled = true
			s.Status.Bridges.Heisenbridge.Name = h.Name
			break
		}
	}

	msList := &synapsev1alpha1.MautrixSignalList{}
	if err := r.List(ctx, msList); err != nil {
		return subreconciler.RequeueWithError(err)
	}
	for _, ms := range msList.Items {
		if ms.Spec.Synapse.Name == s.Name && ms.GetDeletionTimestamp() == nil {
			s.Status.Bridges.MautrixSignal.Enabled = true
			s.Status.Bridges.MautrixSignal.Name = ms.Name
			break
		}
	}

	err := utils.UpdateResourceStatus(ctx, r.Client, s, &synapsev1alpha1.Synapse{})
	if err != nil {
		log.Error(err, "Error updating Synapse Status")
		return subreconciler.RequeueWithError(err)
	}

	return subreconciler.ContinueReconciling()
}

// SetupWithManager sets up the controller with the Manager.
func (r *SynapseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&synapsev1alpha1.Synapse{}).
		Owns(&corev1.Service{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Named("synapse-synapse").
		Complete(r)
}
