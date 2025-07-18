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

package utils

import (
	"context"
	"errors"
	"reflect"
	"strings"

	"github.com/opdev/subreconciler"
	synapsev1alpha1 "github.com/opdev/synapse-operator/api/synapse/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const synapseBridgeFinalizer = "synapse.opdev.io/finalizer"

func ComputeFQDN(name string, namespace string) string {
	return strings.Join([]string{name, namespace, "svc", "cluster", "local"}, ".")
}

func GetSynapseServerName(s synapsev1alpha1.Synapse) (string, error) {
	if s.Status.HomeserverConfiguration.ServerName != "" {
		return s.Status.HomeserverConfiguration.ServerName, nil
	}
	err := errors.New("ServerName not yet populated")
	return "", err
}

func UpdateSynapseStatus(ctx context.Context, kubeClient client.Client, s *synapsev1alpha1.Synapse) error {
	current := &synapsev1alpha1.Synapse{}

	if err := kubeClient.Get(
		ctx,
		types.NamespacedName{Name: s.Name, Namespace: s.Namespace},
		current,
	); err != nil {
		return err
	}

	if !reflect.DeepEqual(s.Status, current.Status) {
		if err := kubeClient.Status().Patch(ctx, s, client.MergeFrom(current)); err != nil {
			return err
		}
	}

	return nil
}

// Matrix Bridges should implement the Bridge interface
type Bridge interface {
	client.Object // *synapsev1alpha1.Heisenbridge | *synapsev1alpha1.MautrixSignal
	GetSynapseName() string
	GetSynapseNamespace() string
}

func FetchSynapseInstance(
	ctx context.Context,
	kubeClient client.Client,
	resource Bridge,
	s *synapsev1alpha1.Synapse,
) error {
	// Validate Synapse instance exists
	keyForSynapse := types.NamespacedName{
		Name:      resource.GetSynapseName(),
		Namespace: ComputeNamespace(resource.GetNamespace(), resource.GetSynapseNamespace()),
	}
	return kubeClient.Get(ctx, keyForSynapse, s)
}

// TriggerSynapseReconciliation returns a function of type subreconciler.FnWithRequest
// Bridges should trigger the reconciliation of their associated Synapse server
// so that Synapse can add the bridge as an application service in its configuration.
func TriggerSynapseReconciliation(
	kubeClient client.Client,
	resource Bridge,
) func(context.Context, ctrl.Request) (*ctrl.Result, error) {
	return func(ctx context.Context, req ctrl.Request) (*ctrl.Result, error) {
		log := logf.FromContext(ctx)

		if r, err := GetResource(ctx, kubeClient, req, resource); subreconciler.ShouldHaltOrRequeue(r, err) {
			return r, err
		}

		s := synapsev1alpha1.Synapse{}
		if err := FetchSynapseInstance(ctx, kubeClient, resource, &s); err != nil {
			log.Error(err, "Error getting Synapse instance")
			return subreconciler.RequeueWithError(err)
		}

		s.Status.NeedsReconcile = true

		if err := UpdateSynapseStatus(ctx, kubeClient, &s); err != nil {
			return subreconciler.RequeueWithError(err)
		}

		return subreconciler.ContinueReconciling()
	}
}

// HandleDelete returns a function of type subreconciler.FnWithRequest
//
// Bridges need to trigger the reconciliation of their associated Synapse homeserver
// so that Synapse can remove the bridge from the list of application services in its configuration.
func HandleDelete(kubeClient client.Client, resource Bridge) func(context.Context, ctrl.Request) (*ctrl.Result, error) {
	return func(ctx context.Context, req ctrl.Request) (*ctrl.Result, error) {
		log := logf.FromContext(ctx)

		if r, err := GetResource(ctx, kubeClient, req, resource); subreconciler.ShouldHaltOrRequeue(r, err) {
			return r, err
		}

		// Check if resource is marked to be deleted
		if resource.GetDeletionTimestamp() != nil {
			if controllerutil.ContainsFinalizer(resource, synapseBridgeFinalizer) {
				// Trigger the reconciliation of the associated Synapse instance
				s := synapsev1alpha1.Synapse{}
				if err := FetchSynapseInstance(ctx, kubeClient, resource, &s); err != nil {
					log.Error(err, "Error getting Synapse instance. Continuing clean-up without triggering reconciliation")
				} else {
					s.Status.NeedsReconcile = true

					if err := UpdateSynapseStatus(ctx, kubeClient, &s); err != nil {
						log.Error(err, "Error updating Synapse Status")
						return subreconciler.RequeueWithError(err)
					}
				}

				// Remove finalizer
				if ok := controllerutil.RemoveFinalizer(resource, synapseBridgeFinalizer); !ok {
					err := errors.New("error removing finalizer")
					log.Error(err, "Error removing finalizer")
					return subreconciler.RequeueWithError(err)
				}

				if err := kubeClient.Update(ctx, resource); err != nil {
					log.Error(err, "error updating Status")
					return subreconciler.RequeueWithError(err)
				}
			}
			return subreconciler.DoNotRequeue()
		}
		return subreconciler.ContinueReconciling()
	}
}

// AddFinalizer returns a function of type subreconciler.FnWithRequest
func AddFinalizer(kubeClient client.Client, resource Bridge) func(context.Context, ctrl.Request) (*ctrl.Result, error) {
	return func(ctx context.Context, req ctrl.Request) (*ctrl.Result, error) {
		log := logf.FromContext(ctx)

		if r, err := GetResource(ctx, kubeClient, req, resource); subreconciler.ShouldHaltOrRequeue(r, err) {
			return r, err
		}

		if !controllerutil.ContainsFinalizer(resource, synapseBridgeFinalizer) {
			log.Info("Adding Finalizer for Memcached")
			if ok := controllerutil.AddFinalizer(resource, synapseBridgeFinalizer); !ok {
				err := errors.New("error adding finalizer")
				log.Error(err, "Failed to add finalizer into the custom resource")
				return subreconciler.RequeueWithError(err)
			}

			if err := kubeClient.Update(ctx, resource); err != nil {
				log.Error(err, "Failed to update custom resource to add finalizer")
				return subreconciler.RequeueWithError(err)
			}
		}

		return subreconciler.ContinueReconciling()
	}
}
