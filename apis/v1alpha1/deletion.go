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

package v1alpha1

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
)

// TypeDeletion reports how far a NopResource's deletion has progressed. The
// managed reconciler owns the Ready condition while a resource is being
// deleted, so deletion progress goes here.
const TypeDeletion xpv1.ConditionType = "Deletion"

// Reasons a resource is in a particular deletion state.
const (
	ReasonDeletionPending xpv1.ConditionReason = "DeletionPending"
	ReasonDeletionFailed  xpv1.ConditionReason = "DeletionFailed"
)

// DeletionPending returns a condition reporting that a NopResource is still
// being deleted, and the earliest time its deletion can complete.
func DeletionPending(deadline time.Time, after time.Duration) xpv1.Condition {
	return xpv1.Condition{
		Type:               TypeDeletion,
		Status:             corev1.ConditionFalse,
		LastTransitionTime: metav1.Now(),
		Reason:             ReasonDeletionPending,
		Message: fmt.Sprintf("External resource will not be deleted before %s (deleteAfter: %s)",
			deadline.UTC().Format(time.RFC3339), after),
	}
}

// DeletionFailed returns a condition reporting that a NopResource cannot be
// deleted, and why. A resource in this state never goes away; clear
// spec.forProvider.deleteError to release it.
func DeletionFailed(msg string) xpv1.Condition {
	return xpv1.Condition{
		Type:               TypeDeletion,
		Status:             corev1.ConditionFalse,
		LastTransitionTime: metav1.Now(),
		Reason:             ReasonDeletionFailed,
		Message:            msg,
	}
}
