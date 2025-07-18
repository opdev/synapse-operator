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

package heisenbridge

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/opdev/subreconciler"
	synapsev1alpha1 "github.com/opdev/synapse-operator/api/synapse/v1alpha1"
	"github.com/opdev/synapse-operator/helpers/utils"
)

// HeisenbridgeReconciler reconciles a Heisenbridge object
type HeisenbridgeReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=synapse.opdev.io,resources=heisenbridges,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=synapse.opdev.io,resources=heisenbridges/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=synapse.opdev.io,resources=heisenbridges/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Heisenbridge object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.4/pkg/reconcile
func (r *HeisenbridgeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var h synapsev1alpha1.Heisenbridge // The Heisenbridge object being reconciled
	if r, err := utils.GetResource(ctx, r.Client, req, &h); subreconciler.ShouldHaltOrRequeue(r, err) {
		return subreconciler.Evaluate(r, err)
	}

	// The list of subreconcilers for Heisenbridge.
	var subreconcilersForHeisenbridge []subreconciler.FnWithRequest

	// We need to trigger a Synapse reconciliation so that it becomes aware of
	// the Heisenbridge.
	subreconcilersForHeisenbridge = []subreconciler.FnWithRequest{
		utils.HandleDelete(r.Client, &synapsev1alpha1.Heisenbridge{}),
		utils.AddFinalizer(r.Client, &synapsev1alpha1.Heisenbridge{}),
		utils.TriggerSynapseReconciliation(r.Client, &synapsev1alpha1.Heisenbridge{}),
	}

	// The user may specify a ConfigMap, containing the heisenbridge.yaml
	// config file, under Spec.Bridges.Heisenbridge.ConfigMap
	if h.Spec.ConfigMap.Name != "" {
		// If the user provided a custom Heisenbridge configuration via a
		// ConfigMap, we need to validate that the ConfigMap exists, and
		// create a copy. We also need to edit the heisenbridge
		// configuration.
		subreconcilersForHeisenbridge = append(
			subreconcilersForHeisenbridge,
			utils.CopyInputConfigMap(r.Client, r.Scheme, &synapsev1alpha1.Heisenbridge{}),
			r.configureHeisenbridgeConfigMap,
		)
	} else {
		// If the user hasn't provided a ConfigMap with a custom
		// heisenbridge.yaml, we create a new ConfigMap with a default
		// heisenbridge.yaml.
		subreconcilersForHeisenbridge = append(
			subreconcilersForHeisenbridge,
			r.reconcileHeisenbridgeConfigMap,
		)
	}

	// Reconcile Heisenbridge resources: Service and Deployment
	subreconcilersForHeisenbridge = append(
		subreconcilersForHeisenbridge,
		r.reconcileHeisenbridgeService,
		r.reconcileHeisenbridgeDeployment,
	)

	// Run all subreconcilers sequentially
	for _, f := range subreconcilersForHeisenbridge {
		if r, err := f(ctx, req); subreconciler.ShouldHaltOrRequeue(r, err) {
			return subreconciler.Evaluate(r, err)
		}
	}

	return subreconciler.Evaluate(subreconciler.DoNotRequeue())
}

// SetupWithManager sets up the controller with the Manager.
func (r *HeisenbridgeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&synapsev1alpha1.Heisenbridge{}).
		Named("synapse-heisenbridge").
		Complete(r)
}
