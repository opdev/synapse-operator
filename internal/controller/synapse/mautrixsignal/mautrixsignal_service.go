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
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/opdev/subreconciler"
	synapsev1alpha1 "github.com/opdev/synapse-operator/api/synapse/v1alpha1"
	"github.com/opdev/synapse-operator/helpers/reconcile"
	"github.com/opdev/synapse-operator/helpers/utils"
	"github.com/opdev/synapse-operator/internal/templates"
)

// reconcileMautrixSignalService is a function of type FnWithRequest, to
// be called in the main reconciliation loop.
//
// It reconciles the Service for mautrix-signal to its desired state.
func (r *MautrixSignalReconciler) reconcileMautrixSignalService(ctx context.Context, req ctrl.Request) (*ctrl.Result, error) {
	ms := &synapsev1alpha1.MautrixSignal{}
	if r, err := utils.GetResource(ctx, r.Client, req, ms); subreconciler.ShouldHaltOrRequeue(r, err) {
		return r, err
	}

	desiredService, err := r.serviceForMautrixSignal(ms)
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

// serviceForMautrixSignal returns a mautrix-signal Service object
func (r *MautrixSignalReconciler) serviceForMautrixSignal(ms *synapsev1alpha1.MautrixSignal) (*corev1.Service, error) {
	type serviceExtraValues struct {
		synapsev1alpha1.MautrixSignal
		Labels     map[string]string
		PortName   string
		Port       int
		TargetPort int
	}

	extraValues := serviceExtraValues{
		MautrixSignal: *ms,
		Labels:        labelsForMautrixSignal(ms.Name),
		PortName:      "mautrix-signal",
		Port:          29328,
		TargetPort:    29328,
	}

	service, err := templates.ResourceFromTemplate[serviceExtraValues, corev1.Service](&extraValues, "service")
	if err != nil {
		return nil, fmt.Errorf("could not get template: %v", err)
	}

	// Set MautrixSignal instance as the owner and controller
	if err := ctrl.SetControllerReference(ms, service, r.Scheme); err != nil {
		return &corev1.Service{}, err
	}
	return service, nil
}
