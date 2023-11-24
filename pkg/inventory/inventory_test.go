// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package inventory

import (
	"reflect"
	"testing"

	"github.com/onsi/gomega"
	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/triggers/test/stubs"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var buildWithTrigger = stubs.ShipwrightBuildWithTriggers(
	"ghcr.io/shipwright-io",
	"name",
	stubs.TriggerWhenPushToMain,
)

func TestInventory(t *testing.T) {
	g := gomega.NewWithT(t)

	i := NewInventory()

	t.Run("adding empty inventory item", func(_ *testing.T) {
		i.Add(buildWithTrigger)
		g.Expect(len(i.cache)).To(gomega.Equal(1))

		_, exists := i.cache[types.NamespacedName{Namespace: stubs.Namespace, Name: "name"}]
		g.Expect(exists).To(gomega.BeTrue())
	})

	t.Run("remove inventory item", func(_ *testing.T) {
		i.Remove(types.NamespacedName{Namespace: stubs.Namespace, Name: "name"})
		g.Expect(len(i.cache)).To(gomega.Equal(0))
	})
}

func TestInventorySearchForgit(t *testing.T) {
	g := gomega.NewWithT(t)

	i := NewInventory()
	i.Add(buildWithTrigger)

	t.Run("should not find any results", func(_ *testing.T) {
		found := i.SearchForGit(buildapi.GitHubWebHookTrigger, "", "")
		g.Expect(len(found)).To(gomega.Equal(0))

		found = i.SearchForGit(buildapi.GitHubWebHookTrigger, stubs.RepoURL, "")
		g.Expect(len(found)).To(gomega.Equal(0))
	})

	t.Run("should find the build object", func(_ *testing.T) {
		found := i.SearchForGit(buildapi.GitHubWebHookTrigger, stubs.RepoURL, stubs.Branch)
		g.Expect(len(found)).To(gomega.Equal(1))
	})
}

func TestInventory_SearchForObjectRef(t *testing.T) {
	buildWithObjectRefName := buildapi.Build{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: stubs.Namespace,
			Name:      "buildname",
		},
		Spec: buildapi.BuildSpec{
			Trigger: &buildapi.Trigger{
				When: []buildapi.TriggerWhen{{
					Type: buildapi.PipelineTrigger,
					ObjectRef: &buildapi.WhenObjectRef{
						Name:   "name",
						Status: []string{"Successful"},
					},
				}},
			},
		},
	}
	buildWithObjectRefSelector := buildapi.Build{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    map[string]string{"k": "v"},
			Namespace: stubs.Namespace,
			Name:      "buildname",
		},
		Spec: buildapi.BuildSpec{
			Trigger: &buildapi.Trigger{
				When: []buildapi.TriggerWhen{{
					Type: buildapi.PipelineTrigger,
					ObjectRef: &buildapi.WhenObjectRef{
						Status:   []string{"Successful"},
						Selector: map[string]string{"k": "v"},
					},
				}},
			},
		},
	}

	tests := []struct {
		name      string
		builds    []buildapi.Build
		whenType  buildapi.TriggerType
		objectRef buildapi.WhenObjectRef
		want      []SearchResult
	}{{
		name:     "find build by name",
		builds:   []buildapi.Build{buildWithObjectRefName},
		whenType: buildapi.PipelineTrigger,
		objectRef: buildapi.WhenObjectRef{
			Name:   "name",
			Status: []string{"Successful"},
		},
		want: []SearchResult{{
			BuildName: types.NamespacedName{Namespace: stubs.Namespace, Name: "buildname"},
		}},
	}, {
		name:     "find build by label selector",
		builds:   []buildapi.Build{buildWithObjectRefSelector},
		whenType: buildapi.PipelineTrigger,
		objectRef: buildapi.WhenObjectRef{
			Status:   []string{"Successful"},
			Selector: map[string]string{"k": "v"},
		},
		want: []SearchResult{{
			BuildName: types.NamespacedName{Namespace: stubs.Namespace, Name: "buildname"},
		}},
	}, {
		name:     "does not find builds, due to wrong selector",
		builds:   []buildapi.Build{buildWithObjectRefSelector},
		whenType: buildapi.PipelineTrigger,
		objectRef: buildapi.WhenObjectRef{
			Status:   []string{"Successful"},
			Selector: map[string]string{"wrong": "label"},
		},
		want: []SearchResult{},
	}, {
		name:     "does not find builds, due to wrong name",
		builds:   []buildapi.Build{buildWithObjectRefSelector},
		whenType: buildapi.PipelineTrigger,
		objectRef: buildapi.WhenObjectRef{
			Name:   "wrong",
			Status: []string{"Successful"},
		},
		want: []SearchResult{},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := NewInventory()
			for _, b := range tt.builds {
				b := b
				i.Add(&b)
			}

			got := i.SearchForObjectRef(tt.whenType, &tt.objectRef)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Inventory.SearchForObjectRef() = %v, want %v", got, tt.want)
			}
		})
	}
}
