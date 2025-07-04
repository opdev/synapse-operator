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

package heisenbridge

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/opdev/subreconciler"
	synapsev1alpha1 "github.com/opdev/synapse-operator/api/synapse/v1alpha1"
	"github.com/opdev/synapse-operator/helpers/reconcile"
	"github.com/opdev/synapse-operator/helpers/utils"
	"github.com/opdev/synapse-operator/internal/templates"
)

// reconcileHeisenbridgeService is a function of type FnWithRequest, to be
// called in the main reconciliation loop.
//
// It reconciles the Service for Heisenbridge to its desired state.
func (r *HeisenbridgeReconciler) reconcileHeisenbridgeService(ctx context.Context, req ctrl.Request) (*ctrl.Result, error) {
	h := &synapsev1alpha1.Heisenbridge{}
	if r, err := utils.GetResource(ctx, r.Client, req, h); subreconciler.ShouldHaltOrRequeue(r, err) {
		return r, err
	}

	desiredService, err := r.serviceForHeisenbridge(h)
	if err != nil {
		return subreconciler.RequeueWithError(err)
	}

	if err := reconcile.ReconcileResource(
		ctx,
		r.Client,
		desiredService,
		&corev1.Service{},
	); err != nil {
		return subreconciler.RequeueWithError(err)
	}

	return subreconciler.ContinueReconciling()
}

// serviceForSynapse returns a Heisenbridge Service object
func (r *HeisenbridgeReconciler) serviceForHeisenbridge(h *synapsev1alpha1.Heisenbridge) (*corev1.Service, error) {
	type serviceExtraValues struct {
		synapsev1alpha1.Heisenbridge
		Labels     map[string]string
		PortName   string
		Port       int
		TargetPort int
	}

	extraValues := serviceExtraValues{
		Heisenbridge: *h,
		Labels:       labelsForHeisenbridge(h.Name),
		PortName:     "heisenbridge",
		Port:         9898,
		TargetPort:   9898,
	}

	service, err := templates.ResourceFromTemplate[serviceExtraValues, corev1.Service](&extraValues, "service")
	if err != nil {
		return nil, fmt.Errorf("could not get template: %v", err)
	}

	// Set Heisenbridge instance as the owner and controller
	if err := ctrl.SetControllerReference(h, service, r.Scheme); err != nil {
		return &corev1.Service{}, err
	}
	return service, nil
}
