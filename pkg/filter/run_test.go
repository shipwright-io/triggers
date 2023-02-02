package filter

import (
	"reflect"
	"testing"

	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/triggers/test/stubs"

	tknv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tknv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestTektonRunParamsToShipwrightParamValues(t *testing.T) {
	value := "value"

	tests := []struct {
		name string
		run  *tknv1alpha1.Run
		want []v1alpha1.ParamValue
	}{{
		name: "run does not contain params",
		run: &tknv1alpha1.Run{
			Spec: tknv1alpha1.RunSpec{
				Params: []tknv1beta1.Param{},
			},
		},
		want: []v1alpha1.ParamValue{},
	}, {
		name: "run contains an string param",
		run: &tknv1alpha1.Run{
			Spec: tknv1alpha1.RunSpec{
				Params: []tknv1beta1.Param{{
					Name:  "string",
					Value: *tknv1beta1.NewArrayOrString(value),
				}},
			},
		},
		want: []v1alpha1.ParamValue{{
			Name: "string",
			SingleValue: &v1alpha1.SingleValue{
				Value: &value,
			},
		}},
	}, {
		name: "run contains an string-array param",
		run: &tknv1alpha1.Run{
			Spec: tknv1alpha1.RunSpec{
				Params: []tknv1beta1.Param{{
					Name:  "string-array",
					Value: *tknv1beta1.NewArrayOrString(value, value),
				}},
			},
		},
		want: []v1alpha1.ParamValue{{
			Name: "string-array",
			Values: []v1alpha1.SingleValue{{
				Value: &value,
			}, {
				Value: &value,
			}},
		}},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TektonRunParamsToShipwrightParamValues(tt.run); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TektonRunParamsToShipwrightParamValues() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRunEventFilterPredicate(t *testing.T) {
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
		obj:  stubs.TektonRun("not-started", nil),
		want: false,
	}, {
		name: "Run started should be accepted",
		obj:  stubs.TektonRunStarted("run-started", stubs.TektonTaskRefToShipwright("build")),
		want: true,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RunEventFilterPredicate(tt.obj); got != tt.want {
				t.Errorf("RunEventFilterPredicate() = %v, want %v", got, tt.want)
			}
		})
	}
}
