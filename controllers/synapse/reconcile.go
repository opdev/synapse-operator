package synapse

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	synapsev1alpha1 "github.com/opdev/synapse-operator/apis/synapse/v1alpha1"
)

type createResourceFunc func(s *synapsev1alpha1.Synapse, objectMeta metav1.ObjectMeta) client.Object

func setObjectMeta(name string, namespace string, labels map[string]string) metav1.ObjectMeta {
	objectMeta := metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
		Labels:    labels,
	}
	return objectMeta
}

func (r *SynapseReconciler) reconcileResource(
	createResource createResourceFunc,
	s *synapsev1alpha1.Synapse,
	resource client.Object,
	objectMeta metav1.ObjectMeta) error {

	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: objectMeta.Name, Namespace: objectMeta.Namespace}, resource)
	fmt.Printf("%v - Getting sample resource error boolean: %v", time.Now(), errors.IsNotFound(err))
	if err != nil {
		if errors.IsNotFound(err) {
			fmt.Printf("%v - reconcileResource: creating a new resource for Synapse", time.Now())

			resource := createResource(s, objectMeta)
			err = r.Client.Create(context.TODO(), resource)

			if err != nil {
				fmt.Printf("%v - reconcileResource: failed to create new resource, err: %v", time.Now(), err)
				return err
			}

			return nil
		}

		fmt.Printf("%v - reconcileResource: error reading resource, err: %v", time.Now(), err)
		return err
	}

	return nil
}
