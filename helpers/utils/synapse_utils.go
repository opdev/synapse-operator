/*
Copyright 2021.

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

package utils

import (
	"context"

	"github.com/opdev/subreconciler"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

func GetResource(
	ctx context.Context,
	kubeClient client.Client,
	req ctrl.Request,
	resource client.Object,
) (*ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)

	if err := kubeClient.Get(ctx, req.NamespacedName, resource); err != nil {
		if k8serrors.IsNotFound(err) {
			// we'll ignore not-found errors, since they can't be fixed by an immediate
			// requeue (we'll need to wait for a new notification), and we can get them
			// on deleted requests.
			log.Error(
				err,
				"Cannot find resource - has it been deleted ?",
				"Name", resource.GetName(),
				"Namespace", resource.GetNamespace(),
			)
			return subreconciler.DoNotRequeue()
		}
		log.Error(
			err,
			"Error fetching resource",
			"Name", resource.GetName(),
			"Namespace", resource.GetNamespace(),
		)

		return subreconciler.RequeueWithError(err)
	}

	return subreconciler.ContinueReconciling()
}
