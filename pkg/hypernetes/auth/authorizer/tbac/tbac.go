// Tenant based access control
package tbac

import (
	"errors"
	"fmt"

	"k8s.io/kubernetes/pkg/hypernetes/auth"
	"k8s.io/kubernetes/pkg/hypernetes/auth/authorizer"
	"k8s.io/kubernetes/pkg/hypernetes/storage"
)

var UnAuthorizerResource error = errors.New("the resource can not be authorized to use")

type tbacAuthorizer struct {
	stor storage.Interface
}

func NewTbacAuthorizer(stor storage.Interface) (*tbacAuthorizer, error) {
	ta := &tbacAuthorizer{
		stor: stor,
	}
	return ta, nil
}

func (ta *tbacAuthorizer) Authorize(a authorizer.Attributes) error {
	var (
		resource string = a.GetResource()
		name     string = a.GetName()
		// AccessKey
		username string = a.GetUserName()
		// Tenant ID
		tenant string

		authEntry   *auth.AuthItem   = &auth.AuthItem{}
		tenantEntry *auth.TenantItem = &auth.TenantItem{}
		err         error
	)
	// For action without specificed resource
	if name == "" {
		return nil
	}
	err = ta.stor.Get(auth.Database, auth.AuthTable, "accesskey", username, authEntry)
	if err != nil {
		return err
	}
	tenant = authEntry.TenantID
	if tenant == "" {
		return fmt.Errorf("Tenant is null for AccessKey(%s)", username)
	}

	err = ta.stor.Get(auth.Database, auth.TenantTable, "tenant", tenant, tenantEntry)
	if err != nil {
		return err
	}
	switch resource {
	case "containers":
		if _, ok := tenantEntry.Containers[name]; !ok {
			return UnAuthorizerResource
		}
	case "volumes":
		if _, ok := tenantEntry.Volumes[name]; !ok {
			return UnAuthorizerResource
		}
	case "networks":
		if _, ok := tenantEntry.Networks[name]; !ok {
			return UnAuthorizerResource
		}
	default:
		return fmt.Errorf("unknown resource type: %s", resource)
	}

	return nil
}
