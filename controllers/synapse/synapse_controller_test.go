package synapse

import (
	"context"
	"path/filepath"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"
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

	pgov1beta1 "github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
	synapsev1alpha1 "github.com/opdev/synapse-operator/apis/synapse/v1alpha1"
)

// Helper function for struct construction requiring a boolean pointer
func BoolAddr(b bool) *bool {
	boolVar := b
	return &boolVar
}

var _ = Describe("Integration tests for the Synapse controller", Ordered, Label("integration"), func() {
	var k8sClient client.Client
	var testEnv *envtest.Environment
	var ctx context.Context
	var cancel context.CancelFunc

	var _ = BeforeAll(func() {
		logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

		ctx, cancel = context.WithCancel(context.TODO())

		By("bootstrapping test environment")
		testEnv = &envtest.Environment{
			CRDDirectoryPaths: []string{
				filepath.Join("..", "..", "config", "crd", "bases"),
				filepath.Join("..", "..", "postgres-operator-crds"),
			},
			ErrorIfCRDPathMissing: true,
		}

		cfg, err := testEnv.Start()
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg).NotTo(BeNil())

		err = synapsev1alpha1.AddToScheme(scheme.Scheme)
		Expect(err).NotTo(HaveOccurred())

		err = pgov1beta1.AddToScheme(scheme.Scheme)
		Expect(err).NotTo(HaveOccurred())

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
			err = k8sManager.Start(ctx)
			Expect(err).ToNot(HaveOccurred(), "failed to run manager")
		}()
	})

	var _ = AfterAll(func() {
		cancel()
		By("tearing down the test environment")
		err := testEnv.Stop()
		Expect(err).NotTo(HaveOccurred())
	})

	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		SynapseName      = "test-synapse"
		SynapseNamespace = "default"
		ConfigMapName    = "test-configmap"
		ServerName       = "example.com"
		ReportStats      = false

		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

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
			Entry("when Synapse spec is missing HomeserverConfigMapName", map[string]interface{}{
				"spec": map[string]interface{}{"createNewPostgreSQL": true},
			}),
			Entry("when Synapse spec possesses an invalid field", map[string]interface{}{
				"spec": map[string]interface{}{
					"HomeserverConfigMapName": ConfigMapName,
					"InvalidSpecFiels":        "random",
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
			Entry("when all spec fields are specified", map[string]interface{}{
				"spec": map[string]interface{}{
					"homeserverConfigMapName": ConfigMapName,
					"createNewPostreSQL":      true,
				},
			}),
			Entry("when optional CreateNewPostgreSQL is missing", map[string]interface{}{
				"spec": map[string]interface{}{
					"homeserverConfigMapName": ConfigMapName,
				},
			}),
		)
	})

	Context("When creating a valid Synapse instance", func() {
		var synapse *synapsev1alpha1.Synapse
		var configMap *corev1.ConfigMap
		var createdPVC *corev1.PersistentVolumeClaim
		var createdDeployment *appsv1.Deployment
		var createdService *corev1.Service
		var synapseLookupKey types.NamespacedName
		var expectedOwnerReference metav1.OwnerReference

		var configmapData map[string]string
		var synapseSpec synapsev1alpha1.SynapseSpec

		BeforeEach(OncePerOrdered, func() {
			// Init variables
			synapseLookupKey = types.NamespacedName{Name: SynapseName, Namespace: SynapseNamespace}
			createdPVC = &corev1.PersistentVolumeClaim{}
			createdDeployment = &appsv1.Deployment{}
			createdService = &corev1.Service{}
			// The OwnerReference UID must be set after the Synapse instance has been
			// created. See the JustBeforeEach node.
			expectedOwnerReference = metav1.OwnerReference{
				Kind:               "Synapse",
				APIVersion:         "synapse.opdev.io/v1alpha1",
				Name:               SynapseName,
				Controller:         BoolAddr(true),
				BlockOwnerDeletion: BoolAddr(true),
			}
		})

		JustBeforeEach(OncePerOrdered, func() {
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
		})

		AfterEach(OncePerOrdered, func() {
			// Common cleanup steps for tests of valid Synapse instances
			By("Cleaning up ConfigMap")
			Expect(k8sClient.Delete(ctx, configMap)).Should(Succeed())

			By("Cleaning up Synapse CR")
			Expect(k8sClient.Delete(ctx, synapse)).Should(Succeed())

			// Child resources must be manually deleted as the controllers responsible of
			// their lifecycle are not running.
			By("Cleaning up Synapse PVC")
			k8sClient.Get(ctx, synapseLookupKey, createdPVC)
			// Manually remove the PVC finalizers
			createdPVC.ObjectMeta.Finalizers = []string{}
			Expect(k8sClient.Update(ctx, createdPVC)).Should(Succeed())

			// Deleting PVC
			Expect(k8sClient.Delete(ctx, createdPVC)).Should(Succeed())

			// Check PVC was successfully removed
			Eventually(func() bool {
				err := k8sClient.Get(ctx, synapseLookupKey, &corev1.PersistentVolumeClaim{})
				return err == nil
			}, timeout, interval).Should(BeFalse())

			By("Cleaning up Synapse Deployment")
			// Deleting Deployment
			k8sClient.Get(ctx, synapseLookupKey, createdDeployment)
			Expect(k8sClient.Delete(ctx, createdDeployment)).Should(Succeed())

			// Check Deployment was successfully removed
			Eventually(func() bool {
				err := k8sClient.Get(ctx, synapseLookupKey, &appsv1.Deployment{})
				return err == nil
			}, timeout, interval).Should(BeFalse())

			By("Cleaning up Synapse Service")
			// Deleting Service
			k8sClient.Get(ctx, synapseLookupKey, createdService)
			Expect(k8sClient.Delete(ctx, createdService)).Should(Succeed())

			// Check Service was successfully removed
			Eventually(func() bool {
				err := k8sClient.Get(ctx, synapseLookupKey, &corev1.Service{})
				return err == nil
			}, timeout, interval).Should(BeFalse())
		})

		When("Creating a simple Synapse instance", Ordered, func() {
			// BeforeAll vs BeforeEach ?
			BeforeAll(func() {
				configmapData = map[string]string{
					"homeserver.yaml": "server_name: " + ServerName + "\n" +
						"report_stats: " + strconv.FormatBool(ReportStats),
				}

				synapseSpec = synapsev1alpha1.SynapseSpec{
					HomeserverConfigMapName: ConfigMapName,
				}
			})

			It("Should should update the Synapse Status", func() {
				expectedStatus := synapsev1alpha1.SynapseStatus{
					State: "RUNNING",
					HomeserverConfiguration: synapsev1alpha1.SynapseStatusHomeserverConfiguration{
						ServerName:  ServerName,
						ReportStats: ReportStats,
					},
				}
				// Status may need some time to be updated
				Eventually(func() synapsev1alpha1.SynapseStatus {
					_ = k8sClient.Get(ctx, synapseLookupKey, synapse)
					return synapse.Status
				}, timeout, interval).Should(Equal(expectedStatus))
			})

			It("Should create a Synapse PVC", func() {
				By("Checking that a Synapse PVC exists")
				Eventually(func() bool {
					err := k8sClient.Get(ctx, synapseLookupKey, createdPVC)
					return err == nil
				}, timeout, interval).Should(BeTrue())

				By("Checking that the PVC's OwnerReference contains the Synapse instance")
				Expect(createdPVC.ObjectMeta.OwnerReferences).To(ContainElement(expectedOwnerReference))

			})

			It("Should create a Synapse Deployment", func() {
				By("Checking that a Synapse Deployment exists and is correctly configured")
				Eventually(func() bool {
					err := k8sClient.Get(ctx, synapseLookupKey, createdDeployment)
					return err == nil
				}, timeout, interval).Should(BeTrue())

				By("Checking that the Deployment's OwnerReference contains the Synapse instance")
				Expect(createdDeployment.ObjectMeta.OwnerReferences).To(ContainElement(expectedOwnerReference))

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
				By("Checking that a Synapse Service exists")
				Eventually(func() bool {
					err := k8sClient.Get(ctx, synapseLookupKey, createdService)
					return err == nil
				}, timeout, interval).Should(BeTrue())

				By("Checking that the Service's OwnerReference contains the Synapse instance")
				Expect(createdService.ObjectMeta.OwnerReferences).To(ContainElement(expectedOwnerReference))
			})
		})

		When("Requesting a new PostgreSQL instance to be created for Synapse", Ordered, func() {
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
					HomeserverConfigMapName: ConfigMapName,
					CreateNewPostgreSQL:     true,
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
				// Deleting PostreSQL
				Expect(k8sClient.Delete(ctx, createdPostgresCluster)).Should(Succeed())

				// Check PostgresCluster was successfully removed
				Eventually(func() bool {
					err := k8sClient.Get(ctx, synapseLookupKey, &pgov1beta1.PostgresCluster{})
					return err == nil
				}, timeout, interval).Should(BeFalse())

				Expect(k8sClient.Delete(ctx, &postgresSecret)).Should(Succeed())
			})

			It("Should create a PostgresCluster for Synapse", func() {
				By("Checking that a Synapse PostgresCluster exists")
				Eventually(func() bool {
					err := k8sClient.Get(ctx, synapseLookupKey, createdPostgresCluster)
					return err == nil
				}, timeout, interval).Should(BeTrue())

				By("Checking that the PostgresCluster's OwnerReference contains the Synapse instance")
				Expect(createdPostgresCluster.ObjectMeta.OwnerReferences).To(ContainElement(expectedOwnerReference))
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
						types.NamespacedName{Name: synapse.Spec.HomeserverConfigMapName, Namespace: SynapseNamespace},
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
	})

	Context("When creating an incorrect Synapse instance", func() {
		var synapse *synapsev1alpha1.Synapse

		BeforeEach(func() {
			By("Creating a Synapse instance which referes an absent ConfigMap")
			synapse = &synapsev1alpha1.Synapse{
				ObjectMeta: metav1.ObjectMeta{
					Name:      SynapseName,
					Namespace: SynapseNamespace,
				},
				Spec: synapsev1alpha1.SynapseSpec{
					HomeserverConfigMapName: ConfigMapName,
				},
			}
			Expect(k8sClient.Create(ctx, synapse)).Should(Succeed())
		})

		AfterEach(func() {
			By("Cleaning up Synapse CR")
			Expect(k8sClient.Delete(ctx, synapse)).Should(Succeed())
		})

		It("Should get in a failed state and not create child objects", func() {
			synapseLookupKey := types.NamespacedName{Name: SynapseName, Namespace: SynapseNamespace}

			By("Verifying that the Synapse object was created")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, synapseLookupKey, &synapsev1alpha1.Synapse{})
				return err == nil
			}, timeout, interval).Should(BeTrue())

			// By("Checking the Synapse status")
			//
			// expectedStatus := synapsev1alpha1.SynapseStatus{
			// 	State: "FAILED",
			// }
			// // Status may need some time to be updated
			// Eventually(func() synapsev1alpha1.SynapseStatus {
			// 	_ = k8sClient.Get(ctx, synapseLookupKey, synapse)
			// 	return synapse.Status
			// }, timeout, interval).Should(Equal(expectedStatus))

			By("Checking that the Synapse PVC, Deployment, and Service have not been created")
			Consistently(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, synapseLookupKey, &corev1.PersistentVolumeClaim{})).ShouldNot(Succeed())
				g.Expect(k8sClient.Get(ctx, synapseLookupKey, &appsv1.Deployment{})).ShouldNot(Succeed())
				g.Expect(k8sClient.Get(ctx, synapseLookupKey, &corev1.Service{})).ShouldNot(Succeed())
			}, timeout, interval).Should(Succeed())
		})
	})
})
