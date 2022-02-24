package client

import (
	"github.com/Azure/go-autorest/autorest"
)

func GetAuthorizer() (autorest.Authorizer) {
	return autorest.NewBearerAuthorizerCallback(nil, clientAssertionBearerAuthorizerCallback)
}
