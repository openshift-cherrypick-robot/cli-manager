/*
Copyright The Kubernetes Authors.

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

// Code generated by lister-gen. DO NOT EDIT.

package v1beta1

import (
	v1beta1 "k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/listers"
	"k8s.io/client-go/tools/cache"
)

// IngressClassLister helps list IngressClasses.
// All objects returned here must be treated as read-only.
type IngressClassLister interface {
	// List lists all IngressClasses in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1beta1.IngressClass, err error)
	// Get retrieves the IngressClass from the index for a given name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1beta1.IngressClass, error)
	IngressClassListerExpansion
}

// ingressClassLister implements the IngressClassLister interface.
type ingressClassLister struct {
	listers.ResourceIndexer[*v1beta1.IngressClass]
}

// NewIngressClassLister returns a new IngressClassLister.
func NewIngressClassLister(indexer cache.Indexer) IngressClassLister {
	return &ingressClassLister{listers.New[*v1beta1.IngressClass](indexer, v1beta1.Resource("ingressclass"))}
}
