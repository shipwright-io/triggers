package filter

import (
	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/triggers/pkg/constants"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ExtractBuildRunOwner inspect the object owners for Tekton Run and returns it, otherwise nil.
func ExtractBuildRunOwner(br *v1alpha1.BuildRun) *types.NamespacedName {
	for _, ownerRef := range br.OwnerReferences {
		if ownerRef.APIVersion == constants.TektonAPIv1alpha1 && ownerRef.Kind == "Run" {
			return &types.NamespacedName{Namespace: br.GetNamespace(), Name: ownerRef.Name}
		}
	}
	return nil
}

// BuildRunEventFilterPredicate only allows BuildRuns owned by Tekton Run objects to be reconciled.
func BuildRunEventFilterPredicate(obj client.Object) bool {
	logger := loggerForClientObj(obj, "controller.buildrun-filter")

	br, ok := obj.(*v1alpha1.BuildRun)
	if !ok {
		logger.V(0).Error(nil, "Unable to cast object as Shipwright's BuildRun")
		return false
	}
	return ExtractBuildRunOwner(br) != nil
}
