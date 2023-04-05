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
	"time"

	"github.com/opdev/subreconciler"
	synapsev1alpha1 "github.com/opdev/synapse-operator/apis/synapse/v1alpha1"
	"github.com/opdev/synapse-operator/helpers/reconcile"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

func GetResource(
	ctx context.Context,
	kubeClient client.Client,
	req ctrl.Request,
	resource client.Object,
) (*ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)

	if err := kubeClient.Get(ctx, req.NamespacedName, resource); err != nil {
		if k8serrors.IsNotFound(err) {
			// we'll ignore not-found errors, since they can't be fixed by an immediate
			// requeue (we'll need to wait for a new notification), and we can get them
			// on deleted requests.
			log.Error(
				err,
				"Cannot find resource - has it been deleted ?",
				"Name", resource.GetName(),
				"Namespace", resource.GetNamespace(),
			)
			return subreconciler.DoNotRequeue()
		}
		log.Error(
			err,
			"Error fetching resource",
			"Name", resource.GetName(),
			"Namespace", resource.GetNamespace(),
		)

		return subreconciler.RequeueWithError(err)
	}

	return subreconciler.ContinueReconciling()
}

func UpdateResourceStatus(ctx context.Context, kubeClient client.Client, resource client.Object, current client.Object) error {
	if err := kubeClient.Get(
		ctx,
		types.NamespacedName{Name: resource.GetName(), Namespace: resource.GetNamespace()},
		current,
	); err != nil {
		return err
	}

	if err := kubeClient.Status().Patch(ctx, resource, client.MergeFrom(current)); err != nil {
		return err
	}

	return nil
}

// CopyInputConfigMap returns a function of type FnWithRequest, to
// be called in the main reconciliation loop.
//
// It creates a copy of the user-provided ConfigMap.
func CopyInputConfigMap(kubeClient client.Client, runtimeScheme *runtime.Scheme, resource client.Object) func(context.Context, ctrl.Request) (*ctrl.Result, error) {
	return func(ctx context.Context, req ctrl.Request) (*ctrl.Result, error) {
		log := ctrllog.FromContext(ctx)

		if r, err := GetResource(ctx, kubeClient, req, resource); subreconciler.ShouldHaltOrRequeue(r, err) {
			return r, err
		}

		var keyForInputConfigMap types.NamespacedName
		if err := getInputConfigMapInfo(resource, &keyForInputConfigMap); err != nil {
			return nil, err
		}

		// Get and check the input ConfigMap for MautrixSignal
		if err := kubeClient.Get(ctx, keyForInputConfigMap, &corev1.ConfigMap{}); err != nil {
			reason := "ConfigMap " + keyForInputConfigMap.Name + " does not exist in namespace " + keyForInputConfigMap.Namespace
			SetFailedState(ctx, kubeClient, resource, reason)

			log.Error(
				err,
				"Failed to get ConfigMap",
				"ConfigMap.Namespace",
				keyForInputConfigMap.Namespace,
				"ConfigMap.Name",
				keyForInputConfigMap.Name,
			)

			return subreconciler.RequeueWithDelayAndError(time.Duration(30), err)
		}

		objectMeta := reconcile.SetObjectMeta(resource.GetName(), resource.GetNamespace(), map[string]string{})

		desiredConfigMap, err := configMapForCopy(ctx, objectMeta, kubeClient, resource, runtimeScheme)
		if err != nil {
			return subreconciler.RequeueWithError(err)
		}

		// Create a copy of the inputConfigMap
		if err := reconcile.ReconcileResource(
			ctx,
			kubeClient,
			desiredConfigMap,
			&corev1.ConfigMap{},
		); err != nil {
			return subreconciler.RequeueWithError(err)
		}

		return subreconciler.ContinueReconciling()
	}
}

// The ConfigMap returned by configMapForCopy is a copy of the user-defined
// ConfigMap.
func configMapForCopy(
	ctx context.Context,
	objectMeta metav1.ObjectMeta,
	kubeClient client.Client,
	resource client.Object,
	runtimeScheme *runtime.Scheme,
) (*corev1.ConfigMap, error) {
	var copyConfigMap *corev1.ConfigMap

	var keyForInputConfigMap types.NamespacedName
	if err := getInputConfigMapInfo(resource, &keyForInputConfigMap); err != nil {
		return nil, err
	}

	copyConfigMap, err := GetConfigMapCopy(
		kubeClient,
		keyForInputConfigMap.Name,
		keyForInputConfigMap.Namespace,
		objectMeta,
	)
	if err != nil {
		return &corev1.ConfigMap{}, err
	}

	// Set owner references
	if err := ctrl.SetControllerReference(resource, copyConfigMap, runtimeScheme); err != nil {
		return &corev1.ConfigMap{}, err
	}

	return copyConfigMap, nil
}

func SetFailedState(ctx context.Context, kubeClient client.Client, resource client.Object, reason string) {
	log := ctrllog.FromContext(ctx)
	var err error

	switch v := resource.(type) {
	case *synapsev1alpha1.Synapse:
		v.Status.State = "FAILED"
		v.Status.Reason = reason

		err = UpdateResourceStatus(ctx, kubeClient, v, &synapsev1alpha1.Synapse{})
	case *synapsev1alpha1.Heisenbridge:
		v.Status.State = "FAILED"
		v.Status.Reason = reason

		err = UpdateResourceStatus(ctx, kubeClient, v, &synapsev1alpha1.Heisenbridge{})
	case *synapsev1alpha1.MautrixSignal:
		v.Status.State = "FAILED"
		v.Status.Reason = reason

		err = UpdateResourceStatus(ctx, kubeClient, v, &synapsev1alpha1.MautrixSignal{})
	default:
		err = errors.New("error in type assertion")
	}

	if err != nil {
		log.Error(err, "Error updating mautrix-signal State")
	}
}

func getInputConfigMapInfo(resource client.Object, keys *types.NamespacedName) error {
	var inputConfigMapName, inputConfigMapNamespace string

	switch v := resource.(type) {
	case *synapsev1alpha1.Synapse:
		inputConfigMapName = v.Spec.Homeserver.ConfigMap.Name
		inputConfigMapNamespace = ComputeNamespace(resource.GetNamespace(), v.Spec.Homeserver.ConfigMap.Namespace)
	case *synapsev1alpha1.Heisenbridge:
		inputConfigMapName = v.Spec.ConfigMap.Name
		inputConfigMapNamespace = ComputeNamespace(resource.GetNamespace(), v.Spec.ConfigMap.Namespace)
	case *synapsev1alpha1.MautrixSignal:
		inputConfigMapName = v.Spec.ConfigMap.Name
		inputConfigMapNamespace = ComputeNamespace(resource.GetNamespace(), v.Spec.ConfigMap.Namespace)
	default:
		err := errors.New("error in type assertion")
		return err
	}

	keys.Name = inputConfigMapName
	keys.Namespace = inputConfigMapNamespace

	return nil
}
