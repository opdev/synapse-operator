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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	synapsev1alpha1 "github.com/opdev/synapse-operator/apis/synapse/v1alpha1"
)

// labelsForSynapse returns the labels for selecting the resources
// belonging to the given synapse CR name.
func labelsForHeisenbridge(name string) map[string]string {
	return map[string]string{"app": "heisenbridge", "synapse_cr": name}
}

// deploymentForSynapse returns a synapse Deployment object
func (r *SynapseReconciler) deploymentForHeisenbridge(s *synapsev1alpha1.Synapse, objectMeta metav1.ObjectMeta) client.Object {
	ls := labelsForHeisenbridge(s.Name)
	replicas := int32(1)

	command := r.craftHeisenbridgeCommad(*s)

	dep := &appsv1.Deployment{
		ObjectMeta: objectMeta,
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: "hif1/heisenbridge:1.10.1",
						Name:  "heisenbridge",
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "data-heisenbridge",
							MountPath: "/data-heisenbridge",
						}},
						Ports: []corev1.ContainerPort{{
							ContainerPort: 9898,
						}},
						Command: command,
					}},
					Volumes: []corev1.Volume{{
						Name: "data-heisenbridge",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: objectMeta.Name,
								},
							},
						},
					}},
				},
			},
		},
	}
	// Set Synapse instance as the owner and controller
	ctrl.SetControllerReference(s, dep, r.Scheme)
	return dep
}

func (r *SynapseReconciler) craftHeisenbridgeCommad(s synapsev1alpha1.Synapse) []string {
	command := []string{
		"python",
		"-m",
		"heisenbridge",
	}

	if s.Spec.Bridges.Heisenbridge.VerboseLevel > 0 {
		verbosity := "-"
		for i := 1; i <= s.Spec.Bridges.Heisenbridge.VerboseLevel; i++ {
			verbosity = verbosity + "v"
		}
		command = append(command, verbosity)
	}

	command = append(
		command,
		"-c",
		"/data-heisenbridge/heisenbridge.yaml",
		"-l",
		"0.0.0.0",
		"http://"+s.Status.IP+":8008",
	)

	return command
}
