package containerapps

import (
	"context"

	azurev1alpha1 "github.com/418-cloud/teapot-operator/apis/azure/v1alpha1"
	"github.com/418-cloud/teapot-operator/pkg/utils/to"
	"github.com/Azure/azure-sdk-for-go/services/operationalinsights/mgmt/2020-08-01/operationalinsights"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/web/mgmt/web"
	"github.com/Azure/go-autorest/autorest"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func CreateNewKubeEnvironment(ctx context.Context, authorizer autorest.Authorizer, env azurev1alpha1.ContainerEnvironment, subscription, resourcegroup string) error {
	log := log.FromContext(ctx).WithValues("subscription", subscription, "resourcegroup", resourcegroup)
	id, key, err := CreateNewLogAnalytics(ctx, authorizer, env, subscription, resourcegroup)
	if err != nil {
		log.Error(err, "failed to create new log analytics workspace")
		return err
	}
	log.Info("created new log analytics workspace", "workspace", id)
	client := web.NewKubeEnvironmentsClient(subscription)
	client.Authorizer = authorizer
	kubeenv := web.KubeEnvironment{
		KubeEnvironmentProperties: &web.KubeEnvironmentProperties{
			InternalLoadBalancerEnabled: to.BoolPtr(false),
			AppLogsConfiguration: &web.AppLogsConfiguration{
				LogAnalyticsConfiguration: &web.LogAnalyticsConfiguration{
					CustomerID: &id,
					SharedKey: &key,
				},
			},
		},
	}
	future, err := client.CreateOrUpdate(ctx, resourcegroup, env.Name, kubeenv)
	if err != nil {
		log.Error(err, "failed to create new kube environment")
		return err
	}
	if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
		log.Error(err, "failed to wait for completion")
		return err
	}
	_, err = future.Result(client)
	if err != nil {
		log.Error(err, "failed to get result")
		return err
	}
	
	return nil
}

func CreateNewLogAnalytics(ctx context.Context, authorizer autorest.Authorizer, env azurev1alpha1.ContainerEnvironment, subscription, resourcegroup string) (customerID, sharedKey string, err error) {
	client := operationalinsights.NewWorkspacesClient(subscription)
	client.Authorizer = authorizer
	workspace := operationalinsights.Workspace{
		WorkspaceProperties: &operationalinsights.WorkspaceProperties{
			RetentionInDays: to.Int32Ptr(10),
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