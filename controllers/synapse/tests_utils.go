package synapse

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2" //lint:ignore ST1001 Ginkgo and gomega are usually dot-imported
	. "github.com/onsi/gomega"    //lint:ignore ST1001 Ginkgo and gomega are usually dot-imported
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

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

func deleteResourceFunc(
	k8sClient client.Client,
	ctx context.Context,
	timeout time.Duration,
	interval time.Duration,
) func(client.Object, types.NamespacedName, bool) {
	return func(resource client.Object, lookupKey types.NamespacedName, removeFinalizers bool) {
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
}

func checkSubresourceAbsenceFunc(
	SynapseName string,
	SynapseNamespace string,
	k8sClient client.Client,
	ctx context.Context,
	timeout time.Duration,
	interval time.Duration,
) func(string) {
	// Verify the absence of Synapse sub-resources
	// This function common to multiple tests
	return func(expectedReason string) {
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
}

func checkResourcePresenceFunc(
	k8sClient client.Client,
	ctx context.Context,
	timeout time.Duration,
	interval time.Duration,
) func(client.Object, types.NamespacedName, metav1.OwnerReference) {
	return func(resource client.Object, lookupKey types.NamespacedName, expectedOwnerReference metav1.OwnerReference) {
		Eventually(func() bool {
			err := k8sClient.Get(ctx, lookupKey, resource)
			return err == nil
		}, timeout, interval).Should(BeTrue())

		Expect(resource.GetOwnerReferences()).To(ContainElement(expectedOwnerReference))
	}

}
