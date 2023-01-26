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

package mautrixsignal

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	synapsev1alpha1 "github.com/opdev/synapse-operator/apis/synapse/v1alpha1"
	"github.com/opdev/synapse-operator/helpers/reconcile"
	reconc "github.com/opdev/synapse-operator/helpers/reconcileresults"
)

// labelsForSignald returns the labels for selecting the resources
// belonging to the given synapse CR name.
func labelsForSignald(name string) map[string]string {
	return map[string]string{"app": "signald", "mautrixsignal_cr": name}
}

// reconcileSignaldDeployment is a function of type subreconcilerFuncs, to be
// called in the main reconciliation loop.
//
// It reconciles the Deployment for signald to its desired state.
func (r *MautrixSignalReconciler) reconcileSignaldDeployment(obj client.Object, ctx context.Context) (*ctrl.Result, error) {
	ms := obj.(*synapsev1alpha1.MautrixSignal)

	objectMetaSignald := reconcile.SetObjectMeta(GetSignaldResourceName(*ms), ms.Namespace, map[string]string{})

	desiredDeployment, err := r.deploymentForSignald(ms, objectMetaSignald)
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

// deploymentForSynapse returns a synapse Deployment object
func (r *MautrixSignalReconciler) deploymentForSignald(ms *synapsev1alpha1.MautrixSignal, objectMeta metav1.ObjectMeta) (*appsv1.Deployment, error) {
	ls := labelsForSignald(ms.Name)
	replicas := int32(1)
	signaldPVCName := objectMeta.Name

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
						Image: "docker.io/signald/signald:0.23.0",
						Name:  "signald",
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "signald",
							MountPath: "/signald",
						}},
					}},
					Volumes: []corev1.Volume{{
						Name: "signald",
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: signaldPVCName,
							},
						},
					}},
				},
			},
		},
	}
	// Set Synapse instance as the owner and controller
	if err := ctrl.SetControllerReference(ms, dep, r.Scheme); err != nil {
		return &appsv1.Deployment{}, err
	}
	return dep, nil
}
