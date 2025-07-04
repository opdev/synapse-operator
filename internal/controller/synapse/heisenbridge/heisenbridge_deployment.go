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

	appsv1 "k8s.io/api/apps/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/opdev/subreconciler"
	synapsev1alpha1 "github.com/opdev/synapse-operator/api/synapse/v1alpha1"
	"github.com/opdev/synapse-operator/helpers/reconcile"
	"github.com/opdev/synapse-operator/helpers/utils"
	"github.com/opdev/synapse-operator/internal/templates"
)

// labelsForSynapse returns the labels for selecting the resources
// belonging to the given synapse CR name.
func labelsForHeisenbridge(name string) map[string]string {
	return map[string]string{"app": "heisenbridge", "heisenbridge_cr": name}
}

// reconcileHeisenbridgeDeployment is a function of type FnWithRequest, to
// be called in the main reconciliation loop.
//
// It reconciles the Deployment for Heisenbridge to its desired state.
func (r *HeisenbridgeReconciler) reconcileHeisenbridgeDeployment(ctx context.Context, req ctrl.Request) (*ctrl.Result, error) {
	h := &synapsev1alpha1.Heisenbridge{}
	if r, err := utils.GetResource(ctx, r.Client, req, h); subreconciler.ShouldHaltOrRequeue(r, err) {
		return r, err
	}

	desiredDeployment, err := r.deploymentForHeisenbridge(h)
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

// deploymentForHeisenbridge returns a Heisenbridge Deployment object
func (r *HeisenbridgeReconciler) deploymentForHeisenbridge(h *synapsev1alpha1.Heisenbridge) (*appsv1.Deployment, error) {
	type deploymentExtraValues struct {
		synapsev1alpha1.Heisenbridge
		Labels  map[string]string
		Command []string
	}

	extraValues := deploymentExtraValues{
		Heisenbridge: *h,
		Labels:       labelsForHeisenbridge(h.Name),
		Command:      r.craftHeisenbridgeCommad(*h),
	}

	dep, err := templates.ResourceFromTemplate[deploymentExtraValues, appsv1.Deployment](&extraValues, "heisenbridge_deployment")
	if err != nil {
		return nil, fmt.Errorf("could not get template: %v", err)
	}

	// Set Heisenbridge instance as the owner and controller
	if err := ctrl.SetControllerReference(h, dep, r.Scheme); err != nil {
		return &appsv1.Deployment{}, err
	}
	return dep, nil
}

func (r *HeisenbridgeReconciler) craftHeisenbridgeCommad(h synapsev1alpha1.Heisenbridge) []string {
	command := []string{
		"python",
		"-m",
		"heisenbridge",
	}

	if h.Spec.VerboseLevel > 0 {
		verbosity := "-"
		for i := 1; i <= h.Spec.VerboseLevel; i++ {
			verbosity = verbosity + "v"
		}
		command = append(command, verbosity)
	}

	SynapseName := h.Spec.Synapse.Name
	SynapseNamespace := utils.ComputeNamespace(h.Namespace, h.Spec.Synapse.Namespace)

	command = append(
		command,
		"-c",
		"/data-heisenbridge/heisenbridge.yaml",
		"-l",
		"0.0.0.0",
		"http://"+utils.ComputeFQDN(SynapseName, SynapseNamespace)+":8008",
	)

	return command
}
