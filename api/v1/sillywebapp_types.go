/*
Copyright 2022.

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

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SillyWebappSpec defines the desired state of SillyWebapp
type SillyWebappSpec struct {
	Frontend  FrontendSpec `json:"frontend"`
	RedisName string       `json:"redisName,omitempty"`
}

// FrontendSpec speficies the frontend container spec
type FrontendSpec struct {
	// +optional
	Resources corev1.ResourceRequirements `json:"resources"`

	// +optional
	// +kubebuilder:default=8080
	// +kubebuilder:validation:Minimum=0
	ServingPort int32 `json:"servingPort"`

	// +optional
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=0
	Replicas *int32 `json:"replicas,omitempty"`
}

// SillyWebappStatus defines the observed state of SillyWebapp
type SillyWebappStatus struct {
	URL string `json:"url"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:JSONPath=".status.url",name="URL",type="string"
//+kubebuilder:printcolumn:JSONPath=".spec.frontend.replicas",name="Replicas",type="integer"

// SillyWebapp is the Schema for the sillywebapps API
type SillyWebapp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SillyWebappSpec   `json:"spec,omitempty"`
	Status SillyWebappStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SillyWebappList contains a list of SillyWebapp
type SillyWebappList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SillyWebapp `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SillyWebapp{}, &SillyWebappList{})
}
