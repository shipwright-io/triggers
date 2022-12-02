package filter

import (
	"bytes"
	"encoding/json"
	"reflect"

	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/triggers/pkg/util"
	tknv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

// TriggeredBuild represents previously triggered builds by storing together the original build name
// and it's objectRef. Both are the criteria needed to find the Builds with matching triggers in the
// Inventory.
type TriggeredBuild struct {
	BuildName string                  `json:"buildName"`
	ObjectRef *v1alpha1.WhenObjectRef `json:"objectRef"`
}

// PipelineRunGetAnnotations extract the annotations, return an empty map otherwise.
func PipelineRunGetAnnotations(pipelineRun *tknv1beta1.PipelineRun) map[string]string {
	annotations := pipelineRun.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	return annotations
}

func PipelineRunAnnotatedNameMatchesObject(pipelineRun *tknv1beta1.PipelineRun) bool {
	annotations := PipelineRunGetAnnotations(pipelineRun)
	value, ok := annotations[TektonPipelineRunName]
	if !ok {
		return false
	}
	return pipelineRun.GetName() == value
}

func PipelineRunAnnotateName(pipelineRun *tknv1beta1.PipelineRun) {
	annotations := PipelineRunGetAnnotations(pipelineRun)
	annotations[TektonPipelineRunName] = pipelineRun.GetName()
	pipelineRun.SetAnnotations(annotations)
}

// UnmarshalIntoTriggeredAnnotationSlice executes the un-marshalling of the informed string payload
// into a slice of TriggeredBuild type. JSON validation is strict, returns error on unknown fields.
func UnmarshalIntoTriggeredAnnotationSlice(payload string) ([]TriggeredBuild, error) {
	reader := bytes.NewReader([]byte(payload))
	dec := json.NewDecoder(reader)
	dec.DisallowUnknownFields()

	var triggeredBuilds []TriggeredBuild
	if err := dec.Decode(&triggeredBuilds); err != nil {
		return nil, err
	}
	return triggeredBuilds, nil
}

// PipelineRunExtractTriggeredBuildsSlice extracts the triggered-builds annotation and returns a
// valid slice of the type. When the annotation is empty, or not present, an empty slice is returned
// instead.
func PipelineRunExtractTriggeredBuildsSlice(
	pipelineRun *tknv1beta1.PipelineRun,
) ([]TriggeredBuild, error) {
	annotations := PipelineRunGetAnnotations(pipelineRun)
	value, ok := annotations[TektonPipelineRunTriggeredBuilds]
	if !ok {
		return []TriggeredBuild{}, nil
	}
	return UnmarshalIntoTriggeredAnnotationSlice(value)
}

// TriggereBuildsContainsObjectRef asserts if the slice contains the informed entry.
func TriggereBuildsContainsObjectRef(
	triggeredBuilds []TriggeredBuild,
	buildNames []string,
	objectRef *v1alpha1.WhenObjectRef,
) bool {
	for _, entry := range triggeredBuilds {
		// first of all, the build name must be the same
		if !util.StringSliceContains(buildNames, entry.BuildName) {
			return false
		}

		// making sure the objectRef is ready to be compared with incoming struct, and then when both
		// entries are the same it asserts the informed objectRef is contained in the slice
		if entry.ObjectRef != nil && entry.ObjectRef.Selector == nil {
			entry.ObjectRef.Selector = map[string]string{}
		}
		if reflect.DeepEqual(entry.ObjectRef, objectRef) {
			return true
		}
	}
	return false
}

// AppendIntoTriggeredBuildSliceAsAnnotation appends the build names with the objectRef into the
// informed triggered-builds slice, the payload returned is marshalled JSON which can emit errors.
func AppendIntoTriggeredBuildSliceAsAnnotation(
	triggeredBuilds []TriggeredBuild,
	buildNames []string,
	objectRef *v1alpha1.WhenObjectRef,
) (string, error) {
	for _, buildName := range buildNames {
		entry := TriggeredBuild{
			BuildName: buildName,
			ObjectRef: objectRef,
		}
		triggeredBuilds = append(triggeredBuilds, entry)
	}

	annotationBytes, err := json.Marshal(triggeredBuilds)
	if err != nil {
		return "", err
	}
	return string(annotationBytes), nil
}

// PipelineRunAppendTriggeredBuildsAnnotation set or update the triggered-builds annotation.
func PipelineRunAppendTriggeredBuildsAnnotation(
	pipelineRun *tknv1beta1.PipelineRun,
	triggeredBuilds []TriggeredBuild,
	buildNames []string,
	objectRef *v1alpha1.WhenObjectRef,
) error {
	annotations := pipelineRun.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}

	// annotating PipelineRun with the meta-information about which Builds have been triggered, and
	// later on this information is used to filter out objects which have already been processed
	triggeredBuildsAnnotation, err := AppendIntoTriggeredBuildSliceAsAnnotation(
		triggeredBuilds, buildNames, objectRef)
	if err != nil {
		return err
	}
	annotations[TektonPipelineRunTriggeredBuilds] = triggeredBuildsAnnotation

	// updating the instance to reflect the annotations
	pipelineRun.SetAnnotations(annotations)
	return nil
}
