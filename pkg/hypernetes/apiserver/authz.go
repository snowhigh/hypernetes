package apiserver

import (
	"errors"

	"k8s.io/kubernetes/pkg/auth/authorizer"
	"k8s.io/kubernetes/pkg/auth/authorizer/union"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/hypernetes/auth/authorizer/keystone"
)

// Attributes implements authorizer.Attributes interface.
type Attributes struct {
	// TODO: add fields and methods when authorizer.Attributes is completed.
}

// NewAuthorizerFromAuthorizationConfig returns the right sort of union of multiple authorizer.Authorizer objects
// based on the authorizationMode or an error.  authorizationMode should be a comma separated values
// of AuthorizationModeChoices.
func NewAuthorizerFromAuthorizationConfig(client client.Interface, keystoneURL string) (authorizer.Authorizer, error) {
	if len(keystoneURL) == 0 {
		return nil, errors.New("Atleast one authorization mode should be passed")
	}
	keystoneAuthorizer, err := keystone.NewKeystoneAuthorizer(client, keystoneURL)
	if err != nil {
		return nil, err
	}

	return union.New([]authorizer.Authorizer{keystoneAuthorizer}...), nil
}
