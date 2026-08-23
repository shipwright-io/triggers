// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	ctrl "sigs.k8s.io/controller-runtime"
)

// Done returns an empty result, signaling that the reconciliation is complete
// and no requeue is needed.
func Done() (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

// RequeueOnError returns the error as-is. A non-nil error makes controller-runtime
// requeue the request with exponential backoff, so no explicit requeue flag is needed.
func RequeueOnError(err error) (ctrl.Result, error) {
	return ctrl.Result{}, err
}
