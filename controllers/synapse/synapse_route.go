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
	routev1 "github.com/openshift/api/route/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	synapsev1alpha1 "github.com/opdev/synapse-operator/apis/synapse/v1alpha1"
)

// serviceForSynapse returns a synapse Service object
func (r *SynapseReconciler) routeForSynapse(s *synapsev1alpha1.Synapse, objectMeta metav1.ObjectMeta) client.Object {
	weight := int32(100)
	route := &routev1.Route{
		ObjectMeta: objectMeta,
		Spec: routev1.RouteSpec{
			Host: "synapse.apps.alex-test-ocp4.9.23.coreostrain.me",
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromInt(8008),
			},
			To: routev1.RouteTargetReference{
				Kind:   "service",
				Name:   "synapse-sample",
				Weight: &weight,
			},
		},
	}
	// Set Synapse instance as the owner and controller
	ctrl.SetControllerReference(s, route, r.Scheme)
	return route
}
