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

	// +kubebuilder:default:=false

	// Set to true to create a new PostreSQL instance. The homeserver.yaml
	// 'database' section will be overwritten.
	CreateNewPostgreSQL bool `json:"createNewPostgreSQL,omitempty"`

	// +kubebuilder:default:=false

	// Set to true if deploying on OpenShift
	IsOpenshift bool `json:"isOpenshift,omitempty"`
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

// SynapseStatus defines the observed state of Synapse
type SynapseStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Connection information to the external PostgreSQL Database
	DatabaseConnectionInfo SynapseStatusDatabaseConnectionInfo `json:"databaseConnectionInfo,omitempty"`

	// Holds configuration information for Synapse
	HomeserverConfiguration SynapseStatusHomeserverConfiguration `json:"homeserverConfiguration,omitempty"`

	// Information on the bridges deployed alongside Synapse
	Bridges SynapseStatusBridges `json:"bridges,omitempty"`

	// State of the Synapse instance
	State string `json:"state,omitempty"`

	// Reason for the current Synapse State
	Reason string `json:"reason,omitempty"`

	// +kubebuilder:default:=false
	NeedsReconcile bool `json:"needsReconcile,omitempty"`
}

type SynapseStatusBridges struct {
	// Information on the Heisenbridge (IRC Bridge).
	Heisenbridge SynapseStatusBridgesHeisenbridge `json:"heisenbridge,omitempty"`

	// Information on the mautrix-signal bridge.
	MautrixSignal SynapseStatusBridgesMautrixSignal `json:"mautrixsignal,omitempty"`
}

type SynapseStatusBridgesHeisenbridge struct {
	// +kubebuilder:default:=false

	// Whether a Heisenbridge has been deployed for this Synapse instance
	Enabled bool `json:"enabled,omitempty"`

	// Name of the Heisenbridge object
	Name string `json:"name,omitempty"`
}

type SynapseStatusBridgesMautrixSignal struct {
	// Whether a mautrix-signal has been deployed for this Synapse instance
	Enabled bool `json:"enabled,omitempty"`

	// Name of the mautrix-signal bridge object
	Name string `json:"name,omitempty"`
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
