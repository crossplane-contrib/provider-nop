/*
Copyright 2026 The Crossplane Authors.

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

package nopprovider

import (
	"context"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"

	"github.com/crossplane-contrib/provider-nop/apis/v1alpha1"
)

// RequeueAtDeadline wraps a managed reconciler so that a resource is
// reconciled again when its pretend external resource is due to change. The
// managed reconciler can't be told when that is, so it would otherwise only
// look again when it next polls, or on backoff, and the change would land late.
//
// A resource waiting out spec.forProvider.deleteAfter is requeued when its
// deletion is due. A live resource is requeued when its conditions next change
// per spec.forProvider.conditionAfter, if that's sooner than the managed
// reconciler would have looked again anyway. So a condition is set when its
// time comes however long the poll interval is.
func RequeueAtDeadline(r reconcile.Reconciler, c client.Reader, newObj func() client.Object) reconcile.Reconciler {
	return reconcile.Func(func(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
		res, err := r.Reconcile(ctx, req)
		if err != nil {
			return res, err
		}

		obj := newObj()
		if err := c.Get(ctx, req.NamespacedName, obj); err != nil {
			return res, errors.Wrap(client.IgnoreNotFound(err), "cannot get managed resource")
		}

		nop, ok := parameters(obj)
		if !ok {
			return res, nil
		}

		if meta.WasDeleted(obj) {
			// After a successful Delete the managed reconciler asks for a
			// requeue on backoff. Anything else, like a failed Delete, keeps
			// its own result.
			if !res.Requeue {
				return res, nil
			}

			deadline, ok := deletionDeadline(nop, obj.GetDeletionTimestamp().Time)
			if !ok {
				return res, nil
			}

			if d := time.Until(deadline); d > 0 {
				return reconcile.Result{RequeueAfter: d}, nil
			}

			return res, nil
		}

		now := time.Now()
		next, ok := nextConditionChange(nop, obj.GetCreationTimestamp().Time, now)
		if !ok {
			return res, nil
		}

		d := next.Sub(now)

		// Only ever bring a reconcile forward. A requeue on backoff, or a poll
		// that comes first, already looks again in time.
		if res.Requeue || (res.RequeueAfter > 0 && res.RequeueAfter <= d) {
			return res, nil
		}

		return reconcile.Result{RequeueAfter: d}, nil
	})
}

// nextConditionChange returns when a live resource's conditions next change
// per conditionAfter, or false if none are still to come. Observe measures
// the same times from the creation timestamp, so the two agree.
func nextConditionChange(nop v1alpha1.NopParameters, created, now time.Time) (time.Time, bool) {
	var next time.Time

	for _, ca := range nop.ConditionAfter {
		at := created.Add(ca.Time.Duration)
		if !at.After(now) {
			continue
		}

		if next.IsZero() || at.Before(next) {
			next = at
		}
	}

	return next, !next.IsZero()
}

// deletionDeadline returns when a deleted resource's pretend external resource
// goes away, or false if its deletion isn't paced by deleteAfter.
func deletionDeadline(nop v1alpha1.NopParameters, deleted time.Time) (time.Time, bool) {
	if nop.DeleteError != nil || nop.DeleteAfter == nil {
		return time.Time{}, false
	}

	return deleted.Add(nop.DeleteAfter.Duration), true
}

func parameters(obj client.Object) (v1alpha1.NopParameters, bool) {
	switch nop := obj.(type) {
	case *v1alpha1.NopResource:
		return nop.Spec.ForProvider, true
	case *v1alpha1.ClusterNopResource:
		return nop.Spec.ForProvider, true
	default:
		return v1alpha1.NopParameters{}, false
	}
}
