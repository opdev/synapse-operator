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
	b64 "encoding/base64"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	pgov1beta1 "github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
	synapsev1alpha1 "github.com/opdev/synapse-operator/apis/synapse/v1alpha1"
)

// postgresClusterForSynapse returns a synapse Deployment object
func (r *SynapseReconciler) postgresClusterForSynapse(s *synapsev1alpha1.Synapse, objectMeta metav1.ObjectMeta) client.Object {
	postgresCluster := &pgov1beta1.PostgresCluster{
		ObjectMeta: objectMeta,
		Spec: pgov1beta1.PostgresClusterSpec{
			Image:           "registry.developers.crunchydata.com/crunchydata/crunchy-postgres:centos8-13.5-0",
			PostgresVersion: 13,
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
					Image: "registry.developers.crunchydata.com/crunchydata/crunchy-pgbackrest:centos8-2.36-0",
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
	ctrl.SetControllerReference(s, postgresCluster, r.Scheme)
	return postgresCluster
}

func (r *SynapseReconciler) configMapForPostgresCluster(s *synapsev1alpha1.Synapse, objectMeta metav1.ObjectMeta) client.Object {
	configMap := &corev1.ConfigMap{
		ObjectMeta: objectMeta,
		Data:       map[string]string{"createdb.sql": "CREATE DATABASE synapse LOCALE 'C' ENCODING 'UTF-8' TEMPLATE template0;"},
	}
	ctrl.SetControllerReference(s, configMap, r.Scheme)

	return configMap
}

func base64encode(to_encode string) []byte {
	return []byte(b64.StdEncoding.EncodeToString([]byte(to_encode)))
}

func base64decode(to_decode []byte) string {
	decoded_bytes, _ := b64.StdEncoding.DecodeString(string(to_decode))
	return string(decoded_bytes)
}
