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

package synapse

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	synapsev1alpha1 "github.com/opdev/synapse-operator/apis/synapse/v1alpha1"
	reconc "github.com/opdev/synapse-operator/helpers/reconcileresults"
)

// reconcileHeisenbridgeConfigMap is a function of type subreconcilerFuncs, to
// be called in the main reconciliation loop.
//
// It reconciles the heisenbridge ConfigMap to its desired state. It is called
// only if the user hasn't provided its own ConfigMap for heisenbridge
func (r *SynapseReconciler) reconcileHeisenbridgeConfigMap(synapse *synapsev1alpha1.Synapse, ctx context.Context) (*ctrl.Result, error) {
	objectMetaHeisenbridge := setObjectMeta(r.GetHeisenbridgeResourceName(*synapse), synapse.Namespace, map[string]string{})
	if err := r.reconcileResource(
		ctx,
		r.configMapForHeisenbridge,
		synapse,
		&corev1.ConfigMap{},
		objectMetaHeisenbridge,
	); err != nil {
		return reconc.RequeueWithError(err)
	}

	return reconc.ContinueReconciling()
}

// configMapForSynapse returns a synapse ConfigMap object
func (r *SynapseReconciler) configMapForHeisenbridge(s *synapsev1alpha1.Synapse, objectMeta metav1.ObjectMeta) (client.Object, error) {
	heisenbridgeYaml := `
id: heisenbridge
url: http://` + r.GetHeisenbridgeServiceFQDN(*s) + `:9898
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
	if err := ctrl.SetControllerReference(s, cm, r.Scheme); err != nil {
		return &corev1.ConfigMap{}, err
	}

	return cm, nil
}

// copyInputHeisenbridgeConfigMap is a function of type subreconcilerFuncs, to
// be called in the main reconciliation loop.
//
// It creates a copy of the user-provided ConfigMap for heisenbridge, defined
// in synapse.Spec.Bridges.Heisenbridge.ConfigMap
func (r *SynapseReconciler) copyInputHeisenbridgeConfigMap(synapse *synapsev1alpha1.Synapse, ctx context.Context) (*ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)

	inputConfigMapName := synapse.Spec.Bridges.Heisenbridge.ConfigMap.Name
	inputConfigMapNamespace := r.getConfigMapNamespace(
		*synapse,
		synapse.Spec.Bridges.Heisenbridge.ConfigMap.Namespace,
	)
	keyForConfigMap := types.NamespacedName{
		Name:      inputConfigMapName,
		Namespace: inputConfigMapNamespace,
	}

	// Get and check the input ConfigMap for Heisenbridge
	if err := r.Get(ctx, keyForConfigMap, &corev1.ConfigMap{}); err != nil {
		reason := "ConfigMap " + inputConfigMapName + " does not exist in namespace " + inputConfigMapNamespace
		if err := r.setFailedState(ctx, synapse, reason); err != nil {
			log.Error(err, "Error updating Synapse State")
		}

		log.Error(
			err,
			"Failed to get ConfigMap",
			"ConfigMap.Namespace",
			inputConfigMapNamespace,
			"ConfigMap.Name",
			inputConfigMapName,
		)

		return reconc.RequeueWithDelayAndError(time.Duration(30), err)
	}

	objectMetaHeisenbridge := setObjectMeta(r.GetHeisenbridgeResourceName(*synapse), synapse.Namespace, map[string]string{})

	// Create a copy of the inputHeisenbridgeConfigMap defined in Spec.Bridges.Heisenbridge.ConfigMap
	// Here we use the configMapForHeisenbridgeCopy function as createResourceFunc
	if err := r.reconcileResource(
		ctx,
		r.configMapForHeisenbridgeCopy,
		synapse,
		&corev1.ConfigMap{},
		objectMetaHeisenbridge,
	); err != nil {
		return reconc.RequeueWithError(err)
	}

	return reconc.ContinueReconciling()
}

// configMapForHeisenbridgeCopy is a function of type createResourceFunc, to be
// passed as an argument in a call to reconcileResouce.
//
// The ConfigMap returned by configMapForHeisenbridgeCopy is a copy of the ConfigMap
// defined in Spec.Bridges.Heisenbridge.ConfigMap.
func (r *SynapseReconciler) configMapForHeisenbridgeCopy(
	s *synapsev1alpha1.Synapse,
	objectMeta metav1.ObjectMeta,
) (client.Object, error) {
	var copyConfigMap *corev1.ConfigMap

	sourceConfigMapName := s.Spec.Bridges.Heisenbridge.ConfigMap.Name
	sourceConfigMapNamespace := r.getConfigMapNamespace(*s, s.Spec.Bridges.Heisenbridge.ConfigMap.Namespace)

	copyConfigMap, err := r.getConfigMapCopy(sourceConfigMapName, sourceConfigMapNamespace, objectMeta)
	if err != nil {
		return &corev1.ConfigMap{}, err
	}

	// Set Synapse instance as the owner and controller
	if err := ctrl.SetControllerReference(s, copyConfigMap, r.Scheme); err != nil {
		return &corev1.ConfigMap{}, err
	}

	return copyConfigMap, nil
}

// configureHeisenbridgeConfigMap is a function of type subreconcilerFuncs, to
// be called in the main reconciliation loop.
//
// Following the previous copy of the user-provided ConfigMap, it edits the
// content of the copy to ensure that heisenbridge is correctly configured.
func (r *SynapseReconciler) configureHeisenbridgeConfigMap(synapse *synapsev1alpha1.Synapse, ctx context.Context) (*ctrl.Result, error) {
	keyForConfigMap := types.NamespacedName{
		Name:      r.GetHeisenbridgeResourceName(*synapse),
		Namespace: synapse.Namespace,
	}

	// Configure correct URL in Heisenbridge ConfigMap
	if err := r.updateConfigMap(
		ctx,
		keyForConfigMap,
		*synapse,
		r.updateHeisenbridgeWithURL,
		"heisenbridge.yaml",
	); err != nil {
		return reconc.RequeueWithError(err)
	}

	return reconc.ContinueReconciling()
}

// updateHeisenbridgeWithURL is a function of type updateDataFunc function to
// be passed as an argument in a call to updateConfigMap.
//
// It configures the correct Heisenbridge URL, needed for Synapse to reach the
// bridge.
func (r *SynapseReconciler) updateHeisenbridgeWithURL(
	s synapsev1alpha1.Synapse,
	heisenbridge map[string]interface{},
) error {
	heisenbridge["url"] = "http://" + r.GetHeisenbridgeServiceFQDN(s) + ":9898"
	return nil
}
