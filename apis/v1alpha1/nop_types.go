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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
)

// ConditionAfter specifies a condition of a NopResource that should be
// set after a certain duration.
type ConditionAfter struct {
	// Time is the duration after which the condition should be set.
	Time metav1.Duration `json:"time"`

	// ConditionType to set - e.g. Ready.
	ConditionType xpv1.ConditionType `json:"conditionType"`

	// ConditionStatus to set - e.g. True.
	ConditionStatus corev1.ConditionStatus `json:"conditionStatus"`

	// ConditionReason to set - e.g. Available.
	// +optional
	ConditionReason *xpv1.ConditionReason `json:"conditionReason,omitempty"`
}

// ConnectionDetail specifies a connection detail a NopResource should
// emit.
type ConnectionDetail struct {
	// Name of the connection detail.
	Name string `json:"name"`

	// Value of the connection detail.
	Value string `json:"value"`
}

// NopParameters are the configurable fields of a NopResource.
type NopParameters struct {
	// ConditionAfter can be used to set status conditions after a specified
	// time. By default, a NopResource will only have a status condition of
	// Type: Synced. It will never have a status condition of Type: Ready
	// unless one is configured here.
	// +optional
	ConditionAfter []ConditionAfter `json:"conditionAfter,omitempty"`

	// ConnectionDetails that this NopResource should emit on each reconcile.
	// +optional
	ConnectionDetails []ConnectionDetail `json:"connectionDetails,omitempty"`

	// Fields is an arbitrary object you can patch to and from. It has no
	// schema, is not validated, and is not used by the NopResource controller.
	// +optional
	Fields runtime.RawExtension `json:"fields,omitempty"`
}

// NopObservation are the observable fields of a NopResource.
type NopObservation struct {
	// Fields is an arbitrary object you can patch to and from. It has no
	// schema, is not validated, and is not used by the NopResource controller.
	// +optional
	Fields runtime.RawExtension `json:"fields,omitempty"`
}

// A NopSpec defines the desired state of a NopResource.
type NopSpec struct {
	xpv1.ResourceSpec `json:",inline"`

	ForProvider NopParameters `json:"forProvider"`
}

// A NopStatus represents the observed state of a NopResource.
type NopStatus struct {
	xpv1.ResourceStatus `json:",inline"`

	AtProvider NopObservation `json:"atProvider,omitempty"`
}
