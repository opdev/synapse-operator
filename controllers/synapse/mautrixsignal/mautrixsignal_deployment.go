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

// labelsForMautrixSignal returns the labels for selecting the resources
// belonging to the given synapse CR name.
func labelsForMautrixSignal(name string) map[string]string {
	return map[string]string{"app": "mautrix-signal", "mautrixsignal_cr": name}
}

// reconcileMautrixSignalDeployment is a function of type subreconcilerFuncs,
// to be called in the main reconciliation loop.
//
// It reconciles the Deployment for mautrix-signal to its desired state.
func (r *MautrixSignalReconciler) reconcileMautrixSignalDeployment(obj client.Object, ctx context.Context) (*ctrl.Result, error) {
	ms := obj.(*synapsev1alpha1.MautrixSignal)

	objectMetaMautrixSignal := reconcile.SetObjectMeta(ms.Name, ms.Namespace, map[string]string{})

	desiredDeployment, err := r.deploymentForMautrixSignal(ms, objectMetaMautrixSignal)
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

// deploymentForMautrixSignal returns a Deployment object for the mautrix-signal bridge
func (r *MautrixSignalReconciler) deploymentForMautrixSignal(ms *synapsev1alpha1.MautrixSignal, objectMeta metav1.ObjectMeta) (*appsv1.Deployment, error) {
	ls := labelsForMautrixSignal(ms.Name)
	replicas := int32(1)

	// The associated mautrix-signal objects (ConfigMap, PVC, SA) share the
	// same name as the mautrix-signal Deployment
	mautrixSignalConfigMapName := objectMeta.Name
	mautrixSignalPVCName := objectMeta.Name
	mautrixSignalServiceAccountName := objectMeta.Name

	// The Signald PVC name is the Synapse object name with "-signald" appended
	SignaldPVCName := GetSignaldResourceName(*ms)

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
					// The init container is responsible of copying the
					// config.yaml from the read-only ConfigMap to the
					// mautrixsignal-data volume. The mautrixsignal process
					// needs read & write access to the config.yaml file.
					InitContainers: []corev1.Container{{
						Image: "registry.access.redhat.com/ubi8/ubi-minimal:8.7",
						Name:  "initconfig",
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "config",
							MountPath: "/input",
						}, {
							Name:      "mautrixsignal-data",
							MountPath: "/data",
						}},
						Command: []string{"bin/sh", "-c"},
						Args:    []string{"if [ ! -f /data/config.yaml ]; then cp /input/config.yaml /data/config.yaml; fi"},
					}},
					Containers: []corev1.Container{{
						Image: "dock.mau.dev/mautrix/signal:v0.4.1",
						Name:  "mautrix-signal",
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "signald",
							MountPath: "/signald",
						}, {
							Name:      "mautrixsignal-data",
							MountPath: "/data",
						}},
					}},
					Volumes: []corev1.Volume{{
						Name: "config",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: mautrixSignalConfigMapName,
								},
							},
						},
					}, {
						Name: "signald",
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: SignaldPVCName,
							},
						},
					}, {
						Name: "mautrixsignal-data",
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: mautrixSignalPVCName,
							},
						},
					}},
				},
			},
		},
	}

	if ms.Status.IsOpenshift {
		// mautrix-signal must run with user 1337.
		// If deploying on Openshift, we must run the workload with a Service
		// Account associated to the 'anyuid' SCC.
		dep.Spec.Template.Spec.ServiceAccountName = mautrixSignalServiceAccountName
	}

	// Set Synapse instance as the owner and controller
	if err := ctrl.SetControllerReference(ms, dep, r.Scheme); err != nil {
		return &appsv1.Deployment{}, err
	}
	return dep, nil
}
