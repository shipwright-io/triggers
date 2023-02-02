package filter

import (
	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/triggers/pkg/constants"

	tknv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tknv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TektonRunParamsToShipwrightParamValues transforms the informed Tekton Run params into Shipwright
// ParamValues slice.
func TektonRunParamsToShipwrightParamValues(run *tknv1alpha1.Run) []v1alpha1.ParamValue {
	paramValues := []v1alpha1.ParamValue{}
	for _, p := range run.Spec.Params {
		paramValue := v1alpha1.ParamValue{Name: p.Name}
		if p.Value.Type == tknv1beta1.ParamTypeArray {
			paramValue.Values = []v1alpha1.SingleValue{}
			for _, v := range p.Value.ArrayVal {
				v := v
				paramValue.Values = append(paramValue.Values, v1alpha1.SingleValue{
					Value: &v,
				})
			}
		} else {
			paramValue.SingleValue = &v1alpha1.SingleValue{
				Value: &p.Value.StringVal,
			}
		}
		paramValues = append(paramValues, paramValue)
	}
	return paramValues
}

// runReferencesShipwright inspect the run instance looking for a TaskRef pointing to Shipwright.
func runReferencesShipwright(run *tknv1alpha1.Run) bool {
	if run.Spec.Ref == nil {
		return false
	}
	return run.Spec.Ref.APIVersion == constants.ShipwrightAPIVersion &&
		run.Spec.Ref.Kind == "Build"
}

// RunEventFilterPredicate inspects the object expecting a Tekton's Run, filtering out when is not
// yet started, or has already being processed by the custom-tasks controller by instpecting the
// Status' ExtraFields.
func RunEventFilterPredicate(obj client.Object) bool {
	logger := loggerForClientObj(obj, "controller.run-filter")

	br, ok := obj.(*v1alpha1.BuildRun)
	if ok {
		logger.V(0).Info("Inspecting BuildRun instance for Tekton's Run ownership")
		return ExtractBuildRunOwner(br) != nil
	}

	// the custom-tasks controller watches over Tekton's Run and BuildRun objects, thus here we
	// ignore casting errors and skip the object
	run, ok := obj.(*tknv1alpha1.Run)
	if !ok {
		logger.V(0).Error(nil, "Object is not a Tekton Run!")
		return false
	}

	// making sure the object contains a TaskRef to Shipwright
	if !runReferencesShipwright(run) {
		logger.V(0).Info("Run instance does not reference Shipwright, skipping!")
		return false
	}

	return true
}
