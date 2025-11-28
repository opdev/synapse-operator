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

package synapse

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math/big"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	subreconciler "github.com/opdev/subreconciler"
	synapsev1alpha1 "github.com/opdev/synapse-operator/api/synapse/v1alpha1"
	"github.com/opdev/synapse-operator/helpers/reconcile"
	"github.com/opdev/synapse-operator/helpers/utils"
	"github.com/opdev/synapse-operator/internal/templates"
)

// reconcileSynapseConfigMap is a function of type FnWithRequest, to be
// called in the main reconciliation loop.
//
// It reconciles the synapse ConfigMap to its desired state. It is called only
// if the user hasn't provided its own ConfigMap for synapse
func (r *SynapseReconciler) reconcileSynapseConfigMap(ctx context.Context, req ctrl.Request) (*ctrl.Result, error) {
	s := &synapsev1alpha1.Synapse{}
	if r, err := utils.GetResource(ctx, r.Client, req, s); subreconciler.ShouldHaltOrRequeue(r, err) {
		return r, err
	}

	desiredConfigMap, err := r.configMapForSynapse(s)
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
func (r *SynapseReconciler) configMapForSynapse(s *synapsev1alpha1.Synapse) (*corev1.ConfigMap, error) {
	type configmapExtraValues struct {
		synapsev1alpha1.Synapse
		RegistrationSharedSecret string
		MacaroonSecretKey        string
		FormSecret               string
	}

	registrationSharedSecret, err := generateASCIIPassword(defaultGeneratedPasswordLength)
	if err != nil {
		return nil, err
	}
	macaroonSecretKey, err := generateASCIIPassword(defaultGeneratedPasswordLength)
	if err != nil {
		return nil, err
	}
	formSecret, err := generateASCIIPassword(defaultGeneratedPasswordLength)
	if err != nil {
		return nil, err
	}

	extraValues := configmapExtraValues{
		Synapse:                  *s,
		RegistrationSharedSecret: registrationSharedSecret,
		MacaroonSecretKey:        macaroonSecretKey,
		FormSecret:               formSecret,
	}

	cm, err := templates.ResourceFromTemplate[configmapExtraValues, corev1.ConfigMap](&extraValues, "synapse_configmap")
	if err != nil {
		return nil, fmt.Errorf("could not get template: %v", err)
	}

	// Set Synapse instance as the owner and controller
	if err := ctrl.SetControllerReference(s, cm, r.Scheme); err != nil {
		return &corev1.ConfigMap{}, err
	}

	return cm, nil
}

// parseInputSynapseConfigMap is a function of type FnWithRequest, to be
// called in the main reconciliation loop.
//
// It checks that the ConfigMap referenced by
// synapse.Spec.Homeserver.ConfigMap.Name exists and extrats the server_name
// and report_stats values.
func (r *SynapseReconciler) parseInputSynapseConfigMap(ctx context.Context, req ctrl.Request) (*ctrl.Result, error) {
	log := logf.FromContext(ctx)

	s := &synapsev1alpha1.Synapse{}
	if r, err := utils.GetResource(ctx, r.Client, req, s); subreconciler.ShouldHaltOrRequeue(r, err) {
		return r, err
	}

	var inputConfigMap corev1.ConfigMap // the user-provided ConfigMap. It should contain a valid homeserver.yaml
	ConfigMapName := s.Spec.Homeserver.ConfigMap.Name
	ConfigMapNamespace := utils.ComputeNamespace(s.Namespace, s.Spec.Homeserver.ConfigMap.Namespace)
	keyForInputConfigMap := types.NamespacedName{
		Name:      ConfigMapName,
		Namespace: ConfigMapNamespace,
	}

	// Get and validate the inputConfigMap
	if err := r.Get(ctx, keyForInputConfigMap, &inputConfigMap); err != nil {
		reason := "ConfigMap " + ConfigMapName + " does not exist in namespace " + ConfigMapNamespace
		utils.SetFailedState(ctx, r.Client, s, reason)

		log.Error(
			err,
			"Failed to get ConfigMap",
			"ConfigMap.Namespace",
			ConfigMapNamespace,
			"ConfigMap.Name",
			ConfigMapName,
		)
		return subreconciler.RequeueWithDelayAndError(time.Duration(30), err)
	}

	if err := r.ParseHomeserverConfigMap(ctx, s, inputConfigMap); err != nil {
		return subreconciler.RequeueWithDelayAndError(time.Duration(30), err)
	}

	err := utils.UpdateResourceStatus(ctx, r.Client, s, &synapsev1alpha1.Synapse{})
	if err != nil {
		log.Error(err, "Error updating Synapse Status")
		return subreconciler.RequeueWithError(err)
	}

	return subreconciler.ContinueReconciling()
}

// ParseHomeserverConfigMap loads the ConfigMap, which name is determined by
// Spec.Homeserver.ConfigMap.Name, run validation checks and fetch necesarry
// value needed to configure the Synapse Deployment.
func (r *SynapseReconciler) ParseHomeserverConfigMap(
	ctx context.Context,
	synapse *synapsev1alpha1.Synapse,
	cm corev1.ConfigMap,
) error {
	log := logf.FromContext(ctx)

	// TODO:
	// - Ensure that key path is and log config file path are in /data
	// - Otherwise, edit homeserver.yaml with new paths

	// Load and validate homeserver.yaml
	homeserver, err := utils.LoadYAMLFileFromConfigMapData(cm, "homeserver.yaml")
	if err != nil {
		return err
	}

	// Fetch server_name and report_stats
	if _, ok := homeserver["server_name"]; !ok {
		err := errors.New("missing server_name key in homeserver.yaml")
		log.Error(err, "Missing server_name key in homeserver.yaml")
		return err
	}
	server_name, ok := homeserver["server_name"].(string)
	if !ok {
		err := errors.New("error converting server_name to string")
		log.Error(err, "Error converting server_name to string")
		return err
	}

	if _, ok := homeserver["report_stats"]; !ok {
		err := errors.New("missing report_stats key in homeserver.yaml")
		log.Error(err, "Missing report_stats key in homeserver.yaml")
		return err
	}
	report_stats, ok := homeserver["report_stats"].(bool)
	if !ok {
		err := errors.New("error converting report_stats to bool")
		log.Error(err, "Error converting report_stats to bool")
		return err
	}

	// Populate the Status.HomeserverConfiguration with values defined in homeserver.yaml
	synapse.Status.HomeserverConfiguration.ServerName = server_name
	synapse.Status.HomeserverConfiguration.ReportStats = report_stats

	log.Info(
		"Loaded homeserver.yaml from ConfigMap successfully",
		"server_name:", synapse.Status.HomeserverConfiguration.ServerName,
		"report_stats:", synapse.Status.HomeserverConfiguration.ReportStats,
	)

	return nil
}

// updateSynapseConfigMapForBridges is a function of type
// FnWithRequest, to be called in the main reconciliation loop.
//
// It registers the bridges as a application services in the
// homeserver.yaml config file.
func (r *SynapseReconciler) updateSynapseConfigMapForBridges(
	ctx context.Context,
	req ctrl.Request,
) (*ctrl.Result, error) {
	s := &synapsev1alpha1.Synapse{}
	if r, err := utils.GetResource(ctx, r.Client, req, s); subreconciler.ShouldHaltOrRequeue(r, err) {
		return r, err
	}

	keyForSynapse := types.NamespacedName{
		Name:      s.Name,
		Namespace: s.Namespace,
	}

	// Update the Synapse ConfigMap to enable bridges
	if err := utils.UpdateConfigMap(
		ctx,
		r.Client,
		keyForSynapse,
		s,
		r.updateHomeserverWithBridgesInfo,
		"homeserver.yaml",
	); err != nil {
		return subreconciler.RequeueWithError(err)
	}

	return subreconciler.ContinueReconciling()
}

func (r *SynapseReconciler) updateHomeserverWithBridgesInfo(
	obj client.Object,
	homeserver map[string]any,
) error {
	s := obj.(*synapsev1alpha1.Synapse)

	var appServiceConfigPaths []string

	if s.Status.Bridges.Heisenbridge.Enabled {
		appServiceConfigPaths = append(appServiceConfigPaths, "/data-heisenbridge/heisenbridge.yaml")
	}

	if s.Status.Bridges.MautrixSignal.Enabled {
		appServiceConfigPaths = append(appServiceConfigPaths, "/data-mautrixsignal/registration.yaml")
	}

	if len(appServiceConfigPaths) > 0 {
		r.addAppServicesToHomeserver(homeserver, appServiceConfigPaths)
	}

	return nil
}

func (r *SynapseReconciler) addAppServicesToHomeserver(
	homeserver map[string]any,
	configFilePaths []string,
) {
	homeserverAppService, ok := homeserver["app_service_config_files"].([]string)
	if !ok {
		// "app_service_config_files" key not present, or malformed. Overwrite with
		// the given app_service config file.
		homeserver["app_service_config_files"] = configFilePaths
	} else {
		// There are already app services registered. Adding to the list.
		homeserver["app_service_config_files"] = append(homeserverAppService, configFilePaths...)
	}
}

// The following constant is used as a part of password generation.
const (
	// DefaultGeneratedPasswordLength is the default length of what a generated
	// password should be if it's not set elsewhere
	defaultGeneratedPasswordLength = 24
)

// accumulate gathers n bytes from f and returns them as a string. It returns
// an empty string when f returns an error.
func accumulate(n int, f func() (byte, error)) (string, error) {
	result := make([]byte, n)

	for i := range result {
		if b, err := f(); err == nil {
			result[i] = b
		} else {
			return "", err
		}
	}

	return string(result), nil
}

// randomCharacter builds a function that returns random bytes from class.
func randomCharacter(random io.Reader, class string) func() (byte, error) {
	if random == nil {
		panic("requires a random source")
	}
	if len(class) == 0 {
		panic("class cannot be empty")
	}

	size := big.NewInt(int64(len(class)))

	return func() (byte, error) {
		if i, err := rand.Int(random, size); err == nil {
			return class[int(i.Int64())], nil
		} else {
			return 0, err
		}
	}
}

// policyASCII is the list of acceptable characters from which to generate an
// ASCII password.
const policyASCII = `` +
	`()*+,-./` + `:;<=>?@` + `[]^_` + `{|}` +
	`ABCDEFGHIJKLMNOPQRSTUVWXYZ` +
	`abcdefghijklmnopqrstuvwxyz` +
	`0123456789`

var randomASCII = randomCharacter(rand.Reader, policyASCII)

// GenerateASCIIPassword returns a random string of printable ASCII characters.
func generateASCIIPassword(length int) (string, error) {
	return accumulate(length, randomASCII)
}
