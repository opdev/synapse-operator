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

package mautrixsignal

import (
	"context"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/opdev/subreconciler"
	synapsev1alpha1 "github.com/opdev/synapse-operator/api/synapse/v1alpha1"
	"github.com/opdev/synapse-operator/helpers/reconcile"
	"github.com/opdev/synapse-operator/helpers/utils"
	"github.com/opdev/synapse-operator/internal/templates"
)

// reconcileMautrixSignalConfigMap is a function of type FnWithRequest, to
// be called in the main reconciliation loop.
//
// It reconciles the mautrix-signal ConfigMap to its desired state. It is
// called only if the user hasn't provided its own ConfigMap for
// mautrix-signal.
func (r *MautrixSignalReconciler) reconcileMautrixSignalConfigMap(
	ctx context.Context,
	req ctrl.Request,
) (*ctrl.Result, error) {
	ms := &synapsev1alpha1.MautrixSignal{}
	if r, err := utils.GetResource(ctx, r.Client, req, ms); subreconciler.ShouldHaltOrRequeue(r, err) {
		return r, err
	}

	desiredConfigMap, err := r.configMapForMautrixSignal(ms)
	if err != nil {
		return subreconciler.RequeueWithError(err)
	}

	if err := reconcile.ReconcileResource(
		ctx,
		r.Client,
		desiredConfigMap,
		&corev1.ConfigMap{},
	); err != nil {
		return subreconciler.RequeueWithError(err)
	}

	return subreconciler.ContinueReconciling()
}

// configMapForSynapse returns a synapse ConfigMap object
func (r *MautrixSignalReconciler) configMapForMautrixSignal(
	ms *synapsev1alpha1.MautrixSignal,
) (*corev1.ConfigMap, error) {
	synapseName := ms.Spec.Synapse.Name
	synapseNamespace := utils.ComputeNamespace(ms.Namespace, ms.Spec.Synapse.Namespace)

	type configmapExtraValues struct {
		synapsev1alpha1.MautrixSignal
		SynapseFQDN       string
		MautrixsignalFQDN string
	}

	extraValues := configmapExtraValues{
		MautrixSignal:     *ms,
		SynapseFQDN:       utils.ComputeFQDN(synapseName, synapseNamespace),
		MautrixsignalFQDN: utils.ComputeFQDN(ms.Name, ms.Namespace),
	}

	cm, err := templates.ResourceFromTemplate[configmapExtraValues, corev1.ConfigMap](
		&extraValues,
		"mautrixsignal_configmap",
	)
	if err != nil {
		return nil, fmt.Errorf("could not get template: %v", err)
	}

	// Set MautrixSignal instance as the owner and controller
	if err := ctrl.SetControllerReference(ms, cm, r.Scheme); err != nil {
		return &corev1.ConfigMap{}, err
	}

	return cm, nil
}

// configureMautrixSignalConfigMap is a function of type FnWithRequest, to
// be called in the main reconciliation loop.
//
// Following the previous copy of the user-provided ConfigMap, it edits the
// content of the copy to ensure that mautrix-signal is correctly configured.
func (r *MautrixSignalReconciler) configureMautrixSignalConfigMap(
	ctx context.Context, req ctrl.Request,
) (*ctrl.Result, error) {
	ms := &synapsev1alpha1.MautrixSignal{}
	if r, err := utils.GetResource(ctx, r.Client, req, ms); subreconciler.ShouldHaltOrRequeue(r, err) {
		return r, err
	}

	keyForConfigMap := types.NamespacedName{
		Name:      ms.Name,
		Namespace: ms.Namespace,
	}

	// Correct data in mautrix-signal ConfigMap
	if err := utils.UpdateConfigMap(
		ctx,
		r.Client,
		keyForConfigMap,
		ms,
		r.updateMautrixSignalData,
		"config.yaml",
	); err != nil {
		return subreconciler.RequeueWithError(err)
	}

	return subreconciler.ContinueReconciling()
}

// updateMautrixSignalData is a function of type updateDataFunc function to
// be passed as an argument in a call to updateConfigMap.
//
// It configures the user-provided config.yaml with the correct values. Among
// other things, it ensures that the bridge can reach the Synapse homeserver
// and knows the correct path to the signald socket.
func (r *MautrixSignalReconciler) updateMautrixSignalData(
	obj client.Object,
	config map[string]interface{},
) error {
	ms := obj.(*synapsev1alpha1.MautrixSignal)

	synapseName := ms.Spec.Synapse.Name
	synapseNamespace := utils.ComputeNamespace(ms.Namespace, ms.Spec.Synapse.Namespace)
	synapseServerName := ms.Status.Synapse.ServerName

	// Update the homeserver section so that the bridge can reach Synapse
	configHomeserver, ok := config["homeserver"].(map[string]interface{})
	if !ok {
		err := errors.New("cannot parse mautrix-signal config.yaml: error parsing 'homeserver' section")
		return err
	}
	configHomeserver["address"] = "http://" + utils.ComputeFQDN(synapseName, synapseNamespace) + ":8008"
	configHomeserver["domain"] = synapseServerName
	config["homeserver"] = configHomeserver

	// Update the appservice section so that Synapse can reach the bridge
	configAppservice, ok := config["appservice"].(map[string]interface{})
	if !ok {
		err := errors.New("cannot parse mautrix-signal config.yaml: error parsing 'appservice' section")
		return err
	}
	configAppservice["address"] = "http://" + utils.ComputeFQDN(ms.Name, ms.Namespace) + ":29328"
	config["appservice"] = configAppservice

	// Update the path to the signal socket path
	configSignal, ok := config["signal"].(map[string]interface{})
	if !ok {
		err := errors.New("cannot parse mautrix-signal config.yaml: error parsing 'signal' section")
		return err
	}
	configSignal["socket_path"] = "/signald/signald.sock"
	config["signal"] = configSignal

	// Update persmissions to use the correct domain name
	configBridge, ok := config["bridge"].(map[string]interface{})
	if !ok {
		err := errors.New("cannot parse mautrix-signal config.yaml: error parsing 'bridge' section")
		return err
	}
	configBridge["permissions"] = map[string]string{
		"*":                           "relay",
		synapseServerName:             "user",
		"@admin:" + synapseServerName: "admin",
	}
	config["bridge"] = configBridge

	// Update the path to the log file
	configLogging, ok := config["logging"].(map[string]interface{})
	if !ok {
		err := errors.New("cannot parse mautrix-signal config.yaml: error parsing 'logging' section")
		return err
	}
	configLoggingHandlers, ok := configLogging["handlers"].(map[string]interface{})
	if !ok {
		err := errors.New("cannot parse mautrix-signal config.yaml: error parsing 'logging/handlers' section")
		return err
	}
	configLoggingHandlersFile, ok := configLoggingHandlers["file"].(map[string]interface{})
	if !ok {
		err := errors.New("cannot parse mautrix-signal config.yaml: error parsing 'logging/handlers/file' section")
		return err
	}
	configLoggingHandlersFile["filename"] = "/data/mautrix-signal.log"
	configLoggingHandlers["file"] = configLoggingHandlersFile
	configLogging["handlers"] = configLoggingHandlers
	config["logging"] = configLogging

	return nil
}
