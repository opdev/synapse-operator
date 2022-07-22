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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	synapsev1alpha1 "github.com/opdev/synapse-operator/apis/synapse/v1alpha1"
	reconc "github.com/opdev/synapse-operator/helpers/reconcileresults"
)

// reconcileMautrixSignalService is a function of type subreconcilerFuncs, to
// be called in the main reconciliation loop.
//
// It reconciles the Service for mautrix-signal to its desired state.
func (r *SynapseReconciler) reconcileMautrixSignalService(synapse *synapsev1alpha1.Synapse, ctx context.Context) (*ctrl.Result, error) {
	objectMetaMautrixSignal := setObjectMeta(r.GetMautrixSignalResourceName(*synapse), synapse.Namespace, map[string]string{})
	if err := r.reconcileResource(
		ctx,
		r.serviceForMautrixSignal,
		synapse,
		&corev1.Service{},
		objectMetaMautrixSignal,
	); err != nil {
		return reconc.RequeueWithError(err)
	}

	return reconc.ContinueReconciling()
}

// serviceForMautrixSignal returns a mautrix-signal Service object
func (r *SynapseReconciler) serviceForMautrixSignal(s *synapsev1alpha1.Synapse, objectMeta metav1.ObjectMeta) (client.Object, error) {
	service := &corev1.Service{
		ObjectMeta: objectMeta,
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name:       "mautrix-signal",
				Protocol:   corev1.ProtocolTCP,
				Port:       29328,
				TargetPort: intstr.FromInt(29328),
			}},
			Selector: labelsForMautrixSignal(s.Name),
			Type:     corev1.ServiceTypeClusterIP,
		},
	}
	// Set Synapse instance as the owner and controller
	if err := ctrl.SetControllerReference(s, service, r.Scheme); err != nil {
		return &corev1.Service{}, err
	}
	return service, nil
}
