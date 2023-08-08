// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"time"
)

type realClock struct{}

func (realClock) Now() time.Time {
	return time.Now()
}

type Clock interface {
	Now() time.Time
}

// +kubebuilder:docs-gen:collapse=Clock
