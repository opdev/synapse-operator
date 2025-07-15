/*
Copyright 2025.

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

package utils

/* This file puts together generic functions for ConfiMap manipulation */
import (
	"context"
	"errors"

	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Defines a function type. Functions of the updateDataFunc type implements the
// logic to update the data of a configmap, defined by the 'data' argument.
type updateDataFunc func(obj client.Object, data map[string]any) error

// A generic function to update an existing ConfigMap. It takes as arguments:
// * The context
// * The key (name and namespace) of the ConfigMap to update
// * The Synapse object being reconciled
// * The function to be called to actually update the ConfigMap's content
// * The name of the file to update in the ConfigMap
func UpdateConfigMap(
	ctx context.Context,
	kubeClient client.Client,
	key types.NamespacedName,
	obj client.Object,
	updateData updateDataFunc,
	filename string,
) error {
	cm := &corev1.ConfigMap{}

	// Get latest ConfigMap version
	if err := kubeClient.Get(ctx, key, cm); err != nil {
		return err
	}

	if err := UpdateConfigMapData(cm, obj, updateData, filename); err != nil {
		return err
	}

	// Update ConfigMap
	if err := kubeClient.Update(ctx, cm); err != nil {
		return err
	}

	return nil
}

func UpdateConfigMapData(
	cm *corev1.ConfigMap,
	obj client.Object,
	updateData updateDataFunc,
	filename string,
) error {
	// Load file to update from ConfigMap
	data, err := LoadYAMLFileFromConfigMapData(*cm, filename)
	if err != nil {
		return err
	}

	// Update the content of the file
	if err := updateData(obj, data); err != nil {
		return err
	}

	// Write new content into ConfigMap data
	if err := writeYAMLFileToConfigMapData(cm, filename, data); err != nil {
		return err
	}

	return nil
}

func loadFileFromConfigMapData(
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

func LoadYAMLFileFromConfigMapData(
	configMap corev1.ConfigMap,
	filename string,
) (map[string]any, error) {
	yamlContent := map[string]any{}

	content, err := loadFileFromConfigMapData(configMap, filename)
	if err != nil {
		return yamlContent, err
	}
	if err := yaml.Unmarshal([]byte(content), yamlContent); err != nil {
		return yamlContent, err
	}

	return yamlContent, nil
}

func writeFileToConfigMapData(
	configMap *corev1.ConfigMap,
	filename string,
	content string,
) {
	configMap.Data = map[string]string{filename: content}
}

func writeYAMLFileToConfigMapData(
	configMap *corev1.ConfigMap,
	filename string,
	yamlContent map[string]any,
) error {
	bytesContent, err := yaml.Marshal(yamlContent)
	if err != nil {
		return err
	}

	writeFileToConfigMapData(configMap, filename, string(bytesContent))
	return nil
}

// getConfigMapCopy is a generic function which creates a copy of a given
// source ConfigMap. The resulting copy is a ConfigMap with similar data, and
// with metadata set by the 'copyConfigMapObjectMeta' argument.
func GetConfigMapCopy(
	kubeClient client.Client,
	sourceConfigMapName string,
	sourceConfigMapNamespace string,
	copyConfigMapObjectMeta metav1.ObjectMeta,
) (*corev1.ConfigMap, error) {
	sourceConfigMap := &corev1.ConfigMap{}

	ctx := context.TODO()

	// Get sourceConfigMap
	if err := kubeClient.Get(
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

// ConfigMap that are created by the user could be living in a different
// namespace as Synapse. getConfigMapNamespace provides a way to default to the
// Synapse namespace if none is provided.
func ComputeNamespace(defaultNamespace string, newNamespace string) string {
	if newNamespace != "" {
		return newNamespace
	}
	return defaultNamespace
}
