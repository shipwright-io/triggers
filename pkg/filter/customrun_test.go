// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package filter

import (
	"reflect"
	"testing"

	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/triggers/test/stubs"

	tektonapibeta "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestTektonCustomRunParamsToShipwrightParamValues(t *testing.T) {
	value := "value"

	tests := []struct {
		name      string
		customRun *tektonapibeta.CustomRun
		want      []buildapi.ParamValue
	}{{
		name: "run does not contain params",
		customRun: &tektonapibeta.CustomRun{
			Spec: tektonapibeta.CustomRunSpec{
				Params: []tektonapibeta.Param{},
			},
		},
		want: []buildapi.ParamValue{},
	}, {
		name: "run contains an string param",
		customRun: &tektonapibeta.CustomRun{
			Spec: tektonapibeta.CustomRunSpec{
				Params: []tektonapibeta.Param{{
					Name:  "string",
					Value: *tektonapibeta.NewArrayOrString(value),
				}},
			},
		},
		want: []buildapi.ParamValue{{
			Name: "string",
			SingleValue: &buildapi.SingleValue{
				Value: &value,
			},
		}},
	}, {
		name: "run contains an string-array param",
		customRun: &tektonapibeta.CustomRun{
			Spec: tektonapibeta.CustomRunSpec{
				Params: []tektonapibeta.Param{{
					Name:  "string-array",
					Value: *tektonapibeta.NewArrayOrString(value, value),
				}},
			},
		},
		want: []buildapi.ParamValue{{
			Name: "string-array",
			Values: []buildapi.SingleValue{{
				Value: &value,
			}, {
				Value: &value,
			}},
		}},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TektonCustomRunParamsToShipwrightParamValues(tt.customRun); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TektonCustomRunParamsToShipwrightParamValues() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCustomRunEventFilterPredicate(t *testing.T) {
	tests := []struct {
		name string
		obj  client.Object
		want bool
	}{{
		name: "BuildRun should be ignored",
		obj:  stubs.ShipwrightBuildRun("buildrun"),
		want: false,
	}, {
		name: "Run without reference to Shipwright should be ignored",
		obj:  stubs.TektonCustomRun("not-started", nil),
		want: false,
	}, {
		name: "Run started should be accepted",
		obj:  stubs.TektonCustomRunStarted("run-started", stubs.TektonTaskRefToShipwright("build")),
		want: true,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CustomRunEventFilterPredicate(tt.obj); got != tt.want {
				t.Errorf("CustomRunEventFilterPredicate() = %v, want %v", got, tt.want)
			}
		})
	}
}
