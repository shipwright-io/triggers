package filter

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/triggers/pkg/constants"
	clock "k8s.io/utils/clock/testing"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tknv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"knative.dev/pkg/apis"
)

// Prefix prefix used in all annotations.
const Prefix = "triggers.shipwright.io"

var (
	// OwnedByTektonRun annotates the BuildRun as owned by Tekton Run.
	OwnedByTektonRun = fmt.Sprintf("%s/owned-by-run", Prefix)
	// OwnedByTektonPipelineRun lables the BuildRun as owned by Tekton PipelineRun.
	OwnedByTektonPipelineRun = fmt.Sprintf("%s/owned-by-pipelinerun", Prefix)
	// BuildRunsCreated annotates the PipelineRun with the BuildRuns created.
	BuildRunsCreated = fmt.Sprintf("%s/buildrun-names", Prefix)

	// TektonPipelineRunName annotates PipelineRuns with its current name, avoid object reprocessing.
	TektonPipelineRunName = fmt.Sprintf("%s/pipelinerun-name", Prefix)
	// TektonPipelineRunTriggeredBuilds contains references for all Builds triggered, JSON formatted
	TektonPipelineRunTriggeredBuilds = fmt.Sprintf("%s/pipelinerun-triggered-builds", Prefix)
)

// pipelineRunReferencesShipwright checks if the informed PipelineRun is reffering to a Shipwright
// resource via TaskRef.
func pipelineRunReferencesShipwright(pipelineRun *tknv1beta1.PipelineRun) bool {
	if pipelineRun.Status.PipelineSpec == nil {
		return false
	}
	for _, task := range pipelineRun.Status.PipelineSpec.Tasks {
		if task.TaskRef == nil {
			continue
		}
		if task.TaskRef.APIVersion == constants.ShipwrightAPIVersion {
			return true
		}
	}
	return false
}

// PipelineRunEventFilterPredicate predicate filter for the basic inspections in the object,
// filtering only what needs to go through reconciliation. PipelineRun objects referencing
// Custom-Tasks are also skipped.
func PipelineRunEventFilterPredicate(obj client.Object) bool {
	logger := loggerForClientObj(obj, "controller.pipelinerun-filter")

	pipelineRun, ok := obj.(*tknv1beta1.PipelineRun)
	if !ok {
		logger.V(0).Error(nil, "Unable to cast object as Tekton PipelineRun")
		return false
	}

	if pipelineRun.Spec.PipelineRef == nil {
		logger.V(0).Info("Skipping due nil .Spec.PipelineRef")
		return false
	}

	if pipelineRun.Status.PipelineSpec == nil {
		logger.V(0).Info("Skipping due to nil .Status.PipelineSpec")
		return false
	}

	if pipelineRunReferencesShipwright(pipelineRun) {
		logger.V(0).Info("Skipping due to being part of a Custom-Task")
		return false
	}
	return true
}

// ParsePipelineRunStatus parte the informed object status to extract its status.
func ParsePipelineRunStatus(
	ctx context.Context,
	now time.Time,
	pipelineRun *tknv1beta1.PipelineRun,
) (string, error) {
	switch {
	case pipelineRun.IsDone():
		if pipelineRun.Status.GetCondition(apis.ConditionSucceeded).IsTrue() {
			return tknv1beta1.PipelineRunReasonSuccessful.String(), nil
		}
		return tknv1beta1.PipelineRunReasonFailed.String(), nil
	case pipelineRun.IsCancelled():
		return tknv1beta1.PipelineRunReasonCancelled.String(), nil
	case pipelineRun.HasTimedOut(ctx, clock.NewFakePassiveClock(now)):
		return "TimedOut", nil
	case pipelineRun.HasStarted():
		return tknv1beta1.PipelineRunReasonStarted.String(), nil
	default:
		return "", fmt.Errorf("unable to parse pipelinerun %q current status",
			pipelineRun.GetNamespacedName())
	}
}

// PipelineRunToObjectRef transforms the informed PipelineRun instance to a ObjectRef.
func PipelineRunToObjectRef(
	ctx context.Context,
	now time.Time,
	pipelineRun *tknv1beta1.PipelineRun,
) (*v1alpha1.WhenObjectRef, error) {
	status, err := ParsePipelineRunStatus(ctx, now, pipelineRun)
	if err != nil {
		return nil, err
	}

	// sanitizing label set to not use the labels added by triggers
	labels := PipelineRunGetLabels(pipelineRun)
	for k := range labels {
		if strings.HasPrefix(k, Prefix) {
			delete(labels, k)
		}
	}

	return &v1alpha1.WhenObjectRef{
		Name:     pipelineRun.Spec.PipelineRef.Name,
		Status:   []string{status},
		Selector: labels,
	}, nil
}
