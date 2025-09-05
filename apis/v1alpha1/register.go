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
	"reflect"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"

	"github.com/crossplane/crossplane-runtime/v2/pkg/webhook"
)

// Package type metadata.
const (
	Group   = "nop.crossplane.io"
	Version = "v1alpha1"
)

var (
	// SchemeGroupVersion is group version used to register these objects.
	SchemeGroupVersion = schema.GroupVersion{Group: Group, Version: Version}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme.
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}
)

// NopResource type metadata.
var (
	ClusterNopResourceKind             = reflect.TypeOf(ClusterNopResource{}).Name()
	ClusterNopResourceGroupKind        = schema.GroupKind{Group: Group, Kind: ClusterNopResourceKind}.String()
	ClusterNopResourceKindAPIVersion   = NopResourceKind + "." + SchemeGroupVersion.String()
	ClusterNopResourceGroupVersionKind = SchemeGroupVersion.WithKind(ClusterNopResourceKind)

	NopResourceKind             = reflect.TypeOf(NopResource{}).Name()
	NopResourceGroupKind        = schema.GroupKind{Group: Group, Kind: NopResourceKind}.String()
	NopResourceKindAPIVersion   = NopResourceKind + "." + SchemeGroupVersion.String()
	NopResourceGroupVersionKind = SchemeGroupVersion.WithKind(NopResourceKind)

	// NopResourceValidator is doing nothing on purpose at the moment, you now... a nop validator.
	NopResourceValidator = webhook.NewValidator()

	// ClusterNopResourceValidator is doing nothing on purpose at the moment, you now... a nop validator.
	ClusterNopResourceValidator = webhook.NewValidator()
)

func init() {
	SchemeBuilder.Register(
		&NopResource{}, &NopResourceList{},
		&ClusterNopResource{}, &ClusterNopResourceList{},
	)
}

// +kubebuilder:webhook:verbs=create;update,path=/validate-nop-crossplane-io-v1alpha1-nopresource,mutating=false,failurePolicy=fail,groups=nop.crossplane.io,resources=nopresources,versions=v1alpha1,name=nopresources.nop.crossplane.io,sideEffects=None,admissionReviewVersions=v1
// +kubebuilder:webhook:verbs=create;update,path=/validate-nop-crossplane-io-v1alpha1-clusternopresource,mutating=false,failurePolicy=fail,groups=nop.crossplane.io,resources=clusternopresources,versions=v1alpha1,name=clusternopresources.nop.crossplane.io,sideEffects=None,admissionReviewVersions=v1
