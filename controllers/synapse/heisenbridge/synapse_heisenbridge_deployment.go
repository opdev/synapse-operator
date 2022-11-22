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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	synapsev1alpha1 "github.com/opdev/synapse-operator/apis/synapse/v1alpha1"
	"github.com/opdev/synapse-operator/helpers/reconcile"
	reconc "github.com/opdev/synapse-operator/helpers/reconcileresults"
	"github.com/opdev/synapse-operator/helpers/utils"
)

// labelsForSynapse returns the labels for selecting the resources
// belonging to the given synapse CR name.
func labelsForHeisenbridge(name string) map[string]string {
	return map[string]string{"app": "heisenbridge", "heisenbridge_cr": name}
}

// reconcileHeisenbridgeDeployment is a function of type subreconcilerFuncs, to
// be called in the main reconciliation loop.
//
// It reconciles the Deployment for Heisenbridge to its desired state.
func (r *HeisenbridgeReconciler) reconcileHeisenbridgeDeployment(i interface{}, ctx context.Context) (*ctrl.Result, error) {
	h := i.(*synapsev1alpha1.Heisenbridge)

	objectMetaHeisenbridge := reconcile.SetObjectMeta(h.Name, h.Namespace, map[string]string{})

	desiredDeployment, err := r.deploymentForHeisenbridge(h, objectMetaHeisenbridge)
	if err != nil {
		return reconc.RequeueWithError(err)
	}

	if err := reconcile.ReconcileResource(
		ctx,
		r.Client,
		desiredDeployment,
		&appsv1.Deployment{},
	); err != nil {
		return reconc.RequeueWithError(err)
	}

	return reconc.ContinueReconciling()
}

// deploymentForHeisenbridge returns a Heisenbridge Deployment object
func (r *HeisenbridgeReconciler) deploymentForHeisenbridge(h *synapsev1alpha1.Heisenbridge, objectMeta metav1.ObjectMeta) (*appsv1.Deployment, error) {
	ls := labelsForHeisenbridge(h.Name)
	replicas := int32(1)

	command := r.craftHeisenbridgeCommad(*h)
	// The created Heisenbridge ConfigMap Name share the same name as the
	// Heisenbridge Deployment
	heisenbridgeConfigMapName := objectMeta.Name

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
						Image: "hif1/heisenbridge:1.14",
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
									Name: heisenbridgeConfigMapName,
								},
							},
						},
					}},
				},
			},
		},
	}
	// Set Synapse instance as the owner and controller
	if err := ctrl.SetControllerReference(h, dep, r.Scheme); err != nil {
		return &appsv1.Deployment{}, err
	}
	return dep, nil
}

func (r *HeisenbridgeReconciler) craftHeisenbridgeCommad(h synapsev1alpha1.Heisenbridge) []string {
	command := []string{
		"python",
		"-m",
		"heisenbridge",
	}

	if h.Spec.VerboseLevel > 0 {
		verbosity := "-"
		for i := 1; i <= h.Spec.VerboseLevel; i++ {
			verbosity = verbosity + "v"
		}
		command = append(command, verbosity)
	}

	SynapseName := h.Spec.Synapse.Name
	SynapseNamespace := utils.ComputeNamespace(h.Namespace, h.Spec.Synapse.Namespace)

	command = append(
		command,
		"-c",
		"/data-heisenbridge/heisenbridge.yaml",
		"-l",
		"0.0.0.0",
		"http://"+utils.ComputeFQDN(SynapseName, SynapseNamespace)+":8008",
	)

	return command
}
