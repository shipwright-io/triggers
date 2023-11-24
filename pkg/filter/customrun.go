// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package filter

import (
	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/triggers/pkg/constants"

	tektonapibeta "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TektonCustomRunParamsToShipwrightParamValues transforms the informed Tekton Run params into Shipwright
// ParamValues slice.
func TektonCustomRunParamsToShipwrightParamValues(customRun *tektonapibeta.CustomRun) []buildapi.ParamValue {
	paramValues := []buildapi.ParamValue{}
	for i, p := range customRun.Spec.Params {
		paramValue := buildapi.ParamValue{Name: p.Name}
		if p.Value.Type == tektonapibeta.ParamTypeArray {
			paramValue.Values = []buildapi.SingleValue{}
			for _, v := range p.Value.ArrayVal {
				v := v
				paramValue.Values = append(paramValue.Values, buildapi.SingleValue{
					Value: &v,
				})
			}
		} else {
			paramValue.SingleValue = &buildapi.SingleValue{
				Value: &customRun.Spec.Params[i].Value.StringVal,
			}
		}
		paramValues = append(paramValues, paramValue)
	}
	return paramValues
}

// customRunReferencesShipwright inspect the CustomRun instance looking for a TaskRef pointing to Shipwright.
func customRunReferencesShipwright(customRun *tektonapibeta.CustomRun) bool {
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

	br, ok := obj.(*buildapi.BuildRun)
	if ok {
		logger.V(0).Info("Inspecting BuildRun instance for Tekton's CustomRun ownership")
		return ExtractBuildRunCustomRunOwner(br) != nil
	}

	// the custom-tasks controller watches over Tekton's CustomRun and BuildRun objects, thus here we
	// ignore casting errors and skip the object
	customRun, ok := obj.(*tektonapibeta.CustomRun)
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
