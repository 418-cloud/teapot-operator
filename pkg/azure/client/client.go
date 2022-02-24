package client

import (
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
)

type Config struct {
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	TenantId     string `json:"tenantId"`
	Subscription string `json:"subscriptionId"`
}

var (
	cloudName string = "AzurePublicCloud"
)

func GetAuthorizer(config Config) (autorest.Authorizer, error) {
	environment, err := azure.EnvironmentFromName(cloudName)
	if err != nil {
		return nil, err
	}
	oauthConfig, err := adal.NewOAuthConfig(environment.ActiveDirectoryEndpoint, config.TenantId)
	if err != nil {
		return nil, err
	}
	token, err := adal.NewServicePrincipalToken(
		*oauthConfig, config.ClientID, config.ClientSecret, environment.ResourceManagerEndpoint,
	)
	if err != nil {
		return nil, err
	}
	return autorest.NewBearerAuthorizer(token), nil
}
