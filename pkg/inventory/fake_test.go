// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package inventory

import (
	"log"
	"sync"

	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/triggers/test/stubs"
	"k8s.io/apimachinery/pkg/types"
)

// FakeInventory testing instance of Inventory, adds all objects o the local cache, and returns all
// of them on the search queries.
type FakeInventory struct {
	m sync.Mutex

	cache map[types.NamespacedName]*buildapi.Build
}

var _ Interface = &FakeInventory{}

// Contains checks if the informed key is in the cache.
func (i *FakeInventory) Contains(name string) bool {
	i.m.Lock()
	defer i.m.Unlock()

	log.Printf("Cheking if Build %q is cached", name)
	_, ok := i.cache[types.NamespacedName{Namespace: stubs.Namespace, Name: name}]
	return ok
}

// Add adds a Build to the cache.
func (i *FakeInventory) Add(b *buildapi.Build) {
	i.m.Lock()
	defer i.m.Unlock()

	key := types.NamespacedName{Namespace: b.GetNamespace(), Name: b.GetName()}
	log.Printf("Adding Build %q to the inventory", key)
	i.cache[key] = b
}

// Remove removes a Build from the cache.
func (i *FakeInventory) Remove(key types.NamespacedName) {
	i.m.Lock()
	defer i.m.Unlock()

	log.Printf("Removing Build %q from the inventory", key)
	delete(i.cache, key)
}

// search returns all instances as SearchResult slice.
func (i *FakeInventory) search() []SearchResult {
	searchResults := []SearchResult{}
	if len(i.cache) == 0 {
		return searchResults
	}
	for _, b := range i.cache {
		secretName := types.NamespacedName{}
		if b.Spec.Trigger != nil &&
			b.Spec.Trigger.TriggerSecret != nil {
			secretName.Namespace = b.GetNamespace()
			secretName.Name = *b.Spec.Trigger.TriggerSecret
		}
		searchResults = append(searchResults, SearchResult{
			BuildName:  types.NamespacedName{Namespace: b.GetNamespace(), Name: b.GetName()},
			SecretName: secretName,
		})
	}
	return searchResults
}

// SearchForObjectRef returns all Builds in cache.
func (i *FakeInventory) SearchForObjectRef(
	buildapi.TriggerType,
	*buildapi.WhenObjectRef,
) []SearchResult {
	i.m.Lock()
	defer i.m.Unlock()

	return i.search()
}

// SearchForGit returns all Builds in cache.
func (i *FakeInventory) SearchForGit(buildapi.TriggerType, string, string) []SearchResult {
	i.m.Lock()
	defer i.m.Unlock()

	return i.search()
}

// NewFakeInventory instante a fake inventory for testing.
func NewFakeInventory() *FakeInventory {
	return &FakeInventory{
		cache: map[types.NamespacedName]*buildapi.Build{},
	}
}
