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

package mautrixsignal

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

// MautrixSignalReconciler reconciles a MautrixSignal object
type MautrixSignalReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func GetSignaldResourceName(ms synapsev1alpha1.MautrixSignal) string {
	return strings.Join([]string{ms.Name, "signald"}, "-")
}

func GetMautrixSignalServiceFQDN(ms synapsev1alpha1.MautrixSignal) string {
	return strings.Join([]string{ms.Name, ms.Namespace, "svc", "cluster", "local"}, ".")
}

//+kubebuilder:rbac:groups=synapse.opdev.io,resources=mautrixsignals,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=synapse.opdev.io,resources=mautrixsignals/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=synapse.opdev.io,resources=mautrixsignals/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MautrixSignal object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *MautrixSignalReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)

	var ms synapsev1alpha1.MautrixSignal // The mautrix-signal object being reconciled

	// Load the mautrix-signal by name
	if err := r.Get(ctx, req.NamespacedName, &ms); err != nil {
		if k8serrors.IsNotFound(err) {
			// we'll ignore not-found errors, since they can't be fixed by an immediate
			// requeue (we'll need to wait for a new notification), and we can get them
			// on deleted requests.
			log.Error(
				err,
				"Cannot find mautrix-signal - has it been deleted ?",
				"mautrix-signal Name", ms.Name,
				"mautrix-signal Namespace", ms.Namespace,
			)
			return ctrl.Result{}, nil
		}
		log.Error(
			err,
			"Error fetching mautrix-signal",
			"mautrix-signal Name", ms.Name,
			"mautrix-signal Namespace", ms.Namespace,
		)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Build mautrix-signal status
	s, err := r.fetchSynapseInstance(ctx, ms)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Error(
				err,
				"Cannot find Synapse instance",
				"Synapse Name", ms.Spec.Synapse.Name,
				"Synapse Namespace", utils.ComputeNamespace(ms.Namespace, ms.Spec.Synapse.Namespace),
			)
		} else {
			log.Error(
				err,
				"Error fetching Synapse instance",
				"Synapse Name", ms.Spec.Synapse.Name,
				"Synapse Namespace", utils.ComputeNamespace(ms.Namespace, ms.Spec.Synapse.Namespace),
			)
		}
		return ctrl.Result{}, err
	}

	// Get Synapse Status
	// if !isSynapseRunning(s) {
	// 	err = errors.New("Synapse is not ready")
	// 	log.Error(
	// 		err,
	// 		"Synapse is not ready",
	// 		"Synapse Name", ms.Spec.Synapse.Name,
	// 		"Synapse Namespace", utils.ComputeNamespace(ms.Namespace, ms.Spec.Synapse.Namespace),
	// 	)

	// 	return ctrl.Result{}, err
	// }

	// Get Synapse ServerName
	ms.Status.Synapse.ServerName, err = utils.GetSynapseServerName(s)
	if err != nil {
		log.Error(
			err,
			"Error getting Synapse ServerName",
			"Synapse Name", ms.Spec.Synapse.Name,
			"Synapse Namespace", utils.ComputeNamespace(ms.Namespace, ms.Spec.Synapse.Namespace),
		)
		return ctrl.Result{}, err
	}

	if r, err := r.triggerSynapseReconciliation(&s, ctx); reconc.ShouldHaltOrRequeue(r, err) {
		return reconc.Evaluate(r, err)
	}

	if err := r.updateMautrixSignalStatus(ctx, &ms); err != nil {
		log.Error(err, "Error updating mautrix-signal Status")
		return ctrl.Result{}, err
	}

	// The list of subreconcilers for mautrix-signal will be built next.
	// mautrix-signal is composed of a ConfigMap, a Service, a SA, a RB,
	// a PVC and a Deployment.
	// In addition, a Deployment and a PVC are needed for signald.
	var subreconcilersForMautrixSignal []reconc.SubreconcilerFuncs

	// The user may specify a ConfigMap, containing the config.yaml config
	// file, under Spec.Bridges.MautrixSignal.ConfigMap
	if ms.Spec.ConfigMap.Name != "" {
		// If the user provided a custom mautrix-signal configuration via a
		// ConfigMap, we need to validate that the ConfigMap exists, and
		// create a copy. We also need to edit the mautrix-signal
		// configuration.
		subreconcilersForMautrixSignal = []reconc.SubreconcilerFuncs{
			r.copyInputMautrixSignalConfigMap,
			r.configureMautrixSignalConfigMap,
		}

	} else {
		// If the user hasn't provided a ConfigMap with a custom
		// config.yaml, we create a new ConfigMap with a default
		// config.yaml.
		subreconcilersForMautrixSignal = []reconc.SubreconcilerFuncs{
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
		if r, err := f(&ms, ctx); reconc.ShouldHaltOrRequeue(r, err) {
			return reconc.Evaluate(r, err)
		}
	}

	return ctrl.Result{}, nil
}

func (r *MautrixSignalReconciler) fetchSynapseInstance(
	ctx context.Context,
	ms synapsev1alpha1.MautrixSignal,
) (synapsev1alpha1.Synapse, error) {
	// Validate Synapse instance exists
	s := &synapsev1alpha1.Synapse{}
	keyForSynapse := types.NamespacedName{
		Name:      ms.Spec.Synapse.Name,
		Namespace: utils.ComputeNamespace(ms.Namespace, ms.Spec.Synapse.Namespace),
	}
	if err := r.Get(ctx, keyForSynapse, s); err != nil {
		return synapsev1alpha1.Synapse{}, err
	}

	return *s, nil
}

func (r *MautrixSignalReconciler) triggerSynapseReconciliation(i interface{}, ctx context.Context) (*ctrl.Result, error) {
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

func (r *MautrixSignalReconciler) setFailedState(ctx context.Context, ms *synapsev1alpha1.MautrixSignal, reason string) error {
	ms.Status.State = "FAILED"
	ms.Status.Reason = reason

	return r.updateMautrixSignalStatus(ctx, ms)
}

func (r *MautrixSignalReconciler) updateMautrixSignalStatus(ctx context.Context, ms *synapsev1alpha1.MautrixSignal) error {
	current := &synapsev1alpha1.MautrixSignal{}
	if err := r.Get(
		ctx,
		types.NamespacedName{Name: ms.Name, Namespace: ms.Namespace},
		current,
	); err != nil {
		return err
	}

	if !reflect.DeepEqual(ms.Status, current.Status) {
		if err := r.Status().Patch(ctx, ms, client.MergeFrom(current)); err != nil {
			return err
		}
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MautrixSignalReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&synapsev1alpha1.MautrixSignal{}).
		Complete(r)
}
