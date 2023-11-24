// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package filter

import (
	"reflect"
	"testing"

	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/triggers/pkg/constants"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestExtractBuildRunCustomRunOwner(t *testing.T) {
	tests := []struct {
		name string
		br   *buildapi.BuildRun
		want *types.NamespacedName
	}{{
		name: "buildrun not owned by tekton customrun",
		br: &buildapi.BuildRun{
			ObjectMeta: metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{},
			},
		},
		want: nil,
	}, {
		name: "buildrun owned by tekton customrun",
		br: &buildapi.BuildRun{
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
