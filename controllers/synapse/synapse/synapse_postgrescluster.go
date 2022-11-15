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
	b64 "encoding/base64"
	"errors"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	pgov1beta1 "github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
	synapsev1alpha1 "github.com/opdev/synapse-operator/apis/synapse/v1alpha1"
	"github.com/opdev/synapse-operator/helpers/reconcile"
	reconc "github.com/opdev/synapse-operator/helpers/reconcileresults"
)

// reconcilePostgresClusterCR is a function of type subreconcilerFuncs, to be
// called in the main reconciliation loop.
//
// It reconciles the PostgresCluster CR to its desired state, and requeues
// until the PostgreSQL cluster is up.
func (r *SynapseReconciler) reconcilePostgresClusterCR(i interface{}, ctx context.Context) (*ctrl.Result, error) {
	s := i.(*synapsev1alpha1.Synapse)

	createdPostgresCluster := pgov1beta1.PostgresCluster{}
	postgresClusterObjectMeta := reconcile.SetObjectMeta(
		GetPostgresClusterResourceName(*s),
		s.Namespace,
		map[string]string{},
	)
	keyForPostgresCluster := types.NamespacedName{
		Name:      GetPostgresClusterResourceName(*s),
		Namespace: s.Namespace,
	}

	desiredPostgresCluster, err := r.postgresClusterForSynapse(s, postgresClusterObjectMeta)
	if err != nil {
		return reconc.RequeueWithError(err)
	}

	// Create PostgresCluster for Synapse
	if err := reconcile.ReconcileResource(
		ctx,
		r.Client,
		desiredPostgresCluster,
		&createdPostgresCluster,
	); err != nil {
		return reconc.RequeueWithError(err)
	}

	// Wait for PostgresCluster to be up
	// TODO: can be removed ?
	if err := r.Get(ctx, keyForPostgresCluster, &createdPostgresCluster); err != nil {
		return reconc.RequeueWithError(err)
	}
	if !r.isPostgresClusterReady(createdPostgresCluster) {
		r.updateSynapseStatusDatabaseState(ctx, s, "NOT READY")
		err := errors.New("postgreSQL Database not ready yet")
		return reconc.RequeueWithDelayAndError(time.Duration(5), err)
	}

	return reconc.ContinueReconciling()
}

// postgresClusterForSynapse returns a PostgresCluster object
func (r *SynapseReconciler) postgresClusterForSynapse(s *synapsev1alpha1.Synapse, objectMeta metav1.ObjectMeta) (*pgov1beta1.PostgresCluster, error) {
	postgresCluster := &pgov1beta1.PostgresCluster{
		ObjectMeta: objectMeta,
		Spec: pgov1beta1.PostgresClusterSpec{
			Image:           "registry.developers.crunchydata.com/crunchydata/crunchy-postgres:ubi8-14.5-1",
			PostgresVersion: 14,
			InstanceSets: []pgov1beta1.PostgresInstanceSetSpec{{
				Name: "instance1",
				DataVolumeClaimSpec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{"ReadWriteOnce"},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							"storage": *resource.NewQuantity(1*1024*1024*1024, resource.BinarySI),
						},
					},
				},
			}},
			Backups: pgov1beta1.Backups{
				PGBackRest: pgov1beta1.PGBackRestArchive{
					Image: "registry.developers.crunchydata.com/crunchydata/crunchy-pgbackrest:ubi8-2.40-1",
					Repos: []pgov1beta1.PGBackRestRepo{{
						Name: "repo1",
						Volume: &pgov1beta1.RepoPVC{
							VolumeClaimSpec: corev1.PersistentVolumeClaimSpec{
								AccessModes: []corev1.PersistentVolumeAccessMode{"ReadWriteOnce"},
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										"storage": *resource.NewQuantity(1*1024*1024*1024, resource.BinarySI),
									},
								},
							},
						},
					}},
				},
			},
			Users: []pgov1beta1.PostgresUserSpec{{
				Name:      "synapse",
				Databases: []pgov1beta1.PostgresIdentifier{"dummy"},
			}},
			// See https://github.com/opdev/synapse-operator/issues/12
			DatabaseInitSQL: &pgov1beta1.DatabaseInitSQL{
				Name: objectMeta.Name,
				Key:  "createdb.sql",
			},
		},
	}

	// Set Synapse instance as the owner and controller
	if err := ctrl.SetControllerReference(s, postgresCluster, r.Scheme); err != nil {
		return &pgov1beta1.PostgresCluster{}, err
	}
	return postgresCluster, nil
}

func (r *SynapseReconciler) isPostgresClusterReady(p pgov1beta1.PostgresCluster) bool {
	var status_found bool

	// Going through instance Specs
	for _, instance_spec := range p.Spec.InstanceSets {
		status_found = false
		for _, instance_status := range p.Status.InstanceSets {
			if instance_status.Name == instance_spec.Name {
				desired_replicas := *instance_spec.Replicas
				if instance_status.Replicas != desired_replicas ||
					instance_status.ReadyReplicas != desired_replicas ||
					instance_status.UpdatedReplicas != desired_replicas {
					return false
				}
				// Found instance in Status, breaking out of for loop
				status_found = true
				break
			}
		}

		// Instance found in spec, but not in status
		if !status_found {
			return false
		}
	}

	// All instances have the correct number of replicas
	return true
}

// reconcilePostgresClusterConfigMap is a function of type subreconcilerFuncs,
// to be called in the main reconciliation loop.
//
// It reconciles the PostgresCluster ConfigMap to its desired state.
func (r *SynapseReconciler) reconcilePostgresClusterConfigMap(i interface{}, ctx context.Context) (*ctrl.Result, error) {
	s := i.(*synapsev1alpha1.Synapse)

	postgresClusterObjectMeta := reconcile.SetObjectMeta(
		GetPostgresClusterResourceName(*s),
		s.Namespace,
		map[string]string{},
	)

	desiredConfigMap, err := r.configMapForPostgresCluster(s, postgresClusterObjectMeta)
	if err != nil {
		return reconc.RequeueWithError(err)
	}

	// Create ConfigMap for PostgresCluster
	if err := reconcile.ReconcileResource(
		ctx,
		r.Client,
		desiredConfigMap,
		&corev1.ConfigMap{},
	); err != nil {
		return reconc.RequeueWithError(err)
	}

	return reconc.ContinueReconciling()
}

// configMapForPostgresCluster returns a ConfigMap object
func (r *SynapseReconciler) configMapForPostgresCluster(s *synapsev1alpha1.Synapse, objectMeta metav1.ObjectMeta) (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{
		ObjectMeta: objectMeta,
		Data:       map[string]string{"createdb.sql": "CREATE DATABASE synapse LOCALE 'C' ENCODING 'UTF-8' TEMPLATE template0;"},
	}

	if err := ctrl.SetControllerReference(s, configMap, r.Scheme); err != nil {
		return &corev1.ConfigMap{}, err
	}

	return configMap, nil
}

func base64encode(to_encode string) []byte {
	return []byte(b64.StdEncoding.EncodeToString([]byte(to_encode)))
}

func base64decode(to_decode []byte) string {
	decoded_bytes, _ := b64.StdEncoding.DecodeString(string(to_decode))
	return string(decoded_bytes)
}
