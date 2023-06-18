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

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

// ResourceConditionAfter specifies a condition of a NopResource that should be
// set after a certain duration.
type ResourceConditionAfter struct {
	// Time is the duration after which the condition should be set.
	Time metav1.Duration `json:"time"`

	// ConditionType to set - e.g. Ready.
	ConditionType xpv1.ConditionType `json:"conditionType"`

	// ConditionStatus to set - e.g. True.
	ConditionStatus corev1.ConditionStatus `json:"conditionStatus"`
}

// ResourceConnectionDetail specifies a connection detail a NopResource should
// emit.
type ResourceConnectionDetail struct {
	// Name of the connection detail.
	Name string `json:"key"`

	// Value of the connection detail.
	Value string `json:"value"`
}

// NopResourceParameters are the configurable fields of a NopResource.
type NopResourceParameters struct {
	// ConditionAfter can be used to set status conditions after a specified
	// time. By default a NopResource will only have a status condition of Type:
	// Synced. It will never have a status condition of Type: Ready unless one
	// is configured here.
	// +optional
	ConditionAfter []ResourceConditionAfter `json:"conditionAfter,omitempty"`

	// ConnectionDetails that this NopResource should emit on each reconcile.
	// +optional
	ConnectionDetails []ResourceConnectionDetail `json:"connectionDetails,omitempty"`

	// Fields is an arbitrary object you can patch to and from. It has no
	// schema, is not validated, and is not used by the NopResource controller.
	// +kubebuilder:validation:Schemaless
	// +optional
	Fields runtime.RawExtension `json:"fields,omitempty"`
}

// NopResourceObservation are the observable fields of a NopResource.
type NopResourceObservation struct {
	// Fields is an arbitrary object you can patch to and from. It has no
	// schema, is not validated, and is not used by the NopResource controller.
	// +kubebuilder:validation:Schemaless
	// +optional
	Fields runtime.RawExtension `json:"fields,omitempty"`
}

// A NopResourceSpec defines the desired state of a NopResource.
type NopResourceSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       NopResourceParameters `json:"forProvider"`
}

// A NopResourceStatus represents the observed state of a NopResource.
type NopResourceStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          NopResourceObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A NopResource is an example API type.
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,nop}
type NopResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NopResourceSpec   `json:"spec"`
	Status NopResourceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NopResourceList contains a list of NopResource
type NopResourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NopResource `json:"items"`
}
