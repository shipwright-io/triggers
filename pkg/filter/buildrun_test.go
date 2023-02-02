package filter

import (
	"reflect"
	"testing"

	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/triggers/pkg/constants"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestExtractBuildRunOwner(t *testing.T) {
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
			if got := ExtractBuildRunOwner(tt.br); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("searchBuildRunForRunOwner() = %v, want %v", got, tt.want)
			}
		})
	}
}
