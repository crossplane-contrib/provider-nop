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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

// ResourceConditionAfter is a configurable field of NopResource.
type ResourceConditionAfter struct {
	Time            string `json:"time"`
	ConditionType   string `json:"conditionType"`
	ConditionStatus string `json:"conditionStatus"`
}

// NopResourceParameters are the configurable fields of a NopResource.
type NopResourceParameters struct {
	ConditionAfter []ResourceConditionAfter `json:"conditionAfter"`
}

// NopResourceObservation are the observable fields of a NopResource.
type NopResourceObservation struct {
	ObservableField string `json:"observableField,omitempty"`
	// ObservableArrays []string
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
// Please replace `PROVIDER-NAME` with your actual provider name, like `aws`, `azure`, `gcp`, `alibaba`
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
