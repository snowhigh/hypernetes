package apiserver

import (
	"k8s.io/kubernetes/pkg/hypernetes/auth/authorizer"
	"k8s.io/kubernetes/pkg/hypernetes/auth/authorizer/tbac"
	"k8s.io/kubernetes/pkg/hypernetes/auth/authorizer/union"
	"k8s.io/kubernetes/pkg/hypernetes/storage"
)

// Attributes implements authorizer.Attributes interface.
type Attributes struct {
	// TODO: add fields and methods when authorizer.Attributes is completed.
}

// NewAuthorizerFromAuthorizationConfig returns the right sort of union of multiple authorizer.Authorizer objects
// based on the authorizationMode or an error.  authorizationMode should be a comma separated values
// of AuthorizationModeChoices.
func NewAuthorizerFromAuthorizationConfig(stor storage.Interface) (authorizer.Authorizer, error) {
	tbacAuthorizer, err := tbac.NewTbacAuthorizer(stor)
	if err != nil {
		return nil, err
	}

	return union.New([]authorizer.Authorizer{tbacAuthorizer}...), nil
}
