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

	appsv1 "k8s.io/api/apps/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/opdev/subreconciler"
	synapsev1alpha1 "github.com/opdev/synapse-operator/api/synapse/v1alpha1"
	"github.com/opdev/synapse-operator/helpers/reconcile"
	"github.com/opdev/synapse-operator/helpers/utils"
	"github.com/opdev/synapse-operator/internal/templates"
)

// labelsForMautrixSignal returns the labels for selecting the resources
// belonging to the given synapse CR name.
func labelsForMautrixSignal(name string) map[string]string {
	return map[string]string{"app": "mautrix-signal", "mautrixsignal_cr": name}
}

// reconcileMautrixSignalDeployment is a function of type FnWithRequest,
// to be called in the main reconciliation loop.
//
// It reconciles the Deployment for mautrix-signal to its desired state.
func (r *MautrixSignalReconciler) reconcileMautrixSignalDeployment(
	ctx context.Context,
	req ctrl.Request,
) (*ctrl.Result, error) {
	ms := &synapsev1alpha1.MautrixSignal{}
	if r, err := utils.GetResource(ctx, r.Client, req, ms); subreconciler.ShouldHaltOrRequeue(r, err) {
		return r, err
	}

	desiredDeployment, err := r.deploymentForMautrixSignal(ms)
	if err != nil {
		return subreconciler.RequeueWithError(err)
	}

	if err := reconcile.ReconcileResource(
		ctx,
		r.Client,
		desiredDeployment,
		&appsv1.Deployment{},
	); err != nil {
		return subreconciler.RequeueWithError(err)
	}

	return subreconciler.ContinueReconciling()
}

// deploymentForMautrixSignal returns a Deployment object for the mautrix-signal bridge
func (r *MautrixSignalReconciler) deploymentForMautrixSignal(
	ms *synapsev1alpha1.MautrixSignal,
) (*appsv1.Deployment, error) {
	type deploymentExtraValues struct {
		synapsev1alpha1.MautrixSignal
		Labels map[string]string
	}

	extraValues := deploymentExtraValues{
		MautrixSignal: *ms,
		Labels:        labelsForMautrixSignal(ms.Name),
	}

	dep, err := templates.ResourceFromTemplate[deploymentExtraValues, appsv1.Deployment](
		&extraValues,
		"mautrixsignal_deployment",
	)
	if err != nil {
		return nil, fmt.Errorf("could not get template: %v", err)
	}

	// Set MautrixSignal instance as the owner and controller
	if err := ctrl.SetControllerReference(ms, dep, r.Scheme); err != nil {
		return &appsv1.Deployment{}, err
	}
	return dep, nil
}
