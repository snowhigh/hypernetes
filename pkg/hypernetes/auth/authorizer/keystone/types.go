package keystone

import (
	"github.com/rackspace/gophercloud/openstack/identity/v2/tenants"
)

// Interface is an abstract interface for testability.  It abstracts the interface to Keystone.
type OpenstackInterface interface {
	getTenant() (*tenants.Tenant, error)
}
