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

package utils

import (
	"errors"
	"strings"

	synapsev1alpha1 "github.com/opdev/synapse-operator/apis/synapse/v1alpha1"
)

func ComputeFQDN(name string, namespace string) string {
	return strings.Join([]string{name, namespace, "svc", "cluster", "local"}, ".")
}

func GetSynapseServerName(s synapsev1alpha1.Synapse) (string, error) {
	if s.Status.HomeserverConfiguration.ServerName != "" {
		return s.Status.HomeserverConfiguration.ServerName, nil
	}
	err := errors.New("ServerName not yet populated")
	return "", err
}
