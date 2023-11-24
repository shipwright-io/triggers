// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package filter

import (
	"testing"

	"github.com/shipwright-io/triggers/test/stubs"
	tektonapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

func TestAppendIssuedBuildRunsLabel(t *testing.T) {
	pipelineRunLabeled := stubs.TektonPipelineRun("pipeline")
	pipelineRunLabeled.SetLabels(map[string]string{
		BuildRunsCreated: "existing-buildrun",
	})

	tests := []struct {
		name            string
		pipelineRun     tektonapi.PipelineRun
		buildRunsIssued []string
		want            string
	}{{
		name:            "PipelineRun without BuildRun labeled",
		pipelineRun:     stubs.TektonPipelineRun("pipeline"),
		buildRunsIssued: []string{"buildrun"},
		want:            "buildrun",
	}, {
		name:            "PipelineRun with BuildRun labeled",
		pipelineRun:     pipelineRunLabeled,
		buildRunsIssued: []string{"buildrun"},
		want:            "buildrun,existing-buildrun",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AppendIssuedBuildRunsLabel(&tt.pipelineRun, tt.buildRunsIssued)
			labels := PipelineRunGetLabels(&tt.pipelineRun)
			got, ok := labels[BuildRunsCreated]
			if !ok {
				t.Errorf(
					"AppendIssuedBuildRunsLabel() doesn't have the expected label %q",
					BuildRunsCreated,
				)
			}
			if got != tt.want {
				t.Errorf("AppendIssuedBuildRunsLabel() = %q, want %q", got, tt.want)
			}
		})
	}
}
