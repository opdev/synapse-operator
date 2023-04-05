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

package utils

import (
	"context"
	"errors"
	"reflect"
	"strings"

	"github.com/opdev/subreconciler"
	synapsev1alpha1 "github.com/opdev/synapse-operator/apis/synapse/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

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

type resourceKeyType int
type kubeClientKeyType int
type runtimeSchemeKeyType int

var resourceKey resourceKeyType
var kubeClientKey kubeClientKeyType
var runtimeSchemeKey runtimeSchemeKeyType

func AddValuesToContext(ctx context.Context, kubeClient client.Client, runtimeScheme *runtime.Scheme, resource client.Object) context.Context {
	newContext := context.WithValue(ctx, resourceKey, resource)
	newContext = context.WithValue(newContext, kubeClientKey, kubeClient)
	newContext = context.WithValue(newContext, runtimeSchemeKey, runtimeScheme)

	return newContext
}

func GetValuesFromContext(ctx context.Context) (client.Client, client.Object, error) {
	var resource client.Object

	kubeClient, ok := ctx.Value(kubeClientKey).(client.Client)
	if !ok {
		err := errors.New("error getting Kubernetes client from context")
		return nil, resource, err
	}

	resource, ok = ctx.Value(resourceKey).(client.Object)
	if !ok {
		err := errors.New("error getting bridge resource from context")
		return nil, resource, err
	}

	return kubeClient, resource, nil
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
		Namespace: ComputeNamespace(resource.GetName(), resource.GetSynapseNamespace()),
	}
	return kubeClient.Get(ctx, keyForSynapse, s)
}

// TriggerSynapseReconciliation is a function of type subreconciler.FnWithRequest
// Bridges should trigger the reconciliation of their associated Synapse server
// so that Synapse can add the bridge as an application service in its configuration.
func TriggerSynapseReconciliation(ctx context.Context, req ctrl.Request) (*ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)
	kubeClient, resourceObject, err := GetValuesFromContext(ctx)
	if err != nil {
		return subreconciler.RequeueWithError(err)
	}

	resource, ok := resourceObject.(Bridge)
	if !ok {
		err := errors.New("error in type assertion")
		log.Error(err, "Error in type assertion")
		return subreconciler.RequeueWithError(err)
	}

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
