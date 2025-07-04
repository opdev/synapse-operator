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

package synapse

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	subreconciler "github.com/opdev/subreconciler"
	synapsev1alpha1 "github.com/opdev/synapse-operator/api/synapse/v1alpha1"
	"github.com/opdev/synapse-operator/helpers/reconcile"
	"github.com/opdev/synapse-operator/helpers/utils"
	"github.com/opdev/synapse-operator/internal/templates"
)

// reconcileSynapsePVC is a function of type FnWithRequest, to be called
// in the main reconciliation loop.
//
// It reconciles the PVC for synapse to its desired state.
func (r *SynapseReconciler) reconcileSynapsePVC(ctx context.Context, req ctrl.Request) (*ctrl.Result, error) {
	s := &synapsev1alpha1.Synapse{}
	if r, err := utils.GetResource(ctx, r.Client, req, s); subreconciler.ShouldHaltOrRequeue(r, err) {
		return r, err
	}

	desiredPVC, err := r.persistentVolumeClaimForSynapse(s)
	if err != nil {
		return subreconciler.RequeueWithError(err)
	}

	if err := reconcile.ReconcileResource(
		ctx,
		r.Client,
		desiredPVC,
		&corev1.PersistentVolumeClaim{},
	); err != nil {
		return subreconciler.RequeueWithError(err)
	}

	return subreconciler.ContinueReconciling()
}

// persistentVolumeClaimForSynapse returns a synapse PVC object
func (r *SynapseReconciler) persistentVolumeClaimForSynapse(s *synapsev1alpha1.Synapse) (*corev1.PersistentVolumeClaim, error) {
	pvc, err := templates.ResourceFromTemplate[synapsev1alpha1.Synapse, corev1.PersistentVolumeClaim](s, "pvc")
	if err != nil {
		return nil, fmt.Errorf("could not get template: %v", err)
	}

	// Set Synapse instance as the owner and controller
	if err := ctrl.SetControllerReference(s, pvc, r.Scheme); err != nil {
		return &corev1.PersistentVolumeClaim{}, err
	}
	return pvc, nil
}
