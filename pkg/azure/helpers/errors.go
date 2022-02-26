package helpers

import "github.com/Azure/go-autorest/autorest"

func ResourceNotFound(err error) bool {
	if e, ok := err.(autorest.DetailedError); ok && e.Response.StatusCode == 404 {
		return true
	}
	return false
}
