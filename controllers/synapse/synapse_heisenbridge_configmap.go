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
func (r *SynapseReconciler) configMapForHeisenbridge(s *synapsev1alpha1.Synapse, objectMeta metav1.ObjectMeta) client.Object {
	heisenbridgeYaml := `
id: heisenbridge
url: http://` + s.Status.BridgesConfiguration.Heisenbridge.IP + `:9898
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
	ctrl.SetControllerReference(s, cm, r.Scheme)

	return cm
}

func (r *SynapseReconciler) updateHeisenbridgeWithURL(
	s synapsev1alpha1.Synapse,
	heisenbridge map[string]interface{},
) error {
	heisenbridge["url"] = "http://" + s.Status.BridgesConfiguration.Heisenbridge.IP + ":9898"
	return nil
}
