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

// RequeueAtDeadline wraps a managed reconciler so that a resource waiting out
// spec.forProvider.deleteAfter is reconciled again when it's due. The managed
// reconciler can't be told when that is, so it would otherwise requeue on
// backoff and finish the deletion late.
func RequeueAtDeadline(r reconcile.Reconciler, c client.Reader, newObj func() client.Object) reconcile.Reconciler {
	return reconcile.Func(func(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
		res, err := r.Reconcile(ctx, req)
		if err != nil || !res.Requeue {
			return res, err
		}

		obj := newObj()
		if err := c.Get(ctx, req.NamespacedName, obj); err != nil {
			return res, errors.Wrap(client.IgnoreNotFound(err), "cannot get managed resource")
		}

		if !meta.WasDeleted(obj) {
			return res, nil
		}

		nop, ok := parameters(obj)
		if !ok {
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
	})
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
