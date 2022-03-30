package synapse

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	synapsev1alpha1 "github.com/opdev/synapse-operator/apis/synapse/v1alpha1"
)

type createResourceFunc func(s *synapsev1alpha1.Synapse, objectMeta metav1.ObjectMeta) (client.Object, error)

func setObjectMeta(name string, namespace string, labels map[string]string) metav1.ObjectMeta {
	objectMeta := metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
		Labels:    labels,
	}
	return objectMeta
}

func (r *SynapseReconciler) reconcileResource(
	ctx context.Context,
	createResource createResourceFunc,
	s *synapsev1alpha1.Synapse,
	resource client.Object,
	objectMeta metav1.ObjectMeta) error {

	log := ctrllog.FromContext(ctx)
	log.Info(
		"Reconciling resource",
		"Kind", resource.GetObjectKind(),
		"Name", objectMeta.Name,
		"Namespace", objectMeta.Namespace,
	)

	if err := r.Client.Get(ctx, types.NamespacedName{Name: objectMeta.Name, Namespace: objectMeta.Namespace}, resource); err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info(
				"Creating a new resource for Synapse",
				"Kind", resource.GetObjectKind(),
				"Name", objectMeta.Name,
				"Namespace", objectMeta.Namespace,
			)

			resource, err := createResource(s, objectMeta)
			if err != nil {
				log.Error(
					err,
					"Failed to generate a new resource for Synapse",
					"Kind", resource.GetObjectKind(),
					"Name", objectMeta.Name,
					"Namespace", objectMeta.Namespace,
				)
				return err
			}

			err = r.Client.Create(ctx, resource)
			if err != nil {
				log.Error(
					err,
					"Failed to create a new resource for Synapse",
					"Kind", resource.GetObjectKind(),
					"Name", objectMeta.Name,
					"Namespace", objectMeta.Namespace,
				)
				return err
			}

			return nil
		}

		log.Error(
			err,
			"Error reading resource",
			"Kind", resource.GetObjectKind(),
			"Name", objectMeta.Name,
			"Namespace", objectMeta.Namespace,
		)
		return err
	}

	log.Info(
		"Reconciling resource",
		"Kind", resource.GetObjectKind(),
		"Name", objectMeta.Name,
		"Namespace", objectMeta.Namespace,
	)
	return nil
}
