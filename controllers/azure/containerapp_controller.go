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
	"fmt"
	"time"

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

// ContainerAppReconciler reconciles a ContainerApp object
type ContainerAppReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	Config       configv2.ProjectConfig
	ClientConfig azureclient.ClientConfig
}

//+kubebuilder:rbac:groups=azure.418.cloud,resources=containerapps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=azure.418.cloud,resources=containerapps/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=azure.418.cloud,resources=containerapps/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ContainerApp object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *ContainerAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	logger := log.FromContext(ctx)

	// TODO(user): your logic here
	var containerapp azurev1alpha1.ContainerApp
	if err := r.Get(ctx, req.NamespacedName, &containerapp); err != nil {
		if !errors.IsNotFound(err) {
			logger.Error(err, "unable to fetch TeapotApp")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	var env azurev1alpha1.ContainerEnvironment
	if err := r.Get(ctx, client.ObjectKey{Namespace: containerapp.Namespace, Name: containerapp.Spec.ContainerEnvironmentName}, &env); err != nil {
		if !errors.IsNotFound(err) {
			logger.Error(err, "unable to fetch ContainerEnvironment")
			containerapp.Status.LatestStatus = fmt.Sprintf("Failed to get ContainerEnvironment %s", err)
			if err := r.Status().Update(ctx, &containerapp); err != nil {
				logger.Error(err, "unable to update ContainerApp status")
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		containerapp.Status.LatestStatus = fmt.Sprintf("ContainerEnvironment %s not found", containerapp.Spec.ContainerEnvironmentName)
		if err := r.Status().Update(ctx, &containerapp); err != nil {
			logger.Error(err, "unable to update ContainerApp status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	a, err := azureclient.GetAuthorizer(r.ClientConfig)
	if err != nil {
		logger.Error(err, "unable to get authorizer")
		containerapp.Status.LatestStatus = fmt.Sprintf("Failed to get authorizer %s", err)
		if err := r.Status().Update(ctx, &containerapp); err != nil {
			logger.Error(err, "unable to update ContainerApp status")
		}
		return ctrl.Result{}, err
	}
	app, err := containerapps.GetAzureContainerApp(ctx, a, containerapp, r.Config.Subscription, r.Config.ResourceGroupName)
	if e, ok := err.(autorest.DetailedError); ok && e.Response.StatusCode == 404 {
		logger.Info("Creating new kube environment")
		app, err = containerapps.CreateAzureContainerApp(ctx, a, containerapp, r.Config.Subscription, r.Config.ResourceGroupName, env.Spec.Location, env.Status.EnvironmentID)
		if err != nil {
			logger.Error(err, "unable to create containerapp")
			return ctrl.Result{}, err
		}
	} else if err != nil {
		logger.Error(err, "unable to get containerapp")
		return ctrl.Result{}, err
	}
	containerapp.Status.FQDN = fmt.Sprintf("https://%s", *app.LatestRevisionFqdn)
	containerapp.Status.LatestStatus = fmt.Sprintf("ContainerApp %s is %s", containerapp.Name, app.ProvisioningState)
	if err := r.Status().Update(ctx, &containerapp); err != nil {
		logger.Error(err, "unable to update ContainerApp status")
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: 60 * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ContainerAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&azurev1alpha1.ContainerApp{}).
		Complete(r)
}
