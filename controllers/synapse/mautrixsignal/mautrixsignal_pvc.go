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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	synapsev1alpha1 "github.com/opdev/synapse-operator/apis/synapse/v1alpha1"
	"github.com/opdev/synapse-operator/helpers/reconcile"
	reconc "github.com/opdev/synapse-operator/helpers/reconcileresults"
)

// reconcileMautrixSignalPVC is a function of type subreconcilerFuncs, to be
// called in the main reconciliation loop.
//
// It reconciles the PVC for mautrix-signal to its desired state.
func (r *MautrixSignalReconciler) reconcileMautrixSignalPVC(obj client.Object, ctx context.Context) (*ctrl.Result, error) {
	ms := obj.(*synapsev1alpha1.MautrixSignal)

	objectMetaMautrixSignal := reconcile.SetObjectMeta(ms.Name, ms.Namespace, map[string]string{})

	desiredPVC, err := r.persistentVolumeClaimForMautrixSignal(ms, objectMetaMautrixSignal)
	if err != nil {
		return reconc.RequeueWithError(err)
	}

	if err := reconcile.ReconcileResource(
		ctx,
		r.Client,
		desiredPVC,
		&corev1.PersistentVolumeClaim{},
	); err != nil {
		return reconc.RequeueWithError(err)
	}

	return reconc.ContinueReconciling()
}

// persistentVolumeClaimForMautrixSignal returns a mautrix-signal PVC object
func (r *MautrixSignalReconciler) persistentVolumeClaimForMautrixSignal(ms *synapsev1alpha1.MautrixSignal, objectMeta metav1.ObjectMeta) (*corev1.PersistentVolumeClaim, error) {
	pvcmode := corev1.PersistentVolumeFilesystem

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: objectMeta,
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{"ReadWriteOnce"},
			VolumeMode:  &pvcmode,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					"storage": *resource.NewQuantity(5*1024*1024*1024, resource.BinarySI),
				},
			},
		},
	}

	// Set Synapse instance as the owner and controller
	if err := ctrl.SetControllerReference(ms, pvc, r.Scheme); err != nil {
		return &corev1.PersistentVolumeClaim{}, err
	}
	return pvc, nil
}
