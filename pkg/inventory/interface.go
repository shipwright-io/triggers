// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package inventory

import (
	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"k8s.io/apimachinery/pkg/types"
)

type Interface interface {
	Add(*buildapi.Build)
	Remove(types.NamespacedName)
	SearchForObjectRef(buildapi.TriggerType, *buildapi.WhenObjectRef) []SearchResult
	SearchForGit(buildapi.TriggerType, string, string) []SearchResult
}
