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
	"errors"

	"gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	synapsev1alpha1 "github.com/opdev/synapse-operator/apis/synapse/v1alpha1"
)

// SynapseReconciler reconciles a Synapse object
type SynapseReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=synapse.opdev.io,resources=synapses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=synapse.opdev.io,resources=synapses/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=synapse.opdev.io,resources=synapses/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *SynapseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)

	// Load the Synapse by name
	var synapse synapsev1alpha1.Synapse
	if err := r.Get(ctx, req.NamespacedName, &synapse); err != nil {
		log.Error(err, "unable to fetch synapse")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err := r.ParseHomeserverConfigMap(&synapse, ctx); err != nil {
		return ctrl.Result{}, err
	}

	// Reconcile Synapse resources: PVC, Deployment and Service
	var objectMeta metav1.ObjectMeta

	objectMeta = setObjectMeta(synapse.Name, synapse.Namespace, map[string]string{})
	r.reconcileResource(r.persistentVolumeClaimForSynapse, &synapse, &corev1.PersistentVolumeClaim{}, objectMeta)

	objectMeta = setObjectMeta(synapse.Name, synapse.Namespace, map[string]string{})
	r.reconcileResource(r.deploymentForSynapse, &synapse, &appsv1.Deployment{}, objectMeta)
	// TODO: If a deployment is found, check that its Spec are correct.

	objectMeta = setObjectMeta(synapse.Name, synapse.Namespace, map[string]string{})
	r.reconcileResource(r.serviceForSynapse, &synapse, &corev1.Service{}, objectMeta)

	// Update the Synapse status if needed
	if synapse.Status.State != "RUNNING" {
		synapse.Status.State = "RUNNING"
		if err := r.Status().Update(ctx, &synapse); err != nil {
			log.Error(err, "Failed to update Synapse status")
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

// labelsForSynapse returns the labels for selecting the resources
// belonging to the given synapse CR name.
func labelsForSynapse(name string) map[string]string {
	return map[string]string{"app": "synapse", "synapse_cr": name}
}

// ParseHomeserverConfigMap loads the ConfigMap, which name is determined by
// Spec.HomeserverConfigMapName, run validation checks and fetch necesarry
// value needed to configure the Synapse Deployment.
func (r *SynapseReconciler) ParseHomeserverConfigMap(synapse *synapsev1alpha1.Synapse, ctx context.Context) error {
	log := ctrllog.FromContext(ctx)

	// Get and validate homeserver ConfigMap
	var cm corev1.ConfigMap
	if err := r.Get(context.TODO(), types.NamespacedName{Name: synapse.Spec.HomeserverConfigMapName, Namespace: synapse.Namespace}, &cm); err != nil {
		log.Error(err, "Failed to get ConfigMap", "ConfigMap.Namespace", synapse.Namespace, "ConfigMap.Name", synapse.Spec.HomeserverConfigMapName)
		return err
	} else {
		// TODO:
		// - Ensure that key path is and log config file path are in /data
		// - Otherwise, edit homeserver.yaml with new paths

		// Load and validate homeserver.yaml
		homeserver := make(map[interface{}]interface{})
		if cm_data, ok := cm.Data["homeserver.yaml"]; !ok {
			err := errors.New("missing homeserver.yaml in ConfigMap")
			log.Error(err, "Missing homeserver.yaml in ConfigMap", "ConfigMap.Namespace", synapse.Namespace, "ConfigMap.Name", synapse.Spec.HomeserverConfigMapName)
			return err
		} else {
			// YAML Validation
			if err := yaml.Unmarshal([]byte(cm_data), homeserver); err != nil {
				log.Error(err, "Malformed homeserver.yaml")
				return err
			}
		}

		// Fetch server_name and report_stats
		if _, ok := homeserver["server_name"]; !ok {
			err := errors.New("missing server_name key in homeserver.yaml")
			log.Error(err, "Missing server_name key in homeserver.yaml")
			return err
		}
		if _, ok := homeserver["report_stats"]; !ok {
			err := errors.New("missing report_stats key in homeserver.yaml")
			log.Error(err, "Missing report_stats key in homeserver.yaml")
			return err
		}

		if server_name, ok := homeserver["server_name"].(string); !ok {
			err := errors.New("error converting server_name to string")
			log.Error(err, "Error converting server_name to string")
			return err
		} else {
			synapse.Status.HomeserverConfiguration.ServerName = server_name
		}
		if report_stats, ok := homeserver["report_stats"].(bool); !ok {
			err := errors.New("error converting report_stats to bool")
			log.Error(err, "Error converting report_stats to bool")
			return err
		} else {
			synapse.Status.HomeserverConfiguration.ReportStats = report_stats
		}

		log.Info(
			"Loaded homeserver.yaml from ConfigMap successfully",
			"server_name:", synapse.Status.HomeserverConfiguration.ServerName,
			"report_stats:", synapse.Status.HomeserverConfiguration.ReportStats,
		)
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SynapseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&synapsev1alpha1.Synapse{}).
		Owns(&corev1.Service{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Complete(r)
}
