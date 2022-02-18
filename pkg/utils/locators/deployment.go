package locators

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

//GetContainerInDeployment returns the container with the given name
func GetContainerInDeployment(d appsv1.Deployment, name string) *corev1.Container {
	for _, container := range d.Spec.Template.Spec.Containers {
		if container.Name == name {
			return &container
		}
	}
	return nil
}

//GetContainer returns the container with the given name
func GetContainerIndexByName(d appsv1.Deployment, name string) int {
	for i, container := range d.Spec.Template.Spec.Containers {
		if container.Name == name {
			return i
		}
	}
	return -1
}
