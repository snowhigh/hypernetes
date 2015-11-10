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
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/registry/generic"
	etcdgeneric "k8s.io/kubernetes/pkg/registry/generic/etcd"
	"k8s.io/kubernetes/pkg/registry/network"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/storage"
)

// rest implements a RESTStorage for networks against etcd
type REST struct {
	*etcdgeneric.Etcd
	status *etcdgeneric.Etcd
}

// StatusREST implements the REST endpoint for changing the status of a network.
type StatusREST struct {
	store *etcdgeneric.Etcd
}

// NewREST returns a RESTStorage object that will work against networks.
func NewREST(s storage.Interface, storageFactory storage.StorageFactory) (*REST, *StatusREST) {
	prefix := "/networks"

	newListFunc := func() runtime.Object { return &api.NetworkList{} }
	storageInterface := storageFactory(
		s, 100, nil, &api.Network{}, prefix, true, newListFunc)

	store := &etcdgeneric.Etcd{
		NewFunc:     func() runtime.Object { return &api.Network{} },
		NewListFunc: newListFunc,
		KeyRootFunc: func(ctx api.Context) string {
			return prefix
		},
		KeyFunc: func(ctx api.Context, name string) (string, error) {
			return prefix + "/" + name, nil
		},
		ObjectNameFunc: func(obj runtime.Object) (string, error) {
			return obj.(*api.Network).Name, nil
		},
		PredicateFunc: func(label labels.Selector, field fields.Selector) generic.Matcher {
			return network.MatchNetwork(label, field)
		},
		EndpointName: "networks",

		CreateStrategy:      network.Strategy,
		UpdateStrategy:      network.Strategy,
		ReturnDeletedObject: true,

		Storage: storageInterface,
	}

	statusStore := *store
	statusStore.UpdateStrategy = network.StatusStrategy

	return &REST{Etcd: store, status: &statusStore}, &StatusREST{store: &statusStore}
}

// Delete enforces life-cycle rules for network termination
func (r *REST) Delete(ctx api.Context, name string, options *api.DeleteOptions) (runtime.Object, error) {
	return r.Etcd.Delete(ctx, name, nil)
}

func (r *StatusREST) New() runtime.Object {
	return r.store.New()
}

// Update alters the status subset of an object.
func (r *StatusREST) Update(ctx api.Context, obj runtime.Object) (runtime.Object, bool, error) {
	return r.store.Update(ctx, obj)
}
