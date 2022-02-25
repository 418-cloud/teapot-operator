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

// ContainerAppSpec defines the desired state of ContainerApp
type ContainerAppSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	//ContainerEnvironmentName name of the ContainerEnvironment where the ContainerApp is deployed
	ContainerEnvironmentName string `json:"containerEnvironmentName"`
	//ContainersTemplate template for the deployed containers
	ContainersTemplate ContainersTemplate `json:"containersTemplate"`
	//TargetPort port the container listens to
	TargetPort int32 `json:"targetPort"`
}

// ContainerAppStatus defines the observed state of ContainerApp
type ContainerAppStatus struct {
	LatestStatus string `json:"latestStatus,omitempty"`
	FQDN         string `json:"fqdn,omitempty"`
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:printcolumn:name="FQDN",type=string,JSONPath=`.status.fqdn`
//+kubebuilder:subresource:status

// ContainerApp is the Schema for the containerapps API
type ContainerApp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ContainerAppSpec   `json:"spec,omitempty"`
	Status ContainerAppStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ContainerAppList contains a list of ContainerApp
type ContainerAppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ContainerApp `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ContainerApp{}, &ContainerAppList{})
}

//ContainersTemplate defines the desired state of Container inside a ContainerApp
type ContainersTemplate struct {
	Name      string                       `json:"name"`
	Image     string                       `json:"image"`
	Resources *ContainersTemplateResources `json:"resources,omitempty"`
}

//ContainersTemplateResources defines the desired state of resources inside a ContainerTemplate
type ContainersTemplateResources struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
}
