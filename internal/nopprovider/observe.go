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

// Package nopprovider is logic for the controllers for a managed resource that does nothing.
package nopprovider

import (
	"sort"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/conditions"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"

	"github.com/crossplane-contrib/provider-nop/apis/v1alpha1"
)

// Observe doesn't actually observe an external resource. Instead, it sets the
// most recent conditions that should occur per nop.conditionAfter.
func Observe(nop v1alpha1.NopParameters, age time.Duration, status conditions.ConditionSet) (managed.ExternalObservation, error) {
	// Sort conditions, with those that should occur latest appearing first.
	// We rely on the fact that the managed.Reconciler won't persist this sorted
	// array because it occurred during the Observe function, and we didn't
	// return ResourceLateInitialized: true.
	sort.SliceStable(nop.ConditionAfter, func(i, j int) bool {
		return nop.ConditionAfter[i].Time.Duration > nop.ConditionAfter[j].Time.Duration
	})

	set := map[xpv1.ConditionType]bool{}
	for _, ca := range nop.ConditionAfter {
		if ca.Time.Duration > age {
			// This condition should not occur yet.
			continue
		}

		if set[ca.ConditionType] {
			// We already encountered and set a condition of this type.
			continue
		}

		// This is the latest condition of this type that should be set.
		var reason xpv1.ConditionReason
		if ca.ConditionReason != nil {
			reason = *ca.ConditionReason
		}
		status.MarkConditions(xpv1.Condition{
			Type:               ca.ConditionType,
			Status:             ca.ConditionStatus,
			Reason:             reason,
			LastTransitionTime: metav1.Now(),
		})

		set[ca.ConditionType] = true
	}

	// Emit any connection details we were asked to.
	cd := managed.ConnectionDetails{}
	for _, nv := range nop.ConnectionDetails {
		cd[nv.Name] = []byte(nv.Value)
	}

	// If our managed resource has not been deleted we report that our
	// pretend external resource exists and is up-to-date. This means
	// we'll never call the CreateFn or UpdateFn.
	return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true, ConnectionDetails: cd}, nil
}
