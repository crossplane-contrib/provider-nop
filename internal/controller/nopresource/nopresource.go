/*
Copyright 2020 The Crossplane Authors.

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

// Package nopresource is a controller for a managed resource that does nothing.
package nopresource

import (
	"context"
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	"github.com/crossplane-contrib/provider-nop/apis/v1alpha1"
)

// Setup adds a controller that reconciles NopResource managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.NopResourceGroupKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.NopResourceGroupVersionKind),
		managed.WithPollInterval(o.PollInterval),
		managed.WithExternalConnecter(&connecter{}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.NopResource{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

type connecter struct{}

func (c *connecter) Connect(_ context.Context, _ resource.Managed) (managed.ExternalClient, error) {
	return managed.ExternalClientFns{ObserveFn: Observe}, nil
}

// Observe doesn't actually observe an external resource. Instead it sets the
// most recent conditions that should occur per spec.forProvider.conditionAfter.
func Observe(_ context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	// If our managed resource has been deleted we need to report that
	// our pretend external resource is gone in order for the delete
	// process to complete. This means we'll never call the DeleteFn.
	if meta.WasDeleted(mg) {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	nop, ok := mg.(*v1alpha1.NopResource)
	if !ok {
		return managed.ExternalObservation{}, errors.Errorf("managed resource was not a %T", &v1alpha1.NopResource{})
	}

	pending := map[xpv1.ConditionType]int{}
	for i, ca := range nop.Spec.ForProvider.ConditionAfter {
		if time.Since(nop.ObjectMeta.CreationTimestamp.Time) < ca.Time.Duration {
			// This condition should not occur yet.
			continue
		}

		// No condition of this type is pending, so it should be.
		idx, ok := pending[ca.ConditionType]
		if !ok {
			pending[ca.ConditionType] = i
			continue
		}

		// This condition should occur after the pending condition of
		// the same type, so replace it as the pending condition.
		if nop.Spec.ForProvider.ConditionAfter[idx].Time.Duration < ca.Time.Duration {
			pending[ca.ConditionType] = i
		}
	}

	// Set our pending conditions.
	for _, idx := range pending {
		nop.SetConditions(xpv1.Condition{
			Type:               nop.Spec.ForProvider.ConditionAfter[idx].ConditionType,
			Status:             nop.Spec.ForProvider.ConditionAfter[idx].ConditionStatus,
			LastTransitionTime: metav1.Now(),
		})
	}

	// If our managed resource has not been deleted we report that our
	// pretend external resource exists and is up-to-date. This means
	// we'll never call the CreateFn or UpdateFn.
	return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, nil
}
