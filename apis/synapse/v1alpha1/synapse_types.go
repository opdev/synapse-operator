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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SynapseSpec defines the desired state of Synapse
type SynapseSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:Required

	// Holds information related to the homeserver.yaml configuration file.
	// The user can either specify an existing ConfigMap by its Name and
	// Namespace containing a homeserver.yaml, or provide a set of values for
	// the creation of a configuration file from scratch.
	Homeserver SynapseHomeserver `json:"homeserver"`

	// Configuration options for optional matrix bridges
	Bridges SynapseBridges `json:"bridges,omitempty"`

	// +kubebuilder:default:=false

	// Set to true to create a new PostreSQL instance. The homeserver.yaml
	// 'database' section will be overwritten.
	CreateNewPostgreSQL bool `json:"createNewPostgreSQL,omitempty"`
}

type SynapseHomeserver struct {
	// Holds information about the ConfigMap containing the homeserver.yaml
	// configuration file to be used as input for the configuration of the
	// Synapse server.
	ConfigMap *SynapseHomeserverConfigMap `json:"configMap,omitempty"`

	// Holds the required values for the creation of a homeserver.yaml
	// configuration file by the Synapse Operator
	Values *SynapseHomeserverValues `json:"values,omitempty"`
}

type SynapseHomeserverConfigMap struct {
	// +kubebuilder:validation:Required

	// Name of the ConfigMap in the given Namespace.
	Name string `json:"name"`

	// Namespace in which the ConfigMap is living. If left empty, the Synapse
	// namespace is used.
	Namespace string `json:"namespace,omitempty"`
}

type SynapseHomeserverValues struct {
	// +kubebuilder:validation:Required

	// The public-facing domain of the server
	ServerName string `json:"serverName"`

	// +kubebuilder:validation:Required

	// Whether or not to report anonymized homeserver usage statistics
	ReportStats bool `json:"reportStats"`
}

type SynapseBridges struct {
	// Configuration options for the IRC bridge Heisenbridge. The user can
	// either:
	// * disable the deployment of the bridge.
	// * enable the bridge, without specifying additional configuration
	//   options. The bridge will be deployed with a default configuration.
	// * enable the bridge and specify an existing ConfigMap by its Name and
	//   Namespace containing a heisenbridge.yaml.
	Heisenbridge SynapseHeisenbridge `json:"heisenbridge,omitempty"`

	// Configuration options for the mautrix-signal bridge. The user can
	// either:
	// * disable the deployment of the bridge.
	// * enable the bridge, without specifying additional configuration
	//   options. The bridge will be deployed with a default configuration.
	// * enable the bridge and specify an existing ConfigMap by its Name and
	//   Namespace containing a config.yaml file.
	MautrixSignal SynapseMautrixSignal `json:"mautrixSignal,omitempty"`
}

type SynapseHeisenbridge struct {
	// +kubebuilder:default:=false

	// Whether to deploy Heisenbridge or not
	Enabled bool `json:"enabled,omitempty"`

	// Holds information about the ConfigMap containing the heisenbridge.yaml
	// configuration file to be used as input for the configuration of the
	// Heisenbridge IRC Bridge.
	ConfigMap SynapseHeisenbridgeConfigMap `json:"configMap,omitempty"`

	// +kubebuilder:default:=0

	// Controls the verbosity of the Heisenbrige:
	// * 0 corresponds to normal level of logs
	// * 1 corresponds to "-v"
	// * 2 corresponds to "-vv"
	// * 3 corresponds to "-vvv"
	VerboseLevel int `json:"verboseLevel,omitempty"`
}

type SynapseHeisenbridgeConfigMap struct {
	// +kubebuilder:validation:Required

	// Name of the ConfigMap in the given Namespace.
	Name string `json:"name"`

	// Namespace in which the ConfigMap is living. If left empty, the Synapse
	// namespace is used.
	Namespace string `json:"namespace,omitempty"`
}

type SynapseMautrixSignal struct {
	// +kubebuilder:default:=false

	// Whether to deploy mautrix-signal or not
	Enabled bool `json:"enabled,omitempty"`

	// Holds information about the ConfigMap containing the config.yaml
	// configuration file to be used as input for the configuration of the
	// mautrix-signal Bridge.
	ConfigMap SynapseMautrixSignalConfigMap `json:"configMap,omitempty"`
}

type SynapseMautrixSignalConfigMap struct {
	// +kubebuilder:validation:Required

	// Name of the ConfigMap in the given Namespace.
	Name string `json:"name"`

	// Namespace in which the ConfigMap is living. If left empty, the Synapse
	// namespace is used.
	Namespace string `json:"namespace,omitempty"`
}

// SynapseStatus defines the observed state of Synapse
type SynapseStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Connection information to the external PostgreSQL Database
	DatabaseConnectionInfo SynapseStatusDatabaseConnectionInfo `json:"databaseConnectionInfo,omitempty"`

	// Holds configuration information for Synapse
	HomeserverConfiguration SynapseStatusHomeserverConfiguration `json:"homeserverConfiguration,omitempty"`

	// State of the Synapse instance
	State string `json:"state,omitempty"`

	// Reason for the current Synapse State
	Reason string `json:"reason,omitempty"`
}

type SynapseStatusDatabaseConnectionInfo struct {
	// Endpoint to connect to the PostgreSQL database
	ConnectionURL string `json:"connectionURL,omitempty"`

	// Name of the database to connect to
	DatabaseName string `json:"databaseName,omitempty"`

	// User allowed to query the given database
	User string `json:"user,omitempty"`

	// Base64 encoded password
	Password string `json:"password,omitempty"`

	// State of the PostgreSQL database
	State string `json:"State,omitempty"`
}

type SynapseStatusHomeserverConfiguration struct {
	// The public-facing domain of the server
	ServerName string `json:"serverName,omitempty"`

	// Whether or not to report anonymized homeserver usage statistics
	ReportStats bool `json:"reportStats,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Synapse is the Schema for the synapses API
type Synapse struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +kubebuilder:validation:Required
	Spec   SynapseSpec   `json:"spec"`
	Status SynapseStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SynapseList contains a list of Synapse
type SynapseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Synapse `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Synapse{}, &SynapseList{})
}
