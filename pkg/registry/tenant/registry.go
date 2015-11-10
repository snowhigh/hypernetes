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

package tenant

import (
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/rest"
	"k8s.io/kubernetes/pkg/watch"
)

// Registry is an interface implemented by things that know how to store Tenant objects.
type Registry interface {
	// ListTenants obtains a list of tenants having labels which match selector.
	ListTenants(ctx api.Context, options *api.ListOptions) (*api.TenantList, error)
	// Watch for new/changed/deleted tenants
	WatchTenants(ctx api.Context, options *api.ListOptions) (watch.Interface, error)
	// Get a specific tenant
	GetTenant(ctx api.Context, tenantID string) (*api.Tenant, error)
	// Create a tenant based on a specification.
	CreateTenant(ctx api.Context, tenant *api.Tenant) error
	// Update an existing tenant
	UpdateTenant(ctx api.Context, tenant *api.Tenant) error
	// Delete an existing tenant
	DeleteTenant(ctx api.Context, tenantID string) error
}

// storage puts strong typing around storage calls
type storage struct {
	rest.StandardStorage
}

// NewRegistry returns a new Registry interface for the given Storage. Any mismatched
// types will panic.
func NewRegistry(s rest.StandardStorage) Registry {
	return &storage{s}
}

func (s *storage) ListTenants(ctx api.Context, options *api.ListOptions) (*api.TenantList, error) {
	obj, err := s.List(ctx, options)
	if err != nil {
		return nil, err
	}
	return obj.(*api.TenantList), nil
}

func (s *storage) WatchTenants(ctx api.Context, options *api.ListOptions) (watch.Interface, error) {
	return s.Watch(ctx, options)
}

func (s *storage) GetTenant(ctx api.Context, tenantName string) (*api.Tenant, error) {
	obj, err := s.Get(ctx, tenantName)
	if err != nil {
		return nil, err
	}
	return obj.(*api.Tenant), nil
}

func (s *storage) CreateTenant(ctx api.Context, tenant *api.Tenant) error {
	_, err := s.Create(ctx, tenant)
	return err
}

func (s *storage) UpdateTenant(ctx api.Context, tenant *api.Tenant) error {
	_, _, err := s.Update(ctx, tenant)
	return err
}

func (s *storage) DeleteTenant(ctx api.Context, tenantID string) error {
	_, err := s.Delete(ctx, tenantID, nil)
	return err
}
