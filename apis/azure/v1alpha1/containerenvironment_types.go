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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ContainerEnvironmentSpec defines the desired state of ContainerEnvironment
type ContainerEnvironmentSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of ContainerEnvironment. Edit containerenvironment_types.go to remove/update
	Name     string `json:"name,omitempty"`
	Location string `json:"location,omitempty"`
}

// ContainerEnvironmentStatus defines the observed state of ContainerEnvironment
type ContainerEnvironmentStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	EnvironmentID     string `json:"environmentID,omitempty"`
	EnvironmentName   string `json:"environmentName,omitempty"`
	EnvironmentIP     string `json:"environmentIP,omitempty"`
	EnvironmentStatus string `json:"environmentStatus,omitempty"`
	EnvironmentFQDN   string `json:"environemntFQDN,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ContainerEnvironment is the Schema for the containerenvironments API
type ContainerEnvironment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ContainerEnvironmentSpec   `json:"spec,omitempty"`
	Status ContainerEnvironmentStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ContainerEnvironmentList contains a list of ContainerEnvironment
type ContainerEnvironmentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ContainerEnvironment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ContainerEnvironment{}, &ContainerEnvironmentList{})
}
