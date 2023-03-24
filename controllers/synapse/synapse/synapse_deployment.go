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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/opdev/subreconciler"
	synapsev1alpha1 "github.com/opdev/synapse-operator/apis/synapse/v1alpha1"
	"github.com/opdev/synapse-operator/helpers/reconcile"
	"github.com/opdev/synapse-operator/helpers/utils"
)

// reconcileSynapseDeployment is a function of type FnWithRequest, to be
// called in the main reconciliation loop.
//
// It reconciles the Deployment for Synapse to its desired state.
func (r *SynapseReconciler) reconcileSynapseDeployment(ctx context.Context, req ctrl.Request) (*ctrl.Result, error) {
	s := &synapsev1alpha1.Synapse{}
	if r, err := r.getLatestSynapse(ctx, req, s); subreconciler.ShouldHaltOrRequeue(r, err) {
		return r, err
	}

	objectMetaForSynapse := reconcile.SetObjectMeta(s.Name, s.Namespace, map[string]string{})
	depl, err := r.deploymentForSynapse(s, objectMetaForSynapse)
	if err != nil {
		return subreconciler.RequeueWithError(err)
	}

	if err := reconcile.ReconcileResource(
		ctx,
		r.Client,
		depl,
		&appsv1.Deployment{},
	); err != nil {
		return subreconciler.RequeueWithError(err)
	}

	return subreconciler.ContinueReconciling()
}

// deploymentForSynapse returns a synapse Deployment object
func (r *SynapseReconciler) deploymentForSynapse(s *synapsev1alpha1.Synapse, objectMeta metav1.ObjectMeta) (*appsv1.Deployment, error) {
	ls := labelsForSynapse(s.Name)
	replicas := int32(1)

	server_name := s.Status.HomeserverConfiguration.ServerName
	report_stats := s.Status.HomeserverConfiguration.ReportStats
	// The created Synapse ConfigMap shares the same name as the Synapse deployment
	synapseConfigMapName := objectMeta.Name

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
					InitContainers: []corev1.Container{{
						Image: "matrixdotorg/synapse:v1.71.0",
						Name:  "synapse-generate",
						Args:  []string{"generate"},
						Env: []corev1.EnvVar{{
							Name:  "SYNAPSE_CONFIG_PATH",
							Value: "/data-homeserver/homeserver.yaml",
						}, {
							Name:  "SYNAPSE_SERVER_NAME",
							Value: server_name,
						}, {
							Name:  "SYNAPSE_REPORT_STATS",
							Value: utils.BoolToYesNo(report_stats),
						}},
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "homeserver",
							MountPath: "/data-homeserver",
						}, {
							Name:      "data-pv",
							MountPath: "/data",
						}},
					}},
					Containers: []corev1.Container{{
						Image: "matrixdotorg/synapse:v1.71.0",
						Name:  "synapse",
						Env: []corev1.EnvVar{{
							Name:  "SYNAPSE_CONFIG_PATH",
							Value: "/data-homeserver/homeserver.yaml",
						}},
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "homeserver",
							MountPath: "/data-homeserver",
						}, {
							Name:      "data-pv",
							MountPath: "/data",
						}},
						Ports: []corev1.ContainerPort{{
							ContainerPort: 8008,
						}},
					}},
					Volumes: []corev1.Volume{{
						Name: "homeserver",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: synapseConfigMapName,
								},
							},
						},
					}, {
						Name: "data-pv",
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: s.Name,
							},
						},
					}},
				},
			},
		},
	}

	if s.Spec.IsOpenshift {
		// Synapse must run with user 991.
		// If deploying on Openshift, we must run the workload with a Service
		// Account associated to the 'anyuid' SCC.
		dep.Spec.Template.Spec.ServiceAccountName = s.Name
	}

	if s.Status.Bridges.Heisenbridge.Enabled {
		heisenbridgeConfigMapName := s.Status.Bridges.Heisenbridge.Name

		dep.Spec.Template.Spec.Containers[0].VolumeMounts = append(
			dep.Spec.Template.Spec.Containers[0].VolumeMounts,
			corev1.VolumeMount{
				Name:      "data-heisenbridge",
				MountPath: "/data-heisenbridge",
			},
		)

		dep.Spec.Template.Spec.Volumes = append(
			dep.Spec.Template.Spec.Volumes,
			corev1.Volume{
				Name: "data-heisenbridge",
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: heisenbridgeConfigMapName,
						},
					},
				},
			},
		)
	}

	if s.Status.Bridges.MautrixSignal.Enabled {
		// If the mautrix-signal bridge is enabled, then Synapse needs access
		// to the registration.yaml file, containing all information to
		// register the mautrix-signal bridge as an application service in
		// homeserver.yaml. This registration file is generated by the bridge
		// the first time it runs and located alongside the config.yaml (config
		// file for mautrix-signal), that is it's in the mautrix-signal PV
		mautrixSignalPVCName := s.Status.Bridges.MautrixSignal.Name

		dep.Spec.Template.Spec.Containers[0].VolumeMounts = append(
			dep.Spec.Template.Spec.Containers[0].VolumeMounts,
			corev1.VolumeMount{
				Name:      "data-mautrixsignal",
				MountPath: "/data-mautrixsignal",
			},
		)

		dep.Spec.Template.Spec.Volumes = append(
			dep.Spec.Template.Spec.Volumes,
			corev1.Volume{
				Name: "data-mautrixsignal",
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: mautrixSignalPVCName,
					},
				},
			},
		)
	}

	// Set Synapse instance as the owner and controller
	if err := ctrl.SetControllerReference(s, dep, r.Scheme); err != nil {
		return &appsv1.Deployment{}, err
	}

	return dep, nil
}
