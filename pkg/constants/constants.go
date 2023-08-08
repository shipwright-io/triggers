// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package constants

import (
	"fmt"

	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"

	tknv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

var (
	TektonAPIv1beta1 = fmt.Sprintf("%s/%s",
		tknv1beta1.SchemeGroupVersion.Group,
		tknv1beta1.SchemeGroupVersion.Version,
	)
	ShipwrightAPIVersion = fmt.Sprintf(
		"%s/%s",
		v1alpha1.SchemeGroupVersion.Group,
		v1alpha1.SchemeGroupVersion.Version,
	)
)
