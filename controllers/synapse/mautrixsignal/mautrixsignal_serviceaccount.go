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

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	synapsev1alpha1 "github.com/opdev/synapse-operator/apis/synapse/v1alpha1"
	"github.com/opdev/synapse-operator/helpers/reconcile"
	reconc "github.com/opdev/synapse-operator/helpers/reconcileresults"
)

// reconcileMautrixSignalServiceAccount is a function of type
// subreconcilerFuncs, to be called in the main reconciliation loop.
//
// It reconciles the ServiceAccount for mautrix-signal to its desired state.
func (r *MautrixSignalReconciler) reconcileMautrixSignalServiceAccount(obj client.Object, ctx context.Context) (*ctrl.Result, error) {
	ms := obj.(*synapsev1alpha1.MautrixSignal)

	objectMetaMautrixSignal := reconcile.SetObjectMeta(ms.Name, ms.Namespace, map[string]string{})

	desiredServiceAccount, err := r.serviceAccountForMautrixSignal(ms, objectMetaMautrixSignal)
	if err != nil {
		return reconc.RequeueWithError(err)
	}

	if err := reconcile.ReconcileResource(
		ctx,
		r.Client,
		desiredServiceAccount,
		&corev1.ServiceAccount{},
	); err != nil {
		return reconc.RequeueWithError(err)
	}

	return reconc.ContinueReconciling()
}

// serviceAccountForMautrixSignal returns a ServiceAccount object for running the mautrix-signal bridge
func (r *MautrixSignalReconciler) serviceAccountForMautrixSignal(obj client.Object, objectMeta metav1.ObjectMeta) (client.Object, error) {
	ms := obj.(*synapsev1alpha1.MautrixSignal)

	// TODO: https://github.com/opdev/synapse-operator/issues/19
	sa := &corev1.ServiceAccount{
		ObjectMeta: objectMeta,
	}

	// Set Synapse instance as the owner and controller
	if err := ctrl.SetControllerReference(ms, sa, r.Scheme); err != nil {
		return &corev1.ServiceAccount{}, err
	}
	return sa, nil
}

// reconcileMautrixSignalRoleBinding is a function of type subreconcilerFuncs,
// to be called in the main reconciliation loop.
//
// It reconciles the RoleBinding for mautrix-signal to its desired state.
func (r *MautrixSignalReconciler) reconcileMautrixSignalRoleBinding(obj client.Object, ctx context.Context) (*ctrl.Result, error) {
	ms := obj.(*synapsev1alpha1.MautrixSignal)

	objectMetaMautrixSignal := reconcile.SetObjectMeta(ms.Name, ms.Namespace, map[string]string{})

	desiredRoleBinding, err := r.roleBindingForMautrixSignal(ms, objectMetaMautrixSignal)
	if err != nil {
		return reconc.RequeueWithError(err)
	}

	if err := reconcile.ReconcileResource(
		ctx,
		r.Client,
		desiredRoleBinding,
		&rbacv1.RoleBinding{},
	); err != nil {
		return reconc.RequeueWithError(err)
	}

	return reconc.ContinueReconciling()
}

// roleBindingForMautrixSignal returns a RoleBinding object for the mautrix-signal bridge
func (r *MautrixSignalReconciler) roleBindingForMautrixSignal(ms *synapsev1alpha1.MautrixSignal, objectMeta metav1.ObjectMeta) (*rbacv1.RoleBinding, error) {
	// TODO: https://github.com/opdev/synapse-operator/issues/19
	rb := &rbacv1.RoleBinding{
		ObjectMeta: objectMeta,
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "system:openshift:scc:anyuid",
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      objectMeta.Name,
			Namespace: objectMeta.Namespace,
		}},
	}

	// Set Synapse instance as the owner and controller
	if err := ctrl.SetControllerReference(ms, rb, r.Scheme); err != nil {
		return &rbacv1.RoleBinding{}, err
	}
	return rb, nil
}
