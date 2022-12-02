package filter

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/triggers/pkg/constants"
	"github.com/shipwright-io/triggers/test/stubs"

	tknv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func Test_searchBuildRunForRunOwner(t *testing.T) {
	tests := []struct {
		name string
		br   *v1alpha1.BuildRun
		want *types.NamespacedName
	}{{
		name: "buildrun not owned by tekton run",
		br: &v1alpha1.BuildRun{
			ObjectMeta: metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{},
			},
		},
		want: nil,
	}, {
		name: "buildrun owned by tekton run",
		br: &v1alpha1.BuildRun{
			ObjectMeta: metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{{
					APIVersion: constants.TektonAPIv1alpha1,
					Kind:       "Run",
					Name:       "run",
				}},
				Namespace: "namespace",
				Name:      "buildrun",
			},
		},
		want: &types.NamespacedName{Namespace: "namespace", Name: "run"},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := searchBuildRunForRunOwner(tt.br); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("searchBuildRunForRunOwner() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_pipelineRunReferencesShipwright(t *testing.T) {
	tests := []struct {
		name        string
		pipelineRun *tknv1beta1.PipelineRun
		want        bool
	}{{
		name: "pipelinerun has status.pipelinespec nil",
		pipelineRun: &tknv1beta1.PipelineRun{
			Status: tknv1beta1.PipelineRunStatus{
				PipelineRunStatusFields: tknv1beta1.PipelineRunStatusFields{
					PipelineSpec: nil,
				},
			},
		},
		want: false,
	}, {
		name: "pipelinerun does not references shipwright build",
		pipelineRun: &tknv1beta1.PipelineRun{
			Status: tknv1beta1.PipelineRunStatus{
				PipelineRunStatusFields: tknv1beta1.PipelineRunStatusFields{
					PipelineSpec: &tknv1beta1.PipelineSpec{
						Tasks: []tknv1beta1.PipelineTask{{}},
					},
				},
			},
		},
		want: false,
	}, {
		name: "pipelinerun references shipwright build",
		pipelineRun: &tknv1beta1.PipelineRun{
			Status: tknv1beta1.PipelineRunStatus{
				PipelineRunStatusFields: tknv1beta1.PipelineRunStatusFields{
					PipelineSpec: &tknv1beta1.PipelineSpec{
						Tasks: []tknv1beta1.PipelineTask{{
							Name: "task",
							TaskRef: &tknv1beta1.TaskRef{
								Name:       "shipwright-build",
								APIVersion: constants.ShipwrightAPIVersion,
								Kind:       "Build",
							},
						}},
					},
				},
			},
		},
		want: true,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pipelineRunReferencesShipwright(tt.pipelineRun); got != tt.want {
				t.Errorf("pipelineRunReferencesShipwright() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParsePipelineRunStatus(t *testing.T) {
	tests := []struct {
		name        string
		pipelineRun tknv1beta1.PipelineRun
		want        string
		wantErr     bool
	}{{
		name:        "cancelled",
		pipelineRun: stubs.TektonPipelineRunCanceled("name"),
		want:        "Cancelled",
		wantErr:     false,
	}, {
		name:        "started",
		pipelineRun: stubs.TektonPipelineRunRunning("name"),
		want:        "Started",
		wantErr:     false,
	}, {
		name:        "timedout",
		pipelineRun: stubs.TektonPipelineRunTimedOut("name"),
		want:        "TimedOut",
		wantErr:     false,
	}, {
		name:        "succeeded",
		pipelineRun: stubs.TektonPipelineRunSucceeded("name"),
		want:        "Succeeded",
		wantErr:     false,
	}, {
		name:        "failed",
		pipelineRun: stubs.TektonPipelineRunFailed("name"),
		want:        "Failed",
		wantErr:     false,
	}}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePipelineRunStatus(ctx, time.Now(), &tt.pipelineRun)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParsePipelineRunStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParsePipelineRunStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}
