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

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	synapsev1alpha1 "github.com/opdev/synapse-operator/apis/synapse/v1alpha1"
	reconc "github.com/opdev/synapse-operator/helpers/reconcileresults"
)

// reconcileSynapseServiceAccount is a function of type subreconcilerFuncs, to
// be called in the main reconciliation loop.
//
// It reconciles the ServiceAccount for synapse to its desired state.
func (r *SynapseReconciler) reconcileSynapseServiceAccount(synapse *synapsev1alpha1.Synapse, ctx context.Context) (*ctrl.Result, error) {
	objectMetaForSynapse := setObjectMeta(synapse.Name, synapse.Namespace, map[string]string{})
	if err := r.reconcileResource(
		ctx,
		r.serviceAccountForSynapse,
		synapse,
		&corev1.ServiceAccount{},
		objectMetaForSynapse,
	); err != nil {
		return reconc.RequeueWithError(err)
	}
	return reconc.ContinueReconciling()
}

// serviceAccountForSynapse returns a synapse ServiceAccount object
func (r *SynapseReconciler) serviceAccountForSynapse(s *synapsev1alpha1.Synapse, objectMeta metav1.ObjectMeta) (client.Object, error) {
	// TODO: https://github.com/opdev/synapse-operator/issues/19
	sa := &corev1.ServiceAccount{
		ObjectMeta: objectMeta,
	}

	// Set Synapse instance as the owner and controller
	if err := ctrl.SetControllerReference(s, sa, r.Scheme); err != nil {
		return &corev1.ServiceAccount{}, err
	}
	return sa, nil
}

// reconcileSynapseRoleBinding is a function of type subreconcilerFuncs, to be
// called in the main reconciliation loop.
//
// It reconciles the RoleBinding for synapse to its desired state.
func (r *SynapseReconciler) reconcileSynapseRoleBinding(synapse *synapsev1alpha1.Synapse, ctx context.Context) (*ctrl.Result, error) {
	objectMetaForSynapse := setObjectMeta(synapse.Name, synapse.Namespace, map[string]string{})

	if err := r.reconcileResource(
		ctx,
		r.roleBindingForSynapse,
		synapse,
		&rbacv1.RoleBinding{},
		objectMetaForSynapse,
	); err != nil {
		return reconc.RequeueWithError(err)
	}
	return reconc.ContinueReconciling()
}

// roleBindingForSynapse returns a synapse RoleBinding object
func (r *SynapseReconciler) roleBindingForSynapse(s *synapsev1alpha1.Synapse, objectMeta metav1.ObjectMeta) (client.Object, error) {
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
	if err := ctrl.SetControllerReference(s, rb, r.Scheme); err != nil {
		return &rbacv1.RoleBinding{}, err
	}
	return rb, nil
}
