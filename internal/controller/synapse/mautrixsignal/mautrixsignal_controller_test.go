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
	rbacv1 "k8s.io/api/rbac/v1"
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

var _ = Describe("Integration tests for the MautrixSignal controller", Ordered, Label("integration"), func() {
	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		MautrixSignalName      = "test-mautrixsignal"
		MautrixSignalNamespace = "default"
		InputConfigMapName     = "test-configmap"

		// Name and namespace of the MautrixSignal instance refered by the MautrixSignal Bridge
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

		err = (&MautrixSignalReconciler{
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
					filepath.Join("..", "..", "..", "..", "bundle", "manifests", "synapse.opdev.io_mautrixsignals.yaml"),
					filepath.Join("..", "..", "..", "..", "bundle", "manifests", "synapse.opdev.io_synapses.yaml"),
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

		Context("Validating MautrixSignal CRD Schema", func() {
			var obj map[string]interface{}

			BeforeEach(func() {
				obj = map[string]interface{}{
					"apiVersion": "synapse.opdev.io/v1alpha1",
					"kind":       "MautrixSignal",
					"metadata": map[string]interface{}{
						"name":      MautrixSignalName,
						"namespace": MautrixSignalNamespace,
					},
				}
			})

			DescribeTable("Creating a misconfigured MautrixSignal instance",
				func(mautrixsignal map[string]interface{}) {
					// Augment base mautrixsignal obj with additional fields
					for key, value := range mautrixsignal {
						obj[key] = value
					}
					// Create Unstructured object from mautrixsignal obj
					u := unstructured.Unstructured{Object: obj}
					Expect(k8sClient.Create(ctx, &u)).ShouldNot(Succeed())
				},
				Entry("when MautrixSignal spec is missing", map[string]interface{}{}),
				Entry("when MautrixSignal spec is empty", map[string]interface{}{
					"spec": map[string]interface{}{},
				}),
				Entry("when MautrixSignal spec is missing Synapse reference", map[string]interface{}{
					"spec": map[string]interface{}{
						"configMap": map[string]interface{}{
							"name": "dummy",
						},
					},
				}),
				Entry("when MautrixSignal spec Synapse doesn't has a name", map[string]interface{}{
					"spec": map[string]interface{}{
						"synapse": map[string]interface{}{
							"namespase": "dummy",
						},
					},
				}),
				Entry("when MautrixSignal spec ConfigMap doesn't specify a Name", map[string]interface{}{
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
				PEntry("when MautrixSignal spec possesses an invalid field", map[string]interface{}{
					"spec": map[string]interface{}{
						"synapse": map[string]interface{}{
							"name": "dummy",
						},
						"invalidSpecFiels": "random",
					},
				}),
			)

			DescribeTable("Creating a correct MautrixSignal instance",
				func(mautrixsignal map[string]interface{}) {
					// Augment base mautrixsignal obj with additional fields
					for key, value := range mautrixsignal {
						obj[key] = value
					}
					// Create Unstructured object from mautrixsignal obj
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

		Context("When creating a valid MautrixSignal instance", func() {
			var mautrixsignal *synapsev1alpha1.MautrixSignal
			var createdConfigMap *corev1.ConfigMap
			var createdPVC *corev1.PersistentVolumeClaim
			var createdDeployment *appsv1.Deployment
			var createdService *corev1.Service
			var createdServiceAccount *corev1.ServiceAccount
			var createdRoleBinding *rbacv1.RoleBinding

			var mautrixsignalLookupKey types.NamespacedName
			var expectedOwnerReference metav1.OwnerReference
			var mautrixsignalSpec synapsev1alpha1.MautrixSignalSpec

			var synapse *synapsev1alpha1.Synapse
			var synapseLookupKey types.NamespacedName

			var initMautrixSignalVariables = func() {
				// Init variables
				mautrixsignalLookupKey = types.NamespacedName{
					Name:      MautrixSignalName,
					Namespace: MautrixSignalNamespace,
				}
				createdConfigMap = &corev1.ConfigMap{}
				createdPVC = &corev1.PersistentVolumeClaim{}
				createdDeployment = &appsv1.Deployment{}
				createdService = &corev1.Service{}
				createdServiceAccount = &corev1.ServiceAccount{}
				createdRoleBinding = &rbacv1.RoleBinding{}

				// The OwnerReference UID must be set after the MautrixSignal instance
				// has been created.
				expectedOwnerReference = metav1.OwnerReference{
					Kind:               "MautrixSignal",
					APIVersion:         "synapse.opdev.io/v1alpha1",
					Name:               MautrixSignalName,
					Controller:         utils.BoolAddr(true),
					BlockOwnerDeletion: utils.BoolAddr(true),
				}

				synapseLookupKey = types.NamespacedName{
					Name:      SynapseName,
					Namespace: SynapseNamespace,
				}
			}

			var createSynapseInstanceForMautrixSignal = func() {
				By("Creating a Synapse instance for MautrixSignal")
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
						IsOpenshift: true,
					},
				}
				Expect(k8sClient.Create(ctx, synapse)).Should(Succeed())
				k8sClient.Get(ctx, synapseLookupKey, synapse)
				// Manually populating the status as the Synapse controller is not running
				synapse.Status = synapsev1alpha1.SynapseStatus{
					HomeserverConfiguration: synapsev1alpha1.SynapseStatusHomeserverConfiguration{
						ServerName:  SynapseServerName,
						ReportStats: false,
					},
				}
				Expect(k8sClient.Status().Update(ctx, synapse)).Should(Succeed())

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

			var createMautrixSignalInstance = func() {
				By("Creating the MautrixSignal instance")
				mautrixsignal = &synapsev1alpha1.MautrixSignal{
					ObjectMeta: metav1.ObjectMeta{
						Name:      MautrixSignalName,
						Namespace: MautrixSignalNamespace,
					},
					Spec: mautrixsignalSpec,
				}
				Expect(k8sClient.Create(ctx, mautrixsignal)).Should(Succeed())

				By("Verifying that the MautrixSignal object was created")
				Eventually(func() bool {
					err := k8sClient.Get(ctx, mautrixsignalLookupKey, mautrixsignal)
					return err == nil
				}, timeout, interval).Should(BeTrue())

				expectedOwnerReference.UID = mautrixsignal.GetUID()
			}

			var cleanupMautrixSignalChildResources = func() {
				// Child resources must be manually deleted as the controllers responsible of
				// their lifecycle are not running.
				By("Cleaning up MautrixSignal ConfigMap")
				deleteResource(createdConfigMap, mautrixsignalLookupKey, false)

				By("Cleaning up MautrixSignal PVC")
				deleteResource(createdPVC, mautrixsignalLookupKey, true)

				By("Cleaning up MautrixSignal Deployment")
				deleteResource(createdDeployment, mautrixsignalLookupKey, false)

				By("Cleaning up MautrixSignal Service")
				deleteResource(createdService, mautrixsignalLookupKey, false)

				By("Cleaning up MautrixSignal RoleBinding")
				deleteResource(createdRoleBinding, mautrixsignalLookupKey, false)

				By("Cleaning up MautrixSignal ServiceAccount")
				deleteResource(createdServiceAccount, mautrixsignalLookupKey, false)
			}

			var cleanupMautrixSignalResources = func() {
				By("Cleaning up MautrixSignal CR")
				Expect(k8sClient.Delete(ctx, mautrixsignal)).Should(Succeed())

				cleanupMautrixSignalChildResources()
			}

			When("No MautrixSignal ConfigMap is provided", func() {
				BeforeAll(func() {
					initMautrixSignalVariables()

					mautrixsignalSpec = synapsev1alpha1.MautrixSignalSpec{
						Synapse: synapsev1alpha1.MautrixSignalSynapseSpec{
							Name:      SynapseName,
							Namespace: SynapseNamespace,
						},
					}

					createSynapseInstanceForMautrixSignal()
					createMautrixSignalInstance()
				})

				AfterAll(func() {
					cleanupMautrixSignalResources()
					cleanupSynapseCR()
				})

				It("Should should update the MautrixSignal Status", func() {
					expectedStatus := synapsev1alpha1.MautrixSignalStatus{
						State:  "",
						Reason: "",
						Synapse: synapsev1alpha1.MautrixSignalStatusSynapse{
							ServerName: SynapseServerName,
						},
						IsOpenshift: true,
					}

					// Status may need some time to be updated
					Eventually(func() synapsev1alpha1.MautrixSignalStatus {
						_ = k8sClient.Get(ctx, mautrixsignalLookupKey, mautrixsignal)

						return mautrixsignal.Status
					}, timeout, interval).Should(Equal(expectedStatus))
				})

				It("Should create a MautrixSignal ConfigMap", func() {
					checkResourcePresence(createdConfigMap, mautrixsignalLookupKey, expectedOwnerReference)
				})

				It("Should create a MautrixSignal PVC", func() {
					checkResourcePresence(createdPVC, mautrixsignalLookupKey, expectedOwnerReference)
				})

				It("Should create a MautrixSignal Deployment", func() {
					checkResourcePresence(createdDeployment, mautrixsignalLookupKey, expectedOwnerReference)
				})

				It("Should create a MautrixSignal Service", func() {
					checkResourcePresence(createdService, mautrixsignalLookupKey, expectedOwnerReference)
				})

				It("Should create a MautrixSignal ServiceAccount", func() {
					checkResourcePresence(createdServiceAccount, mautrixsignalLookupKey, expectedOwnerReference)
				})

				It("Should create a MautrixSignal RoleBinding", func() {
					checkResourcePresence(createdRoleBinding, mautrixsignalLookupKey, expectedOwnerReference)
				})
			})

			When("Specifying the MautrixSignal configuration via a ConfigMap", func() {
				var inputConfigMap *corev1.ConfigMap
				var configYaml string = ""

				const mautrixsignalFQDN = MautrixSignalName + "." + MautrixSignalNamespace + ".svc.cluster.local"
				const synapseFQDN = SynapseName + "." + SynapseNamespace + ".svc.cluster.local"
				const mautrixsignalPort = 29328

				var createMautrixSignalConfigMap = func() {
					By("Creating a ConfigMap containing a basic config.yaml")
					// Incomplete config.yaml, containing only the required data for our
					// tests. We test that those values are correctly updated.
					configYaml = `
homeserver:
    address: http://localhost:8008
    domain: mydomain.com
appservice:
    address: http://localhost:29328
signal:
    socket_path: /var/run/signald/signald.sock
bridge:
    permissions:
        "*": "relay"
        "mydomain.com": "user"
        "@admin:mydomain.com": "admin"
logging:
    handlers:
        file:
            filename: ./mautrix-signal.log`

					inputConfigmapData := map[string]string{
						"config.yaml": configYaml,
					}

					// Populate the ConfigMap with the minimum data needed
					inputConfigMap = &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      InputConfigMapName,
							Namespace: MautrixSignalNamespace,
						},
						Data: inputConfigmapData,
					}
					Expect(k8sClient.Create(ctx, inputConfigMap)).Should(Succeed())
				}

				var cleanupMautrixSignalConfigMap = func() {
					By("Cleaning up ConfigMap")
					Expect(k8sClient.Delete(ctx, inputConfigMap)).Should(Succeed())
				}

				BeforeAll(func() {
					initMautrixSignalVariables()

					mautrixsignalSpec = synapsev1alpha1.MautrixSignalSpec{
						ConfigMap: synapsev1alpha1.MautrixSignalConfigMap{
							Name:      InputConfigMapName,
							Namespace: MautrixSignalNamespace,
						},
						Synapse: synapsev1alpha1.MautrixSignalSynapseSpec{
							Name:      SynapseName,
							Namespace: SynapseNamespace,
						},
					}

					createSynapseInstanceForMautrixSignal()
					createMautrixSignalConfigMap()
					createMautrixSignalInstance()
				})

				AfterAll(func() {
					cleanupMautrixSignalResources()
					cleanupMautrixSignalConfigMap()
					cleanupSynapseCR()
				})

				It("Should should update the MautrixSignal Status", func() {
					expectedStatus := synapsev1alpha1.MautrixSignalStatus{
						State:  "",
						Reason: "",
						Synapse: synapsev1alpha1.MautrixSignalStatusSynapse{
							ServerName: SynapseServerName,
						},
						IsOpenshift: true,
					}
					// Status may need some time to be updated
					Eventually(func() synapsev1alpha1.MautrixSignalStatus {
						_ = k8sClient.Get(ctx, mautrixsignalLookupKey, mautrixsignal)
						return mautrixsignal.Status
					}, timeout, interval).Should(Equal(expectedStatus))
				})

				It("Should create a MautrixSignal ConfigMap", func() {
					checkResourcePresence(createdConfigMap, mautrixsignalLookupKey, expectedOwnerReference)
				})

				It("Should create a MautrixSignal PVC", func() {
					checkResourcePresence(createdPVC, mautrixsignalLookupKey, expectedOwnerReference)
				})

				It("Should create a MautrixSignal Deployment", func() {
					checkResourcePresence(createdDeployment, mautrixsignalLookupKey, expectedOwnerReference)
				})

				It("Should create a MautrixSignal Service", func() {
					checkResourcePresence(createdService, mautrixsignalLookupKey, expectedOwnerReference)
				})

				It("Should create a MautrixSignal ServiceAccount", func() {
					checkResourcePresence(createdServiceAccount, mautrixsignalLookupKey, expectedOwnerReference)
				})

				It("Should create a MautrixSignal RoleBinding", func() {
					checkResourcePresence(createdRoleBinding, mautrixsignalLookupKey, expectedOwnerReference)
				})

				It("Should overwrite necessary values in the created mautrix-signal ConfigMap", func() {
					Eventually(func(g Gomega) {
						By("Verifying that the mautrixsignal ConfigMap exists")
						g.Expect(k8sClient.Get(ctx, mautrixsignalLookupKey, createdConfigMap)).Should(Succeed())

						ConfigMapdata, ok := createdConfigMap.Data["config.yaml"]
						g.Expect(ok).Should(BeTrue())

						config := make(map[string]interface{})
						g.Expect(yaml.Unmarshal([]byte(ConfigMapdata), config)).Should(Succeed())

						By("Verifying that the homeserver configuration has been updated")
						configHomeserver, ok := config["homeserver"].(map[string]interface{})
						g.Expect(ok).Should(BeTrue())
						g.Expect(configHomeserver["address"]).To(Equal("http://" + synapseFQDN + ":8008"))
						g.Expect(configHomeserver["domain"]).To(Equal(SynapseServerName))

						By("Verifying that the appservice configuration has been updated")
						configAppservice, ok := config["appservice"].(map[string]interface{})
						g.Expect(ok).Should(BeTrue())
						g.Expect(configAppservice["address"]).To(Equal("http://" + mautrixsignalFQDN + ":" + strconv.Itoa(mautrixsignalPort)))

						By("Verifying that the signal configuration has been updated")
						configSignal, ok := config["signal"].(map[string]interface{})
						g.Expect(ok).Should(BeTrue())
						g.Expect(configSignal["socket_path"]).To(Equal("/signald/signald.sock"))

						By("Verifying that the permissions have been updated")
						configBridge, ok := config["bridge"].(map[string]interface{})
						g.Expect(ok).Should(BeTrue())
						configBridgePermissions, ok := configBridge["permissions"].(map[string]interface{})
						g.Expect(ok).Should(BeTrue())
						g.Expect(configBridgePermissions).Should(HaveKeyWithValue("*", "relay"))
						g.Expect(configBridgePermissions).Should(HaveKeyWithValue(SynapseServerName, "user"))
						g.Expect(configBridgePermissions).Should(HaveKeyWithValue("@admin:"+SynapseServerName, "admin"))

						By("Verifying that the log configuration file path have been updated")
						configLogging, ok := config["logging"].(map[string]interface{})
						g.Expect(ok).Should(BeTrue())
						configLoggingHandlers, ok := configLogging["handlers"].(map[string]interface{})
						g.Expect(ok).Should(BeTrue())
						configLoggingHandlersFile, ok := configLoggingHandlers["file"].(map[string]interface{})
						g.Expect(ok).Should(BeTrue())
						g.Expect(configLoggingHandlersFile["filename"]).To(Equal("/data/mautrix-signal.log"))
					}, timeout, interval).Should(Succeed())
				})

				It("Should not modify the input ConfigMap", func() {
					Eventually(func(g Gomega) {
						By("Fetching the inputConfigMap data")
						inputConfigMapLookupKey := types.NamespacedName{
							Name:      InputConfigMapName,
							Namespace: MautrixSignalNamespace,
						}
						g.Expect(k8sClient.Get(ctx, inputConfigMapLookupKey, inputConfigMap)).Should(Succeed())

						ConfigMapdata, ok := inputConfigMap.Data["config.yaml"]
						g.Expect(ok).Should(BeTrue())

						By("Verifying the config.yaml hasn't changed")
						g.Expect(ConfigMapdata).To(Equal(configYaml))
					}, timeout, interval).Should(Succeed())
				})
			})

			When("MautrixSignal references a non existing Synapse", func() {
				BeforeAll(func() {
					initMautrixSignalVariables()

					mautrixsignalSpec = synapsev1alpha1.MautrixSignalSpec{
						Synapse: synapsev1alpha1.MautrixSignalSynapseSpec{
							Name:      "not-exist",
							Namespace: SynapseNamespace,
						},
					}

					createMautrixSignalInstance()
				})

				AfterAll(func() {
					By("Cleaning up mautrixsignal CR")
					Expect(k8sClient.Delete(ctx, mautrixsignal)).Should(Succeed())
				})

				It("Should not create any MautrixSignal resources", func() {
					checkStatus("", "", mautrixsignalLookupKey, mautrixsignal)
					checkSubresourceAbsence(
						mautrixsignalLookupKey,
						createdConfigMap,
						createdPVC,
						createdDeployment,
						createdService,
						createdServiceAccount,
						createdRoleBinding,
					)
				})
			})

			When("Creating and deleting MautrixSignal resources", func() {
				const bridgeFinalizer = "synapse.opdev.io/finalizer"

				BeforeAll(func() {
					initMautrixSignalVariables()

					mautrixsignalSpec = synapsev1alpha1.MautrixSignalSpec{
						Synapse: synapsev1alpha1.MautrixSignalSynapseSpec{
							Name:      SynapseName,
							Namespace: SynapseNamespace,
						},
					}

					createSynapseInstanceForMautrixSignal()
				})

				AfterAll(func() {
					cleanupMautrixSignalChildResources()
					cleanupSynapseCR()
				})

				When("Creating MautrixSignal", func() {
					BeforeAll(func() {
						createMautrixSignalInstance()
					})

					It("Should add a Finalizer", func() {
						Eventually(func() []string {
							_ = k8sClient.Get(ctx, mautrixsignalLookupKey, mautrixsignal)
							return mautrixsignal.Finalizers
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

				When("Deleting MautrixSignal", func() {
					BeforeAll(func() {
						By("Sending delete request")
						Expect(k8sClient.Delete(ctx, mautrixsignal)).Should(Succeed())
					})

					// MautrixSignal controller is too fast and complete cleanup
					// tasks (trigger Synapse reconciliation) and effectively
					// deletes the MautrixSignal instance before we have time to
					// check that the finalizer was removed.
					// Instead, we check that the MautrixSignal instance is
					// indeed absent.
					PIt("Should remove the Finalizer", func() {
						Eventually(func() []string {
							_ = k8sClient.Get(ctx, mautrixsignalLookupKey, mautrixsignal)
							return mautrixsignal.Finalizers
						}, timeout, interval).ShouldNot(ContainElement(bridgeFinalizer))
					})

					It("Should effectively remove the MautrixSignal instance", func() {
						Eventually(func() bool {
							err := k8sClient.Get(ctx, mautrixsignalLookupKey, mautrixsignal)
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
