// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	ctrl "sigs.k8s.io/controller-runtime"
)

func Done() (ctrl.Result, error) {
	return ctrl.Result{Requeue: false}, nil
}

func RequeueOnError(err error) (ctrl.Result, error) {
	requeue := true
	if err == nil {
		requeue = false
	}
	return ctrl.Result{Requeue: requeue}, err
}
