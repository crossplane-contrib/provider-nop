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
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/crossplane/crossplane-runtime/pkg/test"

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

	c := []v1alpha1.ResourceConditionAfter{
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
			mg: &v1alpha1.NopResource{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: metav1.NewTime(now.Add(-1 * time.Second)),
				},
				Spec: v1alpha1.NopResourceSpec{
					ForProvider: v1alpha1.NopResourceParameters{
						ConditionAfter: c,
					},
				},
			},
			want: &v1alpha1.NopResource{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: metav1.NewTime(now.Add(-1 * time.Second)),
				},
				Spec: v1alpha1.NopResourceSpec{
					ForProvider: v1alpha1.NopResourceParameters{
						ConditionAfter: c,
					},
				},
			},
		},
		"ReadyForOneDesiredCondition": {
			reason: "Only one condition should be set if enough time has passed for only one desired condition.",
			mg: &v1alpha1.NopResource{
				ObjectMeta: metav1.ObjectMeta{
					// The earliest condition (5) should be set at two seconds.
					CreationTimestamp: metav1.NewTime(now.Add(-2 * time.Second)),
				},
				Spec: v1alpha1.NopResourceSpec{
					ForProvider: v1alpha1.NopResourceParameters{
						ConditionAfter: c,
					},
				},
			},
			want: &v1alpha1.NopResource{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: metav1.NewTime(now.Add(-2 * time.Second)),
				},
				Spec: v1alpha1.NopResourceSpec{
					ForProvider: v1alpha1.NopResourceParameters{
						ConditionAfter: c,
					},
				},
				Status: v1alpha1.NopResourceStatus{
					ResourceStatus: xpv1.ResourceStatus{
						ConditionedStatus: xpv1.ConditionedStatus{
							Conditions: []xpv1.Condition{
								{
									Type:               c[5].ConditionType,
									Status:             c[5].ConditionStatus,
									LastTransitionTime: metav1.Now(),
								},
							},
						},
					},
				},
			},
		},
		"OnlyLatestConditionsAreSet": {
			reason: "When there are many conditions of the same time, only the latest eligible conditions should be set.",
			mg: &v1alpha1.NopResource{
				ObjectMeta: metav1.ObjectMeta{
					// After 8 seconds conditions 2 (Ready=True) and 3
					// (Synced=False) should be set. Condition 2 supercedes
					// conditions 1 and 5 (both Ready=False), which happen
					// earlier.
					CreationTimestamp: metav1.NewTime(now.Add(-8 * time.Second)),
				},
				Spec: v1alpha1.NopResourceSpec{
					ForProvider: v1alpha1.NopResourceParameters{
						ConditionAfter: c,
					},
				},
			},
			want: &v1alpha1.NopResource{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: metav1.NewTime(now.Add(-8 * time.Second)),
				},
				Spec: v1alpha1.NopResourceSpec{
					ForProvider: v1alpha1.NopResourceParameters{
						ConditionAfter: c,
					},
				},
				Status: v1alpha1.NopResourceStatus{
					ResourceStatus: xpv1.ResourceStatus{
						ConditionedStatus: xpv1.ConditionedStatus{
							Conditions: []xpv1.Condition{
								{
									Type:               c[2].ConditionType,
									Status:             c[2].ConditionStatus,
									LastTransitionTime: metav1.Now(),
								},
								{
									Type:               c[3].ConditionType,
									Status:             c[3].ConditionStatus,
									LastTransitionTime: metav1.Now(),
								},
							},
						},
					},
				},
			},
		},
		"LongTimeReconcileBehaviour": {
			reason: "Indexes with last set status of each condition type should be returned till given time elapsed.",
			mg: &v1alpha1.NopResource{
				ObjectMeta: metav1.ObjectMeta{
					// After 8 seconds conditions 2 (Ready=True) and 3
					// (Synced=False) should be set. Condition 2 supercedes
					// conditions 1 and 5 (both Ready=False), which happen
					// earlier.
					CreationTimestamp: metav1.NewTime(now.Add(-50 * time.Second)),
				},
				Spec: v1alpha1.NopResourceSpec{
					ForProvider: v1alpha1.NopResourceParameters{
						ConditionAfter: c,
					},
				},
			},
			want: &v1alpha1.NopResource{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: metav1.NewTime(now.Add(-50 * time.Second)),
				},
				Spec: v1alpha1.NopResourceSpec{
					ForProvider: v1alpha1.NopResourceParameters{
						ConditionAfter: c,
					},
				},
				Status: v1alpha1.NopResourceStatus{
					ResourceStatus: xpv1.ResourceStatus{
						ConditionedStatus: xpv1.ConditionedStatus{
							Conditions: []xpv1.Condition{
								{
									Type:               c[0].ConditionType,
									Status:             c[0].ConditionStatus,
									LastTransitionTime: metav1.Now(),
								},
								{
									Type:               c[4].ConditionType,
									Status:             c[4].ConditionStatus,
									LastTransitionTime: metav1.Now(),
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
