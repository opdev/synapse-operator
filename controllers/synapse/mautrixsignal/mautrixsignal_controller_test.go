package mautrixsignal

// import (
// 	"context"
// 	"path/filepath"

// 	// "strconv"
// 	"time"

// 	. "github.com/onsi/ginkgo/v2"
// 	. "github.com/onsi/gomega"

// 	// appsv1 "k8s.io/api/apps/v1"
// 	// corev1 "k8s.io/api/core/v1"
// 	// rbacv1 "k8s.io/api/rbac/v1"

// 	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
// 	"k8s.io/client-go/kubernetes/scheme"
// 	ctrl "sigs.k8s.io/controller-runtime"
// 	"sigs.k8s.io/controller-runtime/pkg/client"
// 	"sigs.k8s.io/controller-runtime/pkg/envtest"
// 	logf "sigs.k8s.io/controller-runtime/pkg/log"
// 	"sigs.k8s.io/controller-runtime/pkg/log/zap"

// 	synapsev1alpha1 "github.com/opdev/synapse-operator/apis/synapse/v1alpha1"
// )

// var _ = Describe("Integration tests for the Synapse controller", Ordered, Label("integration"), func() {
// 	// Define utility constants for object names and testing timeouts/durations and intervals.
// 	const (
// 		MautrixSignalName      = "test-mautrixsignal"
// 		MautrixSignalNamespace = "default"
// 		InputConfigMapName     = "test-configmap"

// 		timeout  = time.Second * 2
// 		duration = time.Second * 2
// 		interval = time.Millisecond * 250
// 	)

// 	var k8sClient client.Client
// 	var testEnv *envtest.Environment
// 	var ctx context.Context
// 	var cancel context.CancelFunc

// 	// var deleteResource func(client.Object, types.NamespacedName, bool)
// 	// var checkSubresourceAbsence func(string)
// 	// var checkResourcePresence func(client.Object, types.NamespacedName, metav1.OwnerReference)

// 	// Common function to start envTest
// 	var startenvTest = func() {
// 		cfg, err := testEnv.Start()
// 		Expect(err).NotTo(HaveOccurred())
// 		Expect(cfg).NotTo(BeNil())

// 		Expect(synapsev1alpha1.AddToScheme(scheme.Scheme)).NotTo(HaveOccurred())

// 		//+kubebuilder:scaffold:scheme

// 		k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
// 		Expect(err).NotTo(HaveOccurred())
// 		Expect(k8sClient).NotTo(BeNil())

// 		k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
// 			Scheme: scheme.Scheme,
// 		})
// 		Expect(err).ToNot(HaveOccurred())

// 		err = (&MautrixSignalReconciler{
// 			Client: k8sManager.GetClient(),
// 			Scheme: k8sManager.GetScheme(),
// 		}).SetupWithManager(k8sManager)
// 		Expect(err).ToNot(HaveOccurred())

// 		// deleteResource = utils.DeleteResourceFunc(k8sClient, ctx, timeout, interval)
// 		// checkSubresourceAbsence = utils.CheckSubresourceAbsenceFunc(MautrixSignalName, MautrixSignalNamespace, k8sClient, ctx, timeout, interval)
// 		// checkResourcePresence = utils.CheckResourcePresenceFunc(k8sClient, ctx, timeout, interval)

// 		go func() {
// 			defer GinkgoRecover()
// 			Expect(k8sManager.Start(ctx)).ToNot(HaveOccurred(), "failed to run manager")
// 		}()
// 	}

// 	Context("When a corectly configured Kubernetes cluster is present", func() {
// 		var _ = BeforeAll(func() {
// 			logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

// 			ctx, cancel = context.WithCancel(context.TODO())

// 			By("bootstrapping test environment")
// 			testEnv = &envtest.Environment{
// 				CRDDirectoryPaths: []string{
// 					filepath.Join("..", "..", "..", "bundle", "manifests", "synapse.opdev.io_synapses.yaml"),
// 				},
// 				ErrorIfCRDPathMissing: true,
// 			}

// 			startenvTest()
// 		})

// 		var _ = AfterAll(func() {
// 			cancel()
// 			By("tearing down the test environment")
// 			Expect(testEnv.Stop()).NotTo(HaveOccurred())
// 		})

// 		Context("Validating MautixSignal CRD Schema", func() {
// 			var obj map[string]interface{}

// 			BeforeEach(func() {
// 				obj = map[string]interface{}{
// 					"apiVersion": "synapse.opdev.io/v1alpha1",
// 					"kind":       "MautrixSignal",
// 					"metadata": map[string]interface{}{
// 						"name":      MautrixSignalName,
// 						"namespace": MautrixSignalNamespace,
// 					},
// 				}
// 			})

// 			DescribeTable("Creating a misconfigured MautrixSignal instance",
// 				func(mautrixsignal map[string]interface{}) {
// 					// Augment base mautrixsignal obj with additional fields
// 					for key, value := range mautrixsignal {
// 						obj[key] = value
// 					}
// 					// Create Unstructured object from mautrixsignal obj
// 					u := unstructured.Unstructured{Object: obj}
// 					Expect(k8sClient.Create(ctx, &u)).ShouldNot(Succeed())
// 				},
// 				Entry("when MautrixSignal spec is missing", map[string]interface{}{}),
// 				Entry("when MautrixSignal spec is empty", map[string]interface{}{
// 					"spec": map[string]interface{}{},
// 				}),
// 				Entry("when MautrixSignal spec is missing Synapse reference", map[string]interface{}{
// 					"spec": map[string]interface{}{
// 						"configMap": map[string]interface{}{
// 							"name": "dummy",
// 						},
// 					},
// 				}),
// 				Entry("when MautrixSignal spec Synapse doesn't has a name", map[string]interface{}{
// 					"spec": map[string]interface{}{
// 						"synapse": map[string]interface{}{
// 							"namespase": "dummy",
// 						},
// 					},
// 				}),
// 				Entry("when MautrixSignal spec ConfigMap doesn't specify a Name", map[string]interface{}{
// 					"spec": map[string]interface{}{
// 						"configMap": map[string]interface{}{
// 							"namespace": "dummy",
// 						},
// 						"synapse": map[string]interface{}{
// 							"name": "dummy",
// 						},
// 					},
// 				}),
// 				// This should not work but passes
// 				PEntry("when MautrixSignal spec possesses an invalid field", map[string]interface{}{
// 					"spec": map[string]interface{}{
// 						"synapse": map[string]interface{}{
// 							"name": "dummy",
// 						},
// 						"invalidSpecFiels": "random",
// 					},
// 				}),
// 			)

// 			DescribeTable("Creating a correct MautrixSignal instance",
// 				func(mautrixsignal map[string]interface{}) {
// 					// Augment base mautrixsignal obj with additional fields
// 					for key, value := range mautrixsignal {
// 						obj[key] = value
// 					}
// 					// Create Unstructured object from mautrixsignal obj
// 					u := unstructured.Unstructured{Object: obj}
// 					// Use DryRun option to avoid cleaning up resources
// 					opt := client.CreateOptions{DryRun: []string{"All"}}
// 					Expect(k8sClient.Create(ctx, &u, &opt)).Should(Succeed())
// 				},
// 				Entry(
// 					"when the Configuration file is provided via a ConfigMap",
// 					map[string]interface{}{
// 						"spec": map[string]interface{}{
// 							"configMap": map[string]interface{}{
// 								"name":      "dummy",
// 								"namespace": "dummy",
// 							},
// 							"synapse": map[string]interface{}{
// 								"name":      "dummy",
// 								"namespace": "dummy",
// 							},
// 						},
// 					},
// 				),
// 				Entry(
// 					"when optional Synapse Namespace and ConfigMap Namespace are missing",
// 					map[string]interface{}{
// 						"spec": map[string]interface{}{
// 							"configMap": map[string]interface{}{
// 								"name": "dummy",
// 							},
// 							"synapse": map[string]interface{}{
// 								"name": "dummy",
// 							},
// 						},
// 					},
// 				),
// 			)
// 		})

// 	Context("When creating a valid Synapse instance", func() {
// 		var synapse *synapsev1alpha1.Synapse
// 		var createdConfigMap *corev1.ConfigMap
// 		var createdPVC *corev1.PersistentVolumeClaim
// 		var createdDeployment *appsv1.Deployment
// 		var createdService *corev1.Service
// 		var createdServiceAccount *corev1.ServiceAccount
// 		var createdRoleBinding *rbacv1.RoleBinding
// 		var synapseLookupKey types.NamespacedName
// 		var expectedOwnerReference metav1.OwnerReference
// 		var synapseSpec synapsev1alpha1.SynapseSpec

// 		var initSynapseVariables = func() {
// 			// Init variables
// 			synapseLookupKey = types.NamespacedName{Name: SynapseName, Namespace: SynapseNamespace}
// 			createdConfigMap = &corev1.ConfigMap{}
// 			createdPVC = &corev1.PersistentVolumeClaim{}
// 			createdDeployment = &appsv1.Deployment{}
// 			createdService = &corev1.Service{}
// 			createdServiceAccount = &corev1.ServiceAccount{}
// 			createdRoleBinding = &rbacv1.RoleBinding{}
// 			// The OwnerReference UID must be set after the Synapse instance has been
// 			// created. See the JustBeforeEach node.
// 			expectedOwnerReference = metav1.OwnerReference{
// 				Kind:               "Synapse",
// 				APIVersion:         "synapse.opdev.io/v1alpha1",
// 				Name:               SynapseName,
// 				Controller:         utils.BoolAddr(true),
// 				BlockOwnerDeletion: utils.BoolAddr(true),
// 			}
// 		}

// 		var createSynapseInstance = func() {
// 			By("Creating the Synapse instance")
// 			synapse = &synapsev1alpha1.Synapse{
// 				ObjectMeta: metav1.ObjectMeta{
// 					Name:      SynapseName,
// 					Namespace: SynapseNamespace,
// 				},
// 				Spec: synapseSpec,
// 			}
// 			Expect(k8sClient.Create(ctx, synapse)).Should(Succeed())

// 			By("Verifying that the Synapse object was created")
// 			Eventually(func() bool {
// 				err := k8sClient.Get(ctx, synapseLookupKey, synapse)
// 				return err == nil
// 			}, timeout, interval).Should(BeTrue())

// 			expectedOwnerReference.UID = synapse.GetUID()
// 		}

// 		var cleanupSynapseResources = func() {
// 			By("Cleaning up Synapse CR")
// 			Expect(k8sClient.Delete(ctx, synapse)).Should(Succeed())

// 			// Child resources must be manually deleted as the controllers responsible of
// 			// their lifecycle are not running.
// 			By("Cleaning up Synapse ConfigMap")
// 			deleteResource(createdConfigMap, synapseLookupKey, false)

// 			By("Cleaning up Synapse PVC")
// 			deleteResource(createdPVC, synapseLookupKey, true)

// 			By("Cleaning up Synapse Deployment")
// 			deleteResource(createdDeployment, synapseLookupKey, false)

// 			By("Cleaning up Synapse Service")
// 			deleteResource(createdService, synapseLookupKey, false)

// 			By("Cleaning up Synapse RoleBinding")
// 			deleteResource(createdRoleBinding, synapseLookupKey, false)

// 			By("Cleaning up Synapse ServiceAccount")
// 			deleteResource(createdServiceAccount, synapseLookupKey, false)
// 		}

// 		When("Specifying the Synapse configuration via Values", func() {
// 			BeforeAll(func() {
// 				initSynapseVariables()

// 				synapseSpec = synapsev1alpha1.SynapseSpec{
// 					Homeserver: synapsev1alpha1.SynapseHomeserver{
// 						Values: &synapsev1alpha1.SynapseHomeserverValues{
// 							ServerName:  ServerName,
// 							ReportStats: ReportStats,
// 						},
// 					},
// 				}

// 				createSynapseInstance()
// 			})

// 			AfterAll(func() {
// 				cleanupSynapseResources()
// 			})

// 			It("Should should update the Synapse Status", func() {
// 				expectedStatus := synapsev1alpha1.SynapseStatus{
// 					State:  "RUNNING",
// 					Reason: "",
// 					HomeserverConfiguration: synapsev1alpha1.SynapseStatusHomeserverConfiguration{
// 						ServerName:  ServerName,
// 						ReportStats: ReportStats,
// 					},
// 				}
// 				// Status may need some time to be updated
// 				Eventually(func() synapsev1alpha1.SynapseStatus {
// 					_ = k8sClient.Get(ctx, synapseLookupKey, synapse)
// 					return synapse.Status
// 				}, timeout, interval).Should(Equal(expectedStatus))
// 			})

// 			It("Should create a Synapse ConfigMap", func() {
// 				checkResourcePresence(createdConfigMap, synapseLookupKey, expectedOwnerReference)
// 			})

// 			It("Should create a Synapse PVC", func() {
// 				checkResourcePresence(createdPVC, synapseLookupKey, expectedOwnerReference)
// 			})

// 			It("Should create a Synapse Deployment", func() {
// 				By("Checking that a Synapse Deployment exists and is correctly configured")
// 				checkResourcePresence(createdDeployment, synapseLookupKey, expectedOwnerReference)

// 				By("Checking that initContainers contains the required environment variables")
// 				envVars := []corev1.EnvVar{{
// 					Name:  "SYNAPSE_SERVER_NAME",
// 					Value: ServerName,
// 				}, {
// 					Name:  "SYNAPSE_REPORT_STATS",
// 					Value: utils.BoolToYesNo(ReportStats),
// 				}}
// 				Expect(createdDeployment.Spec.Template.Spec.InitContainers[0].Env).Should(ContainElements(envVars))
// 			})

// 			It("Should create a Synapse Service", func() {
// 				checkResourcePresence(createdService, synapseLookupKey, expectedOwnerReference)
// 			})

// 			It("Should create a Synapse ServiceAccount", func() {
// 				checkResourcePresence(createdServiceAccount, synapseLookupKey, expectedOwnerReference)
// 			})

// 			It("Should create a Synapse RoleBinding", func() {
// 				checkResourcePresence(createdRoleBinding, synapseLookupKey, expectedOwnerReference)
// 			})
// 		})

// 		When("Specifying the Synapse configuration via a ConfigMap", func() {
// 			var inputConfigMap *corev1.ConfigMap
// 			var inputConfigmapData map[string]string

// 			var createSynapseConfigMap = func() {
// 				By("Creating a ConfigMap containing a basic homeserver.yaml")
// 				// Populate the ConfigMap with the minimum data needed
// 				inputConfigMap = &corev1.ConfigMap{
// 					ObjectMeta: metav1.ObjectMeta{
// 						Name:      InputConfigMapName,
// 						Namespace: SynapseNamespace,
// 					},
// 					Data: inputConfigmapData,
// 				}
// 				Expect(k8sClient.Create(ctx, inputConfigMap)).Should(Succeed())
// 			}

// 			var cleanupSynapseConfigMap = func() {
// 				By("Cleaning up ConfigMap")
// 				Expect(k8sClient.Delete(ctx, inputConfigMap)).Should(Succeed())
// 			}

// 			When("Creating a simple Synapse instance", func() {
// 				BeforeAll(func() {
// 					initSynapseVariables()

// 					inputConfigmapData = map[string]string{
// 						"homeserver.yaml": "server_name: " + ServerName + "\n" +
// 							"report_stats: " + strconv.FormatBool(ReportStats),
// 					}

// 					synapseSpec = synapsev1alpha1.SynapseSpec{
// 						Homeserver: synapsev1alpha1.SynapseHomeserver{
// 							ConfigMap: &synapsev1alpha1.SynapseHomeserverConfigMap{
// 								Name: InputConfigMapName,
// 							},
// 						},
// 					}

// 					createSynapseConfigMap()
// 					createSynapseInstance()
// 				})

// 				AfterAll(func() {
// 					cleanupSynapseResources()
// 					cleanupSynapseConfigMap()
// 				})

// 				It("Should should update the Synapse Status", func() {
// 					expectedStatus := synapsev1alpha1.SynapseStatus{
// 						State:  "RUNNING",
// 						Reason: "",
// 						HomeserverConfiguration: synapsev1alpha1.SynapseStatusHomeserverConfiguration{
// 							ServerName:  ServerName,
// 							ReportStats: ReportStats,
// 						},
// 					}
// 					// Status may need some time to be updated
// 					Eventually(func() synapsev1alpha1.SynapseStatus {
// 						_ = k8sClient.Get(ctx, synapseLookupKey, synapse)
// 						return synapse.Status
// 					}, timeout, interval).Should(Equal(expectedStatus))
// 				})

// 				It("Should create a Synapse ConfigMap", func() {
// 					checkResourcePresence(createdConfigMap, synapseLookupKey, expectedOwnerReference)
// 				})

// 				It("Should create a Synapse PVC", func() {
// 					checkResourcePresence(createdPVC, synapseLookupKey, expectedOwnerReference)
// 				})

// 				It("Should create a Synapse Deployment", func() {
// 					By("Checking that a Synapse Deployment exists and is correctly configured")
// 					checkResourcePresence(createdDeployment, synapseLookupKey, expectedOwnerReference)

// 					By("Checking that initContainers contains the required environment variables")
// 					envVars := []corev1.EnvVar{{
// 						Name:  "SYNAPSE_SERVER_NAME",
// 						Value: ServerName,
// 					}, {
// 						Name:  "SYNAPSE_REPORT_STATS",
// 						Value: utils.BoolToYesNo(ReportStats),
// 					}}
// 					Expect(createdDeployment.Spec.Template.Spec.InitContainers[0].Env).Should(ContainElements(envVars))
// 				})

// 				It("Should create a Synapse Service", func() {
// 					checkResourcePresence(createdService, synapseLookupKey, expectedOwnerReference)
// 				})

// 				It("Should create a Synapse ServiceAccount", func() {
// 					checkResourcePresence(createdServiceAccount, synapseLookupKey, expectedOwnerReference)
// 				})

// 				It("Should create a Synapse RoleBinding", func() {
// 					checkResourcePresence(createdRoleBinding, synapseLookupKey, expectedOwnerReference)
// 				})
// 			})

// 			When("Requesting a new PostgreSQL instance to be created for Synapse", func() {
// 				var createdPostgresCluster *pgov1beta1.PostgresCluster
// 				var postgresSecret corev1.Secret
// 				var postgresLookupKeys types.NamespacedName

// 				BeforeAll(func() {
// 					initSynapseVariables()

// 					postgresLookupKeys = types.NamespacedName{
// 						Name:      synapseLookupKey.Name + "-pgsql",
// 						Namespace: synapseLookupKey.Namespace,
// 					}

// 					// Init variable
// 					createdPostgresCluster = &pgov1beta1.PostgresCluster{}

// 					inputConfigmapData = map[string]string{
// 						"homeserver.yaml": "server_name: " + ServerName + "\n" +
// 							"report_stats: " + strconv.FormatBool(ReportStats),
// 					}

// 					synapseSpec = synapsev1alpha1.SynapseSpec{
// 						Homeserver: synapsev1alpha1.SynapseHomeserver{
// 							ConfigMap: &synapsev1alpha1.SynapseHomeserverConfigMap{
// 								Name: InputConfigMapName,
// 							},
// 						},
// 						CreateNewPostgreSQL: true,
// 					}

// 					createSynapseConfigMap()
// 					createSynapseInstance()
// 				})

// 				doPostgresControllerJob := func() {
// 					// The postgres-operator is responsible for creating a Secret holding
// 					// information on how to connect to the synapse Database with the synapse
// 					// user. As this controller is not running during our integration tests,
// 					// we have to manually create this secret here.
// 					postgresSecret = corev1.Secret{
// 						ObjectMeta: metav1.ObjectMeta{
// 							Name:      SynapseName + "-pgsql-pguser-synapse",
// 							Namespace: SynapseNamespace,
// 						},
// 						Data: map[string][]byte{
// 							"host":     []byte("hostname.postgresql.url"),
// 							"port":     []byte("5432"),
// 							"dbname":   []byte("synapse"),
// 							"user":     []byte("synapse"),
// 							"password": []byte("VerySecureSyn@psePassword!"),
// 						},
// 					}
// 					Expect(k8sClient.Create(ctx, &postgresSecret)).Should(Succeed())

// 					// The portgres-operator is responsible for updating the PostgresCluster
// 					// status, with the number of Pods being ready. This is used a part of
// 					// the 'isPostgresClusterReady' method in the Synapse controller.
// 					createdPostgresCluster.Status.InstanceSets = []pgov1beta1.PostgresInstanceSetStatus{{
// 						Name:            "instance1",
// 						Replicas:        1,
// 						ReadyReplicas:   1,
// 						UpdatedReplicas: 1,
// 					}}
// 					Expect(k8sClient.Status().Update(ctx, createdPostgresCluster)).Should(Succeed())
// 				}

// 				AfterAll(func() {
// 					By("Cleaning up the Synapse PostgresCluster")
// 					deleteResource(createdPostgresCluster, postgresLookupKeys, false)

// 					cleanupSynapseResources()
// 					cleanupSynapseConfigMap()
// 				})

// 				It("Should create a PostgresCluster for Synapse", func() {
// 					By("Checking that a Synapse PostgresCluster exists")
// 					checkResourcePresence(createdPostgresCluster, postgresLookupKeys, expectedOwnerReference)
// 				})

// 				It("Should update the Synapse status", func() {
// 					By("Checking that the controller detects the Database as not ready")
// 					Expect(k8sClient.Get(ctx, synapseLookupKey, synapse)).Should(Succeed())
// 					Expect(synapse.Status.DatabaseConnectionInfo.State).Should(Equal("NOT READY"))

// 					// Once the PostgresCluster has been created, we simulate the
// 					// postgres-operator reconciliation.
// 					By("Simulating the postgres-operator controller job")
// 					doPostgresControllerJob()

// 					By("Checking that the Synapse Status is correctly updated")
// 					Eventually(func(g Gomega) {
// 						g.Expect(k8sClient.Get(ctx, synapseLookupKey, synapse)).Should(Succeed())

// 						g.Expect(synapse.Status.DatabaseConnectionInfo.ConnectionURL).Should(Equal("hostname.postgresql.url:5432"))
// 						g.Expect(synapse.Status.DatabaseConnectionInfo.DatabaseName).Should(Equal("synapse"))
// 						g.Expect(synapse.Status.DatabaseConnectionInfo.User).Should(Equal("synapse"))
// 						g.Expect(synapse.Status.DatabaseConnectionInfo.Password).Should(Equal(string(base64encode("VerySecureSyn@psePassword!"))))
// 						g.Expect(synapse.Status.DatabaseConnectionInfo.State).Should(Equal("READY"))
// 					}, timeout, interval).Should(Succeed())
// 				})

// 				It("Should update the ConfigMap Data", func() {
// 					Eventually(func(g Gomega) {
// 						// Fetching database section of the homeserver.yaml configuration file
// 						g.Expect(k8sClient.Get(ctx,
// 							types.NamespacedName{Name: SynapseName, Namespace: SynapseNamespace},
// 							createdConfigMap,
// 						)).Should(Succeed())

// 						cm_data, ok := createdConfigMap.Data["homeserver.yaml"]
// 						g.Expect(ok).Should(BeTrue())

// 						homeserver := make(map[string]interface{})
// 						g.Expect(yaml.Unmarshal([]byte(cm_data), homeserver)).Should(Succeed())

// 						_, ok = homeserver["database"]
// 						g.Expect(ok).Should(BeTrue())

// 						marshalled_homeserver_database, err := yaml.Marshal(homeserver["database"])
// 						g.Expect(err).ShouldNot(HaveOccurred())

// 						var hs_database HomeserverPgsqlDatabase
// 						g.Expect(yaml.Unmarshal(marshalled_homeserver_database, &hs_database)).Should(Succeed())

// 						// hs_database, ok := homeserver["database"].(HomeserverPgsqlDatabase)
// 						// g.Expect(ok).Should(BeTrue())

// 						// Testing that the database section is correctly configured for using
// 						// the PostgreSQL DB
// 						g.Expect(hs_database.Name).Should(Equal("psycopg2"))
// 						g.Expect(hs_database.Args.Host).Should(Equal("hostname.postgresql.url"))

// 						g.Expect(hs_database.Args.Port).Should(Equal(int64(5432)))
// 						g.Expect(hs_database.Args.Database).Should(Equal("synapse"))
// 						g.Expect(hs_database.Args.User).Should(Equal("synapse"))
// 						g.Expect(hs_database.Args.Password).Should(Equal("VerySecureSyn@psePassword!"))

// 						g.Expect(hs_database.Args.CpMin).Should(Equal(int64(5)))
// 						g.Expect(hs_database.Args.CpMax).Should(Equal(int64(10)))
// 					}, timeout, interval).Should(Succeed())
// 				})
// 			})

// 			// When("Enabling the Heisenbridge", func() {
// 			// 	const (
// 			// 		heisenbridgePort = 9898
// 			// 	)

// 			// 	var createdHeisenbridgeDeployment *appsv1.Deployment
// 			// 	var createdHeisenbridgeService *corev1.Service
// 			// 	var createdHeisenbridgeConfigMap *corev1.ConfigMap
// 			// 	var heisenbridgeLookupKey types.NamespacedName

// 			// 	var initHeisenbridgeVariables = func() {
// 			// 		// Init vars
// 			// 		createdHeisenbridgeDeployment = &appsv1.Deployment{}
// 			// 		createdHeisenbridgeService = &corev1.Service{}
// 			// 		createdHeisenbridgeConfigMap = &corev1.ConfigMap{}

// 			// 		heisenbridgeLookupKey = types.NamespacedName{Name: SynapseName + "-heisenbridge", Namespace: SynapseNamespace}
// 			// 	}

// 			// 	var cleanupHeisenbridgeResources = func() {
// 			// 		By("Cleaning up the Heisenbridge Deployment")
// 			// 		deleteResource(createdHeisenbridgeDeployment, heisenbridgeLookupKey, false)

// 			// 		By("Cleaning up the Heisenbridge Service")
// 			// 		deleteResource(createdHeisenbridgeService, heisenbridgeLookupKey, false)

// 			// 		By("Cleaning up the Heisenbridge ConfigMap")
// 			// 		deleteResource(createdHeisenbridgeConfigMap, heisenbridgeLookupKey, false)
// 			// 	}

// 			// 	When("Using the default configuration", func() {
// 			// 		BeforeAll(func() {
// 			// 			initSynapseVariables()
// 			// 			initHeisenbridgeVariables()

// 			// 			inputConfigmapData = map[string]string{
// 			// 				"homeserver.yaml": "server_name: " + ServerName + "\n" +
// 			// 					"report_stats: " + strconv.FormatBool(ReportStats),
// 			// 			}

// 			// 			synapseSpec = synapsev1alpha1.SynapseSpec{
// 			// 				Homeserver: synapsev1alpha1.SynapseHomeserver{
// 			// 					ConfigMap: &synapsev1alpha1.SynapseHomeserverConfigMap{
// 			// 						Name: InputConfigMapName,
// 			// 					},
// 			// 				},
// 			// 				Bridges: synapsev1alpha1.SynapseBridges{
// 			// 					Heisenbridge: synapsev1alpha1.SynapseHeisenbridge{
// 			// 						Enabled: true,
// 			// 					},
// 			// 				},
// 			// 			}

// 			// 			createSynapseConfigMap()
// 			// 			createSynapseInstance()
// 			// 		})

// 			// 		AfterAll(func() {
// 			// 			// Cleanup Heisenbridge resources
// 			// 			cleanupSynapseResources()
// 			// 			cleanupSynapseConfigMap()
// 			// 			cleanupHeisenbridgeResources()
// 			// 		})

// 			// 		It("Should create a ConfigMap for Heisenbridge", func() {
// 			// 			checkResourcePresence(createdHeisenbridgeConfigMap, heisenbridgeLookupKey, expectedOwnerReference)
// 			// 		})

// 			// 		It("Should create a Deployment for Heisenbridge", func() {
// 			// 			checkResourcePresence(createdHeisenbridgeDeployment, heisenbridgeLookupKey, expectedOwnerReference)
// 			// 		})

// 			// 		It("Should create a Service for Heisenbridge", func() {
// 			// 			checkResourcePresence(createdHeisenbridgeService, heisenbridgeLookupKey, expectedOwnerReference)
// 			// 		})

// 			// 		It("Should update the Synapse homeserver.yaml", func() {
// 			// 			Eventually(func(g Gomega) {
// 			// 				g.Expect(k8sClient.Get(ctx,
// 			// 					types.NamespacedName{Name: SynapseName, Namespace: SynapseNamespace},
// 			// 					createdConfigMap,
// 			// 				)).Should(Succeed())

// 			// 				cm_data, ok := createdConfigMap.Data["homeserver.yaml"]
// 			// 				g.Expect(ok).Should(BeTrue())

// 			// 				homeserver := make(map[string]interface{})
// 			// 				g.Expect(yaml.Unmarshal([]byte(cm_data), homeserver)).Should(Succeed())

// 			// 				_, ok = homeserver["app_service_config_files"]
// 			// 				g.Expect(ok).Should(BeTrue())

// 			// 				g.Expect(homeserver["app_service_config_files"]).Should(ContainElement("/data-heisenbridge/heisenbridge.yaml"))
// 			// 			}, timeout, interval).Should(Succeed())
// 			// 		})
// 			// 	})

// 			// 	When("The user provides an input ConfigMap", func() {
// 			// 		var inputHeisenbridgeConfigMap *corev1.ConfigMap
// 			// 		var inputHeisenbridgeConfigMapData map[string]string

// 			// 		const InputHeisenbridgeConfigMapName = "heisenbridge-input"
// 			// 		const heisenbridgeFQDN = SynapseName + "-heisenbridge." + SynapseNamespace + ".svc.cluster.local"

// 			// 		BeforeAll(func() {
// 			// 			initSynapseVariables()
// 			// 			initHeisenbridgeVariables()

// 			// 			inputConfigmapData = map[string]string{
// 			// 				"homeserver.yaml": "server_name: " + ServerName + "\n" +
// 			// 					"report_stats: " + strconv.FormatBool(ReportStats),
// 			// 			}

// 			// 			synapseSpec = synapsev1alpha1.SynapseSpec{
// 			// 				Homeserver: synapsev1alpha1.SynapseHomeserver{
// 			// 					ConfigMap: &synapsev1alpha1.SynapseHomeserverConfigMap{
// 			// 						Name: InputConfigMapName,
// 			// 					},
// 			// 				},
// 			// 				Bridges: synapsev1alpha1.SynapseBridges{
// 			// 					Heisenbridge: synapsev1alpha1.SynapseHeisenbridge{
// 			// 						Enabled: true,
// 			// 						ConfigMap: synapsev1alpha1.SynapseHeisenbridgeConfigMap{
// 			// 							Name: InputHeisenbridgeConfigMapName,
// 			// 						},
// 			// 					},
// 			// 				},
// 			// 			}

// 			// 			By("Creating a ConfigMap containing a basic heisenbridge.yaml")
// 			// 			// Incomplete heisenbridge.yaml, containing only the
// 			// 			// required data for our tests. In particular, we
// 			// 			// will test if the URL has been correctly updated
// 			// 			inputHeisenbridgeConfigMapData = map[string]string{
// 			// 				"heisenbridge.yaml": "url: http://10.217.5.134:" + strconv.Itoa(heisenbridgePort),
// 			// 			}

// 			// 			inputHeisenbridgeConfigMap = &corev1.ConfigMap{
// 			// 				ObjectMeta: metav1.ObjectMeta{
// 			// 					Name:      InputHeisenbridgeConfigMapName,
// 			// 					Namespace: SynapseNamespace,
// 			// 				},
// 			// 				Data: inputHeisenbridgeConfigMapData,
// 			// 			}
// 			// 			Expect(k8sClient.Create(ctx, inputHeisenbridgeConfigMap)).Should(Succeed())

// 			// 			createSynapseConfigMap()
// 			// 			createSynapseInstance()
// 			// 		})

// 			// 		AfterAll(func() {
// 			// 			// Cleanup Heisenbridge resources
// 			// 			By("Cleaning up the Heisenbridge ConfigMap")
// 			// 			heisenbridgeConfigMapLookupKey := types.NamespacedName{
// 			// 				Name:      InputHeisenbridgeConfigMapName,
// 			// 				Namespace: SynapseNamespace,
// 			// 			}

// 			// 			deleteResource(inputHeisenbridgeConfigMap, heisenbridgeConfigMapLookupKey, false)

// 			// 			cleanupSynapseResources()
// 			// 			cleanupSynapseConfigMap()
// 			// 			cleanupHeisenbridgeResources()
// 			// 		})

// 			// 		It("Should create a ConfigMap for Heisenbridge", func() {
// 			// 			checkResourcePresence(createdHeisenbridgeConfigMap, heisenbridgeLookupKey, expectedOwnerReference)
// 			// 		})

// 			// 		It("Should create a Deployment for Heisenbridge", func() {
// 			// 			checkResourcePresence(createdHeisenbridgeDeployment, heisenbridgeLookupKey, expectedOwnerReference)
// 			// 		})

// 			// 		It("Should create a Service for Heisenbridge", func() {
// 			// 			checkResourcePresence(createdHeisenbridgeService, heisenbridgeLookupKey, expectedOwnerReference)
// 			// 		})

// 			// 		It("Should add url value to the created Heisenbridge ConfigMap", func() {
// 			// 			Eventually(func(g Gomega) {
// 			// 				g.Expect(k8sClient.Get(ctx, heisenbridgeLookupKey, inputHeisenbridgeConfigMap)).Should(Succeed())

// 			// 				cm_data, ok := inputHeisenbridgeConfigMap.Data["heisenbridge.yaml"]
// 			// 				g.Expect(ok).Should(BeTrue())

// 			// 				heisenbridge := make(map[string]interface{})
// 			// 				g.Expect(yaml.Unmarshal([]byte(cm_data), heisenbridge)).Should(Succeed())

// 			// 				_, ok = heisenbridge["url"]
// 			// 				g.Expect(ok).Should(BeTrue())

// 			// 				g.Expect(heisenbridge["url"]).To(Equal("http://" + heisenbridgeFQDN + ":" + strconv.Itoa(heisenbridgePort)))
// 			// 			}, timeout, interval).Should(Succeed())
// 			// 		})
// 			// 	})
// 			// })

// 			// When("Enabling the mautrix-signal", func() {
// 			// 	const (
// 			// 		mautrixSignalPort = 29328
// 			// 	)

// 			// 	var createdSignaldDeployment *appsv1.Deployment
// 			// 	var createdSignaldPVC *corev1.PersistentVolumeClaim
// 			// 	var createdMautrixSignalServiceAccount *corev1.ServiceAccount
// 			// 	var createdMautrixSignalRoleBinding *rbacv1.RoleBinding
// 			// 	var createdMautrixSignalDeployment *appsv1.Deployment
// 			// 	var createdMautrixSignalPVC *corev1.PersistentVolumeClaim
// 			// 	var createdMautrixSignalService *corev1.Service
// 			// 	var createdMautrixSignalConfigMap *corev1.ConfigMap
// 			// 	var mautrixSignalLookupKey types.NamespacedName
// 			// 	var signaldLookupKey types.NamespacedName

// 			// 	var initMautrixSignalVariables = func() {
// 			// 		// Init vars
// 			// 		createdSignaldDeployment = &appsv1.Deployment{}
// 			// 		createdSignaldPVC = &corev1.PersistentVolumeClaim{}
// 			// 		createdMautrixSignalServiceAccount = &corev1.ServiceAccount{}
// 			// 		createdMautrixSignalRoleBinding = &rbacv1.RoleBinding{}
// 			// 		createdMautrixSignalDeployment = &appsv1.Deployment{}
// 			// 		createdMautrixSignalPVC = &corev1.PersistentVolumeClaim{}
// 			// 		createdMautrixSignalService = &corev1.Service{}
// 			// 		createdMautrixSignalConfigMap = &corev1.ConfigMap{}

// 			// 		signaldLookupKey = types.NamespacedName{Name: SynapseName + "-signald", Namespace: SynapseNamespace}
// 			// 		mautrixSignalLookupKey = types.NamespacedName{Name: SynapseName + "-mautrixsignal", Namespace: SynapseNamespace}
// 			// 	}

// 			// 	var cleanupMautrixSignalResources = func() {
// 			// 		By("Cleaning up the signald Deployment")
// 			// 		deleteResource(createdSignaldDeployment, signaldLookupKey, false)

// 			// 		By("Cleaning up the mautrix-signal Deployment")
// 			// 		deleteResource(createdMautrixSignalDeployment, mautrixSignalLookupKey, false)

// 			// 		By("Cleaning up the signald PVC")
// 			// 		deleteResource(createdSignaldPVC, signaldLookupKey, true)

// 			// 		By("Cleaning up the mautrix-signal PVC")
// 			// 		deleteResource(createdMautrixSignalPVC, mautrixSignalLookupKey, true)

// 			// 		By("Cleaning up the mautrix-signal Service")
// 			// 		deleteResource(createdMautrixSignalService, mautrixSignalLookupKey, false)

// 			// 		By("Cleaning up the mautrix-signal ConfigMap")
// 			// 		deleteResource(createdMautrixSignalConfigMap, mautrixSignalLookupKey, false)

// 			// 		By("Cleaning up mautrix-signal RoleBinding")
// 			// 		deleteResource(createdMautrixSignalRoleBinding, mautrixSignalLookupKey, false)

// 			// 		By("Cleaning up mautrix-signal ServiceAccount")
// 			// 		deleteResource(createdMautrixSignalServiceAccount, mautrixSignalLookupKey, false)
// 			// 	}

// 			// 	When("Using the default configuration", func() {
// 			// 		BeforeAll(func() {
// 			// 			initSynapseVariables()
// 			// 			initMautrixSignalVariables()

// 			// 			inputConfigmapData = map[string]string{
// 			// 				"homeserver.yaml": "server_name: " + ServerName + "\n" +
// 			// 					"report_stats: " + strconv.FormatBool(ReportStats),
// 			// 			}

// 			// 			synapseSpec = synapsev1alpha1.SynapseSpec{
// 			// 				Homeserver: synapsev1alpha1.SynapseHomeserver{
// 			// 					ConfigMap: &synapsev1alpha1.SynapseHomeserverConfigMap{
// 			// 						Name: InputConfigMapName,
// 			// 					},
// 			// 				},
// 			// 				Bridges: synapsev1alpha1.SynapseBridges{
// 			// 					MautrixSignal: synapsev1alpha1.SynapseMautrixSignal{
// 			// 						Enabled: true,
// 			// 					},
// 			// 				},
// 			// 			}

// 			// 			createSynapseConfigMap()
// 			// 			createSynapseInstance()
// 			// 		})

// 			// 		AfterAll(func() {
// 			// 			// Cleanup mautrix-signal resources
// 			// 			cleanupSynapseResources()
// 			// 			cleanupSynapseConfigMap()
// 			// 			cleanupMautrixSignalResources()
// 			// 		})

// 			// 		It("Should create a Deployment for signald", func() {
// 			// 			checkResourcePresence(createdSignaldDeployment, signaldLookupKey, expectedOwnerReference)
// 			// 		})

// 			// 		It("Should create a PVC for signald", func() {
// 			// 			checkResourcePresence(createdSignaldPVC, signaldLookupKey, expectedOwnerReference)
// 			// 		})

// 			// 		It("Should create a ConfigMap for mautrix-signal", func() {
// 			// 			checkResourcePresence(createdMautrixSignalConfigMap, mautrixSignalLookupKey, expectedOwnerReference)
// 			// 		})

// 			// 		It("Should create a Role Binding for mautrix-signal", func() {
// 			// 			checkResourcePresence(createdMautrixSignalRoleBinding, mautrixSignalLookupKey, expectedOwnerReference)
// 			// 		})

// 			// 		It("Should create a Service Account for mautrix-signal", func() {
// 			// 			checkResourcePresence(createdMautrixSignalServiceAccount, mautrixSignalLookupKey, expectedOwnerReference)
// 			// 		})

// 			// 		It("Should create a Deployment for mautrix-signal", func() {
// 			// 			checkResourcePresence(createdMautrixSignalDeployment, mautrixSignalLookupKey, expectedOwnerReference)
// 			// 		})

// 			// 		It("Should create a PVC for mautrix-signal", func() {
// 			// 			checkResourcePresence(createdMautrixSignalPVC, mautrixSignalLookupKey, expectedOwnerReference)
// 			// 		})

// 			// 		It("Should create a Service for mautrix-signal", func() {
// 			// 			checkResourcePresence(createdMautrixSignalService, mautrixSignalLookupKey, expectedOwnerReference)
// 			// 		})

// 			// 		It("Should update the Synapse homeserver.yaml", func() {
// 			// 			Eventually(func(g Gomega) {
// 			// 				g.Expect(k8sClient.Get(ctx,
// 			// 					types.NamespacedName{Name: SynapseName, Namespace: SynapseNamespace},
// 			// 					createdConfigMap,
// 			// 				)).Should(Succeed())

// 			// 				cm_data, ok := createdConfigMap.Data["homeserver.yaml"]
// 			// 				g.Expect(ok).Should(BeTrue())

// 			// 				homeserver := make(map[string]interface{})
// 			// 				g.Expect(yaml.Unmarshal([]byte(cm_data), homeserver)).Should(Succeed())

// 			// 				_, ok = homeserver["app_service_config_files"]
// 			// 				g.Expect(ok).Should(BeTrue())

// 			// 				g.Expect(homeserver["app_service_config_files"]).Should(ContainElement("/data-mautrixsignal/registration.yaml"))
// 			// 			}, timeout, interval).Should(Succeed())
// 			// 		})
// 			// 	})

// 			// 	When("The user provides an input ConfigMap", func() {
// 			// 		var inputMautrixSignalConfigMap *corev1.ConfigMap
// 			// 		var inputMautrixSignalConfigMapData map[string]string

// 			// 		const InputMautrixSignalConfigMapName = "mautrix-signal-input"
// 			// 		const mautrixSignalFQDN = SynapseName + "-mautrixsignal." + SynapseNamespace + ".svc.cluster.local"
// 			// 		const synapseFQDN = SynapseName + "." + SynapseNamespace + ".svc.cluster.local"

// 			// 		BeforeAll(func() {
// 			// 			initSynapseVariables()
// 			// 			initMautrixSignalVariables()

// 			// 			inputConfigmapData = map[string]string{
// 			// 				"homeserver.yaml": "server_name: " + ServerName + "\n" +
// 			// 					"report_stats: " + strconv.FormatBool(ReportStats),
// 			// 			}

// 			// 			synapseSpec = synapsev1alpha1.SynapseSpec{
// 			// 				Homeserver: synapsev1alpha1.SynapseHomeserver{
// 			// 					ConfigMap: &synapsev1alpha1.SynapseHomeserverConfigMap{
// 			// 						Name: InputConfigMapName,
// 			// 					},
// 			// 				},
// 			// 				Bridges: synapsev1alpha1.SynapseBridges{
// 			// 					MautrixSignal: synapsev1alpha1.SynapseMautrixSignal{
// 			// 						Enabled: true,
// 			// 						ConfigMap: synapsev1alpha1.SynapseMautrixSignalConfigMap{
// 			// 							Name: InputMautrixSignalConfigMapName,
// 			// 						},
// 			// 					},
// 			// 				},
// 			// 			}

// 			// 			By("Creating a ConfigMap containing a basic config.yaml")
// 			// 			// Incomplete config.yaml, containing only the
// 			// 			// required data for our tests. We test that those
// 			// 			// values are correctly updated
// 			// 			configYaml := `
// 			//             homeserver:
// 			//                 address: http://localhost:8008
// 			//                 domain: mydomain.com
// 			//             appservice:
// 			//                 address: http://localhost:29328
// 			//             signal:
// 			//                 socket_path: /var/run/signald/signald.sock
// 			//             bridge:
// 			//                 permissions:
// 			//                     "*": "relay"
// 			//                     "mydomain.com": "user"
// 			//                     "@admin:mydomain.com": "admin"
// 			//             logging:
// 			//                 handlers:
// 			//                     file:
// 			//                         filename: ./mautrix-signal.log`

// 			// 			inputMautrixSignalConfigMapData = map[string]string{
// 			// 				"config.yaml": configYaml,
// 			// 			}

// 			// 			inputMautrixSignalConfigMap = &corev1.ConfigMap{
// 			// 				ObjectMeta: metav1.ObjectMeta{
// 			// 					Name:      InputMautrixSignalConfigMapName,
// 			// 					Namespace: SynapseNamespace,
// 			// 				},
// 			// 				Data: inputMautrixSignalConfigMapData,
// 			// 			}
// 			// 			Expect(k8sClient.Create(ctx, inputMautrixSignalConfigMap)).Should(Succeed())

// 			// 			createSynapseConfigMap()
// 			// 			createSynapseInstance()
// 			// 		})

// 			// 		AfterAll(func() {
// 			// 			// Cleanup mautrix-signal resources
// 			// 			By("Cleaning up the mautrix-signal ConfigMap")
// 			// 			mautrixSignalConfigMapLookupKey := types.NamespacedName{
// 			// 				Name:      InputMautrixSignalConfigMapName,
// 			// 				Namespace: SynapseNamespace,
// 			// 			}

// 			// 			deleteResource(inputMautrixSignalConfigMap, mautrixSignalConfigMapLookupKey, false)

// 			// 			cleanupSynapseResources()
// 			// 			cleanupSynapseConfigMap()
// 			// 			cleanupMautrixSignalResources()
// 			// 		})

// 			// 		It("Should create a Deployment for signald", func() {
// 			// 			checkResourcePresence(createdMautrixSignalDeployment, signaldLookupKey, expectedOwnerReference)
// 			// 		})

// 			// 		It("Should create a PVC for signald", func() {
// 			// 			checkResourcePresence(createdSignaldPVC, signaldLookupKey, expectedOwnerReference)
// 			// 		})

// 			// 		It("Should create a ConfigMap for mautrix-signal", func() {
// 			// 			checkResourcePresence(createdMautrixSignalConfigMap, mautrixSignalLookupKey, expectedOwnerReference)
// 			// 		})

// 			// 		It("Should create a Role Binding for mautrix-signal", func() {
// 			// 			checkResourcePresence(createdMautrixSignalRoleBinding, mautrixSignalLookupKey, expectedOwnerReference)
// 			// 		})

// 			// 		It("Should create a Service Account for mautrix-signal", func() {
// 			// 			checkResourcePresence(createdMautrixSignalServiceAccount, mautrixSignalLookupKey, expectedOwnerReference)
// 			// 		})

// 			// 		It("Should create a Deployment for mautrix-signal", func() {
// 			// 			checkResourcePresence(createdMautrixSignalDeployment, mautrixSignalLookupKey, expectedOwnerReference)
// 			// 		})

// 			// 		It("Should create a PVC for mautrix-signal", func() {
// 			// 			checkResourcePresence(createdMautrixSignalPVC, mautrixSignalLookupKey, expectedOwnerReference)
// 			// 		})

// 			// 		It("Should create a Service for mautrix-signal", func() {
// 			// 			checkResourcePresence(createdMautrixSignalService, mautrixSignalLookupKey, expectedOwnerReference)
// 			// 		})

// 			// 		It("Should overwrite necessary values in the created mautrix-signal ConfigMap", func() {
// 			// 			Eventually(func(g Gomega) {
// 			// 				By("Verifying that the mautrixsignal ConfigMap exists")
// 			// 				g.Expect(k8sClient.Get(ctx, mautrixSignalLookupKey, inputMautrixSignalConfigMap)).Should(Succeed())

// 			// 				cm_data, ok := inputMautrixSignalConfigMap.Data["config.yaml"]
// 			// 				g.Expect(ok).Should(BeTrue())

// 			// 				config := make(map[string]interface{})
// 			// 				g.Expect(yaml.Unmarshal([]byte(cm_data), config)).Should(Succeed())

// 			// 				By("Verifying that the homeserver configuration has been updated")
// 			// 				configHomeserver, ok := config["homeserver"].(map[interface{}]interface{})
// 			// 				g.Expect(ok).Should(BeTrue())
// 			// 				g.Expect(configHomeserver["address"]).To(Equal("http://" + synapseFQDN + ":8008"))
// 			// 				g.Expect(configHomeserver["domain"]).To(Equal(ServerName))

// 			// 				By("Verifying that the appservice configuration has been updated")
// 			// 				configAppservice, ok := config["appservice"].(map[interface{}]interface{})
// 			// 				g.Expect(ok).Should(BeTrue())
// 			// 				g.Expect(configAppservice["address"]).To(Equal("http://" + mautrixSignalFQDN + ":" + strconv.Itoa(mautrixSignalPort)))

// 			// 				By("Verifying that the signal configuration has been updated")
// 			// 				configSignal, ok := config["signal"].(map[interface{}]interface{})
// 			// 				g.Expect(ok).Should(BeTrue())
// 			// 				g.Expect(configSignal["socket_path"]).To(Equal("/signald/signald.sock"))

// 			// 				By("Verifying that the permissions have been updated")
// 			// 				configBridge, ok := config["bridge"].(map[interface{}]interface{})
// 			// 				g.Expect(ok).Should(BeTrue())
// 			// 				configBridgePermissions, ok := configBridge["permissions"].(map[interface{}]interface{})
// 			// 				g.Expect(ok).Should(BeTrue())
// 			// 				g.Expect(configBridgePermissions).Should(HaveKeyWithValue("*", "relay"))
// 			// 				g.Expect(configBridgePermissions).Should(HaveKeyWithValue(ServerName, "user"))
// 			// 				g.Expect(configBridgePermissions).Should(HaveKeyWithValue("@admin:"+ServerName, "admin"))

// 			// 				By("Verifying that the log configuration file path have been updated")
// 			// 				configLogging, ok := config["logging"].(map[interface{}]interface{})
// 			// 				g.Expect(ok).Should(BeTrue())
// 			// 				configLoggingHandlers, ok := configLogging["handlers"].(map[interface{}]interface{})
// 			// 				g.Expect(ok).Should(BeTrue())
// 			// 				configLoggingHandlersFile, ok := configLoggingHandlers["file"].(map[interface{}]interface{})
// 			// 				g.Expect(ok).Should(BeTrue())
// 			// 				g.Expect(configLoggingHandlersFile["filename"]).To(Equal("/data/mautrix-signal.log"))
// 			// 			}, timeout, interval).Should(Succeed())
// 			// 		})
// 			// 	})
// 			// })

// 		})
// 	})

// 	Context("When creating an incorrect Synapse instance", func() {
// 		var synapse *synapsev1alpha1.Synapse

// 		BeforeEach(func() {
// 			By("Creating a Synapse instance which refers an absent ConfigMap")
// 			synapse = &synapsev1alpha1.Synapse{
// 				ObjectMeta: metav1.ObjectMeta{
// 					Name:      SynapseName,
// 					Namespace: SynapseNamespace,
// 				},
// 				Spec: synapsev1alpha1.SynapseSpec{
// 					Homeserver: synapsev1alpha1.SynapseHomeserver{
// 						ConfigMap: &synapsev1alpha1.SynapseHomeserverConfigMap{
// 							Name: InputConfigMapName,
// 						},
// 					},
// 				},
// 			}
// 			Expect(k8sClient.Create(ctx, synapse)).Should(Succeed())
// 		})

// 		AfterEach(func() {
// 			By("Cleaning up Synapse CR")
// 			Expect(k8sClient.Delete(ctx, synapse)).Should(Succeed())
// 		})

// 		It("Should get in a failed state and not create child objects", func() {
// 			reason := "ConfigMap " + InputConfigMapName + " does not exist in namespace " + SynapseNamespace
// 			checkSubresourceAbsence(reason)
// 		})
// 	})
// })

// Context("When the Kubernetes cluster is missing the postgres-operator", func() {
// 	BeforeAll(func() {
// 		logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

// 		ctx, cancel = context.WithCancel(context.TODO())

// 		By("bootstrapping test environment")
// 		testEnv = &envtest.Environment{
// 			CRDDirectoryPaths: []string{
// 				filepath.Join("..", "..", "..", "bundle", "manifests", "synapse.opdev.io_synapses.yaml"),
// 			},
// 			ErrorIfCRDPathMissing: true,
// 		}

// 		startenvTest()
// 	})

// 	AfterAll(func() {
// 		cancel()
// 		By("tearing down the test environment")
// 		err := testEnv.Stop()
// 		Expect(err).NotTo(HaveOccurred())
// 	})

// 	When("Requesting a new PostgreSQL instance to be created for Synapse", func() {
// 		var synapse *synapsev1alpha1.Synapse
// 		var configMap *corev1.ConfigMap

// 		BeforeAll(func() {
// 			configMap = &corev1.ConfigMap{
// 				ObjectMeta: metav1.ObjectMeta{
// 					Name:      InputConfigMapName,
// 					Namespace: SynapseNamespace,
// 				},
// 				Data: map[string]string{
// 					"homeserver.yaml": "server_name: " + ServerName + "\n" +
// 						"report_stats: " + strconv.FormatBool(ReportStats),
// 				},
// 			}
// 			Expect(k8sClient.Create(ctx, configMap)).Should(Succeed())

// 			By("Creating the Synapse instance")
// 			synapse = &synapsev1alpha1.Synapse{
// 				ObjectMeta: metav1.ObjectMeta{
// 					Name:      SynapseName,
// 					Namespace: SynapseNamespace,
// 				},
// 				Spec: synapsev1alpha1.SynapseSpec{
// 					Homeserver: synapsev1alpha1.SynapseHomeserver{
// 						ConfigMap: &synapsev1alpha1.SynapseHomeserverConfigMap{
// 							Name: InputConfigMapName,
// 						},
// 					},
// 					CreateNewPostgreSQL: true,
// 				},
// 			}
// 			Expect(k8sClient.Create(ctx, synapse)).Should(Succeed())
// 		})

// 		AfterAll(func() {
// 			By("Cleaning up ConfigMap")
// 			Expect(k8sClient.Delete(ctx, configMap)).Should(Succeed())

// 			By("Cleaning up Synapse CR")
// 			Expect(k8sClient.Delete(ctx, synapse)).Should(Succeed())
// 		})

// 		It("Should not create Synapse sub-resources", func() {
// 			reason := "Cannot create PostgreSQL instance for synapse. Postgres-operator is not installed."
// 			checkSubresourceAbsence(reason)
// 		})
// 	})
// 	})
// })
