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

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/conditions"
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/crossplane/crossplane-runtime/v2/pkg/statemetrics"

	"github.com/crossplane-contrib/provider-nop/apis/v1alpha1"
	"github.com/crossplane-contrib/provider-nop/internal/nopprovider"
)

// SetupGated registers Setup with the Gate, waiting for the NopResource GKV.
func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	o.Gate.Register(func() {
		if err := Setup(mgr, o); err != nil {
			panic(err)
		}
	}, v1alpha1.NopResourceGroupVersionKind)
	return nil
}

// Setup adds a controller that reconciles NopResource managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.NopResourceGroupKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.NopResourceGroupVersionKind),
		managed.WithPollInterval(o.PollInterval),
		managed.WithExternalConnector(&connector{}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithMetricRecorder(o.MetricOptions.MRMetrics),
	)

	if err := ctrl.NewWebhookManagedBy(mgr).
		For(&v1alpha1.NopResource{}).
		WithValidator(v1alpha1.NopResourceValidator).
		Complete(); err != nil {
		return errors.Wrap(err, "cannot set up webhooks")
	}

	if err := mgr.Add(statemetrics.NewMRStateRecorder(
		mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &v1alpha1.NopResourceList{}, o.MetricOptions.PollStateMetricInterval)); err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.NopResource{}).
		Complete(ratelimiter.NewReconciler(name,
			nopprovider.RequeueAtDeadline(r, mgr.GetClient(), func() client.Object { return &v1alpha1.NopResource{} }),
			o.GlobalRateLimiter))
}

type connector struct{}

func (c *connector) Connect(_ context.Context, _ resource.Managed) (managed.ExternalClient, error) {
	return managed.ExternalClientFns{
		ObserveFn: Observe,
		DeleteFn:  Delete,
		DisconnectFn: func(_ context.Context) error {
			return nil
		},
	}, nil
}

// Observe doesn't actually observe an external resource. Instead, it sets the
// most recent conditions that should occur per spec.forProvider.conditionAfter.
func Observe(_ context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	nop, ok := mg.(*v1alpha1.NopResource)
	if !ok {
		return managed.ExternalObservation{}, errors.Errorf("managed resource was not a %T", &v1alpha1.NopResource{})
	}

	status := conditions.ObservedGenerationPropagationManager{}.For(nop)

	// Our pretend external resource takes at least
	// spec.forProvider.deleteAfter to go away, and never goes away at all if
	// spec.forProvider.deleteError is set. With neither it's gone at once.
	if meta.WasDeleted(mg) {
		return nopprovider.ObserveDeleted(nop.Spec.ForProvider, nop.DeletionTimestamp.Time, status), nil
	}

	age := time.Since(nop.CreationTimestamp.Time)

	return nopprovider.Observe(nop.Spec.ForProvider, age, status)
}

// Delete doesn't actually delete an external resource. It fails if
// spec.forProvider.deleteError is set, which is how a NopResource pretends to
// be a resource that cannot be deleted.
func Delete(_ context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	nop, ok := mg.(*v1alpha1.NopResource)
	if !ok {
		return managed.ExternalDelete{}, errors.Errorf("managed resource was not a %T", &v1alpha1.NopResource{})
	}

	return managed.ExternalDelete{}, nopprovider.Delete(nop.Spec.ForProvider)
}
