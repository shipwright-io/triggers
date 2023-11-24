// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package stubs

import (
	"fmt"
	"time"

	"github.com/shipwright-io/triggers/pkg/constants"

	tektonapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	tektonapibeta "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var TektonPipelineRunStatusCustomTaskShipwright = &tektonapi.PipelineSpec{
	Tasks: []tektonapi.PipelineTask{TektonPipelineTaskRefToShipwright},
}

var TektonPipelineTaskRefToShipwright = tektonapi.PipelineTask{
	Name: "shipwright",
	TaskRef: &tektonapi.TaskRef{
		APIVersion: constants.ShipwrightAPIVersion,
		Name:       "name",
	},
}

var TektonTaskRefToTekton = &tektonapibeta.TaskRef{
	Name: "task-ex",
}

func TektonTaskRefToShipwright(name string) *tektonapibeta.TaskRef {
	return &tektonapibeta.TaskRef{
		APIVersion: constants.ShipwrightAPIVersion,
		Kind:       "Build",
		Name:       name,
	}
}

func TektonCustomRun(name string, ref *tektonapibeta.TaskRef) *tektonapibeta.CustomRun {
	return &tektonapibeta.CustomRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: Namespace,
			Name:      name,
		},
		Spec: tektonapibeta.CustomRunSpec{
			CustomRef: ref,
		},
	}
}

// TektonCustomRunStarted returns a started (now) CustomRun instance using the name and TaskRef
// informed.
func TektonCustomRunStarted(name string, ref *tektonapibeta.TaskRef) *tektonapibeta.CustomRun {
	customRun := TektonCustomRun(name, ref)
	customRun.Status = tektonapibeta.CustomRunStatus{
		CustomRunStatusFields: tektonapibeta.CustomRunStatusFields{
			StartTime: &metav1.Time{Time: time.Now()},
		},
	}
	return customRun
}

func TektonPipelineRunCanceled(name string) tektonapi.PipelineRun {
	pipelineRun := TektonPipelineRun(name)
	pipelineRun.Spec.Status = tektonapi.PipelineRunSpecStatus(
		tektonapi.PipelineRunReasonCancelled,
	)
	pipelineRun.Status.PipelineRunStatusFields = tektonapi.PipelineRunStatusFields{
		PipelineSpec: &tektonapi.PipelineSpec{Description: "testing"},
	}
	return pipelineRun
}

func TektonPipelineRunRunning(name string) tektonapi.PipelineRun {
	pipelineRun := TektonPipelineRun(name)
	pipelineRun.Status.StartTime = &metav1.Time{Time: time.Now()}
	pipelineRun.Status.PipelineRunStatusFields = tektonapi.PipelineRunStatusFields{
		StartTime:    &metav1.Time{Time: time.Now()},
		PipelineSpec: &tektonapi.PipelineSpec{Description: "testing"},
	}
	return pipelineRun
}

func TektonPipelineRunTimedOut(name string) tektonapi.PipelineRun {
	pipelineRun := TektonPipelineRun(name)
	pipelineRun.Spec.Timeouts = &tektonapi.TimeoutFields{
		Pipeline: &metav1.Duration{Duration: time.Second},
	}
	pipelineRun.Status.PipelineRunStatusFields = tektonapi.PipelineRunStatusFields{
		StartTime: &metav1.Time{
			Time: time.Date(1982, time.January, 1, 0, 0, 0, 0, time.Local),
		},
		PipelineSpec: &tektonapi.PipelineSpec{Description: "testing"},
	}
	return pipelineRun
}

func TektonPipelineRunSucceeded(name string) tektonapi.PipelineRun {
	pipelineRun := TektonPipelineRun(name)
	pipelineRun.Status.MarkSucceeded("Succeeded", fmt.Sprintf("PipelineRun %q has succeeded", name))
	pipelineRun.Status.PipelineRunStatusFields = tektonapi.PipelineRunStatusFields{
		PipelineSpec: &tektonapi.PipelineSpec{Description: "testing"},
	}
	return pipelineRun
}

func TektonPipelineRunFailed(name string) tektonapi.PipelineRun {
	pipelineRun := TektonPipelineRun(name)
	pipelineRun.Status.MarkFailed("Failed", fmt.Sprintf("PipelineRun %q has failed", name))
	pipelineRun.Status.PipelineRunStatusFields = tektonapi.PipelineRunStatusFields{
		PipelineSpec: &tektonapi.PipelineSpec{Description: "testing"},
	}
	return pipelineRun
}

func TektonPipelineRun(name string) tektonapi.PipelineRun {
	return tektonapi.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: Namespace,
			Name:      name,
		},
		Spec: tektonapi.PipelineRunSpec{
			PipelineRef: &tektonapi.PipelineRef{
				Name: name,
			},
		},
		Status: tektonapi.PipelineRunStatus{},
	}
}
