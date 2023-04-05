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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	subreconciler "github.com/opdev/subreconciler"
	synapsev1alpha1 "github.com/opdev/synapse-operator/apis/synapse/v1alpha1"
	"github.com/opdev/synapse-operator/helpers/reconcile"
	"github.com/opdev/synapse-operator/helpers/utils"
	"github.com/opdev/synapse-operator/internal/templates"
)

// reconcileSynapseServiceAccount is a function of type FnWithRequest, to
// be called in the main reconciliation loop.
//
// It reconciles the ServiceAccount for synapse to its desired state.
func (r *SynapseReconciler) reconcileSynapseServiceAccount(ctx context.Context, req ctrl.Request) (*ctrl.Result, error) {
	s := &synapsev1alpha1.Synapse{}
	if r, err := utils.GetResource(ctx, r.Client, req, s); subreconciler.ShouldHaltOrRequeue(r, err) {
		return r, err
	}

	desiredServiceAccount, err := r.serviceAccountForSynapse(s)
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

// serviceAccountForSynapse returns a synapse ServiceAccount object
func (r *SynapseReconciler) serviceAccountForSynapse(s *synapsev1alpha1.Synapse) (*corev1.ServiceAccount, error) {
	// TODO: https://github.com/opdev/synapse-operator/issues/19
	sa, err := templates.ResourceFromTemplate[synapsev1alpha1.Synapse, corev1.ServiceAccount](s, "serviceaccount")
	if err != nil {
		return nil, fmt.Errorf("could not get template: %v", err)
	}

	// Set Synapse instance as the owner and controller
	if err := ctrl.SetControllerReference(s, sa, r.Scheme); err != nil {
		return &corev1.ServiceAccount{}, err
	}
	return sa, nil
}

// reconcileSynapseRoleBinding is a function of type FnWithRequest, to be
// called in the main reconciliation loop.
//
// It reconciles the RoleBinding for synapse to its desired state.
func (r *SynapseReconciler) reconcileSynapseRoleBinding(ctx context.Context, req ctrl.Request) (*ctrl.Result, error) {
	s := &synapsev1alpha1.Synapse{}
	if r, err := utils.GetResource(ctx, r.Client, req, s); subreconciler.ShouldHaltOrRequeue(r, err) {
		return r, err
	}

	desiredRoleBinding, err := r.roleBindingForSynapse(s)
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

// roleBindingForSynapse returns a synapse RoleBinding object
func (r *SynapseReconciler) roleBindingForSynapse(s *synapsev1alpha1.Synapse) (*rbacv1.RoleBinding, error) {
	// TODO: https://github.com/opdev/synapse-operator/issues/19
	rb, err := templates.ResourceFromTemplate[synapsev1alpha1.Synapse, rbacv1.RoleBinding](s, "rolebinding")
	if err != nil {
		return nil, fmt.Errorf("could not get template: %v", err)
	}

	// Set Synapse instance as the owner and controller
	if err := ctrl.SetControllerReference(s, rb, r.Scheme); err != nil {
		return &rbacv1.RoleBinding{}, err
	}
	return rb, nil
}
