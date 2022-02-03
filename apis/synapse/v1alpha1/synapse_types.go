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

	// Name of the ConfigMap holding the homeserver.yaml config file. It is
	// used as an input for the configuration of the Synapse Server. It can be
	// modified by the Synapse Operator (e.g. the DB section)
	HomeserverConfigMapName string `json:"homeserverConfigMapName"`

	// +kubebuilder:default:=false

	// Set to true to create a new PostreSQL instance.
	CreateNewPostgreSQL bool `json:"createNewPostgreSQL,omitempty"`
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
