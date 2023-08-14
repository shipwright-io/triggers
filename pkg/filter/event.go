// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package filter

import (
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// loggerForClientObj returns a logger instance using the object attributes and setting up the
// component name.
func loggerForClientObj(obj client.Object, name string) logr.Logger {
	return logr.New(log.Log.GetSink()).
		WithName(name).
		WithValues(
			"generation", obj.GetGeneration(),
			"namespace", obj.GetNamespace(),
			"name", obj.GetName(),
		)
}
