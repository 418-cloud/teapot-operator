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

package controllers

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	configv2 "github.com/418-cloud/teapot-operator/apis/config/v2"
	k8sv1alpha1 "github.com/418-cloud/teapot-operator/apis/k8s/v1alpha1"
	"github.com/418-cloud/teapot-operator/pkg/utils/locators"
	"github.com/418-cloud/teapot-operator/pkg/utils/to"
	traefikv1alpha1 "github.com/traefik/traefik/v2/pkg/provider/kubernetes/crd/traefik/v1alpha1"
	traefiktype "github.com/traefik/traefik/v2/pkg/types"
)

// TeapotAppReconciler reconciles a TeapotApp object
type TeapotAppReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Config configv2.ProjectConfig
}

//+kubebuilder:rbac:groups=k8s.418.cloud,resources=teapotapps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=k8s.418.cloud,resources=teapotapps/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=k8s.418.cloud,resources=teapotapps/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get
//+kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers/status,verbs=get
//+kubebuilder:rbac:groups=traefik.containo.us,resources=ingressroutes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=traefik.containo.us,resources=ingressroutes/status,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the TeapotApp object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *TeapotAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var teapotapp k8sv1alpha1.TeapotApp
	if err := r.Get(ctx, req.NamespacedName, &teapotapp); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err, "unable to fetch TeapotApp")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	//Deployment
	var deployment appsv1.Deployment
	err := r.Get(ctx, types.NamespacedName{Namespace: teapotapp.Namespace, Name: teapotapp.Status.DeploymentName}, &deployment)
	if client.IgnoreNotFound(err) != nil {
		log.Error(err, "unable to fetch Deployment")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if errors.IsNotFound(err) {
		log.Info("Deployment not found, creating a new one")
		return ctrl.Result{}, r.createNewDeployment(ctx, &teapotapp)
	}
	if teapotapp.IsDeploymentOutdated(deployment) {
		log.Info("Deployment outdated, updating Deployment")
		return ctrl.Result{}, r.updateDeployment(ctx, &teapotapp, deployment)
	}
	//Autoscaling
	var autoscaling autoscalingv2beta2.HorizontalPodAutoscaler
	err = r.Get(ctx, types.NamespacedName{Namespace: teapotapp.Namespace, Name: teapotapp.Name}, &autoscaling)
	if client.IgnoreNotFound(err) != nil {
		log.Error(err, "unable to fetch HorizontalPodAutoscaler")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if teapotapp.AutoscalingEnabled() && errors.IsNotFound(err) {
		log.Info("HorizontalPodAutoscaler not found, creating a new one")
		return ctrl.Result{}, r.createNewAutoscaling(ctx, &teapotapp)
	}
	if teapotapp.IsAutoscalingOutdated(autoscaling) && !errors.IsNotFound(err) {
		log.Info("HorizontalPodAutoscaler outdated, updating HorizontalPodAutoscaler")
		return ctrl.Result{}, r.updateAutoscaling(ctx, &teapotapp, autoscaling)
	}
	//Service
	var service corev1.Service
	err = r.Get(ctx, types.NamespacedName{Namespace: teapotapp.Namespace, Name: teapotapp.Name}, &service)
	if client.IgnoreNotFound(err) != nil {
		log.Error(err, "unable to fetch Service")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if errors.IsNotFound(err) {
		log.Info("Service not found, creating a new one")
		return ctrl.Result{}, r.createNewService(ctx, &teapotapp)
	}
	//IngressRoute
	var ingressRoute traefikv1alpha1.IngressRoute
	err = r.Get(ctx, types.NamespacedName{Namespace: teapotapp.Namespace, Name: teapotapp.Name}, &ingressRoute)
	if client.IgnoreNotFound(err) != nil {
		log.Error(err, "unable to fetch IngressRoute")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if errors.IsNotFound(err) {
		log.Info("IngressRoute not found, creating a new one")
		return ctrl.Result{}, r.createNewIngressRoute(ctx, &teapotapp)
	}
	if teapotapp.IsIngressRouteOutdated(ingressRoute) {
		log.Info("IngressRoute outdated, updating IngressRoute")
		return ctrl.Result{}, r.updateIngressRoute(ctx, &teapotapp, ingressRoute)
	}
	teapotapp.Status.Route = fmt.Sprintf("%s://%s%s", r.Config.Protocol, r.Config.Domain, teapotapp.GetPath())
	teapotapp.Status.DeploymentStatus = deployment.Status
	if err := r.Status().Update(ctx, &teapotapp); err != nil {
		log.Error(err, "unable to update TeapotApp status")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TeapotAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1alpha1.TeapotApp{}).
		Owns(&appsv1.Deployment{}).
		Owns(&autoscalingv2beta2.HorizontalPodAutoscaler{}).
		Owns(&corev1.Service{}).
		Owns(&traefikv1alpha1.IngressRoute{}).
		Complete(r)
}

func (r *TeapotAppReconciler) createNewDeployment(ctx context.Context, teapotapp *k8sv1alpha1.TeapotApp) error {
	d := newDeployment(teapotapp)
	if err := r.Create(ctx, &d); err != nil {
		return fmt.Errorf("unable to create deployment. %v", err)
	}
	teapotapp.Status.DeploymentName = d.Name
	teapotapp.Status.DeploymentStatus = d.Status
	if err := r.Status().Update(ctx, teapotapp); err != nil {
		return fmt.Errorf("unable to update TeapotApp status. %v", err)
	}
	return nil
}

func (r *TeapotAppReconciler) createNewAutoscaling(ctx context.Context, teapotapp *k8sv1alpha1.TeapotApp) error {
	a := newAutoscaling(teapotapp)
	if err := r.Create(ctx, &a); err != nil {
		return fmt.Errorf("unable to create HorizontalPodAutoscaler. %v", err)
	}
	return nil
}

func (r *TeapotAppReconciler) createNewService(ctx context.Context, teapotapp *k8sv1alpha1.TeapotApp) error {
	s := newService(teapotapp)
	if err := r.Create(ctx, &s); err != nil {
		return fmt.Errorf("unable to create service. %v", err)
	}
	return nil
}

func (r *TeapotAppReconciler) createNewIngressRoute(ctx context.Context, teapotapp *k8sv1alpha1.TeapotApp) error {
	s := newIngressRoute(teapotapp, r.Config.Domain, r.Config.TLSSecret)
	if err := r.Create(ctx, &s); err != nil {
		return fmt.Errorf("unable to create ingressroute. %v", err)
	}
	return nil
}

func (r *TeapotAppReconciler) updateDeployment(ctx context.Context, teapotapp *k8sv1alpha1.TeapotApp, deployment appsv1.Deployment) error {
	deployment.Spec.Replicas = teapotapp.GetReplicas()
	i := locators.GetContainerIndexByName(deployment, teapotapp.Name)
	if i == -1 {
		return fmt.Errorf("unable to find container with name %s", teapotapp.Name)
	}
	deployment.Spec.Template.Spec.Containers[i].Resources.Limits = teapotapp.GetLimits()
	deployment.Spec.Template.Spec.Containers[i].Resources.Requests = teapotapp.GetRequests()
	deployment.Spec.Template.Spec.Containers[i].Args = teapotapp.Spec.Args
	if err := r.Update(ctx, &deployment); err != nil {
		return fmt.Errorf("unable to update Deployment. %v", err)
	}
	return nil
}

func (r *TeapotAppReconciler) updateAutoscaling(ctx context.Context, teapotapp *k8sv1alpha1.TeapotApp, autoscaling autoscalingv2beta2.HorizontalPodAutoscaler) error {
	if !teapotapp.AutoscalingEnabled() {
		if err := r.Delete(ctx, &autoscaling); err != nil {
			return fmt.Errorf("unable to delete HorizontalPodAutoscaler. %v", err)
		}
		return nil
	}
	autoscaling.Spec.MinReplicas = teapotapp.GetMinReplicas()
	autoscaling.Spec.MaxReplicas = teapotapp.GetMaxReplicas()
	if err := r.Update(ctx, &autoscaling); err != nil {
		return fmt.Errorf("unable to update HorizontalPodAutoscaler. %v", err)
	}
	return nil
}

func (r *TeapotAppReconciler) updateIngressRoute(ctx context.Context, teapotapp *k8sv1alpha1.TeapotApp, ingressRoute traefikv1alpha1.IngressRoute) error {
	ingressRoute.Spec.Routes[0].Match = "Host(`localhost`) && PathPrefix(`" + teapotapp.Spec.Path + "`)"
	if err := r.Update(ctx, &ingressRoute); err != nil {
		return fmt.Errorf("unable to update IngressRoute. %v", err)
	}
	return nil
}

func newDeployment(teapotapp *k8sv1alpha1.TeapotApp) appsv1.Deployment {
	d := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      teapotapp.Name,
			Namespace: teapotapp.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         teapotapp.APIVersion,
					Kind:               teapotapp.Kind,
					Name:               teapotapp.Name,
					UID:                teapotapp.UID,
					BlockOwnerDeletion: to.BoolPtr(true),
					Controller:         to.BoolPtr(true),
				},
			},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": teapotapp.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": teapotapp.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  teapotapp.Name,
							Image: teapotapp.Spec.Image,
							Resources: corev1.ResourceRequirements{
								Limits:   teapotapp.GetLimits(),
								Requests: teapotapp.GetRequests(),
							},
							Args: teapotapp.Spec.Args,
						},
					},
				},
			},
		},
	}
	if !teapotapp.AutoscalingEnabled() {
		d.Spec.Replicas = teapotapp.GetReplicas()
	}
	return d
}

func newAutoscaling(teapotapp *k8sv1alpha1.TeapotApp) autoscalingv2beta2.HorizontalPodAutoscaler {
	a := autoscalingv2beta2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      teapotapp.Name,
			Namespace: teapotapp.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         teapotapp.APIVersion,
					Kind:               teapotapp.Kind,
					Name:               teapotapp.Name,
					UID:                teapotapp.UID,
					BlockOwnerDeletion: to.BoolPtr(true),
					Controller:         to.BoolPtr(true),
				},
			},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "HorizontalPodAutoscaler",
			APIVersion: "autoscaling/v2beta2",
		},
		Spec: autoscalingv2beta2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2beta2.CrossVersionObjectReference{
				Kind:       "Deployment",
				Name:       teapotapp.Name,
				APIVersion: "apps/v1",
			},
			MinReplicas: teapotapp.GetMinReplicas(),
			MaxReplicas: teapotapp.GetMaxReplicas(),
			Metrics: []autoscalingv2beta2.MetricSpec{
				{
					Type: autoscalingv2beta2.ResourceMetricSourceType,
					Resource: &autoscalingv2beta2.ResourceMetricSource{
						Name: corev1.ResourceCPU,
						Target: autoscalingv2beta2.MetricTarget{
							Type:         autoscalingv2beta2.AverageValueMetricType,
							AverageValue: resource.NewQuantity(80, resource.DecimalSI),
						},
					},
				},
			},
		},
	}
	return a
}

func newService(teapotapp *k8sv1alpha1.TeapotApp) corev1.Service {
	s := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      teapotapp.Name,
			Namespace: teapotapp.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         teapotapp.APIVersion,
					Kind:               teapotapp.Kind,
					Name:               teapotapp.Name,
					UID:                teapotapp.UID,
					BlockOwnerDeletion: to.BoolPtr(true),
					Controller:         to.BoolPtr(true),
				},
			},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       80,
					TargetPort: intstr.FromInt(8080),
				},
			},
			Selector: map[string]string{
				"app": teapotapp.Name,
			},
		},
	}
	return s
}

func newIngressRoute(teapotapp *k8sv1alpha1.TeapotApp, domain, secret string) traefikv1alpha1.IngressRoute {
	return traefikv1alpha1.IngressRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      teapotapp.Name,
			Namespace: teapotapp.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         teapotapp.APIVersion,
					Kind:               teapotapp.Kind,
					Name:               teapotapp.Name,
					UID:                teapotapp.UID,
					BlockOwnerDeletion: to.BoolPtr(true),
					Controller:         to.BoolPtr(true),
				},
			},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "IngressRoute",
			APIVersion: "traefik.containo.us/v1alpha1",
		},
		Spec: traefikv1alpha1.IngressRouteSpec{
			EntryPoints: []string{"web", "websecure"},
			Routes: []traefikv1alpha1.Route{
				{
					Match: "Host(`" + domain + "`) && PathPrefix(`" + teapotapp.Spec.Path + "`)",
					Kind:  "Rule",
					Services: []traefikv1alpha1.Service{
						{
							LoadBalancerSpec: traefikv1alpha1.LoadBalancerSpec{
								Name: teapotapp.Name,
								Port: intstr.FromInt(80),
							},
						},
					},
				},
			},
			TLS: &traefikv1alpha1.TLS{
				SecretName: secret,
				Domains: []traefiktype.Domain{
					{
						Main: domain,
					},
				},
			},
		},
	}
}
