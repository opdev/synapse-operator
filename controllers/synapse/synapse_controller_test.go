package synapse

import (
	"context"
	"path/filepath"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

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
			CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
			ErrorIfCRDPathMissing: true,
		}

		cfg, err := testEnv.Start()
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg).NotTo(BeNil())

		err = synapsev1alpha1.AddToScheme(scheme.Scheme)
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

	Context("When creating a Synapse instance", func() {
		var synapse *synapsev1alpha1.Synapse
		var configMap *corev1.ConfigMap
		var createdPVC *corev1.PersistentVolumeClaim
		var createdDeployment *appsv1.Deployment
		var createdService *corev1.Service
		var synapseLookupKey types.NamespacedName

		BeforeEach(func() {
			// Init variables
			synapseLookupKey = types.NamespacedName{Name: SynapseName, Namespace: SynapseNamespace}
			createdPVC = &corev1.PersistentVolumeClaim{}
			createdDeployment = &appsv1.Deployment{}
			createdService = &corev1.Service{}

			By("Creating a ConfigMap containing a basic homeserver.yaml")
			// Populate the ConfigMap with the minimum data needed
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
					HomeserverConfigMapName: ConfigMapName,
				},
			}
			Expect(k8sClient.Create(ctx, synapse)).Should(Succeed())
		})

		AfterEach(func() {
			// Cleanup
			By("Cleaning up ConfigMap")
			Expect(k8sClient.Delete(ctx, configMap)).Should(Succeed())

			By("Cleaning up Synapse CR")
			Expect(k8sClient.Delete(ctx, synapse)).Should(Succeed())

			// Child resources must be manually deleted as the controllers responsible of
			// their lifecycle are not running.
			By("Cleaning up Synapse PVC")
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
			Expect(k8sClient.Delete(ctx, createdDeployment)).Should(Succeed())

			// Check Deployment was successfully removed
			Eventually(func() bool {
				err := k8sClient.Get(ctx, synapseLookupKey, &appsv1.Deployment{})
				return err == nil
			}, timeout, interval).Should(BeFalse())

			By("Cleaning up Synapse Service")
			// Deleting Service
			Expect(k8sClient.Delete(ctx, createdService)).Should(Succeed())

			// Check Service was successfully removed
			Eventually(func() bool {
				err := k8sClient.Get(ctx, synapseLookupKey, &corev1.Service{})
				return err == nil
			}, timeout, interval).Should(BeFalse())
		})

		It("Should create a PVC, a Deployment and a Service", func() {
			By("Verifying that the Synapse object was created")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, synapseLookupKey, synapse)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Checking the Synapse Status")
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

			By("Checking that a Synapse PVC exists")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, synapseLookupKey, createdPVC)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Checking that the PVC's OwnerReference contains the Synapse instance")
			expectedOwnerReference := metav1.OwnerReference{
				Kind:               "Synapse",
				APIVersion:         "synapse.opdev.io/v1alpha1",
				UID:                synapse.GetUID(),
				Name:               SynapseName,
				Controller:         BoolAddr(true),
				BlockOwnerDeletion: BoolAddr(true),
			}
			Expect(createdPVC.ObjectMeta.OwnerReferences).To(ContainElement(expectedOwnerReference))

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

			By("Checking that a Synapse Service exists")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, synapseLookupKey, createdService)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Checking that the Service's OwnerReference contains the Synapse instance")
			Expect(createdService.ObjectMeta.OwnerReferences).To(ContainElement(expectedOwnerReference))
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
