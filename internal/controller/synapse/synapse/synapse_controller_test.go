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
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	synapsev1alpha1 "github.com/opdev/synapse-operator/api/synapse/v1alpha1"
	"github.com/opdev/synapse-operator/helpers/utils"
)

var _ = Describe("Integration tests for the Synapse controller", Ordered, Label("integration"), func() {
	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		SynapseName        = "test-synapse"
		SynapseNamespace   = "default"
		InputConfigMapName = "test-configmap"
		ServerName         = "example.com"
		ReportStats        = false

		timeout  = time.Second * 2
		duration = time.Second * 2
		interval = time.Millisecond * 250
	)

	var k8sClient client.Client
	var k8sManager manager.Manager
	var testEnv *envtest.Environment
	var ctx context.Context
	var cancel context.CancelFunc

	var deleteResource func(client.Object, types.NamespacedName, bool)
	var checkSubresourceAbsence func(types.NamespacedName, ...client.Object)
	var checkResourcePresence func(client.Object, types.NamespacedName, metav1.OwnerReference)
	var checkStatus func(string, string, types.NamespacedName, client.Object)

	// Common function to start envTest
	var startenvTest = func() {
		// Retrieve the first found binary directory to allow running tests from IDEs
		if getFirstFoundEnvTestBinaryDir() != "" {
			testEnv.BinaryAssetsDirectory = getFirstFoundEnvTestBinaryDir()
		}

		cfg, err := testEnv.Start()
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg).NotTo(BeNil())

		Expect(synapsev1alpha1.AddToScheme(scheme.Scheme)).NotTo(HaveOccurred())
		// +kubebuilder:scaffold:scheme

		k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
		Expect(err).NotTo(HaveOccurred())
		Expect(k8sClient).NotTo(BeNil())

		k8sManager, err = ctrl.NewManager(cfg, ctrl.Options{
			Scheme: scheme.Scheme,
			Metrics: server.Options{
				BindAddress: "0",
			},
			Controller: config.Controller{
				SkipNameValidation: utils.BoolAddr(true),
			},
		})
		Expect(err).ToNot(HaveOccurred())

		err = (&SynapseReconciler{
			Client: k8sManager.GetClient(),
			Scheme: k8sManager.GetScheme(),
		}).SetupWithManager(k8sManager)
		Expect(err).ToNot(HaveOccurred())

		deleteResource = utils.DeleteResourceFunc(k8sClient, ctx, timeout, interval)
		checkSubresourceAbsence = utils.CheckSubresourceAbsenceFunc(k8sClient, ctx, timeout, interval)
		checkResourcePresence = utils.CheckResourcePresenceFunc(k8sClient, ctx, timeout, interval)
		checkStatus = utils.CheckStatusFunc(k8sClient, ctx, timeout, interval)

		go func() {
			defer GinkgoRecover()
			Expect(k8sManager.Start(ctx)).ToNot(HaveOccurred(), "failed to run manager")
		}()
	}

	Context("When a corectly configured Kubernetes cluster is present", func() {
		var _ = BeforeAll(func() {
			logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

			ctx, cancel = context.WithCancel(context.TODO())

			By("bootstrapping test environment")
			testEnv = &envtest.Environment{
				CRDDirectoryPaths: []string{
					filepath.Join("..", "..", "..", "..", "bundle", "manifests", "synapse.opdev.io_synapses.yaml"),
					filepath.Join("..", "..", "..", "..", "bundle", "manifests", "synapse.opdev.io_heisenbridges.yaml"),
					filepath.Join("..", "..", "..", "..", "bundle", "manifests", "synapse.opdev.io_mautrixsignals.yaml"),
				},
				ErrorIfCRDPathMissing: true,
				BinaryAssetsDirectory: filepath.Join("..", "..", "..", "bin", "k8s",
					fmt.Sprintf("1.31.0-%s-%s", runtime.GOOS, runtime.GOARCH)),
				AttachControlPlaneOutput: true,
			}

			startenvTest()
		})

		var _ = AfterAll(func() {
			cancel()
			By("tearing down the test environment")
			Expect(testEnv.Stop()).NotTo(HaveOccurred())
		})

		Context("Validating Synapse CRD Schema", func() {
			var obj map[string]any

			BeforeEach(func() {
				obj = map[string]any{
					"apiVersion": "synapse.opdev.io/v1alpha1",
					"kind":       "Synapse",
					"metadata": map[string]any{
						"name":      SynapseName,
						"namespace": SynapseNamespace,
					},
				}
			})

			DescribeTable("Creating a misconfigured Synapse instance",
				func(synapse_data map[string]any) {
					// Augment base synapse obj with additional fields
					for key, value := range synapse_data {
						obj[key] = value
					}
					// Create Unstructured object from synapse obj
					u := unstructured.Unstructured{Object: obj}
					Expect(k8sClient.Create(ctx, &u)).ShouldNot(Succeed())
				},
				Entry("when Synapse spec is missing", map[string]any{}),
				Entry("when Synapse spec is empty", map[string]any{
					"spec": map[string]any{},
				}),
				Entry("when Synapse spec is missing Homeserver", map[string]any{
					"spec": map[string]any{"IsOpenshift": true},
				}),
				Entry("when Synapse spec Homeserver is empty", map[string]any{
					"spec": map[string]any{
						"homeserver": map[string]any{},
					},
				}),
				Entry("when Synapse spec Homeserver possess both Values and ConfigMap", map[string]any{
					"spec": map[string]any{
						"homeserver": map[string]any{
							"configMap": map[string]any{
								"name":      InputConfigMapName,
								"namespace": SynapseNamespace,
							},
							"values": map[string]any{
								"serverName":  ServerName,
								"reportStats": ReportStats,
							},
						}},
				}),
				Entry("when Synapse spec Homeserver ConfigMap doesn't specify a Name", map[string]any{
					"spec": map[string]any{
						"homeserver": map[string]any{
							"configMap": map[string]any{
								"namespace": SynapseNamespace,
							},
						}},
				}),
				Entry("when Synapse spec Homeserver Values is missing ServerName", map[string]any{
					"spec": map[string]any{
						"homeserver": map[string]any{
							"values": map[string]any{
								"reportStats": ReportStats,
							},
						}},
				}),
				Entry("when Synapse spec Homeserver Values is missing ReportStats", map[string]any{
					"spec": map[string]any{
						"homeserver": map[string]any{
							"values": map[string]any{
								"serverName": ServerName,
							},
						}},
				}),
				// This should not work but passes
				PEntry("when Synapse spec possesses an invalid field", map[string]any{
					"spec": map[string]any{
						"homeserver": map[string]any{
							"configMap": map[string]any{
								"name":      InputConfigMapName,
								"namespace": SynapseNamespace,
							},
						},
						"invalidSpecFiels": "random",
					},
				}),
			)

			DescribeTable("Creating a correct Synapse instance",
				func(synapse_data map[string]any) {
					// Augment base synapse obj with additional fields
					for key, value := range synapse_data {
						obj[key] = value
					}
					// Create Unstructured object from synapse obj
					u := unstructured.Unstructured{Object: obj}
					// Use DryRun option to avoid cleaning up resources
					opt := client.CreateOptions{DryRun: []string{"All"}}
					Expect(k8sClient.Create(ctx, &u, &opt)).Should(Succeed())
				},
				Entry(
					"when the Homeserver Configuration file is provided via a ConfigMap",
					map[string]any{
						"spec": map[string]any{
							"homeserver": map[string]any{
								"configMap": map[string]any{
									"name":      InputConfigMapName,
									"namespace": SynapseNamespace,
								},
							},
							"createNewPostreSQL": true,
						},
					},
				),
				Entry(
					"when the Homeserver Configuration values are provided",
					map[string]any{
						"spec": map[string]any{
							"homeserver": map[string]any{
								"values": map[string]any{
									"serverName":  ServerName,
									"reportStats": ReportStats,
								},
							},
							"createNewPostreSQL": true,
						},
					},
				),
				Entry(
					"when registration option is provided",
					map[string]any{
						"spec": map[string]any{
							"homeserver": map[string]any{
								"values": map[string]any{
									"serverName":         ServerName,
									"reportStats":        ReportStats,
									"enableRegistration": true,
								},
							},
						},
					},
				),
			)
		})

		Context("When creating a valid Synapse instance", func() {
			var synapse *synapsev1alpha1.Synapse
			var createdConfigMap *corev1.ConfigMap
			var createdPVC *corev1.PersistentVolumeClaim
			var createdDeployment *appsv1.Deployment
			var createdService *corev1.Service
			var createdServiceAccount *corev1.ServiceAccount
			var createdRoleBinding *rbacv1.RoleBinding
			var synapseLookupKey types.NamespacedName
			var expectedOwnerReference metav1.OwnerReference
			var synapseSpec synapsev1alpha1.SynapseSpec

			var initSynapseVariables = func() {
				// Init variables
				synapseLookupKey = types.NamespacedName{Name: SynapseName, Namespace: SynapseNamespace}
				createdConfigMap = &corev1.ConfigMap{}
				createdPVC = &corev1.PersistentVolumeClaim{}
				createdDeployment = &appsv1.Deployment{}
				createdService = &corev1.Service{}
				createdServiceAccount = &corev1.ServiceAccount{}
				createdRoleBinding = &rbacv1.RoleBinding{}
				// The OwnerReference UID must be set after the Synapse instance has been
				// created.
				expectedOwnerReference = metav1.OwnerReference{
					Kind:               "Synapse",
					APIVersion:         "synapse.opdev.io/v1alpha1",
					Name:               SynapseName,
					Controller:         utils.BoolAddr(true),
					BlockOwnerDeletion: utils.BoolAddr(true),
				}
			}

			var createSynapseInstance = func() {
				By("Creating the Synapse instance")
				synapse = &synapsev1alpha1.Synapse{
					ObjectMeta: metav1.ObjectMeta{
						Name:      SynapseName,
						Namespace: SynapseNamespace,
					},
					Spec: synapseSpec,
				}
				Expect(k8sClient.Create(ctx, synapse)).Should(Succeed())

				By("Verifying that the Synapse object was created")
				Eventually(func() bool {
					err := k8sClient.Get(ctx, synapseLookupKey, synapse)
					return err == nil
				}, timeout, interval).Should(BeTrue())

				expectedOwnerReference.UID = synapse.GetUID()
			}

			var cleanupSynapseResources = func() {
				By("Cleaning up Synapse CR")
				Expect(k8sClient.Delete(ctx, synapse)).Should(Succeed())

				// Child resources must be manually deleted as the controllers responsible of
				// their lifecycle are not running.
				By("Cleaning up Synapse ConfigMap")
				deleteResource(createdConfigMap, synapseLookupKey, false)

				By("Cleaning up Synapse PVC")
				deleteResource(createdPVC, synapseLookupKey, true)

				By("Cleaning up Synapse Deployment")
				deleteResource(createdDeployment, synapseLookupKey, false)

				By("Cleaning up Synapse Service")
				deleteResource(createdService, synapseLookupKey, false)

				By("Cleaning up Synapse RoleBinding")
				deleteResource(createdRoleBinding, synapseLookupKey, false)

				By("Cleaning up Synapse ServiceAccount")
				deleteResource(createdServiceAccount, synapseLookupKey, false)
			}

			When("Specifying the Synapse configuration via Values", func() {
				When("Creating a simple Synapse instance", func() {
					BeforeAll(func() {
						initSynapseVariables()

						synapseSpec = synapsev1alpha1.SynapseSpec{
							Homeserver: synapsev1alpha1.SynapseHomeserver{
								Values: &synapsev1alpha1.SynapseHomeserverValues{
									ServerName:  ServerName,
									ReportStats: ReportStats,
								},
							},
							IsOpenshift: true,
						}

						createSynapseInstance()
					})

					AfterAll(func() {
						cleanupSynapseResources()
					})

					It("Should should update the Synapse Status", func() {
						expectedStatus := synapsev1alpha1.SynapseStatus{
							State:  "RUNNING",
							Reason: "",
							HomeserverConfiguration: synapsev1alpha1.SynapseStatusHomeserverConfiguration{
								ServerName:          ServerName,
								ReportStats:         ReportStats,
								RegistrationEnabled: false, // Default value
							},
						}
						// Status may need some time to be updated
						Eventually(func() synapsev1alpha1.SynapseStatus {
							_ = k8sClient.Get(ctx, synapseLookupKey, synapse)
							return synapse.Status
						}, timeout, interval).Should(Equal(expectedStatus))
					})

					It("Should create a Synapse ConfigMap", func() {
						checkResourcePresence(createdConfigMap, synapseLookupKey, expectedOwnerReference)
					})

					It("Should create a Synapse PVC", func() {
						checkResourcePresence(createdPVC, synapseLookupKey, expectedOwnerReference)
					})

					It("Should create a Synapse Deployment", func() {
						By("Checking that a Synapse Deployment exists and is correctly configured")
						checkResourcePresence(createdDeployment, synapseLookupKey, expectedOwnerReference)

						By("Checking that initContainer for generating config file contains the required environment variables")
						envVars := []corev1.EnvVar{{
							Name:  "SYNAPSE_SERVER_NAME",
							Value: ServerName,
						}, {
							Name:  "SYNAPSE_REPORT_STATS",
							Value: utils.BoolToYesNo(ReportStats),
						}}
						Expect(createdDeployment.Spec.Template.Spec.InitContainers[1].Env).Should(ContainElements(envVars))
					})

					It("Should create a Synapse Service", func() {
						checkResourcePresence(createdService, synapseLookupKey, expectedOwnerReference)
					})

					It("Should create a Synapse ServiceAccount", func() {
						checkResourcePresence(createdServiceAccount, synapseLookupKey, expectedOwnerReference)
					})

					It("Should create a Synapse RoleBinding", func() {
						checkResourcePresence(createdRoleBinding, synapseLookupKey, expectedOwnerReference)
					})
				})

				When("Specifying a registration option", func() {
					DescribeTable("Should create a ConfigMap with the correct enable_registration setting",
						func(enableRegistration *bool, expectedValue bool) {
							initSynapseVariables()

							synapseSpec = synapsev1alpha1.SynapseSpec{
								Homeserver: synapsev1alpha1.SynapseHomeserver{
									Values: &synapsev1alpha1.SynapseHomeserverValues{
										ServerName:  ServerName,
										ReportStats: ReportStats,
									},
								},
								IsOpenshift: true,
							}

							// Only set enableRegistration if provided
							if enableRegistration != nil {
								synapseSpec.Homeserver.Values.EnableRegistration = enableRegistration
							}

							createSynapseInstance()

							By("Checking that the Synapse Status is correctly updated")
							expectedStatus := synapsev1alpha1.SynapseStatus{
								State:  "RUNNING",
								Reason: "",
								HomeserverConfiguration: synapsev1alpha1.SynapseStatusHomeserverConfiguration{
									ServerName:          ServerName,
									ReportStats:         ReportStats,
									RegistrationEnabled: expectedValue,
								},
							}
							// Status may need some time to be updated
							Eventually(func() synapsev1alpha1.SynapseStatus {
								_ = k8sClient.Get(ctx, synapseLookupKey, synapse)
								return synapse.Status
							}, timeout, interval).Should(Equal(expectedStatus))

							// Verify ConfigMap content
							Eventually(func(g Gomega) {
								By("Checking that the Synapse ConfigMap exists")
								checkResourcePresence(createdConfigMap, synapseLookupKey, expectedOwnerReference)

								By("Checking that the ConfigMap contains the enable_registration setting")
								ConfigMapData, ok := createdConfigMap.Data["homeserver.yaml"]
								g.Expect(ok).Should(BeTrue())

								homeserver := make(map[string]any)
								g.Expect(yaml.Unmarshal([]byte(ConfigMapData), homeserver)).Should(Succeed())

								// Check enable_registration setting
								enableRegistrationValue, exists := homeserver["enable_registration"]
								if exists {
									g.Expect(enableRegistrationValue).Should(Equal(expectedValue))
								} else {
									// If not present, Synapse defaults to false
									g.Expect(expectedValue).Should(BeFalse())
								}

								// Check enable_registration_without_verification setting
								enableRegistrationWithoutVerificationValue, withoutVerificationExists := homeserver["enable_registration_without_verification"]
								if expectedValue {
									// When registration is enabled, enable_registration_without_verification should also be true
									g.Expect(withoutVerificationExists).Should(BeTrue())
									g.Expect(enableRegistrationWithoutVerificationValue).Should(BeTrue())
								} else {
									// When registration is disabled, enable_registration_without_verification should not be present
									g.Expect(withoutVerificationExists).Should(BeFalse())
								}
							}, timeout, interval).Should(Succeed())

							cleanupSynapseResources()
						},
						Entry("when enableRegistration is nil (default)", nil, false),
						Entry("when enableRegistration is false", utils.BoolAddr(false), false),
						Entry("when enableRegistration is true", utils.BoolAddr(true), true),
					)
				})
			})

			When("Specifying the Synapse configuration via a ConfigMap", func() {
				var inputConfigMap *corev1.ConfigMap
				var inputConfigmapData map[string]string

				var createSynapseConfigMap = func() {
					By("Creating a ConfigMap containing a basic homeserver.yaml")
					// Populate the ConfigMap with the minimum data needed
					inputConfigMap = &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      InputConfigMapName,
							Namespace: SynapseNamespace,
						},
						Data: inputConfigmapData,
					}
					Expect(k8sClient.Create(ctx, inputConfigMap)).Should(Succeed())
				}

				var cleanupSynapseConfigMap = func() {
					By("Cleaning up ConfigMap")
					Expect(k8sClient.Delete(ctx, inputConfigMap)).Should(Succeed())
				}

				When("Creating a simple Synapse instance", func() {
					BeforeAll(func() {
						initSynapseVariables()

						inputConfigmapData = map[string]string{
							"homeserver.yaml": "server_name: " + ServerName + "\n" +
								"report_stats: " + strconv.FormatBool(ReportStats),
						}

						synapseSpec = synapsev1alpha1.SynapseSpec{
							Homeserver: synapsev1alpha1.SynapseHomeserver{
								ConfigMap: &synapsev1alpha1.SynapseHomeserverConfigMap{
									Name: InputConfigMapName,
								},
							},
							IsOpenshift: true,
						}

						createSynapseConfigMap()
						createSynapseInstance()
					})

					AfterAll(func() {
						cleanupSynapseResources()
						cleanupSynapseConfigMap()
					})

					It("Should should update the Synapse Status", func() {
						expectedStatus := synapsev1alpha1.SynapseStatus{
							State:  "RUNNING",
							Reason: "",
							HomeserverConfiguration: synapsev1alpha1.SynapseStatusHomeserverConfiguration{
								ServerName:          ServerName,
								ReportStats:         ReportStats,
								RegistrationEnabled: false, // Default value
							},
						}
						// Status may need some time to be updated
						Eventually(func() synapsev1alpha1.SynapseStatus {
							_ = k8sClient.Get(ctx, synapseLookupKey, synapse)
							return synapse.Status
						}, timeout, interval).Should(Equal(expectedStatus))
					})

					It("Should create a Synapse ConfigMap", func() {
						checkResourcePresence(createdConfigMap, synapseLookupKey, expectedOwnerReference)
					})

					It("Should create a Synapse PVC", func() {
						checkResourcePresence(createdPVC, synapseLookupKey, expectedOwnerReference)
					})

					It("Should create a Synapse Deployment", func() {
						By("Checking that a Synapse Deployment exists and is correctly configured")
						checkResourcePresence(createdDeployment, synapseLookupKey, expectedOwnerReference)

						By("Checking that initContainer for generating config file contains the required environment variables")
						envVars := []corev1.EnvVar{{
							Name:  "SYNAPSE_SERVER_NAME",
							Value: ServerName,
						}, {
							Name:  "SYNAPSE_REPORT_STATS",
							Value: utils.BoolToYesNo(ReportStats),
						}}
						Expect(createdDeployment.Spec.Template.Spec.InitContainers[1].Env).Should(ContainElements(envVars))
					})

					It("Should create a Synapse Service", func() {
						checkResourcePresence(createdService, synapseLookupKey, expectedOwnerReference)
					})

					It("Should create a Synapse ServiceAccount", func() {
						checkResourcePresence(createdServiceAccount, synapseLookupKey, expectedOwnerReference)
					})

					It("Should create a Synapse RoleBinding", func() {
						checkResourcePresence(createdRoleBinding, synapseLookupKey, expectedOwnerReference)
					})
				})

				When("Enabling the Heisenbridge", func() {
					const (
						heisenbridgeName      = "test-heisenbridge"
						heisenbridgeNamespace = "default"
					)
					var heisenbridge *synapsev1alpha1.Heisenbridge

					BeforeAll(func() {
						initSynapseVariables()

						inputConfigmapData = map[string]string{
							"homeserver.yaml": "server_name: " + ServerName + "\n" +
								"report_stats: " + strconv.FormatBool(ReportStats),
						}

						synapseSpec = synapsev1alpha1.SynapseSpec{
							Homeserver: synapsev1alpha1.SynapseHomeserver{
								ConfigMap: &synapsev1alpha1.SynapseHomeserverConfigMap{
									Name: InputConfigMapName,
								},
							},
							IsOpenshift: true,
						}

						createSynapseConfigMap()
						createSynapseInstance()
					})

					AfterAll(func() {
						cleanupSynapseResources()
						cleanupSynapseConfigMap()
					})

					It("Should update the Synapse Status", func() {
						expectedStatus := synapsev1alpha1.SynapseStatus{
							State:  "RUNNING",
							Reason: "",
							HomeserverConfiguration: synapsev1alpha1.SynapseStatusHomeserverConfiguration{
								ServerName:          ServerName,
								ReportStats:         ReportStats,
								RegistrationEnabled: false, // Default value
							},
						}
						// Status may need some time to be updated
						Eventually(func() synapsev1alpha1.SynapseStatus {
							_ = k8sClient.Get(ctx, synapseLookupKey, synapse)
							return synapse.Status
						}, timeout, interval).Should(Equal(expectedStatus))
					})

					It("Should register the presence of the bridge in the Synapse status", func() {
						By("Creating the Heisenbridge object")
						heisenbridge = &synapsev1alpha1.Heisenbridge{
							ObjectMeta: metav1.ObjectMeta{
								Name:      heisenbridgeName,
								Namespace: heisenbridgeNamespace,
							},
							Spec: synapsev1alpha1.HeisenbridgeSpec{
								Synapse: synapsev1alpha1.HeisenbridgeSynapseSpec{
									Name: SynapseName,
								},
							},
						}
						Expect(k8sClient.Create(ctx, heisenbridge)).Should(Succeed())

						By("Triggering the Synapse reconciliation")
						synapse.Status.NeedsReconcile = true
						Expect(k8sClient.Status().Update(ctx, synapse)).Should(Succeed())

						By("Checking the Synapse Status")
						expectedStatusBridgesHeisenbridge := synapsev1alpha1.SynapseStatusBridgesHeisenbridge{
							Enabled: true,
							Name:    heisenbridgeName,
						}
						Eventually(func(g Gomega) synapsev1alpha1.SynapseStatusBridgesHeisenbridge {
							_ = k8sClient.Get(ctx, synapseLookupKey, synapse)
							return synapse.Status.Bridges.Heisenbridge
						}, timeout, interval).Should(Equal(expectedStatusBridgesHeisenbridge))
					})

					It("Should update the Synapse homeserver.yaml", func() {
						Eventually(func(g Gomega) {
							g.Expect(k8sClient.Get(ctx,
								types.NamespacedName{Name: SynapseName, Namespace: SynapseNamespace},
								createdConfigMap,
							)).Should(Succeed())

							ConfigMapData, ok := createdConfigMap.Data["homeserver.yaml"]
							g.Expect(ok).Should(BeTrue())

							homeserver := make(map[string]any)
							g.Expect(yaml.Unmarshal([]byte(ConfigMapData), homeserver)).Should(Succeed())

							_, ok = homeserver["app_service_config_files"]
							g.Expect(ok).Should(BeTrue())

							g.Expect(homeserver["app_service_config_files"]).Should(ContainElement("/data-heisenbridge/heisenbridge.yaml"))
						}, timeout, interval).Should(Succeed())
					})

					It("Should mount the Heisenbridge ConfigMap in the Synapse Deployment", func() {
						By("Checking that a Synapse Deployment exists and is correctly configured")
						checkResourcePresence(createdDeployment, synapseLookupKey, expectedOwnerReference)

						By("Checking that the VolumeMount for the Heisenbridge Volume is present")
						heisenbridgeVolumeMount := corev1.VolumeMount{
							Name:      "data-heisenbridge",
							MountPath: "/data-heisenbridge",
						}
						Expect(createdDeployment.Spec.Template.Spec.Containers[0].VolumeMounts).
							Should(ContainElement(heisenbridgeVolumeMount))

						By("Checking that the Volume for the Heisenbridge ConfigMap is present")
						var defaultMode int32 = 420
						heisenbridgeVolume := corev1.Volume{
							Name: "data-heisenbridge",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: heisenbridgeName,
									},
									DefaultMode: &defaultMode,
								},
							},
						}
						Expect(createdDeployment.Spec.Template.Spec.Volumes).
							Should(ContainElement(heisenbridgeVolume))
					})

					It("Should unregister Heisenbridge when deleted", func() {
						By("Deleting the Heisenbridge object")
						Expect(k8sClient.Delete(ctx, heisenbridge)).Should(Succeed())

						By("Triggering the Synapse reconciliation")
						synapse.Status.NeedsReconcile = true
						Expect(k8sClient.Status().Update(ctx, synapse)).Should(Succeed())

						By("Checking the Synapse Status")
						expectedStatusBridgesHeisenbridge := synapsev1alpha1.SynapseStatusBridgesHeisenbridge{
							Enabled: false,
						}
						Eventually(func(g Gomega) synapsev1alpha1.SynapseStatusBridgesHeisenbridge {
							_ = k8sClient.Get(ctx, synapseLookupKey, synapse)
							return synapse.Status.Bridges.Heisenbridge
						}, timeout, interval).Should(Equal(expectedStatusBridgesHeisenbridge))
					})
				})

				When("A Mautrix-Signal bridge refers this Synapse instance", func() {
					const (
						mautrixSignalName      = "test-mautrixsignal"
						mautrixSignalNamespace = "default"
					)

					var mautrixsignal *synapsev1alpha1.MautrixSignal

					BeforeAll(func() {
						initSynapseVariables()

						inputConfigmapData = map[string]string{
							"homeserver.yaml": "server_name: " + ServerName + "\n" +
								"report_stats: " + strconv.FormatBool(ReportStats),
						}

						synapseSpec = synapsev1alpha1.SynapseSpec{
							Homeserver: synapsev1alpha1.SynapseHomeserver{
								ConfigMap: &synapsev1alpha1.SynapseHomeserverConfigMap{
									Name: InputConfigMapName,
								},
							},
							IsOpenshift: true,
						}

						createSynapseConfigMap()
						createSynapseInstance()
					})

					AfterAll(func() {
						cleanupSynapseResources()
						cleanupSynapseConfigMap()
					})

					It("Should update the Synapse Status", func() {
						expectedStatus := synapsev1alpha1.SynapseStatus{
							State:  "RUNNING",
							Reason: "",
							HomeserverConfiguration: synapsev1alpha1.SynapseStatusHomeserverConfiguration{
								ServerName:          ServerName,
								ReportStats:         ReportStats,
								RegistrationEnabled: false, // Default value
							},
						}
						// Status may need some time to be updated
						Eventually(func() synapsev1alpha1.SynapseStatus {
							_ = k8sClient.Get(ctx, synapseLookupKey, synapse)
							return synapse.Status
						}, timeout, interval).Should(Equal(expectedStatus))
					})

					It("Should register the presence of the bridge in the Synapse status", func() {
						By("Creating the MautrixSignal object")
						mautrixsignal = &synapsev1alpha1.MautrixSignal{
							ObjectMeta: metav1.ObjectMeta{
								Name:      mautrixSignalName,
								Namespace: mautrixSignalNamespace,
							},
							Spec: synapsev1alpha1.MautrixSignalSpec{
								Synapse: synapsev1alpha1.MautrixSignalSynapseSpec{
									Name: SynapseName,
								},
							},
						}
						Expect(k8sClient.Create(ctx, mautrixsignal)).Should(Succeed())

						By("Triggering the Synapse reconciliation")
						synapse.Status.NeedsReconcile = true
						Expect(k8sClient.Status().Update(ctx, synapse)).Should(Succeed())

						expectedStatusBridgesMautrixSignal := synapsev1alpha1.SynapseStatusBridgesMautrixSignal{
							Enabled: true,
							Name:    mautrixSignalName,
						}
						Eventually(func(g Gomega) synapsev1alpha1.SynapseStatusBridgesMautrixSignal {
							_ = k8sClient.Get(ctx, synapseLookupKey, synapse)
							return synapse.Status.Bridges.MautrixSignal
						}, timeout, interval).Should(Equal(expectedStatusBridgesMautrixSignal))
					})

					It("Should update the Synapse homeserver.yaml", func() {
						Eventually(func(g Gomega) {
							g.Expect(k8sClient.Get(
								ctx,
								types.NamespacedName{Name: SynapseName, Namespace: SynapseNamespace},
								createdConfigMap,
							)).Should(Succeed())

							ConfigMapData, ok := createdConfigMap.Data["homeserver.yaml"]
							g.Expect(ok).Should(BeTrue())

							homeserver := make(map[string]any)
							g.Expect(yaml.Unmarshal([]byte(ConfigMapData), homeserver)).Should(Succeed())

							_, ok = homeserver["app_service_config_files"]
							g.Expect(ok).Should(BeTrue())

							g.Expect(homeserver["app_service_config_files"]).Should(ContainElement("/data-mautrixsignal/registration.yaml"))
						}, timeout, interval).Should(Succeed())
					})

					It("Should mount the MautrixSignal PVC in the Synapse Deployment", func() {
						By("Checking that a Synapse Deployment exists and is correctly configured")
						checkResourcePresence(createdDeployment, synapseLookupKey, expectedOwnerReference)

						By("Checking that the VolumeMount for the Mautrix-Signal Volume is present")
						mautrixsignalVolumeMount := corev1.VolumeMount{
							Name:      "data-mautrixsignal",
							MountPath: "/data-mautrixsignal",
						}
						Expect(createdDeployment.Spec.Template.Spec.Containers[0].VolumeMounts).
							Should(ContainElement(mautrixsignalVolumeMount))

						By("Checking that the Volume for the Mautrix-Signal PVC is present")
						mautrixsignalVolume := corev1.Volume{
							Name: "data-mautrixsignal",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: mautrixSignalName,
								},
							},
						}
						Expect(createdDeployment.Spec.Template.Spec.Volumes).
							Should(ContainElement(mautrixsignalVolume))
					})

					It("Should unregister MautrixSignal when deleted", func() {
						By("Deleting the MautrixSignal object")
						Expect(k8sClient.Delete(ctx, mautrixsignal)).Should(Succeed())

						By("Triggering the Synapse reconciliation")
						synapse.Status.NeedsReconcile = true
						Expect(k8sClient.Status().Update(ctx, synapse)).Should(Succeed())

						By("Checking the Synapse Status")
						expectedStatusBridgesMautrixSignal := synapsev1alpha1.SynapseStatusBridgesMautrixSignal{
							Enabled: false,
						}
						Eventually(func(g Gomega) synapsev1alpha1.SynapseStatusBridgesMautrixSignal {
							_ = k8sClient.Get(ctx, synapseLookupKey, synapse)
							return synapse.Status.Bridges.MautrixSignal
						}, timeout, interval).Should(Equal(expectedStatusBridgesMautrixSignal))
					})
				})
			})
		})

		Context("When creating an incorrect Synapse instance", func() {
			var synapse *synapsev1alpha1.Synapse
			var createdPVC *corev1.PersistentVolumeClaim
			var createdDeployment *appsv1.Deployment
			var createdService *corev1.Service
			var createdServiceAccount *corev1.ServiceAccount
			var createdRoleBinding *rbacv1.RoleBinding
			var synapseLookupKey types.NamespacedName

			var initSynapseVariables = func() {
				// Init variables
				synapseLookupKey = types.NamespacedName{Name: SynapseName, Namespace: SynapseNamespace}
				createdPVC = &corev1.PersistentVolumeClaim{}
				createdDeployment = &appsv1.Deployment{}
				createdService = &corev1.Service{}
				createdServiceAccount = &corev1.ServiceAccount{}
				createdRoleBinding = &rbacv1.RoleBinding{}
			}

			BeforeEach(func() {
				initSynapseVariables()

				By("Creating a Synapse instance which refers an absent ConfigMap")
				synapse = &synapsev1alpha1.Synapse{
					ObjectMeta: metav1.ObjectMeta{
						Name:      SynapseName,
						Namespace: SynapseNamespace,
					},
					Spec: synapsev1alpha1.SynapseSpec{
						Homeserver: synapsev1alpha1.SynapseHomeserver{
							ConfigMap: &synapsev1alpha1.SynapseHomeserverConfigMap{
								Name: InputConfigMapName,
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, synapse)).Should(Succeed())
			})

			AfterEach(func() {
				By("Cleaning up Synapse CR")
				Expect(k8sClient.Delete(ctx, synapse)).Should(Succeed())
			})

			It("Should get in a failed state and not create child objects", func() {
				reason := "ConfigMap " + InputConfigMapName + " does not exist in namespace " + SynapseNamespace
				checkStatus("FAILED", reason, synapseLookupKey, synapse)
				checkSubresourceAbsence(
					synapseLookupKey,
					createdPVC,
					createdDeployment,
					createdService,
					createdServiceAccount,
					createdRoleBinding,
				)
			})
		})
	})
})

// getFirstFoundEnvTestBinaryDir locates the first binary in the specified path.
// ENVTEST-based tests depend on specific binaries, usually located in paths set by
// controller-runtime. When running tests directly (e.g., via an IDE) without using
// Makefile targets, the 'BinaryAssetsDirectory' must be explicitly configured.
//
// This function streamlines the process by finding the required binaries, similar to
// setting the 'KUBEBUILDER_ASSETS' environment variable. To ensure the binaries are
// properly set up, run 'make setup-envtest' beforehand.
func getFirstFoundEnvTestBinaryDir() string {
	basePath := filepath.Join("..", "..", "..", "..", "bin", "k8s")
	entries, err := os.ReadDir(basePath)
	if err != nil {
		logf.Log.Error(err, "Failed to read directory", "path", basePath)
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() {
			return filepath.Join(basePath, entry.Name())
		}
	}
	return ""
}
