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
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/conditions"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"

	"github.com/crossplane-contrib/provider-nop/apis/v1alpha1"
)

func TestObserveDeleted(t *testing.T) {
	after := func(d time.Duration) *metav1.Duration { return &metav1.Duration{Duration: d} }
	msg := func(s string) *string { return &s }

	now := time.Now()

	cases := map[string]struct {
		reason string
		params v1alpha1.NopParameters
		// deleted is when the NopResource's deletion timestamp was set.
		deleted time.Time
		want    managed.ExternalObservation
		// wantReason and wantMessage, when wantReason is set, must match the
		// Deletion condition. When it isn't, there must be no such condition.
		wantReason  xpv1.ConditionReason
		wantMessage string
	}{
		"NoDelay": {
			reason:  "With no deleteAfter the external resource is gone at once, as it always was.",
			deleted: now,
		},
		"ZeroDelay": {
			reason:  "A deleteAfter of 0s is no delay at all.",
			params:  v1alpha1.NopParameters{DeleteAfter: after(0)},
			deleted: now,
		},
		"NegativeDelay": {
			reason:  "A negative deleteAfter - which the CRD rejects, but which we shouldn't hang on - is already elapsed.",
			params:  v1alpha1.NopParameters{DeleteAfter: after(-time.Minute)},
			deleted: now,
		},
		"StillDeleting": {
			reason:      "Before deleteAfter elapses the external resource is still there, and says when it can go.",
			params:      v1alpha1.NopParameters{DeleteAfter: after(30 * time.Second)},
			deleted:     now.Add(-10 * time.Second),
			want:        managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true},
			wantReason:  v1alpha1.ReasonDeletionPending,
			wantMessage: fmt.Sprintf("External resource will not be deleted before %s (deleteAfter: 30s)", now.Add(20*time.Second).UTC().Format(time.RFC3339)),
		},
		"DeletedAfterDelay": {
			reason:  "Once deleteAfter has elapsed the external resource is gone.",
			params:  v1alpha1.NopParameters{DeleteAfter: after(30 * time.Second)},
			deleted: now.Add(-30 * time.Second),
		},
		"DeleteErrorNeverCompletes": {
			reason:      "A resource whose deletion fails is still there however long we wait, and says why.",
			params:      v1alpha1.NopParameters{DeleteError: msg("cannot delete: still in use")},
			deleted:     now.Add(-24 * time.Hour),
			want:        managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true},
			wantReason:  v1alpha1.ReasonDeletionFailed,
			wantMessage: "cannot delete: still in use",
		},
		"DeleteErrorBeatsDeleteAfter": {
			reason:      "A resource that cannot be deleted doesn't become deletable when its delay elapses.",
			params:      v1alpha1.NopParameters{DeleteAfter: after(time.Second), DeleteError: msg("boom")},
			deleted:     now.Add(-time.Hour),
			want:        managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true},
			wantReason:  v1alpha1.ReasonDeletionFailed,
			wantMessage: "boom",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			nop := &v1alpha1.NopResource{}

			got := ObserveDeleted(tc.params, tc.deleted, conditions.ObservedGenerationPropagationManager{}.For(nop))
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("\n%s\nObserveDeleted(...): -want, +got:\n%s", tc.reason, diff)
			}

			c := nop.GetCondition(v1alpha1.TypeDeletion)

			if tc.wantReason == "" {
				// An unset condition reads back as Unknown with no reason.
				if c.Reason != "" {
					t.Errorf("\n%s\nObserveDeleted(...): unexpected Deletion condition %v", tc.reason, c)
				}

				return
			}

			if diff := cmp.Diff(tc.wantReason, c.Reason); diff != "" {
				t.Errorf("\n%s\nObserveDeleted(...): -want reason, +got reason:\n%s", tc.reason, diff)
			}

			if diff := cmp.Diff(tc.wantMessage, c.Message); diff != "" {
				t.Errorf("\n%s\nObserveDeleted(...): -want message, +got message:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	msg := "cannot delete: still in use"

	cases := map[string]struct {
		reason  string
		params  v1alpha1.NopParameters
		wantErr string
	}{
		"Succeeds": {
			reason: "Deleting is a no-op unless deleteError says otherwise.",
		},
		"Fails": {
			reason:  "deleteError is what makes a resource refuse to delete.",
			params:  v1alpha1.NopParameters{DeleteError: &msg},
			wantErr: msg,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := Delete(tc.params)

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
