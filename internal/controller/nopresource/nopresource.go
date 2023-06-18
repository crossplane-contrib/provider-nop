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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	"github.com/crossplane-contrib/provider-nop/apis/v1alpha1"
)

// TODO(negz): Plumb up this logger and rate-limiter?

// Setup adds a controller that reconciles NopResource managed resources.
func Setup(mgr ctrl.Manager, _ logging.Logger, _ workqueue.RateLimiter) error {
	name := managed.ControllerName(v1alpha1.NopResourceGroupKind)

	r := NewReconciler(mgr)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&v1alpha1.NopResource{}).
		Complete(r)
}

// reconcileLogic returns a slice of indices from conditionAfter which states
// the condition for each type specified till the given timeElapsed duration.
func reconcileLogic(conditionAfter []v1alpha1.ResourceConditionAfter, timeElapsed time.Duration) []int {
	latestTime := make(map[string]time.Duration)
	latestIdx := make(map[string]int)

	for i := 0; i < len(conditionAfter); i++ {
		specTime, _ := time.ParseDuration(conditionAfter[i].Time)

		// For each ConditionType finds the latest time it was updated until the
		// elapsed time and the corresponding index of the same in conditionAfter.
		if timeElapsed >= specTime {
			lastChange, ok := latestTime[conditionAfter[i].ConditionType]
			if !ok || lastChange < specTime {
				latestTime[conditionAfter[i].ConditionType] = specTime
				latestIdx[conditionAfter[i].ConditionType] = i
			}
		}
	}

	idxs := make([]int, 0, len(latestIdx))
	for _, l := range latestIdx {
		idxs = append(idxs, l)
	}

	return idxs
}

// A Reconciler reconciles managed resources by creating and managing the
// lifecycle of an external resource, i.e. a resource in an external system such
// as a cloud provider API. Each controller must watch the managed resource kind
// for which it is responsible.
type Reconciler struct {
	client client.Client

	pollInterval time.Duration
	timeout      time.Duration

	log    logging.Logger
	record event.Recorder
}

const (
	reconcileGracePeriod = 30 * time.Second
	reconcileTimeout     = 1 * time.Minute
	defaultPollInterval  = 1 * time.Second
)

// NewReconciler builds a reconciler for managing NopResource.
func NewReconciler(m manager.Manager) *Reconciler {

	r := &Reconciler{
		client:       m.GetClient(),
		pollInterval: defaultPollInterval,
		timeout:      reconcileTimeout,
		log:          logging.NewNopLogger(),
		record:       event.NewNopRecorder(),
	}

	return r
}

const (
	errGetManaged          = "cannot get managed resource"
	errUpdateManagedStatus = "cannot update managed resource status"
)

// Reconcile a managed resource with an external resource.
func (r *Reconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {

	log := r.log.WithValues("request", req)
	log.Debug("Reconciling")

	ctx, cancel := context.WithTimeout(ctx, r.timeout+reconcileGracePeriod)
	defer cancel()

	managed := &v1alpha1.NopResource{}

	if err := r.client.Get(ctx, req.NamespacedName, managed); err != nil {
		// There's no need to requeue if we no longer exist. Otherwise we'll be
		// requeued implicitly because we return an error.
		log.Debug("Cannot get managed resource", "error", err)
		return reconcile.Result{}, errors.Wrap(resource.IgnoreNotFound(err), errGetManaged)
	}

	startTime := managed.CreationTimestamp

	ci := reconcileLogic(managed.Spec.ForProvider.ConditionAfter, time.Since(startTime.Time))

	for _, l := range ci {

		x := xpv1.Condition{
			Type:               xpv1.ConditionType(managed.Spec.ForProvider.ConditionAfter[l].ConditionType),
			Status:             v1.ConditionStatus(managed.Spec.ForProvider.ConditionAfter[l].ConditionStatus),
			LastTransitionTime: metav1.Now(),
			Reason:             xpv1.ReasonAvailable,
		}

		managed.Status.SetConditions(x)
	}

	log.Debug("Successfully requested update of external resource", "requeue-after", time.Now().Add(r.pollInterval))

	return reconcile.Result{RequeueAfter: r.pollInterval}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
}
