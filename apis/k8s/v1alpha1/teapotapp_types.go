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
	"fmt"
	"strings"

	"github.com/418-cloud/teapot-operator/pkg/utils/locators"
	"github.com/418-cloud/teapot-operator/pkg/utils/to"
	traefikv1alpha1 "github.com/traefik/traefik/v2/pkg/provider/kubernetes/crd/traefik/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TeapotAppSpec defines the desired state of TeapotApp
type TeapotAppSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Image     string             `json:"image,omitempty"`
	Args      []string           `json:"args,omitempty"`
	Scale     TeapotAppScale     `json:"scale,omitempty"`
	Resources TeapotAppResources `json:"resources,omitempty"`
	Path      string             `json:"path,omitempty"`
}

// TeapotAppStatus defines the observed state of TeapotApp
type TeapotAppStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	DeploymentName   string                  `json:"deployment"`
	DeploymentStatus appsv1.DeploymentStatus `json:"deploymentStatus"`
	Route            string                  `json:"route"`
}

//+kubebuilder:object:root=true
//+kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.deploymentStatus.readyReplicas`
//+kubebuilder:printcolumn:name="Route",type=string,JSONPath=`.status.route`
//+kubebuilder:printcolumn:name="Image",type=string,JSONPath=`.spec.image`,priority=1
//+kubebuilder:subresource:status

// TeapotApp is the Schema for the teapotapps API
type TeapotApp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TeapotAppSpec   `json:"spec,omitempty"`
	Status TeapotAppStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TeapotAppList contains a list of TeapotApp
type TeapotAppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TeapotApp `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TeapotApp{}, &TeapotAppList{})
}

//TeapotAppScale defines the desired scale of the deployment
type TeapotAppScale struct {
	Replicas    *int32                `json:"replicas"`
	Autoscaling *TeapotAppAutoscaling `json:"autoscaling,omitempty"`
}

type TeapotAppAutoscaling struct {
	Enabled     *bool  `json:"enabled,omitempty"`
	MinReplicas *int32 `json:"minReplicas,omitempty"`
	MaxReplicas *int32 `json:"maxReplicas,omitempty"`
}

//TeapotAppResources defines the resources of the deployment
type TeapotAppResources struct {
	Limits   *TeapotAppResourceBlock `json:"limits,omitempty"`
	Requests *TeapotAppResourceBlock `json:"requests,omitempty"`
}

//TeapotAppResourceBlock defines CPU and Memory requests and limits of the deployment
type TeapotAppResourceBlock struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
}

//IsDeploymentOutdated returns true if the deployment is outdated
func (t *TeapotApp) IsDeploymentOutdated(deployment appsv1.Deployment) bool {
	if !t.AutoscalingEnabled() && t.GetReplicas() != nil && *t.GetReplicas() != *deployment.Spec.Replicas {
		fmt.Println("Replicas are not equal")
		return true
	}
	if t.Spec.Resources.Requests != nil &&
		(t.Spec.Resources.Requests.CPU != locators.GetContainerInDeployment(deployment, t.Name).Resources.Requests.Cpu().String() ||
			t.Spec.Resources.Requests.Memory != locators.GetContainerInDeployment(deployment, t.Name).Resources.Requests.Memory().String()) {
		return true
	}

	if t.Spec.Resources.Limits != nil &&
		(t.Spec.Resources.Limits.CPU != locators.GetContainerInDeployment(deployment, t.Name).Resources.Limits.Cpu().String() ||
			t.Spec.Resources.Limits.Memory != locators.GetContainerInDeployment(deployment, t.Name).Resources.Limits.Memory().String()) {
		return true
	}
	if t.Spec.Args != nil && !strings.EqualFold(strings.Join(t.Spec.Args, " "), strings.Join(locators.GetContainerInDeployment(deployment, t.Name).Args, " ")) {
		return true
	}
	return false
}

func (t *TeapotApp) AutoscalingEnabled() bool {
	if t.Spec.Scale.Autoscaling == nil || t.Spec.Scale.Autoscaling.Enabled == nil || *t.Spec.Scale.Autoscaling.Enabled {
		return true
	}
	return false
}

func (t *TeapotApp) IsAutoscalingOutdated(autoscaling autoscalingv2beta2.HorizontalPodAutoscaler) bool {
	if !t.AutoscalingEnabled() {
		return true
	}
	if *t.GetMinReplicas() != *autoscaling.Spec.MinReplicas {
		return true
	}
	if t.GetMaxReplicas() != autoscaling.Spec.MaxReplicas {
		return true
	}
	return false
}

func (t *TeapotApp) IsIngressRouteOutdated(ingressroute traefikv1alpha1.IngressRoute) bool {
	if strings.Contains(ingressroute.Spec.Routes[0].Match, fmt.Sprintf("PathPrefix(`%s`)", t.GetPath())) {
		return false
	}
	return true
}

func (t *TeapotApp) GetLimits() corev1.ResourceList {
	if t.Spec.Resources.Limits == nil {
		return corev1.ResourceList{}
	}
	return corev1.ResourceList{
		"cpu":    resource.MustParse(t.Spec.Resources.Limits.CPU),
		"memory": resource.MustParse(t.Spec.Resources.Limits.Memory),
	}
}

func (t *TeapotApp) GetRequests() corev1.ResourceList {
	if t.Spec.Resources.Requests == nil {
		return corev1.ResourceList{
			"cpu":    resource.MustParse("100m"),
			"memory": resource.MustParse("128Mi"),
		}
	}
	return corev1.ResourceList{
		"cpu":    resource.MustParse(t.Spec.Resources.Requests.CPU),
		"memory": resource.MustParse(t.Spec.Resources.Requests.Memory),
	}
}

func (t *TeapotApp) GetReplicas() *int32 {
	if t.Spec.Scale.Replicas == nil {
		return to.Int32Ptr(2)
	}
	return t.Spec.Scale.Replicas
}

func (t *TeapotApp) GetMinReplicas() *int32 {
	if t.Spec.Scale.Autoscaling == nil || t.Spec.Scale.Autoscaling.MinReplicas == nil {
		return to.Int32Ptr(1)
	}
	return t.Spec.Scale.Autoscaling.MinReplicas
}

func (t *TeapotApp) GetMaxReplicas() int32 {
	if t.Spec.Scale.Autoscaling == nil || t.Spec.Scale.Autoscaling.MaxReplicas == nil {
		return 3
	}
	return *t.Spec.Scale.Autoscaling.MaxReplicas
}

func (t *TeapotApp) GetPath() string {
	if t.Spec.Path == "" {
		return "/"
	}
	return t.Spec.Path
}
