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

	appsv1 "k8s.io/api/apps/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/opdev/subreconciler"
	synapsev1alpha1 "github.com/opdev/synapse-operator/api/synapse/v1alpha1"
	"github.com/opdev/synapse-operator/helpers/reconcile"
	"github.com/opdev/synapse-operator/helpers/utils"
	"github.com/opdev/synapse-operator/internal/templates"
)

// reconcileSynapseDeployment is a function of type FnWithRequest, to be
// called in the main reconciliation loop.
//
// It reconciles the Deployment for Synapse to its desired state.
func (r *SynapseReconciler) reconcileSynapseDeployment(ctx context.Context, req ctrl.Request) (*ctrl.Result, error) {
	s := &synapsev1alpha1.Synapse{}
	if r, err := utils.GetResource(ctx, r.Client, req, s); subreconciler.ShouldHaltOrRequeue(r, err) {
		return r, err
	}

	depl, err := r.deploymentForSynapse(s)
	if err != nil {
		return subreconciler.RequeueWithError(err)
	}

	if err := reconcile.ReconcileResource(
		ctx,
		r.Client,
		depl,
		&appsv1.Deployment{},
	); err != nil {
		return subreconciler.RequeueWithError(err)
	}

	return subreconciler.ContinueReconciling()
}

// deploymentForSynapse returns a synapse Deployment object
func (r *SynapseReconciler) deploymentForSynapse(s *synapsev1alpha1.Synapse) (*appsv1.Deployment, error) {
	type deploymentExtraValues struct {
		synapsev1alpha1.Synapse
		Labels map[string]string
	}

	extraValues := deploymentExtraValues{
		Synapse: *s,
		Labels:  labelsForSynapse(s.Name),
	}

	dep, err := templates.ResourceFromTemplate[deploymentExtraValues, appsv1.Deployment](&extraValues, "synapse_deployment")
	if err != nil {
		return nil, fmt.Errorf("could not get template: %v", err)
	}

	// Set Synapse instance as the owner and controller
	if err := ctrl.SetControllerReference(s, dep, r.Scheme); err != nil {
		return &appsv1.Deployment{}, err
	}

	return dep, nil
}
