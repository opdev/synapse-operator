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

package heisenbridge

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/opdev/subreconciler"
	synapsev1alpha1 "github.com/opdev/synapse-operator/apis/synapse/v1alpha1"
	"github.com/opdev/synapse-operator/helpers/reconcile"
	"github.com/opdev/synapse-operator/helpers/utils"
)

// reconcileHeisenbridgeConfigMap is a function of type FnWithRequest, to
// be called in the main reconciliation loop.
//
// It reconciles the heisenbridge ConfigMap to its desired state. It is called
// only if the user hasn't provided its own ConfigMap for heisenbridge
func (r *HeisenbridgeReconciler) reconcileHeisenbridgeConfigMap(ctx context.Context, req ctrl.Request) (*ctrl.Result, error) {
	h := &synapsev1alpha1.Heisenbridge{}
	if r, err := utils.GetResource(ctx, r.Client, req, h); subreconciler.ShouldHaltOrRequeue(r, err) {
		return r, err
	}

	objectMetaHeisenbridge := reconcile.SetObjectMeta(h.Name, h.Namespace, map[string]string{})

	desiredConfigMap, err := r.configMapForHeisenbridge(h, objectMetaHeisenbridge)
	if err != nil {
		return subreconciler.RequeueWithError(err)
	}

	if err := reconcile.ReconcileResource(
		ctx,
		r.Client,
		desiredConfigMap,
		&corev1.ConfigMap{},
	); err != nil {
		return subreconciler.RequeueWithError(err)
	}

	return subreconciler.ContinueReconciling()
}

// configMapForSynapse returns a synapse ConfigMap object
func (r *HeisenbridgeReconciler) configMapForHeisenbridge(h *synapsev1alpha1.Heisenbridge, objectMeta metav1.ObjectMeta) (*corev1.ConfigMap, error) {
	heisenbridgeYaml := `
id: heisenbridge
url: http://` + GetHeisenbridgeServiceFQDN(*h) + `:9898
as_token: EUFqSPQusV4mXkPKbwdHyIhthELQ1Xf9S5lSEzTrrlb0uz0ZJRHhwEljT71ByObe
hs_token: If6r2GGlsNN4MnoW3djToADNdq0JuIJ1WNM4rKHO73WuG5QvVubj1Q4JHrmQBcS6
rate_limited: false
sender_localpart: heisenbridge
namespaces:
    users:
    - regex: '@irc_.*'
      exclusive: true
    aliases: []
    rooms: []
  `

	cm := &corev1.ConfigMap{
		ObjectMeta: objectMeta,
		Data:       map[string]string{"heisenbridge.yaml": heisenbridgeYaml},
	}

	// Set Synapse instance as the owner and controller
	if err := ctrl.SetControllerReference(h, cm, r.Scheme); err != nil {
		return &corev1.ConfigMap{}, err
	}

	return cm, nil
}

// configureHeisenbridgeConfigMap is a function of type FnWithRequest, to
// be called in the main reconciliation loop.
//
// Following the previous copy of the user-provided ConfigMap, it edits the
// content of the copy to ensure that heisenbridge is correctly configured.
func (r *HeisenbridgeReconciler) configureHeisenbridgeConfigMap(ctx context.Context, req ctrl.Request) (*ctrl.Result, error) {
	h := &synapsev1alpha1.Heisenbridge{}
	if r, err := utils.GetResource(ctx, r.Client, req, h); subreconciler.ShouldHaltOrRequeue(r, err) {
		return r, err
	}

	keyForConfigMap := types.NamespacedName{
		Name:      h.Name,
		Namespace: h.Namespace,
	}

	// Configure correct URL in Heisenbridge ConfigMap
	if err := utils.UpdateConfigMap(
		ctx,
		r.Client,
		keyForConfigMap,
		h,
		r.updateHeisenbridgeWithURL,
		"heisenbridge.yaml",
	); err != nil {
		return subreconciler.RequeueWithError(err)
	}

	return subreconciler.ContinueReconciling()
}

// updateHeisenbridgeWithURL is a function of type updateDataFunc function to
// be passed as an argument in a call to updateConfigMap.
//
// It configures the correct Heisenbridge URL, needed for Synapse to reach the
// bridge.
func (r *HeisenbridgeReconciler) updateHeisenbridgeWithURL(
	obj client.Object,
	heisenbridge map[string]interface{},
) error {
	h := obj.(*synapsev1alpha1.Heisenbridge)

	heisenbridge["url"] = "http://" + GetHeisenbridgeServiceFQDN(*h) + ":9898"
	return nil
}
