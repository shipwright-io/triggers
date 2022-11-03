package filter

import (
	"strings"

	"github.com/shipwright-io/triggers/pkg/util"
	tknv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

// PipelineRunGetLabels extract labels from informed object, returns an empty map when `nil` labels.
func PipelineRunGetLabels(pipelineRun *tknv1beta1.PipelineRun) map[string]string {
	labels := pipelineRun.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	return labels
}

// AppendIssuedBuildRunsLabel update or add the label to document the BuildRuns issued for the
// PipelineRun instance informed.
func AppendIssuedBuildRunsLabel(pipelineRun *tknv1beta1.PipelineRun, buildRunsIssued []string) {
	labels := PipelineRunGetLabels(pipelineRun)

	// contains all BuildRuns issued for the PipelineRun instance
	pipelineRunBuildRunsIssued := []string{}

	// extracting existing label value to append the BuildRuns issued
	label := labels[BuildRunsCreated]
	if label != "" {
		pipelineRunBuildRunsIssued = strings.Split(label, ",")
	}
	pipelineRunBuildRunsIssued = append(pipelineRunBuildRunsIssued, buildRunsIssued...)

	labels[BuildRunsCreated] = util.JoinReversedStringSliceForK8s(pipelineRunBuildRunsIssued)
	pipelineRun.SetLabels(labels)
}
