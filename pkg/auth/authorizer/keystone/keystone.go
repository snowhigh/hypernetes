/*
Copyright 2015 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package keystone

import (
	"errors"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/auth/authorizer"
	client "k8s.io/kubernetes/pkg/client/unversioned"

	"github.com/golang/glog"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack"
	"github.com/rackspace/gophercloud/openstack/identity/v2/tenants"
	"github.com/rackspace/gophercloud/pagination"
)

type authConfig struct {
	AuthUrl  string `json:"auth-url"`
	Username string `json:"user-name"`
	Password string `json:"password"`
	TokenID  string `json:"token"`
	Tenant   string `json:"tenant"`
	TenantID string `json:"tenantID"`
}

type OpenstackClient struct {
	provider   *gophercloud.ProviderClient
	authClient *gophercloud.ServiceClient
	config     *authConfig
}

type keystoneAuthorizer struct {
	kubeClient client.Interface
	osClient   OpenstackInterface
	authUrl    string
}

func newOpenstackClient(config *authConfig) (*OpenstackClient, error) {

	if config == nil {
		err := errors.New("no OpenStack cloud provider config file given")
		return nil, err
	}

	opts := gophercloud.AuthOptions{
		IdentityEndpoint: config.AuthUrl,
		Username:         config.Username,
		Password:         config.Password,
		TenantName:       config.Tenant,
		TenantID:         config.TenantID,
		AllowReauth:      false,
	}

	provider, err := openstack.AuthenticatedClient(opts)
	if err != nil {
		glog.Info("Failed: Starting openstack authenticate client")
		return nil, err
	}
	authClient := openstack.NewIdentityV2(provider)

	return &OpenstackClient{
		provider,
		authClient,
		config,
	}, nil
}

func NewKeystoneAuthorizer(kubeClient client.Interface, authUrl string) (*keystoneAuthorizer, error) {

	ka := &keystoneAuthorizer{
		kubeClient: kubeClient,
		authUrl:    authUrl,
	}
	return ka, nil
}

// Authorizer implements authorizer.Authorize
func (ka *keystoneAuthorizer) Authorize(a authorizer.Attributes) (string, error) {

	var (
		tenantName string
		ns         *api.Namespace
		err        error
	)
	if a.GetNamespace() != "" {
		ns, err = ka.kubeClient.Namespaces().Get(a.GetNamespace())
		if err != nil {
			return "", err
		}
		tenantName = ns.Tenant
	} else {
		if a.GetTenant() != "" {
			te, err := ka.kubeClient.Tenants().Get(a.GetTenant())
			if err != nil {
				return "", err
			}
			tenantName = te.Name
		}
	}
	if authorizer.IsWhiteListedUser(a.GetUserName()) {
		if a.GetUserName() != api.UserAdmin {
			return tenantName, nil
		} else {
			return api.TenantDefault, nil
		}
	} else {
		if !a.IsReadOnly() && a.GetResource() == "tenants" {
			return "", errors.New("only admin can write tenant")
		}
	}

	authConfig := &authConfig{
		AuthUrl:  ka.authUrl,
		Username: a.GetUserName(),
		Password: a.GetPassword(),
	}
	osClient, err := newOpenstackClient(authConfig)
	if err != nil {
		glog.Errorf("%v", err)
		return "", err
	}

	tenant, err := osClient.getTenant()
	if err != nil {
		glog.Errorf("%v", err)
		return "", err
	}
	if tenantName == "" || tenantName == tenant.Name {
		return tenant.Name, nil
	}
	return "", errors.New("Keystone authorization failed")
}

func (osClient *OpenstackClient) getTenant() (tenant *tenants.Tenant, err error) {
	tenantList := make([]tenants.Tenant, 0)
	opts := tenants.ListOpts{}
	pager := tenants.List(osClient.authClient, &opts)
	err = pager.EachPage(func(page pagination.Page) (bool, error) {
		tenantList, err = tenants.ExtractTenants(page)
		if err != nil {
			return false, err
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	if len(tenantList) > 1 {
		return nil, errors.New("too much tenants")
	} else if len(tenantList) != 1 {
		return nil, errors.New("no tenants")
	}
	return &tenantList[0], nil
}
