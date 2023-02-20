package filter

import (
	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/triggers/pkg/constants"

	"k8s.io/apimachinery/pkg/types"
)

// ExtractBuildRunRunOwner inspect the object owners for Tekton Run and returns it, otherwise nil.
func ExtractBuildRunRunOwner(br *v1alpha1.BuildRun) *types.NamespacedName {
	for _, ownerRef := range br.OwnerReferences {
		if ownerRef.APIVersion == constants.TektonAPIv1alpha1 && ownerRef.Kind == "Run" {
			return &types.NamespacedName{Namespace: br.GetNamespace(), Name: ownerRef.Name}
		}
	}
	return nil
}

// ExtractBuildRunCustomRunOwner inspect the object owners for Tekton CustomRun and returns it, otherwise nil.
func ExtractBuildRunCustomRunOwner(br *v1alpha1.BuildRun) *types.NamespacedName {
	for _, ownerRef := range br.OwnerReferences {
		if ownerRef.APIVersion == constants.TektonAPIv1beta1 && ownerRef.Kind == "CustomRun" {
			return &types.NamespacedName{Namespace: br.GetNamespace(), Name: ownerRef.Name}
		}
	}
	return nil
}
