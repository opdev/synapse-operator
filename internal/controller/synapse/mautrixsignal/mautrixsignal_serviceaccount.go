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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/opdev/subreconciler"
	synapsev1alpha1 "github.com/opdev/synapse-operator/api/synapse/v1alpha1"
	"github.com/opdev/synapse-operator/helpers/reconcile"
	"github.com/opdev/synapse-operator/helpers/utils"
	"github.com/opdev/synapse-operator/internal/templates"
)

// reconcileMautrixSignalServiceAccount is a function of type
// FnWithRequest, to be called in the main reconciliation loop.
//
// It reconciles the ServiceAccount for mautrix-signal to its desired state.
func (r *MautrixSignalReconciler) reconcileMautrixSignalServiceAccount(
	ctx context.Context,
	req ctrl.Request,
) (*ctrl.Result, error) {
	ms := &synapsev1alpha1.MautrixSignal{}
	if r, err := utils.GetResource(ctx, r.Client, req, ms); subreconciler.ShouldHaltOrRequeue(r, err) {
		return r, err
	}

	desiredServiceAccount, err := r.serviceAccountForMautrixSignal(ms)
	if err != nil {
		return subreconciler.RequeueWithError(err)
	}

	if err := reconcile.ReconcileResource(
		ctx,
		r.Client,
		desiredServiceAccount,
		&corev1.ServiceAccount{},
	); err != nil {
		return subreconciler.RequeueWithError(err)
	}

	return subreconciler.ContinueReconciling()
}

// serviceAccountForMautrixSignal returns a ServiceAccount object for running the mautrix-signal bridge
func (r *MautrixSignalReconciler) serviceAccountForMautrixSignal(
	ms *synapsev1alpha1.MautrixSignal,
) (client.Object, error) {
	// TODO: https://github.com/opdev/synapse-operator/issues/19
	sa, err := templates.ResourceFromTemplate[synapsev1alpha1.MautrixSignal, corev1.ServiceAccount](ms, "serviceaccount")
	if err != nil {
		return nil, fmt.Errorf("could not get template: %v", err)
	}

	// Set MautrixSignal instance as the owner and controller
	if err := ctrl.SetControllerReference(ms, sa, r.Scheme); err != nil {
		return &corev1.ServiceAccount{}, err
	}
	return sa, nil
}

// reconcileMautrixSignalRoleBinding is a function of type FnWithRequest,
// to be called in the main reconciliation loop.
//
// It reconciles the RoleBinding for mautrix-signal to its desired state.
func (r *MautrixSignalReconciler) reconcileMautrixSignalRoleBinding(
	ctx context.Context,
	req ctrl.Request,
) (*ctrl.Result, error) {
	ms := &synapsev1alpha1.MautrixSignal{}
	if r, err := utils.GetResource(ctx, r.Client, req, ms); subreconciler.ShouldHaltOrRequeue(r, err) {
		return r, err
	}

	desiredRoleBinding, err := r.roleBindingForMautrixSignal(ms)
	if err != nil {
		return subreconciler.RequeueWithError(err)
	}

	if err := reconcile.ReconcileResource(
		ctx,
		r.Client,
		desiredRoleBinding,
		&rbacv1.RoleBinding{},
	); err != nil {
		return subreconciler.RequeueWithError(err)
	}

	return subreconciler.ContinueReconciling()
}

// roleBindingForMautrixSignal returns a RoleBinding object for the mautrix-signal bridge
func (r *MautrixSignalReconciler) roleBindingForMautrixSignal(
	ms *synapsev1alpha1.MautrixSignal,
) (*rbacv1.RoleBinding, error) {
	// TODO: https://github.com/opdev/synapse-operator/issues/19
	rb, err := templates.ResourceFromTemplate[synapsev1alpha1.MautrixSignal, rbacv1.RoleBinding](ms, "rolebinding")
	if err != nil {
		return nil, fmt.Errorf("could not get template: %v", err)
	}

	// Set MautrixSignal instance as the owner and controller
	if err := ctrl.SetControllerReference(ms, rb, r.Scheme); err != nil {
		return &rbacv1.RoleBinding{}, err
	}
	return rb, nil
}
