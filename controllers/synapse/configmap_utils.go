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

/* This file puts together generic functions for ConfiMap manipulation */
import (
	"context"
	"errors"

	synapsev1alpha1 "github.com/opdev/synapse-operator/apis/synapse/v1alpha1"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Defines a function type. Functions of the updateDataFunc type implements the
// logic to update the data of a configmap, defined by the 'data' argument.
type updateDataFunc func(s synapsev1alpha1.Synapse, data map[string]interface{}) error

// A generic function to update an existing ConfigMap. It takes as arguments:
// * The context
// * The name of the ConfigMap to update
// * The Synapse object being reconciled
// * The function to be called to actually update the ConfigMap's content
// * The name of the file to update in the ConfigMap
func (r *SynapseReconciler) updateConfigMap(
	ctx context.Context,
	cm *corev1.ConfigMap,
	s synapsev1alpha1.Synapse,
	updateData updateDataFunc,
	filename string,
) error {
	// Get latest ConfigMap version
	if err := r.Get(
		ctx,
		types.NamespacedName{Name: cm.Name, Namespace: cm.Namespace},
		cm,
	); err != nil {
		return err
	}

	if err := r.updateConfigMapData(cm, s, updateData, filename); err != nil {
		return err
	}

	// Update ConfigMap
	if err := r.Client.Update(ctx, cm); err != nil {
		return err
	}

	return nil
}

func (r *SynapseReconciler) updateConfigMapData(
	cm *corev1.ConfigMap,
	s synapsev1alpha1.Synapse,
	updateData updateDataFunc,
	filename string,
) error {
	// Load file to update from ConfigMap
	data, err := r.loadYAMLFileFromConfigMapData(*cm, filename)
	if err != nil {
		return err
	}

	// Update the content of the file
	if err := updateData(s, data); err != nil {
		return err
	}

	// Write new content into ConfigMap data
	if err := r.writeYAMLFileToConfigMapData(cm, filename, data); err != nil {
		return err
	}

	return nil
}

func (r *SynapseReconciler) loadFileFromConfigMapData(
	configMap corev1.ConfigMap,
	filename string,
) (string, error) {
	content, ok := configMap.Data[filename]
	if !ok {
		err := errors.New("missing " + filename + " in ConfigMap " + configMap.Name)
		return "", err
	}

	return content, nil
}

func (r *SynapseReconciler) loadYAMLFileFromConfigMapData(
	configMap corev1.ConfigMap,
	filename string,
) (map[string]interface{}, error) {
	yamlContent := map[string]interface{}{}

	content, err := r.loadFileFromConfigMapData(configMap, filename)
	if err != nil {
		return yamlContent, err
	}
	if err := yaml.Unmarshal([]byte(content), yamlContent); err != nil {
		return yamlContent, err
	}

	return yamlContent, nil
}

func (r *SynapseReconciler) writeFileToConfigMapData(
	configMap *corev1.ConfigMap,
	filename string,
	content string,
) {
	configMap.Data = map[string]string{filename: content}
}

func (r *SynapseReconciler) writeYAMLFileToConfigMapData(
	configMap *corev1.ConfigMap,
	filename string,
	yamlContent map[string]interface{},
) error {
	bytesContent, err := yaml.Marshal(yamlContent)
	if err != nil {
		return err
	}

	r.writeFileToConfigMapData(configMap, filename, string(bytesContent))
	return nil
}

// getConfigMapCopy is a generic function which creates a copy of a given
// source ConfigMap. The resulting copy is a ConfigMap with similar data, and
// with metadata set by the 'copyConfigMapObjectMeta' argument.
func (r *SynapseReconciler) getConfigMapCopy(
	sourceConfigMapName string,
	sourceConfigMapNamespace string,
	copyConfigMapObjectMeta metav1.ObjectMeta,
) (*corev1.ConfigMap, error) {
	sourceConfigMap := &corev1.ConfigMap{}

	ctx := context.TODO()

	// Get sourceConfigMap
	if err := r.Get(
		ctx,
		types.NamespacedName{Name: sourceConfigMapName, Namespace: sourceConfigMapNamespace},
		sourceConfigMap,
	); err != nil {
		return &corev1.ConfigMap{}, err
	}

	// Create a copy of the source ConfigMap with the same content.
	copyConfigMap := &corev1.ConfigMap{
		ObjectMeta: copyConfigMapObjectMeta,
		Data:       sourceConfigMap.Data,
	}

	return copyConfigMap, nil
}
