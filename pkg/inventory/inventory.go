// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package inventory

import (
	"sync"

	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/triggers/pkg/util"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Inventory keeps track of Build object details, on which it can find objects that match the
// repository URL and trigger rules.
type Inventory struct {
	m sync.Mutex

	logger logr.Logger                           // component logger
	cache  map[types.NamespacedName]TriggerRules // cache storage
}

var _ Interface = &Inventory{}

// TriggerRules keeps the source and webhook trigger information for each Build instance.
type TriggerRules struct {
	source  v1alpha1.Source
	trigger v1alpha1.Trigger
}

// SearchFn search function signature.
type SearchFn func(TriggerRules) bool

// Add insert or update an existing record.
func (i *Inventory) Add(b *v1alpha1.Build) {
	i.m.Lock()
	defer i.m.Unlock()

	if b.Spec.Trigger == nil {
		b.Spec.Trigger = &v1alpha1.Trigger{}
	}
	buildName := types.NamespacedName{Namespace: b.GetNamespace(), Name: b.GetName()}
	i.logger.V(0).Info(
		"Storing Build on the inventory",
		"build-namespace", b.GetNamespace(),
		"build-name", b.GetName(),
		"generation", b.GetGeneration(),
	)
	i.cache[buildName] = TriggerRules{
		source:  b.Spec.Source,
		trigger: *b.Spec.Trigger,
	}
}

// Remove the informed entry from the cache.
func (i *Inventory) Remove(buildName types.NamespacedName) {
	i.m.Lock()
	defer i.m.Unlock()

	i.logger.V(0).Info("Removing Build from the inventory", "build-name", buildName)
	if _, ok := i.cache[buildName]; !ok {
		i.logger.V(0).Info("Inventory entry is not found, skipping deletion!")
		return
	}
	delete(i.cache, buildName)
}

// loopByWhenType execute the search function informed against each inventory entry, when it returns
// true it returns the build name on the search results instance.
func (i *Inventory) loopByWhenType(triggerType v1alpha1.TriggerType, fn SearchFn) []SearchResult {
	found := []SearchResult{}
	for k, v := range i.cache {
		for _, when := range v.trigger.When {
			if triggerType != when.Type {
				continue
			}
			if fn(v) {
				secretName := types.NamespacedName{}
				if v.trigger.SecretRef != nil {
					secretName.Namespace = k.Namespace
					secretName.Name = v.trigger.SecretRef.Name
				}
				found = append(found, SearchResult{
					BuildName:  k,
					SecretName: secretName,
				})
			}
		}
	}
	i.logger.V(0).Info("Build search results",
		"amount", len(found), "trigger-type", triggerType)
	return found
}

// SearchForObjectRef search for builds using the ObjectRef as query parameters.
func (i *Inventory) SearchForObjectRef(
	triggerType v1alpha1.TriggerType,
	objectRef *v1alpha1.WhenObjectRef,
) []SearchResult {
	i.m.Lock()
	defer i.m.Unlock()

	return i.loopByWhenType(triggerType, func(tr TriggerRules) bool {
		for _, w := range tr.trigger.When {
			if w.ObjectRef == nil {
				continue
			}

			// checking the desired status, it must what's informed on the Build object
			if len(w.ObjectRef.Status) > 0 && len(objectRef.Status) > 0 {
				status := objectRef.Status[0]
				if !util.StringSliceContains(w.ObjectRef.Status, status) {
					continue
				}
			}

			// when name is informed it will try to match it first, otherwise the label selector
			// matching will take place
			if w.ObjectRef.Name != "" {
				if objectRef.Name != w.ObjectRef.Name {
					continue
				}
			} else {
				if len(w.ObjectRef.Selector) == 0 || len(objectRef.Selector) == 0 {
					continue
				}
				// transforming the matching labels passed to this method as a regular label selector
				// instance, which is employed to match against the Build trigger definition
				selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
					MatchLabels: w.ObjectRef.Selector,
				})
				if err != nil {
					i.logger.V(0).Error(err, "Unable to parse ObjectRef as label-selector",
						"ref-selector", w.ObjectRef.Selector)
					continue
				}
				if !selector.Matches(labels.Set(objectRef.Selector)) {
					continue
				}
			}
			return true
		}
		return false
	})
}

// SearchForGit search for builds using the Git repository details, like the URL, branch name and
// such type of information.
func (i *Inventory) SearchForGit(
	triggerType v1alpha1.TriggerType,
	repoURL string,
	branch string,
) []SearchResult {
	i.m.Lock()
	defer i.m.Unlock()

	return i.loopByWhenType(triggerType, func(tr TriggerRules) bool {
		// first thing to compare, is the repository URL, it must match in order to define the actual
		// builds that are representing the repository
		if !CompareURLs(repoURL, *tr.source.URL) {
			return false
		}

		// second part is to search for event-type and compare the informed branch, with the allowed
		// branches, configured for that build
		for _, w := range tr.trigger.When {
			if w.GitHub == nil {
				continue
			}
			for _, b := range w.GitHub.Branches {
				if branch == b {
					i.logger.V(0).Info("GitHub repository URL matches criteria",
						"repo-url", repoURL, "branch", branch)
					return true
				}
			}
		}

		return false
	})
}

// NewInventory instantiate the inventory.
func NewInventory() *Inventory {
	logger := logr.New(log.Log.GetSink())
	return &Inventory{
		logger: logger.WithName("component.inventory"),
		cache:  map[types.NamespacedName]TriggerRules{},
	}
}
