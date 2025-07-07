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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HeisenbridgeSpec defines the desired state of Heisenbridge. The user can
// either:
//   - enable the bridge, without specifying additional configuration options.
//     The bridge will be deployed with a default configuration.
//   - enable the bridge and specify an existing ConfigMap by its Name and
//     Namespace containing a heisenbridge.yaml.
type HeisenbridgeSpec struct {
	// Holds information about the ConfigMap containing the heisenbridge.yaml
	// configuration file to be used as input for the configuration of the
	// Heisenbridge IRC Bridge.
	ConfigMap HeisenbridgeConfigMap `json:"configMap,omitempty"`

	// +kubebuilder:default:=0

	// Controls the verbosity of the Heisenbrige:
	// * 0 corresponds to normal level of logs
	// * 1 corresponds to "-v"
	// * 2 corresponds to "-vv"
	// * 3 corresponds to "-vvv"
	VerboseLevel int `json:"verboseLevel,omitempty"`

	// +kubebuilder:validation:Required

	// Name of the Synapse instance, living in the same namespace.
	Synapse HeisenbridgeSynapseSpec `json:"synapse"`
}

type HeisenbridgeSynapseSpec struct {
	// +kubebuilder:validation:Required

	// Name of the Synapse instance
	Name string `json:"name"`

	// Namespace of the Synapse instance
	// TODO: Complete
	Namespace string `json:"namespace,omitempty"`
}

type HeisenbridgeConfigMap struct {
	// +kubebuilder:validation:Required

	// Name of the ConfigMap in the given Namespace.
	Name string `json:"name"`

	// Namespace in which the ConfigMap is living. If left empty, the
	// Heisenbridge namespace is used.
	Namespace string `json:"namespace,omitempty"`
}

// HeisenbridgeStatus defines the observed state of Heisenbridge.
type HeisenbridgeStatus struct {
	// State of the Heisenbridge instance
	State string `json:"state,omitempty"`

	// Reason for the current Heisenbridge State
	Reason string `json:"reason,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Heisenbridge is the Schema for the heisenbridges API.
type Heisenbridge struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +kubebuilder:validation:Required
	Spec   HeisenbridgeSpec   `json:"spec"`
	Status HeisenbridgeStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// HeisenbridgeList contains a list of Heisenbridge.
type HeisenbridgeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Heisenbridge `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Heisenbridge{}, &HeisenbridgeList{})
}

func (h *Heisenbridge) GetSynapseName() string {
	return h.Spec.Synapse.Name
}

func (h *Heisenbridge) GetSynapseNamespace() string {
	return h.Spec.Synapse.Namespace
}
