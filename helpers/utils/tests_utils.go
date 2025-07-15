package utils

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck // ST1001 Ginkgo and gomega are usually dot-imported
	. "github.com/onsi/gomega"    //nolint:staticcheck // ST1001 Ginkgo and gomega are usually dot-imported
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/opdev/synapse-operator/api/synapse/v1alpha1"
)

// Helper function for struct construction requiring a boolean pointer
func BoolAddr(b bool) *bool {
	boolVar := b
	return &boolVar
}

func Convert(i any) any {
	switch x := i.(type) {
	case map[any]any:
		m2 := map[string]any{}
		for k, v := range x {
			m2[k.(string)] = Convert(v)
		}
		return m2
	case []any:
		for i, v := range x {
			x[i] = Convert(v)
		}
	}
	return i
}

func DeleteResourceFunc(
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

func CheckSubresourceAbsenceFunc(
	k8sClient client.Client,
	ctx context.Context,
	timeout time.Duration,
	interval time.Duration,
) func(types.NamespacedName, ...client.Object) {
	return func(lookupKey types.NamespacedName, subResources ...client.Object) {
		By("Checking sub-resources have not been created")
		Consistently(func(g Gomega) {
			for _, resource := range subResources {
				g.Expect(k8sClient.Get(ctx, lookupKey, resource)).ShouldNot(Succeed())
			}
		}, timeout, interval).Should(Succeed())
	}
}

func CheckStatusFunc(
	k8sClient client.Client,
	ctx context.Context,
	timeout time.Duration,
	interval time.Duration,
) func(string, string, types.NamespacedName, client.Object) {
	return func(expectedState string, expectedReason string, lookupKey types.NamespacedName, object client.Object) {
		By("Checking the Status State and Reason")
		// Status may need some time to be updated
		Eventually(func(g Gomega) {
			var currentState, currentReason string

			switch resource := object.(type) {
			case *v1alpha1.Synapse:
				g.Expect(k8sClient.Get(ctx, lookupKey, resource)).Should(Succeed())
				currentState = resource.Status.State
				currentReason = resource.Status.Reason
			case *v1alpha1.Heisenbridge:
				g.Expect(k8sClient.Get(ctx, lookupKey, resource)).Should(Succeed())
				currentState = resource.Status.State
				currentReason = resource.Status.Reason
			case *v1alpha1.MautrixSignal:
				g.Expect(k8sClient.Get(ctx, lookupKey, resource)).Should(Succeed())
				currentState = resource.Status.State
				currentReason = resource.Status.Reason
			}
			g.Expect(currentState).To(Equal(expectedState))
			g.Expect(currentReason).To(Equal(expectedReason))
		}, timeout, interval).Should(Succeed())
	}
}

func CheckResourcePresenceFunc(
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
