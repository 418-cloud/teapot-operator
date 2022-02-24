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

package azure

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	azurev1alpha1 "github.com/418-cloud/teapot-operator/apis/azure/v1alpha1"
	configv2 "github.com/418-cloud/teapot-operator/apis/config/v2"
	azureclient "github.com/418-cloud/teapot-operator/pkg/azure/client"
	"github.com/418-cloud/teapot-operator/pkg/azure/containerapps"
	"github.com/Azure/go-autorest/autorest"
)

// ContainerEnvironmentReconciler reconciles a ContainerEnvironment object
type ContainerEnvironmentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Config configv2.ProjectConfig
}

//+kubebuilder:rbac:groups=azure.418.cloud,resources=containerenvironments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=azure.418.cloud,resources=containerenvironments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=azure.418.cloud,resources=containerenvironments/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ContainerEnvironment object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *ContainerEnvironmentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// TODO(user): your logic here
	var containerenvironment azurev1alpha1.ContainerEnvironment
	if err := r.Get(ctx, req.NamespacedName, &containerenvironment); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err, "unable to fetch TeapotApp")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	authorizer := azureclient.GetAuthorizer()
	kubeEnv, err := containerapps.GetKubeEnvironment(ctx, authorizer, containerenvironment, r.Config.Subscription, r.Config.ResourceGroupName)
	if e, ok := err.(autorest.DetailedError); ok && e.Response.StatusCode == 404 {
		kubeEnv, err = containerapps.CreateNewKubeEnvironment(ctx, authorizer, containerenvironment, r.Config.Subscription, r.Config.ResourceGroupName)
		if err != nil {
			log.Error(err, "unable to create kube environment")
			return ctrl.Result{}, err
		}
	} else if err != nil {
		log.Error(err, "unable to get kube environment")
		return ctrl.Result{}, err
	}
	containerenvironment.Status.EnvironmentID = *kubeEnv.ID
	containerenvironment.Status.EnvironmentName = *kubeEnv.Name
	containerenvironment.Status.EnvironmentIP = *kubeEnv.StaticIP
	containerenvironment.Status.EnvironmentStatus = kubeEnv.Status

	if err := r.Status().Update(ctx, &containerenvironment); err != nil {
		log.Error(err, "unable to update status")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ContainerEnvironmentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&azurev1alpha1.ContainerEnvironment{}).
		Complete(r)
}
