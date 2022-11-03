package stubs

import (
	"fmt"
	"time"

	"github.com/shipwright-io/triggers/pkg/constants"

	tknv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tknv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var TektonPipelineRunStatusCustomTaskShipwright = &tknv1beta1.PipelineSpec{
	Tasks: []tknv1beta1.PipelineTask{TektonPipelineTaskRefToShipwright},
}

var TektonPipelineTaskRefToShipwright = tknv1beta1.PipelineTask{
	Name: "shipwright",
	TaskRef: &tknv1beta1.TaskRef{
		APIVersion: constants.ShipwrightAPIVersion,
		Name:       "name",
	},
}

var TektonTaskRefToShipwright = &tknv1beta1.TaskRef{
	APIVersion: constants.ShipwrightAPIVersion,
	Kind:       "Build",
	Name:       "shipwright-ex",
}

var TektonTaskRefToTekton = &tknv1beta1.TaskRef{
	Name: "task-ex",
}

func TektonRun(name string, ref *tknv1beta1.TaskRef) tknv1alpha1.Run {
	return tknv1alpha1.Run{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: Namespace,
			Name:      name,
		},
		Spec: tknv1alpha1.RunSpec{
			Ref: ref,
		},
	}
}

func TektonPipelineRunCanceled(name string) tknv1beta1.PipelineRun {
	pipelineRun := TektonPipelineRun(name)
	pipelineRun.Spec.Status = tknv1beta1.PipelineRunSpecStatus(
		tknv1beta1.PipelineRunReasonCancelled,
	)
	pipelineRun.Status.PipelineRunStatusFields = tknv1beta1.PipelineRunStatusFields{
		PipelineSpec: &tknv1beta1.PipelineSpec{Description: "testing"},
	}
	return pipelineRun
}

func TektonPipelineRunRunning(name string) tknv1beta1.PipelineRun {
	pipelineRun := TektonPipelineRun(name)
	pipelineRun.Status.StartTime = &metav1.Time{Time: time.Now()}
	pipelineRun.Status.PipelineRunStatusFields = tknv1beta1.PipelineRunStatusFields{
		StartTime:    &metav1.Time{Time: time.Now()},
		PipelineSpec: &tknv1beta1.PipelineSpec{Description: "testing"},
	}
	return pipelineRun
}

func TektonPipelineRunTimedOut(name string) tknv1beta1.PipelineRun {
	pipelineRun := TektonPipelineRun(name)
	pipelineRun.Spec.Timeout = &metav1.Duration{Duration: time.Second}
	pipelineRun.Status.PipelineRunStatusFields = tknv1beta1.PipelineRunStatusFields{
		StartTime: &metav1.Time{
			Time: time.Date(1982, time.January, 1, 0, 0, 0, 0, time.Local),
		},
		PipelineSpec: &tknv1beta1.PipelineSpec{Description: "testing"},
	}
	return pipelineRun
}

func TektonPipelineRunSucceeded(name string) tknv1beta1.PipelineRun {
	pipelineRun := TektonPipelineRun(name)
	pipelineRun.Status.MarkSucceeded("Succeeded", fmt.Sprintf("PipelineRun %q has succeeded", name))
	pipelineRun.Status.PipelineRunStatusFields = tknv1beta1.PipelineRunStatusFields{
		PipelineSpec: &tknv1beta1.PipelineSpec{Description: "testing"},
	}
	return pipelineRun
}

func TektonPipelineRunFailed(name string) tknv1beta1.PipelineRun {
	pipelineRun := TektonPipelineRun(name)
	pipelineRun.Status.MarkFailed("Failed", fmt.Sprintf("PipelineRun %q has failed", name))
	pipelineRun.Status.PipelineRunStatusFields = tknv1beta1.PipelineRunStatusFields{
		PipelineSpec: &tknv1beta1.PipelineSpec{Description: "testing"},
	}
	return pipelineRun
}

func TektonPipelineRun(name string) tknv1beta1.PipelineRun {
	return tknv1beta1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: Namespace,
			Name:      name,
		},
		Spec: tknv1beta1.PipelineRunSpec{
			PipelineRef: &tknv1beta1.PipelineRef{
				Name: name,
			},
		},
		Status: tknv1beta1.PipelineRunStatus{},
	}
}
