/*
Copyright 2025 The Crossplane Authors.

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

package clusternopresource

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource/fake"
	"github.com/crossplane/crossplane-runtime/v2/pkg/test"

	"github.com/crossplane-contrib/provider-nop/apis/v1alpha1"
)

// Unlike many Kubernetes projects Crossplane does not use third party testing
// libraries, per the common Go test review comments. Crossplane encourages the
// use of table driven unit tests. The tests of the crossplane-runtime project
// are representative of the testing style Crossplane encourages.
//
// https://github.com/golang/go/wiki/TestComments
// https://github.com/crossplane/crossplane/blob/master/CONTRIBUTING.md#contributing-code

func TestReconcileLogic(t *testing.T) {
	c := []v1alpha1.ConditionAfter{
		{Time: metav1.Duration{Duration: 10 * time.Second}, ConditionType: xpv1.TypeReady, ConditionStatus: corev1.ConditionFalse},
		{Time: metav1.Duration{Duration: 5 * time.Second}, ConditionType: xpv1.TypeReady, ConditionStatus: corev1.ConditionFalse},
		{Time: metav1.Duration{Duration: 7 * time.Second}, ConditionType: xpv1.TypeReady, ConditionStatus: corev1.ConditionTrue},
		{Time: metav1.Duration{Duration: 5 * time.Second}, ConditionType: xpv1.TypeSynced, ConditionStatus: corev1.ConditionFalse},
		{Time: metav1.Duration{Duration: 10 * time.Second}, ConditionType: xpv1.TypeSynced, ConditionStatus: corev1.ConditionTrue},
		{Time: metav1.Duration{Duration: 2 * time.Second}, ConditionType: xpv1.TypeReady, ConditionStatus: corev1.ConditionFalse},
	}

	now := time.Now()

	cases := map[string]struct {
		reason string
		mg     resource.Managed
		want   resource.Managed
	}{
		"NoDesiredConditionsYet": {
			reason: "No conditions should be set if not enough time has passed for any desired conditions to be applied.",
			mg: &v1alpha1.ClusterNopResource{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: metav1.NewTime(now.Add(-1 * time.Second)),
					Generation:        42,
				},
				Spec: v1alpha1.NopSpec{
					ForProvider: v1alpha1.NopParameters{
						ConditionAfter: c,
					},
				},
			},
			want: &v1alpha1.ClusterNopResource{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: metav1.NewTime(now.Add(-1 * time.Second)),
					Generation:        42,
				},
				Spec: v1alpha1.NopSpec{
					ForProvider: v1alpha1.NopParameters{
						ConditionAfter: c,
					},
				},
			},
		},
		"ReadyForOneDesiredCondition": {
			reason: "Only one condition should be set if enough time has passed for only one desired condition.",
			mg: &v1alpha1.ClusterNopResource{
				ObjectMeta: metav1.ObjectMeta{
					// The earliest condition (5) should be set at two seconds.
					CreationTimestamp: metav1.NewTime(now.Add(-2 * time.Second)),
					Generation:        42,
				},
				Spec: v1alpha1.NopSpec{
					ForProvider: v1alpha1.NopParameters{
						ConditionAfter: c,
					},
				},
			},
			want: &v1alpha1.ClusterNopResource{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: metav1.NewTime(now.Add(-2 * time.Second)),
					Generation:        42,
				},
				Spec: v1alpha1.NopSpec{
					ForProvider: v1alpha1.NopParameters{
						ConditionAfter: c,
					},
				},
				Status: v1alpha1.NopStatus{
					ResourceStatus: xpv1.ResourceStatus{
						ConditionedStatus: xpv1.ConditionedStatus{
							Conditions: []xpv1.Condition{
								{
									Type:               c[5].ConditionType,
									Status:             c[5].ConditionStatus,
									LastTransitionTime: metav1.Now(),
									ObservedGeneration: 42,
								},
							},
						},
					},
				},
			},
		},
		"OnlyLatestConditionsAreSet": {
			reason: "When there are many conditions of the same time, only the latest eligible conditions should be set.",
			mg: &v1alpha1.ClusterNopResource{
				ObjectMeta: metav1.ObjectMeta{
					// After 8 seconds conditions 2 (Ready=True) and 3
					// (Synced=False) should be set. Condition 2 supercedes
					// conditions 1 and 5 (both Ready=False), which happen
					// earlier.
					CreationTimestamp: metav1.NewTime(now.Add(-8 * time.Second)),
					Generation:        42,
				},
				Spec: v1alpha1.NopSpec{
					ForProvider: v1alpha1.NopParameters{
						ConditionAfter: c,
					},
				},
			},
			want: &v1alpha1.ClusterNopResource{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: metav1.NewTime(now.Add(-8 * time.Second)),
					Generation:        42,
				},
				Spec: v1alpha1.NopSpec{
					ForProvider: v1alpha1.NopParameters{
						ConditionAfter: c,
					},
				},
				Status: v1alpha1.NopStatus{
					ResourceStatus: xpv1.ResourceStatus{
						ConditionedStatus: xpv1.ConditionedStatus{
							Conditions: []xpv1.Condition{
								{
									Type:               c[2].ConditionType,
									Status:             c[2].ConditionStatus,
									LastTransitionTime: metav1.Now(),
									ObservedGeneration: 42,
								},
								{
									Type:               c[3].ConditionType,
									Status:             c[3].ConditionStatus,
									LastTransitionTime: metav1.Now(),
									ObservedGeneration: 42,
								},
							},
						},
					},
				},
			},
		},
		"LongTimeReconcileBehaviour": {
			reason: "Indexes with last set status of each condition type should be returned till given time elapsed.",
			mg: &v1alpha1.ClusterNopResource{
				ObjectMeta: metav1.ObjectMeta{
					// After 8 seconds conditions 2 (Ready=True) and 3
					// (Synced=False) should be set. Condition 2 supercedes
					// conditions 1 and 5 (both Ready=False), which happen
					// earlier.
					CreationTimestamp: metav1.NewTime(now.Add(-50 * time.Second)),
					Generation:        42,
				},
				Spec: v1alpha1.NopSpec{
					ForProvider: v1alpha1.NopParameters{
						ConditionAfter: c,
					},
				},
			},
			want: &v1alpha1.ClusterNopResource{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: metav1.NewTime(now.Add(-50 * time.Second)),
					Generation:        42,
				},
				Spec: v1alpha1.NopSpec{
					ForProvider: v1alpha1.NopParameters{
						ConditionAfter: c,
					},
				},
				Status: v1alpha1.NopStatus{
					ResourceStatus: xpv1.ResourceStatus{
						ConditionedStatus: xpv1.ConditionedStatus{
							Conditions: []xpv1.Condition{
								{
									Type:               c[0].ConditionType,
									Status:             c[0].ConditionStatus,
									LastTransitionTime: metav1.Now(),
									ObservedGeneration: 42,
								},
								{
									Type:               c[4].ConditionType,
									Status:             c[4].ConditionStatus,
									LastTransitionTime: metav1.Now(),
									ObservedGeneration: 42,
								},
							},
						},
					},
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, _ = Observe(context.Background(), tc.mg)
			if diff := cmp.Diff(tc.want, tc.mg, test.EquateConditions()); diff != "" {
				t.Errorf("Observe(...): -want, +got:\n%s\n%s\n", tc.reason, diff)
			}
		})
	}
}

func TestObserveDeleted(t *testing.T) {
	after := func(d time.Duration) *metav1.Duration { return &metav1.Duration{Duration: d} }
	msg := func(s string) *string { return &s }
	deletedAt := func(t time.Time) *metav1.Time { mt := metav1.NewTime(t); return &mt }

	now := time.Now()

	cases := map[string]struct {
		reason string
		mg     resource.Managed
		want   managed.ExternalObservation
		// wantCondition, if set, must match the Deletion condition.
		wantCondition *xpv1.Condition
	}{
		"NotDeleted": {
			reason: "A ClusterNopResource that hasn't been deleted still reports its external resource exists.",
			mg: &v1alpha1.ClusterNopResource{
				ObjectMeta: metav1.ObjectMeta{CreationTimestamp: metav1.NewTime(now)},
			},
			want: managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true, ConnectionDetails: managed.ConnectionDetails{}},
		},
		"DeletedImmediately": {
			reason: "With neither deleteAfter nor deleteError the external resource is gone as soon as the ClusterNopResource is deleted.",
			mg: &v1alpha1.ClusterNopResource{
				ObjectMeta: metav1.ObjectMeta{DeletionTimestamp: deletedAt(now)},
			},
			want: managed.ExternalObservation{ResourceExists: false},
		},
		"DeletePending": {
			reason: "Within deleteAfter of the deletion timestamp the external resource still exists, and the Deletion condition says when it can go.",
			mg: &v1alpha1.ClusterNopResource{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: deletedAt(now.Add(-10 * time.Second)),
					Generation:        42,
				},
				Spec: v1alpha1.NopSpec{ForProvider: v1alpha1.NopParameters{DeleteAfter: after(30 * time.Second)}},
			},
			want: managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true},
			wantCondition: &xpv1.Condition{
				Type:               v1alpha1.TypeDeletion,
				Status:             corev1.ConditionFalse,
				Reason:             v1alpha1.ReasonDeletionPending,
				Message:            fmt.Sprintf("External resource will not be deleted before %s (deleteAfter: 30s)", now.Add(20*time.Second).UTC().Format(time.RFC3339)),
				LastTransitionTime: metav1.Now(),
				ObservedGeneration: 42,
			},
		},
		"DeleteFailed": {
			reason: "A ClusterNopResource with deleteError never reports its external resource gone.",
			mg: &v1alpha1.ClusterNopResource{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: deletedAt(now.Add(-time.Hour)),
					Generation:        42,
				},
				Spec: v1alpha1.NopSpec{ForProvider: v1alpha1.NopParameters{DeleteError: msg("cannot delete: still in use")}},
			},
			want: managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true},
			wantCondition: &xpv1.Condition{
				Type:               v1alpha1.TypeDeletion,
				Status:             corev1.ConditionFalse,
				Reason:             v1alpha1.ReasonDeletionFailed,
				Message:            "cannot delete: still in use",
				LastTransitionTime: metav1.Now(),
				ObservedGeneration: 42,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := Observe(context.Background(), tc.mg)
			if err != nil {
				t.Fatalf("\n%s\nObserve(...): unexpected error: %v", tc.reason, err)
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("\n%s\nObserve(...): -want, +got:\n%s", tc.reason, diff)
			}

			c := tc.mg.(*v1alpha1.ClusterNopResource).GetCondition(v1alpha1.TypeDeletion)

			if tc.wantCondition == nil {
				if c.Reason != "" {
					t.Errorf("\n%s\nObserve(...): unexpected Deletion condition %v", tc.reason, c)
				}

				return
			}

			if diff := cmp.Diff(*tc.wantCondition, c, test.EquateConditions()); diff != "" {
				t.Errorf("\n%s\nObserve(...): -want Deletion condition, +got:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	msg := "cannot delete: still in use"

	cases := map[string]struct {
		reason  string
		mg      resource.Managed
		wantErr string
	}{
		"WrongType": {
			reason:  "Deleting something that isn't a ClusterNopResource is an error.",
			mg:      &fake.Managed{},
			wantErr: "managed resource was not a *v1alpha1.ClusterNopResource",
		},
		"Succeeds": {
			reason: "Deleting is a no-op unless deleteError says otherwise.",
			mg:     &v1alpha1.ClusterNopResource{},
		},
		"Fails": {
			reason:  "deleteError is what makes a ClusterNopResource refuse to delete.",
			mg:      &v1alpha1.ClusterNopResource{Spec: v1alpha1.NopSpec{ForProvider: v1alpha1.NopParameters{DeleteError: &msg}}},
			wantErr: msg,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := Delete(context.Background(), tc.mg)

			switch {
			case tc.wantErr == "" && err != nil:
				t.Errorf("\n%s\nDelete(...): unexpected error: %v", tc.reason, err)
			case tc.wantErr != "" && err == nil:
				t.Errorf("\n%s\nDelete(...): want error %q, got none", tc.reason, tc.wantErr)
			case tc.wantErr != "" && err.Error() != tc.wantErr:
				t.Errorf("\n%s\nDelete(...): want error %q, got %q", tc.reason, tc.wantErr, err.Error())
			}
		})
	}
}
