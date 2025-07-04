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

package heisenbridge

import (
	"context"
	"os"
	"path/filepath"
	"strconv"

	// "strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	synapsev1alpha1 "github.com/opdev/synapse-operator/api/synapse/v1alpha1"
	"github.com/opdev/synapse-operator/helpers/utils"
)

var _ = Describe("Integration tests for the Heisenbridge controller", Ordered, Label("integration"), func() {
	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		HeisenbridgeName      = "test-heisenbridge"
		HeisenbridgeNamespace = "default"
		InputConfigMapName    = "test-configmap"

		// Name and namespace of the Heisenbridge instance refered by the Heisenbridge Bridge
		SynapseName       = "test-synapse"
		SynapseNamespace  = "default"
		SynapseServerName = "my.matrix.host"

		timeout  = time.Second * 2
		duration = time.Second * 2
		interval = time.Millisecond * 250
	)

	var k8sClient client.Client
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

		//+kubebuilder:scaffold:scheme

		k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
		Expect(err).NotTo(HaveOccurred())
		Expect(k8sClient).NotTo(BeNil())

		k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
			Scheme: scheme.Scheme,
			Metrics: server.Options{
				BindAddress: "0",
			},
		})
		Expect(err).ToNot(HaveOccurred())

		err = (&HeisenbridgeReconciler{
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
					filepath.Join("..", "..", "..", "bundle", "manifests", "synapse.opdev.io_heisenbridges.yaml"),
					filepath.Join("..", "..", "..", "bundle", "manifests", "synapse.opdev.io_synapses.yaml"),
				},
				ErrorIfCRDPathMissing: true,
			}

			startenvTest()
		})

		var _ = AfterAll(func() {
			cancel()
			By("tearing down the test environment")
			Expect(testEnv.Stop()).NotTo(HaveOccurred())
		})

		Context("Validating Heisenbridge CRD Schema", func() {
			var obj map[string]interface{}

			BeforeEach(func() {
				obj = map[string]interface{}{
					"apiVersion": "synapse.opdev.io/v1alpha1",
					"kind":       "Heisenbridge",
					"metadata": map[string]interface{}{
						"name":      HeisenbridgeName,
						"namespace": HeisenbridgeNamespace,
					},
				}
			})

			DescribeTable("Creating a misconfigured Heisenbridge instance",
				func(heisenbridge map[string]interface{}) {
					// Augment base heisenbridge obj with additional fields
					for key, value := range heisenbridge {
						obj[key] = value
					}
					// Create Unstructured object from heisenbridge obj
					u := unstructured.Unstructured{Object: obj}
					Expect(k8sClient.Create(ctx, &u)).ShouldNot(Succeed())
				},
				Entry("when Heisenbridge spec is missing", map[string]interface{}{}),
				Entry("when Heisenbridge spec is empty", map[string]interface{}{
					"spec": map[string]interface{}{},
				}),
				Entry("when Heisenbridge spec is missing Synapse reference", map[string]interface{}{
					"spec": map[string]interface{}{
						"configMap": map[string]interface{}{
							"name": "dummy",
						},
					},
				}),
				Entry("when Heisenbridge spec Synapse doesn't has a name", map[string]interface{}{
					"spec": map[string]interface{}{
						"synapse": map[string]interface{}{
							"namespase": "dummy",
						},
					},
				}),
				Entry("when Heisenbridge spec ConfigMap doesn't specify a Name", map[string]interface{}{
					"spec": map[string]interface{}{
						"configMap": map[string]interface{}{
							"namespace": "dummy",
						},
						"synapse": map[string]interface{}{
							"name": "dummy",
						},
					},
				}),
				// This should not work but passes
				PEntry("when Heisenbridge spec possesses an invalid field", map[string]interface{}{
					"spec": map[string]interface{}{
						"synapse": map[string]interface{}{
							"name": "dummy",
						},
						"invalidSpecFiels": "random",
					},
				}),
			)

			DescribeTable("Creating a correct Heisenbridge instance",
				func(heisenbridge map[string]interface{}) {
					// Augment base heisenbridge obj with additional fields
					for key, value := range heisenbridge {
						obj[key] = value
					}
					// Create Unstructured object from heisenbridge obj
					u := unstructured.Unstructured{Object: obj}
					// Use DryRun option to avoid cleaning up resources
					opt := client.CreateOptions{DryRun: []string{"All"}}
					Expect(k8sClient.Create(ctx, &u, &opt)).Should(Succeed())
				},
				Entry(
					"when the Configuration file is provided via a ConfigMap",
					map[string]interface{}{
						"spec": map[string]interface{}{
							"configMap": map[string]interface{}{
								"name":      "dummy",
								"namespace": "dummy",
							},
							"synapse": map[string]interface{}{
								"name":      "dummy",
								"namespace": "dummy",
							},
						},
					},
				),
				Entry(
					"when optional Synapse Namespace and ConfigMap Namespace are missing",
					map[string]interface{}{
						"spec": map[string]interface{}{
							"configMap": map[string]interface{}{
								"name": "dummy",
							},
							"synapse": map[string]interface{}{
								"name": "dummy",
							},
						},
					},
				),
			)
		})

		Context("When creating a valid Heisenbridge instance", func() {
			var heisenbridge *synapsev1alpha1.Heisenbridge
			var createdConfigMap *corev1.ConfigMap
			var createdDeployment *appsv1.Deployment
			var createdService *corev1.Service

			var heisenbridgeLookupKey types.NamespacedName
			var expectedOwnerReference metav1.OwnerReference
			var heisenbridgeSpec synapsev1alpha1.HeisenbridgeSpec

			var synapse *synapsev1alpha1.Synapse
			var synapseLookupKey types.NamespacedName

			var initHeisenbridgeVariables = func() {
				// Init variables
				heisenbridgeLookupKey = types.NamespacedName{
					Name:      HeisenbridgeName,
					Namespace: HeisenbridgeNamespace,
				}
				createdConfigMap = &corev1.ConfigMap{}
				createdDeployment = &appsv1.Deployment{}
				createdService = &corev1.Service{}

				// The OwnerReference UID must be set after the Heisenbridge instance
				// has been created.
				expectedOwnerReference = metav1.OwnerReference{
					Kind:               "Heisenbridge",
					APIVersion:         "synapse.opdev.io/v1alpha1",
					Name:               HeisenbridgeName,
					Controller:         utils.BoolAddr(true),
					BlockOwnerDeletion: utils.BoolAddr(true),
				}

				synapseLookupKey = types.NamespacedName{
					Name:      SynapseName,
					Namespace: SynapseNamespace,
				}
			}

			var createSynapseInstanceForHeisenbridge = func() {
				By("Creating a Synapse instance for Heisenbridge")
				synapse = &synapsev1alpha1.Synapse{
					ObjectMeta: metav1.ObjectMeta{
						Name:      SynapseName,
						Namespace: SynapseNamespace,
					},
					Spec: synapsev1alpha1.SynapseSpec{
						Homeserver: synapsev1alpha1.SynapseHomeserver{
							Values: &synapsev1alpha1.SynapseHomeserverValues{
								ServerName:  SynapseServerName,
								ReportStats: false,
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, synapse)).Should(Succeed())
				k8sClient.Get(ctx, synapseLookupKey, synapse)

				By("Verifying that the Synapse object was created")
				Eventually(func() bool {
					err := k8sClient.Get(ctx, synapseLookupKey, synapse)
					return err == nil
				}, timeout, interval).Should(BeTrue())
			}

			var cleanupSynapseCR = func() {
				By("Cleaning up Synapse CR")
				Expect(k8sClient.Delete(ctx, synapse)).Should(Succeed())
			}

			var createHeisenbridgeInstance = func() {
				By("Creating the Heisenbridge instance")
				heisenbridge = &synapsev1alpha1.Heisenbridge{
					ObjectMeta: metav1.ObjectMeta{
						Name:      HeisenbridgeName,
						Namespace: HeisenbridgeNamespace,
					},
					Spec: heisenbridgeSpec,
				}
				Expect(k8sClient.Create(ctx, heisenbridge)).Should(Succeed())

				By("Verifying that the Heisenbridge object was created")
				Eventually(func() bool {
					err := k8sClient.Get(ctx, heisenbridgeLookupKey, heisenbridge)
					return err == nil
				}, timeout, interval).Should(BeTrue())

				expectedOwnerReference.UID = heisenbridge.GetUID()
			}

			var cleanupHeisenbridgeChildResources = func() {
				// Child resources must be manually deleted as the controllers responsible of
				// their lifecycle are not running.
				By("Cleaning up Heisenbridge ConfigMap")
				deleteResource(createdConfigMap, heisenbridgeLookupKey, false)

				By("Cleaning up Heisenbridge Deployment")
				deleteResource(createdDeployment, heisenbridgeLookupKey, false)

				By("Cleaning up Heisenbridge Service")
				deleteResource(createdService, heisenbridgeLookupKey, false)
			}

			var cleanupHeisenbridgeResources = func() {
				By("Cleaning up Heisenbridge CR")
				Expect(k8sClient.Delete(ctx, heisenbridge)).Should(Succeed())

				cleanupHeisenbridgeChildResources()
			}

			When("No Heisenbridge ConfigMap is provided", func() {
				BeforeAll(func() {
					initHeisenbridgeVariables()

					heisenbridgeSpec = synapsev1alpha1.HeisenbridgeSpec{
						Synapse: synapsev1alpha1.HeisenbridgeSynapseSpec{
							Name:      SynapseName,
							Namespace: SynapseNamespace,
						},
					}

					createSynapseInstanceForHeisenbridge()
					createHeisenbridgeInstance()
				})

				AfterAll(func() {
					cleanupHeisenbridgeResources()
					cleanupSynapseCR()
				})

				It("Should should update the Heisenbridge Status", func() {
					expectedStatus := synapsev1alpha1.HeisenbridgeStatus{
						State:  "",
						Reason: "",
					}

					// Status may need some time to be updated
					Eventually(func() synapsev1alpha1.HeisenbridgeStatus {
						_ = k8sClient.Get(ctx, heisenbridgeLookupKey, heisenbridge)
						return heisenbridge.Status
					}, timeout, interval).Should(Equal(expectedStatus))
				})

				It("Should create a Heisenbridge ConfigMap", func() {
					checkResourcePresence(createdConfigMap, heisenbridgeLookupKey, expectedOwnerReference)
				})

				It("Should create a Heisenbridge Deployment", func() {
					checkResourcePresence(createdDeployment, heisenbridgeLookupKey, expectedOwnerReference)
				})

				It("Should create a Heisenbridge Service", func() {
					checkResourcePresence(createdService, heisenbridgeLookupKey, expectedOwnerReference)
				})
			})

			When("Specifying the Heisenbridge configuration via a ConfigMap", func() {
				var inputConfigMap *corev1.ConfigMap
				var heisenbridgeYaml string = ""

				const heisenbridgeFQDN = HeisenbridgeName + "." + HeisenbridgeNamespace + ".svc.cluster.local"
				const synapseFQDN = SynapseName + "." + SynapseNamespace + ".svc.cluster.local"
				const heisenbridgePort = 9898

				var createHeisenbridgeConfigMap = func() {
					By("Creating a ConfigMap containing a basic heisenbridge.yaml")
					// Incomplete heisenbridge.yaml, containing only the required data for our
					// tests. We test that those values are correctly updated.
					heisenbridgeYaml = "url: http://10.217.5.134:" + strconv.Itoa(heisenbridgePort)

					inputConfigmapData := map[string]string{
						"heisenbridge.yaml": heisenbridgeYaml,
					}

					// Populate the ConfigMap with the minimum data needed
					inputConfigMap = &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      InputConfigMapName,
							Namespace: HeisenbridgeNamespace,
						},
						Data: inputConfigmapData,
					}
					Expect(k8sClient.Create(ctx, inputConfigMap)).Should(Succeed())
				}

				var cleanupHeisenbridgeConfigMap = func() {
					By("Cleaning up ConfigMap")
					Expect(k8sClient.Delete(ctx, inputConfigMap)).Should(Succeed())
				}

				BeforeAll(func() {
					initHeisenbridgeVariables()

					heisenbridgeSpec = synapsev1alpha1.HeisenbridgeSpec{
						ConfigMap: synapsev1alpha1.HeisenbridgeConfigMap{
							Name:      InputConfigMapName,
							Namespace: HeisenbridgeNamespace,
						},
						Synapse: synapsev1alpha1.HeisenbridgeSynapseSpec{
							Name:      SynapseName,
							Namespace: SynapseNamespace,
						},
					}

					createSynapseInstanceForHeisenbridge()
					createHeisenbridgeConfigMap()
					createHeisenbridgeInstance()
				})

				AfterAll(func() {
					cleanupHeisenbridgeResources()
					cleanupHeisenbridgeConfigMap()
					cleanupSynapseCR()
				})

				It("Should should update the Heisenbridge Status", func() {
					expectedStatus := synapsev1alpha1.HeisenbridgeStatus{
						State:  "",
						Reason: "",
					}
					// Status may need some time to be updated
					Eventually(func() synapsev1alpha1.HeisenbridgeStatus {
						_ = k8sClient.Get(ctx, heisenbridgeLookupKey, heisenbridge)
						return heisenbridge.Status
					}, timeout, interval).Should(Equal(expectedStatus))
				})

				It("Should create a Heisenbridge ConfigMap", func() {
					checkResourcePresence(createdConfigMap, heisenbridgeLookupKey, expectedOwnerReference)
				})

				It("Should create a Heisenbridge Deployment", func() {
					checkResourcePresence(createdDeployment, heisenbridgeLookupKey, expectedOwnerReference)
				})

				It("Should create a Heisenbridge Service", func() {
					checkResourcePresence(createdService, heisenbridgeLookupKey, expectedOwnerReference)
				})

				It("Should overwrite necessary values in the created heisenbridge ConfigMap", func() {
					Eventually(func(g Gomega) {
						By("Verifying that the heisenbridge ConfigMap exists")
						g.Expect(k8sClient.Get(ctx, heisenbridgeLookupKey, createdConfigMap)).Should(Succeed())

						configMapdata, ok := createdConfigMap.Data["heisenbridge.yaml"]
						g.Expect(ok).Should(BeTrue())

						heisenbridge := make(map[string]interface{})
						g.Expect(yaml.Unmarshal([]byte(configMapdata), heisenbridge)).Should(Succeed())

						_, ok = heisenbridge["url"]
						g.Expect(ok).Should(BeTrue())

						g.Expect(heisenbridge["url"]).To(Equal("http://" + heisenbridgeFQDN + ":" + strconv.Itoa(heisenbridgePort)))
					}, timeout, interval).Should(Succeed())
				})

				It("Should not modify the input ConfigMap", func() {
					Eventually(func(g Gomega) {
						By("Fetching the inputConfigMap data")
						inputConfigMapLookupKey := types.NamespacedName{
							Name:      InputConfigMapName,
							Namespace: HeisenbridgeNamespace,
						}
						g.Expect(k8sClient.Get(ctx, inputConfigMapLookupKey, inputConfigMap)).Should(Succeed())

						configMapdata, ok := inputConfigMap.Data["heisenbridge.yaml"]
						g.Expect(ok).Should(BeTrue())

						By("Verifying the heisenbridge.yaml hasn't changed")
						g.Expect(configMapdata).To(Equal(heisenbridgeYaml))
					}, timeout, interval).Should(Succeed())
				})
			})

			When("Heisenbridge references a non existing Synapse", func() {
				BeforeAll(func() {
					initHeisenbridgeVariables()

					heisenbridgeSpec = synapsev1alpha1.HeisenbridgeSpec{
						Synapse: synapsev1alpha1.HeisenbridgeSynapseSpec{
							Name:      "not-exist",
							Namespace: SynapseNamespace,
						},
					}

					createHeisenbridgeInstance()
				})

				AfterAll(func() {
					By("Cleaning up heisenbridge CR")
					Expect(k8sClient.Delete(ctx, heisenbridge)).Should(Succeed())
				})

				It("Should not create any Heisenbridge resources", func() {
					checkStatus("", "", heisenbridgeLookupKey, heisenbridge)
					checkSubresourceAbsence(
						heisenbridgeLookupKey,
						createdConfigMap,
						createdDeployment,
						createdService,
					)
				})
			})

			When("Creating and deleting Heisenbridge resources", func() {
				const bridgeFinalizer = "synapse.opdev.io/finalizer"

				BeforeAll(func() {
					initHeisenbridgeVariables()

					heisenbridgeSpec = synapsev1alpha1.HeisenbridgeSpec{
						Synapse: synapsev1alpha1.HeisenbridgeSynapseSpec{
							Name:      SynapseName,
							Namespace: SynapseNamespace,
						},
					}

					createSynapseInstanceForHeisenbridge()
				})

				AfterAll(func() {
					cleanupHeisenbridgeChildResources()
					cleanupSynapseCR()
				})

				When("Creating Heisenbridge", func() {
					BeforeAll(func() {
						createHeisenbridgeInstance()
					})

					It("Should add a Finalizer", func() {
						Eventually(func() []string {
							_ = k8sClient.Get(ctx, heisenbridgeLookupKey, heisenbridge)
							return heisenbridge.Finalizers
						}, timeout, interval).Should(ContainElement(bridgeFinalizer))
					})

					It("Should trigger the reconciliation of Synapse", func() {
						Eventually(func() bool {
							_ = k8sClient.Get(ctx, synapseLookupKey, synapse)
							return synapse.Status.NeedsReconcile
						}, timeout, interval).Should(BeTrue())

						// Set needsReconcile back to False, as the Synapse Controller
						// would do if it would be running
						synapse.Status.NeedsReconcile = false
						Expect(k8sClient.Status().Update(ctx, synapse)).Should(Succeed())

						time.Sleep(time.Second)
					})
				})

				When("Deleting Heisenbridge", func() {
					BeforeAll(func() {
						By("Sending delete request")
						Expect(k8sClient.Delete(ctx, heisenbridge)).Should(Succeed())
					})

					// Heisenbridge controller is too fast and complete cleanup
					// tasks (trigger Synapse reconciliation) and effectively
					// deletes the Heisenbridge instance before we have time to
					// check that the finalizer was removed.
					// Instead, we check that the Heisenbridge instance is
					// indeed absent.
					PIt("Should remove the Finalizer", func() {
						Eventually(func() []string {
							_ = k8sClient.Get(ctx, heisenbridgeLookupKey, heisenbridge)
							return heisenbridge.Finalizers
						}, timeout, interval).ShouldNot(ContainElement(bridgeFinalizer))
					})

					It("Should effectively remove the Heisenbridge instance", func() {
						Eventually(func() bool {
							err := k8sClient.Get(ctx, heisenbridgeLookupKey, heisenbridge)
							return err == nil
						}, timeout, interval).Should(BeFalse())
					})

					It("Should trigger the reconciliation of Synapse", func() {
						Eventually(func() bool {
							_ = k8sClient.Get(ctx, synapseLookupKey, synapse)
							return synapse.Status.NeedsReconcile
						}, timeout, interval).Should(BeTrue())
					})
				})
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
