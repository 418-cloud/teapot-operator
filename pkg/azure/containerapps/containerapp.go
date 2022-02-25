package containerapps

import (
	"context"
	"strconv"

	azurev1alpha1 "github.com/418-cloud/teapot-operator/apis/azure/v1alpha1"
	"github.com/418-cloud/teapot-operator/pkg/utils/to"
	"github.com/Azure/azure-sdk-for-go/services/web/mgmt/2021-03-01/web"
	"github.com/Azure/go-autorest/autorest"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func CreateAzureContainerApp(ctx context.Context, authorizer autorest.Authorizer, app azurev1alpha1.ContainerApp, subscription, resourcegroup, location, kubeenvID string) (*web.ContainerApp, error) {
	logger := log.FromContext(ctx).WithValues("subscription", subscription, "resourcegroup", resourcegroup)
	logger.Info("creating new azure container app", "name", app.Name)
	client := web.NewContainerAppsClient(subscription)
	client.Authorizer = authorizer
	cpu := .25
	memory := ".5Gi"
	if app.Spec.ContainersTemplate.Resources != nil {
		var err error
		cpu, err = strconv.ParseFloat(app.Spec.ContainersTemplate.Resources.CPU, 32)
		if err != nil {
			logger.Error(err, "failed to parse cpu")
			return nil, err
		}
		memory = app.Spec.ContainersTemplate.Resources.Memory
	}
	c := web.ContainerApp{
		Name:     &app.Name,
		Location: &location,
		ContainerAppProperties: &web.ContainerAppProperties{
			KubeEnvironmentID: &kubeenvID,
			Configuration: &web.Configuration{
				Ingress: &web.Ingress{
					External:   to.BoolPtr(true),
					TargetPort: &app.Spec.TargetPort,
					Transport:  web.IngressTransportMethodAuto,
				},
			},
			Template: &web.Template{
				Containers: &[]web.Container{
					{
						Name:    &app.Spec.ContainersTemplate.Name,
						Image:   &app.Spec.ContainersTemplate.Image,
						Command: &[]string{},
						Resources: &web.ContainerResources{
							CPU:    &cpu,
							Memory: &memory,
						},
					},
				},
			},
		},
	}
	future, err := client.CreateOrUpdate(ctx, resourcegroup, app.Name, c)
	if err != nil {
		logger.Error(err, "failed to create new container app")
		return nil, err
	}
	if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
		logger.Error(err, "failed to wait for completion")
		return nil, err
	}
	return &c, nil
}

func GetAzureContainerApp(ctx context.Context, authorizer autorest.Authorizer, env azurev1alpha1.ContainerApp, subscription, resourcegroup string) (*web.ContainerApp, error) {
	logger := log.FromContext(ctx).WithValues("subscription", subscription, "resourcegroup", resourcegroup)
	client := web.NewContainerAppsClient(subscription)
	client.Authorizer = authorizer
	c, err := client.Get(ctx, resourcegroup, env.Name)
	if err != nil {
		logger.Error(err, "failed to get container app")
		return nil, err
	}
	return &c, nil
}
