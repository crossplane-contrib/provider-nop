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
	"sort"
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

	if err := ctrl.NewWebhookManagedBy(mgr).
		For(&v1alpha1.NopResource{}).
		WithValidator(v1alpha1.NopResourceValidator).
		Complete(); err != nil {
		return errors.Wrap(err, "cannot set up webhooks")
	}

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

	// Sort conditions, with those that should occur latest appearing first.
	// We rely on the fact that the managed.Reconciler won't persist this sorted
	// array because it occurred during the Observe function, and we didn't
	// return ResourceLateInitialized: true.
	sort.SliceStable(nop.Spec.ForProvider.ConditionAfter, func(i, j int) bool {
		return nop.Spec.ForProvider.ConditionAfter[i].Time.Duration > nop.Spec.ForProvider.ConditionAfter[j].Time.Duration
	})

	age := time.Since(nop.ObjectMeta.CreationTimestamp.Time)
	set := map[xpv1.ConditionType]bool{}
	for _, ca := range nop.Spec.ForProvider.ConditionAfter {
		if ca.Time.Duration > age {
			// This condition should not occur yet.
			continue
		}

		if set[ca.ConditionType] {
			// We already encountered and set a condition of this type.
			continue
		}

		// This is the latest condition of this type that should be set.
		var r xpv1.ConditionReason
		if ca.ConditionReason != nil {
			r = *ca.ConditionReason
		}
		nop.SetConditions(xpv1.Condition{
			Type:               ca.ConditionType,
			Status:             ca.ConditionStatus,
			Reason:             r,
			LastTransitionTime: metav1.Now(),
		})

		set[ca.ConditionType] = true
	}

	// Emit any connection details we were asked to.
	cd := managed.ConnectionDetails{}
	for _, nv := range nop.Spec.ForProvider.ConnectionDetails {
		cd[nv.Name] = []byte(nv.Value)
	}

	// If our managed resource has not been deleted we report that our
	// pretend external resource exists and is up-to-date. This means
	// we'll never call the CreateFn or UpdateFn.
	return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true, ConnectionDetails: cd}, nil
}
