// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package filter

import (
	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/triggers/pkg/constants"

	tknv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TektonCustomRunParamsToShipwrightParamValues transforms the informed Tekton Run params into Shipwright
// ParamValues slice.
func TektonCustomRunParamsToShipwrightParamValues(customRun *tknv1beta1.CustomRun) []v1alpha1.ParamValue {
	paramValues := []v1alpha1.ParamValue{}
	for _, p := range customRun.Spec.Params {
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

// customRunReferencesShipwright inspect the CustomRun instance looking for a TaskRef pointing to Shipwright.
func customRunReferencesShipwright(customRun *tknv1beta1.CustomRun) bool {
	if customRun.Spec.CustomRef == nil {
		return false
	}
	return customRun.Spec.CustomRef.APIVersion == constants.ShipwrightAPIVersion &&
		customRun.Spec.CustomRef.Kind == "Build"
}

// CustomRunEventFilterPredicate inspects the object expecting a Tekton's CustomRun, filtering out when is not
// yet started, or has already being processed by the custom-tasks controller by inspecting the
// Status' ExtraFields.
func CustomRunEventFilterPredicate(obj client.Object) bool {
	logger := loggerForClientObj(obj, "controller.run-filter")

	br, ok := obj.(*v1alpha1.BuildRun)
	if ok {
		logger.V(0).Info("Inspecting BuildRun instance for Tekton's CustomRun ownership")
		return ExtractBuildRunCustomRunOwner(br) != nil
	}

	// the custom-tasks controller watches over Tekton's CustomRun and BuildRun objects, thus here we
	// ignore casting errors and skip the object
	customRun, ok := obj.(*tknv1beta1.CustomRun)
	if !ok {
		logger.V(0).Error(nil, "Object is not a Tekton CustomRun!")
		return false
	}

	// making sure the object contains a TaskRef to Shipwright
	if !customRunReferencesShipwright(customRun) {
		logger.V(0).Info("CustomRun instance does not reference Shipwright, skipping!")
		return false
	}

	return true
}
