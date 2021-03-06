package containerapps

import (
	"context"

	azurev1alpha1 "github.com/418-cloud/teapot-operator/apis/azure/v1alpha1"
	azurehelpers "github.com/418-cloud/teapot-operator/pkg/azure/helpers"
	"github.com/418-cloud/teapot-operator/pkg/utils/to"
	"github.com/Azure/azure-sdk-for-go/services/operationalinsights/mgmt/2020-08-01/operationalinsights"
	"github.com/Azure/azure-sdk-for-go/services/web/mgmt/2021-03-01/web"
	"github.com/Azure/go-autorest/autorest"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func CreateNewKubeEnvironment(ctx context.Context, authorizer autorest.Authorizer, env azurev1alpha1.ContainerEnvironment, subscription, resourcegroup string) (*web.KubeEnvironment, error) {
	logger := log.FromContext(ctx).WithValues("subscription", subscription, "resourcegroup", resourcegroup)
	id, key, err := getLogWorkspace(ctx, authorizer, env, subscription, resourcegroup)
	if azurehelpers.ResourceNotFound(err) {
		logger.Info("creating new log workspace")
		id, key, err = createNewLogWorkspace(ctx, authorizer, env, subscription, resourcegroup)
		if err != nil {
			logger.Error(err, "failed to create new log analytics workspace")
			return nil, err
		}
		logger.Info("created new log analytics workspace", "workspace", id)
	} else if err != nil {
		logger.Error(err, "failed to get log analytics workspace")
		return nil, err
	}
	client := web.NewKubeEnvironmentsClient(subscription)

	client.Authorizer = authorizer
	logger.Info("created new kube environment client", "subscription", subscription, "resourcegroup", resourcegroup)
	kubeenv := web.KubeEnvironment{
		Location: &env.Spec.Location,
		KubeEnvironmentProperties: &web.KubeEnvironmentProperties{
			EnvironmentType:             to.StringPtr("Managed"),
			InternalLoadBalancerEnabled: to.BoolPtr(false),
			AppLogsConfiguration: &web.AppLogsConfiguration{
				Destination: to.StringPtr("log-analytics"),
				LogAnalyticsConfiguration: &web.LogAnalyticsConfiguration{
					CustomerID: &id,
					SharedKey:  &key,
				},
			},
		},
	}
	future, err := client.CreateOrUpdate(ctx, resourcegroup, env.Name, kubeenv)
	if err != nil {
		logger.Error(err, "failed to create new kube environment")
		return nil, err
	}
	if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
		logger.Error(err, "failed to wait for completion")
		return nil, err
	}
	ke, err := future.Result(client)
	if err != nil {
		logger.Error(err, "failed to get result")
		return nil, err
	}

	return &ke, nil
}

func GetKubeEnvironment(ctx context.Context, authorizer autorest.Authorizer, env azurev1alpha1.ContainerEnvironment, subscription, resourcegroup string) (*web.KubeEnvironment, error) {
	logger := log.FromContext(ctx).WithValues("func", "GetKubeEnvironment", "subscription", subscription, "resourcegroup", resourcegroup)
	client := web.NewKubeEnvironmentsClient(subscription)
	logger.Info("setting authorizer")
	client.Authorizer = authorizer
	logger.Info("fetching kube environment")
	ke, err := client.Get(ctx, resourcegroup, env.Name)
	if err != nil {
		return nil, err
	}
	return &ke, nil
}

func getLogWorkspace(ctx context.Context, authorizer autorest.Authorizer, env azurev1alpha1.ContainerEnvironment, subscription, resourcegroup string) (customerID, sharedKey string, err error) {
	client := operationalinsights.NewWorkspacesClient(subscription)
	client.Authorizer = authorizer
	w, err := client.Get(ctx, resourcegroup, env.Name)
	if err != nil {
		return
	}
	customerID = *w.CustomerID
	sharedKeysClient := operationalinsights.NewSharedKeysClient(subscription)
	sharedKeysClient.Authorizer = authorizer
	k, err := sharedKeysClient.GetSharedKeys(ctx, resourcegroup, env.Name)
	if err != nil {
		return
	}
	sharedKey = *k.PrimarySharedKey
	return
}

func createNewLogWorkspace(ctx context.Context, authorizer autorest.Authorizer, env azurev1alpha1.ContainerEnvironment, subscription, resourcegroup string) (customerID, sharedKey string, err error) {
	client := operationalinsights.NewWorkspacesClient(subscription)
	client.Authorizer = authorizer
	workspace := operationalinsights.Workspace{
		Location: &env.Spec.Location,
		WorkspaceProperties: &operationalinsights.WorkspaceProperties{
			RetentionInDays: to.Int32Ptr(30),
		},
	}
	future, err := client.CreateOrUpdate(ctx, resourcegroup, env.Name, workspace)
	if err != nil {
		return
	}
	if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return
	}
	w, err := future.Result(client)
	if err != nil {
		return
	}
	customerID = *w.CustomerID
	sharedKeysClient := operationalinsights.NewSharedKeysClient(subscription)
	sharedKeysClient.Authorizer = authorizer
	k, err := sharedKeysClient.GetSharedKeys(ctx, resourcegroup, env.Name)
	if err != nil {
		return
	}
	sharedKey = *k.PrimarySharedKey
	return
}
