package inventory

import (
	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
)

type Interface interface {
	Add(*v1alpha1.Build)
	Remove(types.NamespacedName)
	SearchForObjectRef(v1alpha1.TriggerType, *v1alpha1.WhenObjectRef) []SearchResult
	SearchForGit(v1alpha1.TriggerType, string, string) []SearchResult
}
