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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MautrixSignalSpec defines the desired state of MautrixSignal. The user can
// either:
//   - enable the bridge, without specifying additional configuration options.
//     The bridge will be deployed with a default configuration.
//   - enable the bridge and specify an existing ConfigMap by its Name and
//     Namespace containing a config.yaml file.
type MautrixSignalSpec struct {
	// Holds information about the ConfigMap containing the config.yaml
	// configuration file to be used as input for the configuration of the
	// mautrix-signal bridge.
	ConfigMap MautrixSignalConfigMap `json:"configMap,omitempty"`

	// +kubebuilder:validation:Required

	// Name of the Synapse instance, living in the same namespace.
	Synapse MautrixSignalSynapseSpec `json:"synapse"`
}

type MautrixSignalSynapseSpec struct {
	// +kubebuilder:validation:Required

	// Name of the Synapse instance
	Name string `json:"name"`

	// Namespace of the Synapse instance
	// TODO: Complete
	Namespace string `json:"namespace,omitempty"`
}

type MautrixSignalConfigMap struct {
	// +kubebuilder:validation:Required

	// Name of the ConfigMap in the given Namespace.
	Name string `json:"name"`

	// Namespace in which the ConfigMap is living. If left empty, the Synapse
	// namespace is used.
	Namespace string `json:"namespace,omitempty"`
}

// MautrixSignalStatus defines the observed state of MautrixSignal
type MautrixSignalStatus struct {
	// State of the MautrixSignal instance
	State string `json:"state,omitempty"`

	// Reason for the current MautrixSignal State
	Reason string `json:"reason,omitempty"`

	// Information related to the Synapse instance associated with this bridge
	Synapse MautrixSignalStatusSynapse `json:"synapse,omitempty"`
}

type MautrixSignalStatusSynapse struct {
	ServerName string `json:"serverName,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// MautrixSignal is the Schema for the mautrixsignals API
type MautrixSignal struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +kubebuilder:validation:Required
	Spec   MautrixSignalSpec   `json:"spec"`
	Status MautrixSignalStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MautrixSignalList contains a list of MautrixSignal
type MautrixSignalList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MautrixSignal `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MautrixSignal{}, &MautrixSignalList{})
}
