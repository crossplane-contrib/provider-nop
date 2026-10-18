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
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"
	"github.com/crossplane/crossplane-runtime/v2/pkg/test"

	"github.com/crossplane-contrib/provider-nop/apis/v1alpha1"
)

func TestRequeueAtDeadline(t *testing.T) {
	errBoom := errors.New("boom")
	after := func(d time.Duration) *metav1.Duration { return &metav1.Duration{Duration: d} }
	msg := func(s string) *string { return &s }

	now := time.Now()
	deletedAgo := func(d time.Duration) metav1.ObjectMeta {
		ts := metav1.NewTime(now.Add(-d))
		return metav1.ObjectMeta{DeletionTimestamp: &ts}
	}
	createdAgo := func(d time.Duration) metav1.ObjectMeta {
		return metav1.ObjectMeta{CreationTimestamp: metav1.NewTime(now.Add(-d))}
	}

	// readyAfter returns conditionAfter entries that report Ready False at
	// once, then Ready True after d, the way tests usually configure them.
	readyAfter := func(d time.Duration) []v1alpha1.ConditionAfter {
		return []v1alpha1.ConditionAfter{
			{Time: metav1.Duration{}, ConditionType: xpv1.TypeReady, ConditionStatus: corev1.ConditionFalse},
			{Time: metav1.Duration{Duration: d}, ConditionType: xpv1.TypeReady, ConditionStatus: corev1.ConditionTrue},
		}
	}

	// get returns a Get that fills in nop, as the cache would.
	get := func(nop client.Object) test.MockGetFn {
		return test.NewMockGetFn(nil, func(o client.Object) error {
			switch want := nop.(type) {
			case *v1alpha1.NopResource:
				want.DeepCopyInto(o.(*v1alpha1.NopResource))
			case *v1alpha1.ClusterNopResource:
				want.DeepCopyInto(o.(*v1alpha1.ClusterNopResource))
			}
			return nil
		})
	}

	type args struct {
		res    reconcile.Result
		err    error
		get    test.MockGetFn
		newObj func() client.Object
	}

	type want struct {
		res reconcile.Result
		err error
	}

	newNop := func() client.Object { return &v1alpha1.NopResource{} }

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"ReconcileError": {
			reason: "An error from the managed reconciler is returned as is.",
			args: args{
				res:    reconcile.Result{Requeue: true},
				err:    errBoom,
				newObj: newNop,
			},
			want: want{res: reconcile.Result{Requeue: true}, err: errBoom},
		},
		"PollNothingDue": {
			reason: "A poll for a live resource with no conditions still to come is returned as is.",
			args: args{
				res:    reconcile.Result{RequeueAfter: time.Minute},
				get:    get(&v1alpha1.NopResource{ObjectMeta: createdAgo(time.Second)}),
				newObj: newNop,
			},
			want: want{res: reconcile.Result{RequeueAfter: time.Minute}},
		},
		"ConditionPending": {
			reason: "A live resource whose conditions change before the next poll is requeued when they're due.",
			args: args{
				res: reconcile.Result{RequeueAfter: 10 * time.Second},
				get: get(&v1alpha1.NopResource{
					ObjectMeta: createdAgo(time.Second),
					Spec:       v1alpha1.NopSpec{ForProvider: v1alpha1.NopParameters{ConditionAfter: readyAfter(3 * time.Second)}},
				}),
				newObj: newNop,
			},
			want: want{res: reconcile.Result{RequeueAfter: 2 * time.Second}},
		},
		"ClusterConditionPending": {
			reason: "A live ClusterNopResource whose conditions change before the next poll is requeued when they're due.",
			args: args{
				res: reconcile.Result{RequeueAfter: 10 * time.Second},
				get: get(&v1alpha1.ClusterNopResource{
					ObjectMeta: createdAgo(time.Second),
					Spec:       v1alpha1.NopSpec{ForProvider: v1alpha1.NopParameters{ConditionAfter: readyAfter(3 * time.Second)}},
				}),
				newObj: func() client.Object { return &v1alpha1.ClusterNopResource{} },
			},
			want: want{res: reconcile.Result{RequeueAfter: 2 * time.Second}},
		},
		"NextOfSeveral": {
			reason: "With several conditions still to come, the resource is requeued for the earliest.",
			args: args{
				res: reconcile.Result{RequeueAfter: time.Minute},
				get: get(&v1alpha1.NopResource{
					ObjectMeta: createdAgo(5 * time.Second),
					Spec: v1alpha1.NopSpec{ForProvider: v1alpha1.NopParameters{ConditionAfter: []v1alpha1.ConditionAfter{
						{Time: metav1.Duration{Duration: 20 * time.Second}, ConditionType: xpv1.TypeReady, ConditionStatus: corev1.ConditionFalse},
						{Time: metav1.Duration{}, ConditionType: xpv1.TypeReady, ConditionStatus: corev1.ConditionFalse},
						{Time: metav1.Duration{Duration: 8 * time.Second}, ConditionType: xpv1.TypeReady, ConditionStatus: corev1.ConditionTrue},
						{Time: metav1.Duration{Duration: 3 * time.Second}, ConditionType: xpv1.TypeSynced, ConditionStatus: corev1.ConditionTrue},
					}}},
				}),
				newObj: newNop,
			},
			want: want{res: reconcile.Result{RequeueAfter: 3 * time.Second}},
		},
		"ConditionsPast": {
			reason: "Once every condition has had its time the managed reconciler's poll is kept.",
			args: args{
				res: reconcile.Result{RequeueAfter: 10 * time.Second},
				get: get(&v1alpha1.NopResource{
					ObjectMeta: createdAgo(time.Minute),
					Spec:       v1alpha1.NopSpec{ForProvider: v1alpha1.NopParameters{ConditionAfter: readyAfter(3 * time.Second)}},
				}),
				newObj: newNop,
			},
			want: want{res: reconcile.Result{RequeueAfter: 10 * time.Second}},
		},
		"PollFirst": {
			reason: "A poll that comes before the next condition change already looks again in time, so it's kept.",
			args: args{
				res: reconcile.Result{RequeueAfter: 10 * time.Second},
				get: get(&v1alpha1.NopResource{
					ObjectMeta: createdAgo(time.Second),
					Spec:       v1alpha1.NopSpec{ForProvider: v1alpha1.NopParameters{ConditionAfter: readyAfter(time.Minute)}},
				}),
				newObj: newNop,
			},
			want: want{res: reconcile.Result{RequeueAfter: 10 * time.Second}},
		},
		"BackoffFirst": {
			reason: "A requeue on backoff, for example after a failed write, comes first, so it's kept.",
			args: args{
				res: reconcile.Result{Requeue: true},
				get: get(&v1alpha1.NopResource{
					ObjectMeta: createdAgo(time.Second),
					Spec:       v1alpha1.NopSpec{ForProvider: v1alpha1.NopParameters{ConditionAfter: readyAfter(3 * time.Second)}},
				}),
				newObj: newNop,
			},
			want: want{res: reconcile.Result{Requeue: true}},
		},
		"GetError": {
			reason: "An error getting the resource is returned, which requeues it on backoff.",
			args: args{
				res:    reconcile.Result{Requeue: true},
				get:    test.NewMockGetFn(errBoom),
				newObj: newNop,
			},
			want: want{res: reconcile.Result{Requeue: true}, err: errBoom},
		},
		"NotFound": {
			reason: "A resource that's already gone keeps the managed reconciler's requeue.",
			args: args{
				res:    reconcile.Result{Requeue: true},
				get:    test.NewMockGetFn(kerrors.NewNotFound(schema.GroupResource{}, "")),
				newObj: newNop,
			},
			want: want{res: reconcile.Result{Requeue: true}},
		},
		"NotDeleted": {
			reason: "A resource that isn't being deleted keeps the managed reconciler's requeue.",
			args: args{
				res: reconcile.Result{Requeue: true},
				get: get(&v1alpha1.NopResource{
					Spec: v1alpha1.NopSpec{ForProvider: v1alpha1.NopParameters{DeleteAfter: after(30 * time.Second)}},
				}),
				newObj: newNop,
			},
			want: want{res: reconcile.Result{Requeue: true}},
		},
		"DeleteError": {
			reason: "A resource that can't be deleted keeps retrying on backoff, even with deleteAfter set.",
			args: args{
				res: reconcile.Result{Requeue: true},
				get: get(&v1alpha1.NopResource{
					ObjectMeta: deletedAgo(10 * time.Second),
					Spec: v1alpha1.NopSpec{ForProvider: v1alpha1.NopParameters{
						DeleteAfter: after(30 * time.Second),
						DeleteError: msg("boom"),
					}},
				}),
				newObj: newNop,
			},
			want: want{res: reconcile.Result{Requeue: true}},
		},
		"DeletePending": {
			reason: "A resource waiting out deleteAfter is requeued when it's due.",
			args: args{
				res: reconcile.Result{Requeue: true},
				get: get(&v1alpha1.NopResource{
					ObjectMeta: deletedAgo(10 * time.Second),
					Spec:       v1alpha1.NopSpec{ForProvider: v1alpha1.NopParameters{DeleteAfter: after(30 * time.Second)}},
				}),
				newObj: newNop,
			},
			want: want{res: reconcile.Result{RequeueAfter: 20 * time.Second}},
		},
		"ClusterDeletePending": {
			reason: "A ClusterNopResource waiting out deleteAfter is requeued when it's due.",
			args: args{
				res: reconcile.Result{Requeue: true},
				get: get(&v1alpha1.ClusterNopResource{
					ObjectMeta: deletedAgo(10 * time.Second),
					Spec:       v1alpha1.NopSpec{ForProvider: v1alpha1.NopParameters{DeleteAfter: after(30 * time.Second)}},
				}),
				newObj: func() client.Object { return &v1alpha1.ClusterNopResource{} },
			},
			want: want{res: reconcile.Result{RequeueAfter: 20 * time.Second}},
		},
		"DeadlinePassed": {
			reason: "Once deleteAfter has passed the managed reconciler's requeue is kept.",
			args: args{
				res: reconcile.Result{Requeue: true},
				get: get(&v1alpha1.NopResource{
					ObjectMeta: deletedAgo(time.Minute),
					Spec:       v1alpha1.NopSpec{ForProvider: v1alpha1.NopParameters{DeleteAfter: after(30 * time.Second)}},
				}),
				newObj: newNop,
			},
			want: want{res: reconcile.Result{Requeue: true}},
		},
	}

	// RequeueAfter is computed from the wall clock, so allow for the time the
	// test takes to run.
	approx := cmp.Comparer(func(a, b time.Duration) bool {
		d := a - b
		return d > -time.Second && d < time.Second
	})

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			inner := reconcile.Func(func(_ context.Context, _ reconcile.Request) (reconcile.Result, error) {
				return tc.args.res, tc.args.err
			})
			c := &test.MockClient{MockGet: tc.args.get}

			got, err := RequeueAtDeadline(inner, c, tc.args.newObj).Reconcile(context.Background(), reconcile.Request{})
			if diff := cmp.Diff(tc.want.err, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("\n%s\nReconcile(...): -want error, +got error:\n%s", tc.reason, diff)
			}
			if diff := cmp.Diff(tc.want.res, got, approx); diff != "" {
				t.Errorf("\n%s\nReconcile(...): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}
