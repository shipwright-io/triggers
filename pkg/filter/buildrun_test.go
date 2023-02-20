package filter

import (
	"reflect"
	"testing"

	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/triggers/pkg/constants"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestExtractBuildRunRunOwner(t *testing.T) {
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
			if got := ExtractBuildRunRunOwner(tt.br); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExtractBuildRunRunOwner() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractBuildRunCustomRunOwner(t *testing.T) {
	tests := []struct {
		name string
		br   *v1alpha1.BuildRun
		want *types.NamespacedName
	}{{
		name: "buildrun not owned by tekton customrun",
		br: &v1alpha1.BuildRun{
			ObjectMeta: metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{},
			},
		},
		want: nil,
	}, {
		name: "buildrun owned by tekton customrun",
		br: &v1alpha1.BuildRun{
			ObjectMeta: metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{{
					APIVersion: constants.TektonAPIv1beta1,
					Kind:       "CustomRun",
					Name:       "customrun",
				}},
				Namespace: "namespace",
				Name:      "buildrun",
			},
		},
		want: &types.NamespacedName{Namespace: "namespace", Name: "customrun"},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractBuildRunCustomRunOwner(tt.br); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExtractBuildRunCustomRunOwner() = %v, want %v", got, tt.want)
			}
		})
	}
}
