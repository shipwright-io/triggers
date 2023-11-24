// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package constants

import (
	"fmt"

	buildapi "github.com/shipwright-io/build/pkg/apis/build/v1beta1"

	tektonapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	tektonapibeta "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

var (
	TektonAPIv1 = fmt.Sprintf("%s/%s",
		tektonapi.SchemeGroupVersion.Group,
		tektonapi.SchemeGroupVersion.Version,
	)
	TektonAPIv1beta1 = fmt.Sprintf("%s/%s",
		tektonapibeta.SchemeGroupVersion.Group,
		tektonapibeta.SchemeGroupVersion.Version,
	)
	ShipwrightAPIVersion = fmt.Sprintf(
		"%s/%s",
		buildapi.SchemeGroupVersion.Group,
		buildapi.SchemeGroupVersion.Version,
	)
)
