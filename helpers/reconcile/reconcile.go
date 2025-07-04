package reconcile

import (
	"context"

	"dario.cat/mergo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

func SetObjectMeta(name string, namespace string, labels map[string]string) metav1.ObjectMeta {
	objectMeta := metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
		Labels:    labels,
	}
	return objectMeta
}

// Generic function to reconcile a Kubernetes resource
// `current` should be an empty resource (e.g. &appsv1.Deployment{}). It is
// populated by the actual current state of the resource in the initial GET
// request.
func ReconcileResource(
	ctx context.Context,
	rclient client.Client,
	desired client.Object,
	current client.Object,
) error {
	log := ctrllog.FromContext(ctx)
	log.Info(
		"Reconciling child resource",
		"Kind", desired.GetObjectKind().GroupVersionKind().Kind,
		"Name", desired.GetName(),
		"Namespace", desired.GetNamespace(),
	)

	key := types.NamespacedName{Name: desired.GetName(), Namespace: desired.GetNamespace()}
	if err := rclient.Get(ctx, key, current); err != nil {
		if !k8serrors.IsNotFound(err) {
			log.Error(
				err,
				"Error reading child resource",
				"Kind", desired.GetObjectKind().GroupVersionKind().Kind,
				"Name", desired.GetName(),
				"Namespace", desired.GetNamespace(),
			)
			return err
		}

		log.Info(
			"Creating a new child resource",
			"Kind", desired.GetObjectKind().GroupVersionKind().Kind,
			"Name", desired.GetName(),
			"Namespace", desired.GetNamespace(),
		)

		err = rclient.Create(ctx, desired)
		if err != nil {
			log.Error(
				err,
				"Failed to create a new child resource",
				"Kind", desired.GetObjectKind().GroupVersionKind().Kind,
				"Name", desired.GetName(),
				"Namespace", desired.GetNamespace(),
			)
			return err
		}
	} else {
		log.Info(
			"Patching existing child resource",
			"Kind", desired.GetObjectKind().GroupVersionKind().Kind,
			"Name", desired.GetName(),
			"Namespace", desired.GetNamespace(),
		)

		// This ensures that the resource is patched only if there is a
		// difference between desired and current.
		patchDiff := client.MergeFrom(current.DeepCopyObject().(client.Object))
		if err := mergo.Merge(current, desired, mergo.WithOverride); err != nil {
			log.Error(err, "Error in merge")
			return err
		}

		if err := rclient.Patch(ctx, current, patchDiff); err != nil {
			log.Error(
				err,
				"Failed to patch child resource",
				"Kind", desired.GetObjectKind().GroupVersionKind().Kind,
				"Name", desired.GetName(),
				"Namespace", desired.GetNamespace(),
			)
			return err
		}
	}

	log.Info(
		"Finished reconciling resource",
		"Kind", desired.GetObjectKind().GroupVersionKind().Kind,
		"Name", desired.GetName(),
		"Namespace", desired.GetNamespace(),
	)
	return nil
}
