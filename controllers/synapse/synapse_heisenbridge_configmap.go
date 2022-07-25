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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	synapsev1alpha1 "github.com/opdev/synapse-operator/apis/synapse/v1alpha1"
)

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
