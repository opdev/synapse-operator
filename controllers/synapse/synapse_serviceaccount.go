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
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	synapsev1alpha1 "github.com/opdev/synapse-operator/apis/synapse/v1alpha1"
)

// serviceAccountForSynapse returns a synapse ServiceAccount object
func (r *SynapseReconciler) serviceAccountForSynapse(s *synapsev1alpha1.Synapse, objectMeta metav1.ObjectMeta) (client.Object, error) {
	sa := &corev1.ServiceAccount{
		ObjectMeta: objectMeta,
	}

	// Set Synapse instance as the owner and controller
	if err := ctrl.SetControllerReference(s, sa, r.Scheme); err != nil {
		return &corev1.ServiceAccount{}, err
	}
	return sa, nil
}

// roleBindingForSynapse returns a synapse RoleBinding object
func (r *SynapseReconciler) roleBindingForSynapse(s *synapsev1alpha1.Synapse, objectMeta metav1.ObjectMeta) (client.Object, error) {
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
