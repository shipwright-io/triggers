// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package filter

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/triggers/test/stubs"
	tknv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

func TestPipelineRunExtractTriggeredBuildsSlice(t *testing.T) {
	// PipelineRun with a bogus annotation payload, instead of valid JSON
	pipelineRunWithBogusAnnotation := stubs.TektonPipelineRun("pipeline")
	pipelineRunWithBogusAnnotation.SetAnnotations(map[string]string{
		TektonPipelineRunTriggeredBuilds: "bogus stuff",
	})

	// PipelineRun with valid triggered-builds annotation, JSON payload
	pipelineRunWithTriggeredBuildsAnnotation := stubs.TektonPipelineRun("pipeline")
	triggeredBuilds := []TriggeredBuild{{
		BuildName: "build",
		ObjectRef: &v1alpha1.WhenObjectRef{},
	}}
	triggeredBuildsAnnotationBytes, err := json.Marshal(triggeredBuilds)
	if err != nil {
		t.Errorf("failed to prepare triggered-builds JSON representation: %q", err.Error())
	}
	pipelineRunWithTriggeredBuildsAnnotation.SetAnnotations(map[string]string{
		TektonPipelineRunTriggeredBuilds: string(triggeredBuildsAnnotationBytes),
	})

	tests := []struct {
		name        string
		pipelineRun tknv1beta1.PipelineRun
		want        []TriggeredBuild
		wantErr     bool
	}{{
		name:        "PipelineRun without triggered-builds annotation",
		pipelineRun: stubs.TektonPipelineRun("pipeline"),
		want:        []TriggeredBuild{},
		wantErr:     false,
	}, {
		name:        "PipelineRun with bogus annotation",
		pipelineRun: pipelineRunWithBogusAnnotation,
		want:        nil,
		wantErr:     true,
	}, {
		name:        "PipelineRun with triggered-builds annotation",
		pipelineRun: pipelineRunWithTriggeredBuildsAnnotation,
		want:        triggeredBuilds,
		wantErr:     false,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PipelineRunExtractTriggeredBuildsSlice(&tt.pipelineRun)
			if (err != nil) != tt.wantErr {
				t.Errorf("PipelineRunExtractTriggeredBuildsSlice() = %#v, error = %v, wantErr %v",
					got, err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PipelineRunExtractTriggeredBuildsSlice() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestTriggereBuildsContainsObjectRef(t *testing.T) {
	tests := []struct {
		name            string
		triggeredBuilds []TriggeredBuild
		buildNames      []string
		objectRef       *v1alpha1.WhenObjectRef
		want            bool
	}{{
		name: "triggered builds contains objectRef",
		triggeredBuilds: []TriggeredBuild{{
			BuildName: "build",
			ObjectRef: stubs.TriggerWhenPipelineSucceeded.ObjectRef.DeepCopy(),
		}},
		buildNames: []string{"build"},
		objectRef:  stubs.TriggerWhenPipelineSucceeded.ObjectRef.DeepCopy(),
		want:       true,
	}, {
		name:            "empty triggered builds does not contain objectRef",
		triggeredBuilds: []TriggeredBuild{},
		buildNames:      []string{"build"},
		objectRef:       stubs.TriggerWhenPipelineSucceeded.ObjectRef.DeepCopy(),
		want:            false,
	}, {
		name: "triggered builds does not contain objectRef build name",
		triggeredBuilds: []TriggeredBuild{{
			BuildName: "another-build",
			ObjectRef: stubs.TriggerWhenPipelineSucceeded.ObjectRef.DeepCopy(),
		}},
		buildNames: []string{"build"},
		objectRef:  stubs.TriggerWhenPipelineSucceeded.ObjectRef.DeepCopy(),
		want:       false,
	}, {
		name: "triggered builds does not contain objectRef",
		triggeredBuilds: []TriggeredBuild{{
			BuildName: "build",
			ObjectRef: stubs.TriggerWhenPushToMain.ObjectRef.DeepCopy(),
		}},
		buildNames: []string{"build"},
		objectRef:  stubs.TriggerWhenPipelineSucceeded.ObjectRef.DeepCopy(),
		want:       false,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TriggereBuildsContainsObjectRef(
				tt.triggeredBuilds,
				tt.buildNames,
				tt.objectRef,
			)
			if got != tt.want {
				t.Errorf("TriggereBuildsContainsObjectRef() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAppendIntoTriggeredBuildSliceAsAnnotation(t *testing.T) {
	tests := []struct {
		name            string
		triggeredBuilds []TriggeredBuild
		buildNames      []string
		objectRef       *v1alpha1.WhenObjectRef
		want            string
		wantErr         bool
	}{{
		name:            "empty inputs",
		triggeredBuilds: []TriggeredBuild{},
		buildNames:      []string{},
		objectRef:       &v1alpha1.WhenObjectRef{},
		want:            "[]",
		wantErr:         false,
	}, {
		name:            "empty triggered-builds with a single build",
		triggeredBuilds: []TriggeredBuild{},
		buildNames:      []string{"build"},
		objectRef:       &v1alpha1.WhenObjectRef{},
		want:            "[{\"buildName\":\"build\",\"objectRef\":{}}]",
		wantErr:         false,
	}, {
		name: "single triggered-build with single build",
		triggeredBuilds: []TriggeredBuild{{
			BuildName: "previous-build",
			ObjectRef: &v1alpha1.WhenObjectRef{},
		}},
		buildNames: []string{"build"},
		objectRef:  &v1alpha1.WhenObjectRef{},
		want: "[{\"buildName\":\"previous-build\",\"objectRef\":{}}," +
			"{\"buildName\":\"build\",\"objectRef\":{}}]",
		wantErr: false,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AppendIntoTriggeredBuildSliceAsAnnotation(
				tt.triggeredBuilds,
				tt.buildNames,
				tt.objectRef,
			)
			if (err != nil) != tt.wantErr {
				t.Errorf("AppendIntoTriggeredBuildSliceAsAnnotation() error = %v, wantErr %v",
					err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("AppendIntoTriggeredBuildSliceAsAnnotation() = %v, want %v",
					got, tt.want)
			}
		})
	}
}
