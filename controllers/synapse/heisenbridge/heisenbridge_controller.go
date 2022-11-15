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
	"reflect"
	"strings"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	synapsev1alpha1 "github.com/opdev/synapse-operator/apis/synapse/v1alpha1"
	reconc "github.com/opdev/synapse-operator/helpers/reconcileresults"
	"github.com/opdev/synapse-operator/helpers/utils"
)

// HeisenbridgeReconciler reconciles a Heisenbridge object
type HeisenbridgeReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func GetHeisenbridgeServiceFQDN(h synapsev1alpha1.Heisenbridge) string {
	return strings.Join([]string{h.Name, h.Namespace, "svc", "cluster", "local"}, ".")
}

//+kubebuilder:rbac:groups=synapse.opdev.io,resources=heisenbridges,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=synapse.opdev.io,resources=heisenbridges/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=synapse.opdev.io,resources=heisenbridges/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Heisenbridge object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *HeisenbridgeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)

	var h synapsev1alpha1.Heisenbridge // The Heisenbridge object being reconciled

	// Load the Heisenbridge by name
	if err := r.Get(ctx, req.NamespacedName, &h); err != nil {
		if k8serrors.IsNotFound(err) {
			// we'll ignore not-found errors, since they can't be fixed by an immediate
			// requeue (we'll need to wait for a new notification), and we can get them
			// on deleted requests.
			log.Error(
				err,
				"Cannot find Heisenbridge - has it been deleted ?",
				"Heisenbridge Name", h.Name,
				"Heisenbridge Namespace", h.Namespace,
			)
			return ctrl.Result{}, nil
		}
		log.Error(
			err,
			"Error fetching Heisenbridge",
			"Heisenbridge Name", h.Name,
			"Heisenbridge Namespace", h.Namespace,
		)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Build mautrix-signal status
	s, err := r.fetchSynapseInstance(ctx, h)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Error(
				err,
				"Cannot find Synapse instance",
				"Synapse Name", h.Spec.Synapse.Name,
				"Synapse Namespace", utils.ComputeNamespace(h.Namespace, h.Spec.Synapse.Namespace),
			)
		} else {
			log.Error(
				err,
				"Error getting Synapse server name",
				"Synapse Name", h.Spec.Synapse.Name,
				"Synapse Namespace", utils.ComputeNamespace(h.Namespace, h.Spec.Synapse.Namespace),
			)
		}
		return ctrl.Result{}, err
	}

	if r, err := r.triggerSynapseReconciliation(&s, ctx); reconc.ShouldHaltOrRequeue(r, err) {
		return reconc.Evaluate(r, err)
	}

	// The list of subreconcilers for Heisenbridge will be built next.
	// Heisenbridge is composed of a ConfigMap, a Service and a Deployment.
	var subreconcilersForHeisenbridge []reconc.SubreconcilerFuncs

	// The user may specify a ConfigMap, containing the heisenbridge.yaml
	// config file, under Spec.Bridges.Heisenbridge.ConfigMap
	if h.Spec.ConfigMap.Name != "" {
		// If the user provided a custom Heisenbridge configuration via a
		// ConfigMap, we need to validate that the ConfigMap exists, and
		// create a copy. We also need to edit the heisenbridge
		// configuration.
		subreconcilersForHeisenbridge = []reconc.SubreconcilerFuncs{
			r.copyInputHeisenbridgeConfigMap,
			r.configureHeisenbridgeConfigMap,
		}
	} else {
		// If the user hasn't provided a ConfigMap with a custom
		// heisenbridge.yaml, we create a new ConfigMap with a default
		// heisenbridge.yaml.
		subreconcilersForHeisenbridge = []reconc.SubreconcilerFuncs{
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
		if r, err := f(&h, ctx); reconc.ShouldHaltOrRequeue(r, err) {
			return reconc.Evaluate(r, err)
		}
	}

	return ctrl.Result{}, nil
}

func (r *HeisenbridgeReconciler) fetchSynapseInstance(
	ctx context.Context,
	h synapsev1alpha1.Heisenbridge,
) (synapsev1alpha1.Synapse, error) {
	// Validate Synapse instance exists
	s := &synapsev1alpha1.Synapse{}
	keyForSynapse := types.NamespacedName{
		Name:      h.Spec.Synapse.Name,
		Namespace: utils.ComputeNamespace(h.Namespace, h.Spec.Synapse.Namespace),
	}
	if err := r.Get(ctx, keyForSynapse, s); err != nil {
		return synapsev1alpha1.Synapse{}, err
	}

	return *s, nil
}

func (r *HeisenbridgeReconciler) triggerSynapseReconciliation(i interface{}, ctx context.Context) (*ctrl.Result, error) {
	s := i.(*synapsev1alpha1.Synapse)
	s.Status.NeedsReconcile = true

	current := &synapsev1alpha1.Synapse{}
	if err := r.Get(
		ctx,
		types.NamespacedName{Name: s.Name, Namespace: s.Namespace},
		current,
	); err != nil {
		return reconc.RequeueWithError(err)
	}

	if !reflect.DeepEqual(s.Status, current.Status) {
		if err := r.Status().Patch(ctx, s, client.MergeFrom(current)); err != nil {
			return reconc.RequeueWithError(err)
		}
	}

	return reconc.ContinueReconciling()
}

func (r *HeisenbridgeReconciler) setFailedState(ctx context.Context, h *synapsev1alpha1.Heisenbridge, reason string) error {
	h.Status.State = "FAILED"
	h.Status.Reason = reason

	return r.updateHeisenbridgeStatus(ctx, h)
}

func (r *HeisenbridgeReconciler) updateHeisenbridgeStatus(ctx context.Context, h *synapsev1alpha1.Heisenbridge) error {
	current := &synapsev1alpha1.Heisenbridge{}
	if err := r.Get(
		ctx,
		types.NamespacedName{Name: h.Name, Namespace: h.Namespace},
		current,
	); err != nil {
		return err
	}

	if !reflect.DeepEqual(h.Status, current.Status) {
		if err := r.Status().Patch(ctx, h, client.MergeFrom(current)); err != nil {
			return err
		}
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *HeisenbridgeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&synapsev1alpha1.Heisenbridge{}).
		Complete(r)
}
