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

package nopresource

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"

	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/provider-nop/apis/sample/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	errNotNopResource = "managed resource is not a NopResource custom resource"
)

// Setup adds a controller that reconciles NopResource managed resources.
func Setup(mgr ctrl.Manager, l logging.Logger, rl workqueue.RateLimiter) error {
	name := managed.ControllerName(v1alpha1.NopResourceGroupKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.NopResourceGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube: mgr.GetClient(),
		}),
		managed.WithInitializers(managed.NewNameAsExternalName(mgr.GetClient())),
		managed.WithPollInterval(1*time.Second),
		managed.WithLogger(l.WithValues("controller", name)),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&v1alpha1.NopResource{}).
		Complete(r)
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube client.Client
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.NopResource)
	if !ok {
		return nil, errors.New(errNotNopResource)
	}

	// fmt.Printf(string(cr.GetCondition(xpv1.TypeReady).Status) + "\n\n")

	return &external{}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	// A 'client' used to connect to the external resource API. In practice this
	// would be something like an AWS SDK client.
	service interface{}
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.NopResource)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotNopResource)
	}
	startTime := cr.CreationTimestamp

	// If object was deleted, return it does not exist so that managed reconciler removes finalizer
	if meta.WasDeleted(mg) {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	ci := reconcileLogic(cr.Spec.ForProvider.ConditionAfter, time.Since(startTime.Time))

	for _, l := range ci {
		// fmt.Printf("Calling update on index %d\n", l)

		x := xpv1.Condition{
			Type:               xpv1.ConditionType(cr.Spec.ForProvider.ConditionAfter[l].ConditionType),
			Status:             v1.ConditionStatus(cr.Spec.ForProvider.ConditionAfter[l].ConditionStatus),
			LastTransitionTime: metav1.Now(),
			Reason:             xpv1.ReasonAvailable,
		}

		cr.Status.SetConditions(x)
	}

	// x := cr.Status.Conditions
	// fmt.Printf("\n\n	My values	\n\n")
	// fmt.Printf("%v", ci)
	// fmt.Printf(time.Since(startTime.Time).String() + "\n\n")
	// for _, e := range x {
	// 	fmt.Printf("%s %s %s %s", string(e.Reason), string(e.Message), string(e.Status), string(e.Type))
	// 	fmt.Print("\n\n")
	// }
	// These fmt statements should be removed in the real implementation.
	// fmt.Printf("Observing: %+v", cr)

	return managed.ExternalObservation{
		// Return false when the external resource does not exist. This lets
		// the managed resource reconciler know that it needs to call Create to
		// (re)create the resource, or that it has successfully been deleted.
		ResourceExists: true,

		// Return false when the external resource exists, but it not up to date
		// with the desired managed resource state. This lets the managed
		// resource reconciler know that it needs to call Update.
		ResourceUpToDate: true,

		// Return any details that may be required to connect to the external
		// resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.NopResource)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotNopResource)
	}

	fmt.Printf("Creating: %+v", cr)

	return managed.ExternalCreation{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.NopResource)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotNopResource)
	}

	fmt.Printf("Updating: %+v", cr)

	return managed.ExternalUpdate{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.NopResource)
	if !ok {
		return errors.New(errNotNopResource)
	}

	fmt.Printf("Deleting: %+v", cr)

	cr.Status.SetConditions(xpv1.Deleting())

	return nil
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

	var idx []int
	for _, l := range latestIdx {
		idx = append(idx, l)
	}

	return idx
}
