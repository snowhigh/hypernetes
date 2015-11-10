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

package etcd

import (
	"path"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/registry/generic"
	etcdgeneric "k8s.io/kubernetes/pkg/registry/generic/etcd"
	"k8s.io/kubernetes/pkg/registry/tenant"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/storage"
)

// rest implements a RESTStorage for tenants against etcd
type REST struct {
	*etcdgeneric.Etcd
	status *etcdgeneric.Etcd
}

// StatusREST implements the REST endpoint for changing the status of a tenant.
type StatusREST struct {
	store *etcdgeneric.Etcd
}

// NewREST returns a RESTStorage object that will work against tenants.
func NewREST(s storage.Interface) (*REST, *StatusREST) {
	prefix := "/tenants"
	store := &etcdgeneric.Etcd{
		NewFunc:     func() runtime.Object { return &api.Tenant{} },
		NewListFunc: func() runtime.Object { return &api.TenantList{} },
		KeyRootFunc: func(ctx api.Context) string {
			return prefix
		},
		KeyFunc: func(ctx api.Context, name string) (string, error) {
			return path.Join(prefix, name), nil
		},
		ObjectNameFunc: func(obj runtime.Object) (string, error) {
			return obj.(*api.Tenant).Name, nil
		},
		PredicateFunc: func(label labels.Selector, field fields.Selector) generic.Matcher {
			return tenant.MatchTenant(label, field)
		},
		EndpointName: "tenants",

		CreateStrategy:      tenant.Strategy,
		UpdateStrategy:      tenant.Strategy,
		ReturnDeletedObject: true,

		Storage: s,
	}

	statusStore := *store
	statusStore.UpdateStrategy = tenant.StatusStrategy

	return &REST{Etcd: store, status: &statusStore}, &StatusREST{store: &statusStore}
}

// Delete enforces life-cycle rules for tenant termination
func (r *REST) Delete(ctx api.Context, name string, options *api.DeleteOptions) (runtime.Object, error) {
	nsObj, err := r.Get(ctx, name)
	if err != nil {
		return nil, err
	}

	tenant := nsObj.(*api.Tenant)

	// upon first request to delete, we switch the phase to start tenant termination
	if tenant.DeletionTimestamp.IsZero() {
		now := unversioned.Now()
		tenant.DeletionTimestamp = &now
		tenant.Status.Phase = api.TenantTerminating
		result, _, err := r.status.Update(ctx, tenant)
		return result, err
	}

	return r.Etcd.Delete(ctx, name, nil)
}

func (r *StatusREST) New() runtime.Object {
	return r.store.New()
}

// Update alters the status subset of an object.
func (r *StatusREST) Update(ctx api.Context, obj runtime.Object) (runtime.Object, bool, error) {
	return r.store.Update(ctx, obj)
}
