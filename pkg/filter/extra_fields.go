// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package filter

import (
	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"

	"k8s.io/apimachinery/pkg/types"
)

// ExtraFields carry on meta-information linking Tekton Run with Shipwright Build.
type ExtraFields struct {
	BuildRunNamespace string `json:"buildRunNamespace,omitempty"` // buildrun namespace
	BuildRunName      string `json:"buildRunName,omitempty"`      // buildrun name
}

// IsEmpty checks if the BuildRunName is defined.
func (e *ExtraFields) IsEmpty() bool {
	return e.BuildRunName == ""
}

// GetNamespacedName returns a NamespacedName representation.
func (e *ExtraFields) GetNamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Namespace: e.BuildRunNamespace,
		Name:      e.BuildRunName,
	}
}

// NewExtraFields instantiates a ExtraFields with informed BuildRun.
func NewExtraFields(br *v1alpha1.BuildRun) ExtraFields {
	return ExtraFields{
		BuildRunNamespace: br.GetNamespace(),
		BuildRunName:      br.GetName(),
	}
}
