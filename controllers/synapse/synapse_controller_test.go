package synapse

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	pgov1beta1 "github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
	synapsev1alpha1 "github.com/opdev/synapse-operator/apis/synapse/v1alpha1"
)

// Helper function for struct construction requiring a boolean pointer
func BoolAddr(b bool) *bool {
	boolVar := b
	return &boolVar
}

func convert(i interface{}) interface{} {
	switch x := i.(type) {
	case map[interface{}]interface{}:
		m2 := map[string]interface{}{}
		for k, v := range x {
			m2[k.(string)] = convert(v)
		}
		return m2
	case []interface{}:
		for i, v := range x {
			x[i] = convert(v)
		}
	}
	return i
}

var _ = Describe("Integration tests for the Synapse controller", Ordered, Label("integration"), func() {
	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		SynapseName      = "test-synapse"
		SynapseNamespace = "default"
		ConfigMapName    = "test-configmap"
		ServerName       = "example.com"
		ReportStats      = false

		timeout  = time.Second * 2
		duration = time.Second * 2
		interval = time.Millisecond * 250
	)

	var k8sClient client.Client
	var testEnv *envtest.Environment
	var ctx context.Context
	var cancel context.CancelFunc

	// Common function to start envTest
	var startenvTest = func() {
		cfg, err := testEnv.Start()
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg).NotTo(BeNil())

		Expect(synapsev1alpha1.AddToScheme(scheme.Scheme)).NotTo(HaveOccurred())
		Expect(pgov1beta1.AddToScheme(scheme.Scheme)).NotTo(HaveOccurred())

		//+kubebuilder:scaffold:scheme

		k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
		Expect(err).NotTo(HaveOccurred())
		Expect(k8sClient).NotTo(BeNil())

		k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
			Scheme: scheme.Scheme,
		})
		Expect(err).ToNot(HaveOccurred())

		err = (&SynapseReconciler{
			Client: k8sManager.GetClient(),
			Scheme: k8sManager.GetScheme(),
		}).SetupWithManager(k8sManager)
		Expect(err).ToNot(HaveOccurred())

		go func() {
			defer GinkgoRecover()
			Expect(k8sManager.Start(ctx)).ToNot(HaveOccurred(), "failed to run manager")
		}()
	}

	// Verify the absence of Synapse sub-resources
	// This function common to multiple tests
	var checkSubresourceAbsence = func(expectedReason string) {
		s := &synapsev1alpha1.Synapse{}
		synapseLookupKey := types.NamespacedName{Name: SynapseName, Namespace: SynapseNamespace}
		expectedState := "FAILED"

		By("Verifying that the Synapse object was created")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, synapseLookupKey, &synapsev1alpha1.Synapse{})
			return err == nil
		}, timeout, interval).Should(BeTrue())

		By("Checking the Synapse status")
		// Status may need some time to be updated
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, synapseLookupKey, s)).Should(Succeed())
			g.Expect(s.Status.State).To(Equal(expectedState))
			g.Expect(s.Status.Reason).To(Equal(expectedReason))
		}, timeout, interval).Should(Succeed())

		By("Checking that synapse sub-resources have not been created")
		Consistently(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, synapseLookupKey, &corev1.ServiceAccount{})).ShouldNot(Succeed())
			g.Expect(k8sClient.Get(ctx, synapseLookupKey, &rbacv1.RoleBinding{})).ShouldNot(Succeed())
			g.Expect(k8sClient.Get(ctx, synapseLookupKey, &corev1.PersistentVolumeClaim{})).ShouldNot(Succeed())
			g.Expect(k8sClient.Get(ctx, synapseLookupKey, &appsv1.Deployment{})).ShouldNot(Succeed())
			g.Expect(k8sClient.Get(ctx, synapseLookupKey, &corev1.Service{})).ShouldNot(Succeed())
		}, timeout, interval).Should(Succeed())
	}

	var checkResourcePresence = func(resource client.Object, lookupKey types.NamespacedName, expectedOwnerReference metav1.OwnerReference) {
		Eventually(func() bool {
			err := k8sClient.Get(ctx, lookupKey, resource)
			return err == nil
		}, timeout, interval).Should(BeTrue())

		Expect(resource.GetOwnerReferences()).To(ContainElement(expectedOwnerReference))
	}

	var deleteResource = func(resource client.Object, lookupKey types.NamespacedName, removeFinalizers bool) {
		// Using 'Eventually' to eliminate race conditions where the Synapse
		// Operator didn't have time to create a sub resource.
		Eventually(func() bool {
			err := k8sClient.Get(ctx, lookupKey, resource)
			return err == nil
		}, timeout, interval).Should(BeTrue())

		if removeFinalizers {
			// Manually remove the finalizers
			resource.SetFinalizers([]string{})
			Expect(k8sClient.Update(ctx, resource)).Should(Succeed())
		}

		// Deleting
		Expect(k8sClient.Delete(ctx, resource)).Should(Succeed())

		// Check that the resource was successfully removed
		Eventually(func() bool {
			err := k8sClient.Get(ctx, lookupKey, resource)
			return err == nil
		}, timeout, interval).Should(BeFalse())
	}

	Context("When a corectly configured Kubernetes cluster is present", func() {
		var _ = BeforeAll(func() {

			logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

			ctx, cancel = context.WithCancel(context.TODO())

			By("Getting latest version of the PostgresCluster CRD")
			postgresOperatorVersion := "5.0.4"
			postgresClusterURL := "https://raw.githubusercontent.com/redhat-openshift-ecosystem/community-operators-prod/main/operators/postgresql/" +
				postgresOperatorVersion +
				"/manifests/postgresclusters.postgres-operator.crunchydata.com.crd.yaml"

			resp, err := http.Get(postgresClusterURL)
			Expect(err).ShouldNot(HaveOccurred())

			// The CRD is downloaded as a YAML document. The CustomResourceDefinition
			// struct defined in the v1 package only possess json tags. In order to
			// successfully Unmarshal the CRD Document into a
			// CustomResourceDefinition object, it is necessary to first transform the
			// YAML document into a intermediate JSON document.
			defer resp.Body.Close()
			yamlBody, err := io.ReadAll(resp.Body)
			Expect(err).ShouldNot(HaveOccurred())

			// Unmarshal the YAML document into an intermediate map
			var mapBody interface{}
			Expect(yaml.Unmarshal(yamlBody, &mapBody)).ShouldNot(HaveOccurred())

			// The map has to be converted. See https://stackoverflow.com/a/40737676/6133648
			mapBody = convert(mapBody)

			// Marshal the map into an intermediate JSON document
			jsonBody, err := json.Marshal(mapBody)
			Expect(err).ShouldNot(HaveOccurred())

			// Unmarshall the JSON document into the final CustomResourceDefinition object.
			var PostgresClusterCRD v1.CustomResourceDefinition
			Expect(json.Unmarshal(jsonBody, &PostgresClusterCRD)).ShouldNot(HaveOccurred())

			By("bootstrapping test environment")
			testEnv = &envtest.Environment{
				CRDDirectoryPaths: []string{
					filepath.Join("..", "..", "bundle", "manifests", "synapse.opdev.io_synapses.yaml"),
				},
				CRDs:                  []*v1.CustomResourceDefinition{&PostgresClusterCRD},
				ErrorIfCRDPathMissing: true,
			}

			startenvTest()
		})

		var _ = AfterAll(func() {
			cancel()
			By("tearing down the test environment")
			Expect(testEnv.Stop()).NotTo(HaveOccurred())
		})

		Context("Validating Synapse CRD Schema", func() {
			var obj map[string]interface{}

			BeforeEach(func() {
				obj = map[string]interface{}{
					"apiVersion": "synapse.opdev.io/v1alpha1",
					"kind":       "Synapse",
					"metadata": map[string]interface{}{
						"name":      SynapseName,
						"namespace": SynapseNamespace,
					},
				}
			})

			DescribeTable("Creating a misconfigured Synapse instance",
				func(synapse_data map[string]interface{}) {
					// Augment base synapse obj with additional fields
					for key, value := range synapse_data {
						obj[key] = value
					}
					// Create Unstructured object from synapse obj
					u := unstructured.Unstructured{Object: obj}
					Expect(k8sClient.Create(ctx, &u)).ShouldNot(Succeed())
				},
				Entry("when Synapse spec is missing", map[string]interface{}{}),
				Entry("when Synapse spec is empty", map[string]interface{}{
					"spec": map[string]interface{}{},
				}),
				Entry("when Synapse spec is missing Homeserver", map[string]interface{}{
					"spec": map[string]interface{}{"createNewPostgreSQL": true},
				}),
				Entry("when Synapse spec Homeserver is empty", map[string]interface{}{
					"spec": map[string]interface{}{
						"homeserver": map[string]interface{}{},
					},
				}),
				Entry("when Synapse spec Homeserver possess both Values and ConfigMap", map[string]interface{}{
					"spec": map[string]interface{}{
						"homeserver": map[string]interface{}{
							"configMap": map[string]interface{}{
								"name":      ConfigMapName,
								"namespace": SynapseNamespace,
							},
							"values": map[string]interface{}{
								"serverName":  ServerName,
								"reportStats": ReportStats,
							},
						}},
				}),
				Entry("when Synapse spec Homeserver ConfigMap doesn't specify a Name", map[string]interface{}{
					"spec": map[string]interface{}{
						"homeserver": map[string]interface{}{
							"configMap": map[string]interface{}{
								"namespace": SynapseNamespace,
							},
						}},
				}),
				Entry("when Synapse spec Homeserver Values is missing ServerName", map[string]interface{}{
					"spec": map[string]interface{}{
						"homeserver": map[string]interface{}{
							"values": map[string]interface{}{
								"reportStats": ReportStats,
							},
						}},
				}),
				Entry("when Synapse spec Homeserver Values is missing ReportStats", map[string]interface{}{
					"spec": map[string]interface{}{
						"homeserver": map[string]interface{}{
							"values": map[string]interface{}{
								"serverName": ServerName,
							},
						}},
				}),
				Entry("when Heisenbridge ConfigMap doesn't possess a name", map[string]interface{}{
					"spec": map[string]interface{}{
						"homeserver": map[string]interface{}{
							"configMap": map[string]interface{}{
								"name":      ConfigMapName,
								"namespace": SynapseNamespace,
							},
						},
						"bridges": map[string]interface{}{
							"heisenbridge": map[string]interface{}{
								"enabled": false,
								"configMap": map[string]interface{}{
									"namespace": "random-namespace",
								},
							},
						},
					},
				}),
				// This should not work but passes
				PEntry("when Synapse spec possesses an invalid field", map[string]interface{}{
					"spec": map[string]interface{}{
						"homeserver": map[string]interface{}{
							"configMap": map[string]interface{}{
								"name":      ConfigMapName,
								"namespace": SynapseNamespace,
							},
						},
						"invalidSpecFiels": "random",
					},
				}),
			)

			DescribeTable("Creating a correct Synapse instance",
				func(synapse_data map[string]interface{}) {
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
					map[string]interface{}{
						"spec": map[string]interface{}{
							"homeserver": map[string]interface{}{
								"configMap": map[string]interface{}{
									"name":      ConfigMapName,
									"namespace": SynapseNamespace,
								},
							},
							"createNewPostreSQL": true,
						},
					},
				),
				Entry(
					"when the Homeserver Configuration values are provided",
					map[string]interface{}{
						"spec": map[string]interface{}{
							"homeserver": map[string]interface{}{
								"values": map[string]interface{}{
									"serverName":  ServerName,
									"reportStats": ReportStats,
								},
							},
							"createNewPostreSQL": true,
						},
					},
				),
				Entry(
					"when optional CreateNewPostgreSQL and ConfigMap Namespace are missing",
					map[string]interface{}{
						"spec": map[string]interface{}{
							"homeserver": map[string]interface{}{
								"configMap": map[string]interface{}{
									"name": ConfigMapName,
								},
							},
						},
					},
				),
				Entry(
					"when optional Heisenbridge is enabled",
					map[string]interface{}{
						"spec": map[string]interface{}{
							"homeserver": map[string]interface{}{
								"configMap": map[string]interface{}{
									"name": ConfigMapName,
								},
							},
							"bridges": map[string]interface{}{
								"heisenbridge": map[string]interface{}{
									"enabled": true,
								},
							},
						},
					},
				),
				Entry(
					"when optional Heisenbridge is enabled and an input ConfigMap name is given",
					map[string]interface{}{
						"spec": map[string]interface{}{
							"homeserver": map[string]interface{}{
								"configMap": map[string]interface{}{
									"name": ConfigMapName,
								},
							},
							"bridges": map[string]interface{}{
								"heisenbridge": map[string]interface{}{
									"enabled": true,
									"configMap": map[string]interface{}{
										"name": "random-name",
									},
								},
							},
						},
					},
				),
			)
		})

		Context("When creating a valid Synapse instance", func() {
			var synapse *synapsev1alpha1.Synapse
			var createdPVC *corev1.PersistentVolumeClaim
			var createdDeployment *appsv1.Deployment
			var createdService *corev1.Service
			var createdServiceAccount *corev1.ServiceAccount
			var createdRoleBinding *rbacv1.RoleBinding
			var synapseLookupKey types.NamespacedName
			var expectedOwnerReference metav1.OwnerReference
			var synapseSpec synapsev1alpha1.SynapseSpec

			// This node could be a `BeforeEach` with a `OncePerOrdered`
			// decorator, if the last container nodes of each branch would be
			// decorated with `Ordered`, and the root container node wouldn't
			// be `Ordered`. But we need the root container node to have the
			// `Ordered` decorator so we can use `BeforeAll` and `AfterAll`
			// setup nodes within it to start envTest only once. Therefore we
			// are using functions instead, that are called in other setup nodes
			// down the tree.
			var beforeEachCreateSynapseInstance = func() {
				// Init variables
				synapseLookupKey = types.NamespacedName{Name: SynapseName, Namespace: SynapseNamespace}
				createdPVC = &corev1.PersistentVolumeClaim{}
				createdDeployment = &appsv1.Deployment{}
				createdService = &corev1.Service{}
				createdServiceAccount = &corev1.ServiceAccount{}
				createdRoleBinding = &rbacv1.RoleBinding{}
				// The OwnerReference UID must be set after the Synapse instance has been
				// created. See the JustBeforeEach node.
				expectedOwnerReference = metav1.OwnerReference{
					Kind:               "Synapse",
					APIVersion:         "synapse.opdev.io/v1alpha1",
					Name:               SynapseName,
					Controller:         BoolAddr(true),
					BlockOwnerDeletion: BoolAddr(true),
				}
			}

			// This node could be a `JustBeforeEach` with a `OncePerOrdered`
			// decorator, see comment above.
			var justBeforeEachCreateSynapseInstance = func() {
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

			// This node could be a `AfterEach` with a `OncePerOrdered`
			// decorator, see comment above.
			var afterEachCreateSynapseInstance = func() {
				By("Cleaning up Synapse CR")
				Expect(k8sClient.Delete(ctx, synapse)).Should(Succeed())

				// Child resources must be manually deleted as the controllers responsible of
				// their lifecycle are not running.
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
				var createdConfigMap *corev1.ConfigMap

				BeforeAll(func() {
					beforeEachCreateSynapseInstance()

					createdConfigMap = &corev1.ConfigMap{}
					synapseSpec = synapsev1alpha1.SynapseSpec{
						Homeserver: synapsev1alpha1.SynapseHomeserver{
							Values: &synapsev1alpha1.SynapseHomeserverValues{
								ServerName:  ServerName,
								ReportStats: ReportStats,
							},
						},
					}

					justBeforeEachCreateSynapseInstance()
				})

				AfterAll(func() {
					// Delete new ConfigMap
					deleteResource(createdConfigMap, synapseLookupKey, false)

					afterEachCreateSynapseInstance()
				})

				It("Should should update the Synapse Status", func() {
					// Get ServiceIP
					var synapseIP string
					Eventually(func() bool {
						err := k8sClient.Get(ctx, synapseLookupKey, createdService)
						if err != nil {
							return false
						}
						synapseIP = createdService.Spec.ClusterIP
						return synapseIP != ""
					}, timeout, interval).Should(BeTrue())

					expectedStatus := synapsev1alpha1.SynapseStatus{
						State:                   "RUNNING",
						Reason:                  "",
						HomeserverConfigMapName: SynapseName,
						HomeserverConfiguration: synapsev1alpha1.SynapseStatusHomeserverConfiguration{
							ServerName:  ServerName,
							ReportStats: ReportStats,
						},
						IP: synapseIP,
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

					By("Checking that initContainers contains the required environment variables")
					envVars := []corev1.EnvVar{{
						Name:  "SYNAPSE_SERVER_NAME",
						Value: ServerName,
					}, {
						Name:  "SYNAPSE_REPORT_STATS",
						Value: convert_to_yes_no(ReportStats),
					}}
					Expect(createdDeployment.Spec.Template.Spec.InitContainers[0].Env).Should(ContainElements(envVars))
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

			When("Specifying the Synapse configuration via a ConfigMap", func() {
				var configMap *corev1.ConfigMap
				var configmapData map[string]string

				BeforeEach(OncePerOrdered, func() {
					beforeEachCreateSynapseInstance()
				})

				JustBeforeEach(OncePerOrdered, func() {
					justBeforeEachCreateSynapseInstance()

					By("Creating a ConfigMap containing a basic homeserver.yaml")
					// Populate the ConfigMap with the minimum data needed
					configMap = &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      ConfigMapName,
							Namespace: SynapseNamespace,
						},
						Data: configmapData,
					}
					Expect(k8sClient.Create(ctx, configMap)).Should(Succeed())
				})

				AfterEach(OncePerOrdered, func() {
					By("Cleaning up ConfigMap")
					Expect(k8sClient.Delete(ctx, configMap)).Should(Succeed())

					afterEachCreateSynapseInstance()
				})

				When("Creating a simple Synapse instance", func() {
					BeforeAll(func() {
						configmapData = map[string]string{
							"homeserver.yaml": "server_name: " + ServerName + "\n" +
								"report_stats: " + strconv.FormatBool(ReportStats),
						}

						synapseSpec = synapsev1alpha1.SynapseSpec{
							Homeserver: synapsev1alpha1.SynapseHomeserver{
								ConfigMap: &synapsev1alpha1.SynapseHomeserverConfigMap{
									Name: ConfigMapName,
								},
							},
						}
					})

					It("Should should update the Synapse Status", func() {
						// Get ServiceIP
						var synapseIP string
						Eventually(func() bool {
							err := k8sClient.Get(ctx, synapseLookupKey, createdService)
							if err != nil {
								return false
							}
							synapseIP = createdService.Spec.ClusterIP
							return synapseIP != ""
						}, timeout, interval).Should(BeTrue())

						expectedStatus := synapsev1alpha1.SynapseStatus{
							State:                   "RUNNING",
							Reason:                  "",
							HomeserverConfigMapName: ConfigMapName,
							HomeserverConfiguration: synapsev1alpha1.SynapseStatusHomeserverConfiguration{
								ServerName:  ServerName,
								ReportStats: ReportStats,
							},
							IP: synapseIP,
						}
						// Status may need some time to be updated
						Eventually(func() synapsev1alpha1.SynapseStatus {
							_ = k8sClient.Get(ctx, synapseLookupKey, synapse)
							return synapse.Status
						}, timeout, interval).Should(Equal(expectedStatus))
					})

					It("Should create a Synapse PVC", func() {
						checkResourcePresence(createdPVC, synapseLookupKey, expectedOwnerReference)
					})

					It("Should create a Synapse Deployment", func() {
						By("Checking that a Synapse Deployment exists and is correctly configured")
						checkResourcePresence(createdDeployment, synapseLookupKey, expectedOwnerReference)

						By("Checking that initContainers contains the required environment variables")
						envVars := []corev1.EnvVar{{
							Name:  "SYNAPSE_SERVER_NAME",
							Value: ServerName,
						}, {
							Name:  "SYNAPSE_REPORT_STATS",
							Value: convert_to_yes_no(ReportStats),
						}}
						Expect(createdDeployment.Spec.Template.Spec.InitContainers[0].Env).Should(ContainElements(envVars))
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

				When("Requesting a new PostgreSQL instance to be created for Synapse", func() {
					var createdPostgresCluster *pgov1beta1.PostgresCluster
					var postgresSecret corev1.Secret

					BeforeAll(func() {
						// Init variable
						createdPostgresCluster = &pgov1beta1.PostgresCluster{}

						configmapData = map[string]string{
							"homeserver.yaml": "server_name: " + ServerName + "\n" +
								"report_stats: " + strconv.FormatBool(ReportStats),
						}

						synapseSpec = synapsev1alpha1.SynapseSpec{
							Homeserver: synapsev1alpha1.SynapseHomeserver{
								ConfigMap: &synapsev1alpha1.SynapseHomeserverConfigMap{
									Name: ConfigMapName,
								},
							},
							CreateNewPostgreSQL: true,
						}
					})

					doPostgresControllerJob := func() {
						// The postgres-operator is responsible for creating a Secret holding
						// information on how to connect to the synapse Database with the synapse
						// user. As this controller is not running during our integration tests,
						// we have to manually create this secret here.
						postgresSecret = corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Name:      SynapseName + "-pguser-synapse",
								Namespace: SynapseNamespace,
							},
							Data: map[string][]byte{
								"host":     []byte("hostname.postgresql.url"),
								"port":     []byte("5432"),
								"dbname":   []byte("synapse"),
								"user":     []byte("synapse"),
								"password": []byte("VerySecureSyn@psePassword!"),
							},
						}
						Expect(k8sClient.Create(ctx, &postgresSecret)).Should(Succeed())

						// The portgres-operator is responsible for updating the PostgresCluster
						// status, with the number of Pods being ready. This is used a part of
						// the 'isPostgresClusterReady' method in the Synapse controller.
						createdPostgresCluster.Status.InstanceSets = []pgov1beta1.PostgresInstanceSetStatus{{
							Name:            "instance1",
							Replicas:        1,
							ReadyReplicas:   1,
							UpdatedReplicas: 1,
						}}
						Expect(k8sClient.Status().Update(ctx, createdPostgresCluster)).Should(Succeed())
					}

					AfterAll(func() {
						By("Cleaning up the Synapse PostgresCluster")
						deleteResource(createdPostgresCluster, synapseLookupKey, false)
					})

					It("Should create a PostgresCluster for Synapse", func() {
						By("Checking that a Synapse PostgresCluster exists")
						checkResourcePresence(createdPostgresCluster, synapseLookupKey, expectedOwnerReference)
					})

					It("Should update the Synapse status", func() {
						By("Checking that the controller detects the Database as not ready")
						Expect(k8sClient.Get(ctx, synapseLookupKey, synapse)).Should(Succeed())
						Expect(synapse.Status.DatabaseConnectionInfo.State).Should(Equal("NOT READY"))

						// Once the PostgresCluster has been created, we simulate the
						// postgres-operator reconciliation.
						By("Simulating the postgres-operator controller job")
						doPostgresControllerJob()

						By("Checking that the Synapse Status is correctly updated")
						Eventually(func(g Gomega) {
							g.Expect(k8sClient.Get(ctx, synapseLookupKey, synapse)).Should(Succeed())

							g.Expect(synapse.Status.DatabaseConnectionInfo.ConnectionURL).Should(Equal("hostname.postgresql.url:5432"))
							g.Expect(synapse.Status.DatabaseConnectionInfo.DatabaseName).Should(Equal("synapse"))
							g.Expect(synapse.Status.DatabaseConnectionInfo.User).Should(Equal("synapse"))
							g.Expect(synapse.Status.DatabaseConnectionInfo.Password).Should(Equal(string(base64encode("VerySecureSyn@psePassword!"))))
							g.Expect(synapse.Status.DatabaseConnectionInfo.State).Should(Equal("READY"))
						}, timeout, interval).Should(Succeed())
					})

					It("Should update the ConfigMap Data", func() {
						Eventually(func(g Gomega) {
							// Fetching database section of the homeserver.yaml configuration file
							g.Expect(k8sClient.Get(ctx,
								types.NamespacedName{Name: ConfigMapName, Namespace: SynapseNamespace},
								configMap,
							)).Should(Succeed())

							cm_data, ok := configMap.Data["homeserver.yaml"]
							g.Expect(ok).Should(BeTrue())

							homeserver := make(map[string]interface{})
							g.Expect(yaml.Unmarshal([]byte(cm_data), homeserver)).Should(Succeed())

							_, ok = homeserver["database"]
							g.Expect(ok).Should(BeTrue())

							marshalled_homeserver_database, err := yaml.Marshal(homeserver["database"])
							g.Expect(err).ShouldNot(HaveOccurred())

							var hs_database HomeserverPgsqlDatabase
							g.Expect(yaml.Unmarshal(marshalled_homeserver_database, &hs_database)).Should(Succeed())

							// hs_database, ok := homeserver["database"].(HomeserverPgsqlDatabase)
							// g.Expect(ok).Should(BeTrue())

							// Testing that the database section is correctly configured for using
							// the PostgreSQL DB
							g.Expect(hs_database.Name).Should(Equal("psycopg2"))
							g.Expect(hs_database.Args.Host).Should(Equal("hostname.postgresql.url"))

							g.Expect(hs_database.Args.Port).Should(Equal(int64(5432)))
							g.Expect(hs_database.Args.Database).Should(Equal("synapse"))
							g.Expect(hs_database.Args.User).Should(Equal("synapse"))
							g.Expect(hs_database.Args.Password).Should(Equal("VerySecureSyn@psePassword!"))

							g.Expect(hs_database.Args.CpMin).Should(Equal(int64(5)))
							g.Expect(hs_database.Args.CpMax).Should(Equal(int64(10)))
						}, timeout, interval).Should(Succeed())
					})
				})

				When("Enabling the Heisenbridge", func() {
					var createdHeisenbridgeConfigMap *corev1.ConfigMap
					var createdHeisenbridgeDeployment *appsv1.Deployment
					var createdHeisenbridgeService *corev1.Service
					var heisenbridgeLookupKey types.NamespacedName

					BeforeAll(func() {
						createdHeisenbridgeConfigMap = &corev1.ConfigMap{}
						createdHeisenbridgeDeployment = &appsv1.Deployment{}
						createdHeisenbridgeService = &corev1.Service{}

						configmapData = map[string]string{
							"homeserver.yaml": "server_name: " + ServerName + "\n" +
								"report_stats: " + strconv.FormatBool(ReportStats),
						}

						synapseSpec = synapsev1alpha1.SynapseSpec{
							Homeserver: synapsev1alpha1.SynapseHomeserver{
								ConfigMap: &synapsev1alpha1.SynapseHomeserverConfigMap{
									Name: ConfigMapName,
								},
							},
							Bridges: synapsev1alpha1.SynapseBridges{
								Heisenbridge: synapsev1alpha1.SynapseHeisenbridge{
									Enabled: true,
								},
							},
						}

						heisenbridgeLookupKey = types.NamespacedName{Name: SynapseName + "-heisenbridge", Namespace: SynapseNamespace}

					})

					AfterAll(func() {
						// Cleanup Heisenbridge resources
						By("Cleaning up the Heisenbridge ConfigMap")
						deleteResource(createdHeisenbridgeConfigMap, heisenbridgeLookupKey, false)

						By("Cleaning up the Heisenbridge Deployment")
						deleteResource(createdHeisenbridgeDeployment, heisenbridgeLookupKey, false)

						By("Cleaning up the Heisenbridge Service")
						deleteResource(createdHeisenbridgeService, heisenbridgeLookupKey, false)
					})

					It("Should create a ConfigMap for Heisenbridge", func() {
						checkResourcePresence(createdHeisenbridgeConfigMap, heisenbridgeLookupKey, expectedOwnerReference)
					})

					It("Should create a Deployment for Heisenbridge", func() {
						By("Checking that a Synapse Deployment exists and is correctly configured")
						checkResourcePresence(createdHeisenbridgeDeployment, heisenbridgeLookupKey, expectedOwnerReference)
					})

					It("Should create a Service for Heisenbridge", func() {
						checkResourcePresence(createdHeisenbridgeService, heisenbridgeLookupKey, expectedOwnerReference)
					})

					It("Should add the Heisenbridge IP to the Synapse Status", func() {
						// Get Heisenbridge IP
						var heisenbridgeIP string
						Eventually(func() bool {
							err := k8sClient.Get(ctx, heisenbridgeLookupKey, createdHeisenbridgeService)
							if err != nil {
								return false
							}
							heisenbridgeIP = createdHeisenbridgeService.Spec.ClusterIP
							return heisenbridgeIP != ""
						}, timeout, interval).Should(BeTrue())

						Expect(k8sClient.Get(ctx, synapseLookupKey, synapse)).To(Succeed())
						Expect(synapse.Status.BridgesConfiguration.Heisenbridge.IP).To(Equal(heisenbridgeIP))
					})
				})

			})
		})

		Context("When creating an incorrect Synapse instance", func() {
			var synapse *synapsev1alpha1.Synapse

			BeforeEach(func() {
				By("Creating a Synapse instance which refers an absent ConfigMap")
				synapse = &synapsev1alpha1.Synapse{
					ObjectMeta: metav1.ObjectMeta{
						Name:      SynapseName,
						Namespace: SynapseNamespace,
					},
					Spec: synapsev1alpha1.SynapseSpec{
						Homeserver: synapsev1alpha1.SynapseHomeserver{
							ConfigMap: &synapsev1alpha1.SynapseHomeserverConfigMap{
								Name: ConfigMapName,
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
				reason := "ConfigMap " + ConfigMapName + " does not exist in namespace " + SynapseNamespace
				checkSubresourceAbsence(reason)
			})
		})
	})

	Context("When the Kubernetes cluster is missing the postgres-operator", func() {
		BeforeAll(func() {
			logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

			ctx, cancel = context.WithCancel(context.TODO())

			By("bootstrapping test environment")
			testEnv = &envtest.Environment{
				CRDDirectoryPaths: []string{
					filepath.Join("..", "..", "bundle", "manifests", "synapse.opdev.io_synapses.yaml"),
				},
				ErrorIfCRDPathMissing: true,
			}

			startenvTest()
		})

		AfterAll(func() {
			cancel()
			By("tearing down the test environment")
			err := testEnv.Stop()
			Expect(err).NotTo(HaveOccurred())
		})

		When("Requesting a new PostgreSQL instance to be created for Synapse", func() {
			var synapse *synapsev1alpha1.Synapse
			var configMap *corev1.ConfigMap

			BeforeAll(func() {
				configMap = &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ConfigMapName,
						Namespace: SynapseNamespace,
					},
					Data: map[string]string{
						"homeserver.yaml": "server_name: " + ServerName + "\n" +
							"report_stats: " + strconv.FormatBool(ReportStats),
					},
				}
				Expect(k8sClient.Create(ctx, configMap)).Should(Succeed())

				By("Creating the Synapse instance")
				synapse = &synapsev1alpha1.Synapse{
					ObjectMeta: metav1.ObjectMeta{
						Name:      SynapseName,
						Namespace: SynapseNamespace,
					},
					Spec: synapsev1alpha1.SynapseSpec{
						Homeserver: synapsev1alpha1.SynapseHomeserver{
							ConfigMap: &synapsev1alpha1.SynapseHomeserverConfigMap{
								Name: ConfigMapName,
							},
						},
						CreateNewPostgreSQL: true,
					},
				}
				Expect(k8sClient.Create(ctx, synapse)).Should(Succeed())
			})

			AfterAll(func() {
				By("Cleaning up ConfigMap")
				Expect(k8sClient.Delete(ctx, configMap)).Should(Succeed())

				By("Cleaning up Synapse CR")
				Expect(k8sClient.Delete(ctx, synapse)).Should(Succeed())
			})

			It("Should not create Synapse sub-resources", func() {
				reason := "Cannot create PostgreSQL instance for synapse. Postgres-operator is not installed."
				checkSubresourceAbsence(reason)
			})
		})
	})
})
