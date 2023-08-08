// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"fmt"
)

// StringSliceContains assert if the informed slice contains a string.
func StringSliceContains(slice []string, str string) bool {
	for _, s := range slice {
		if str == s {
			return true
		}
	}
	return false
}

// JoinReversedStringSliceForK8s joins the entries of the informed slice reversed using commas in a
// single string limited to 63 characters, a Kubernetes limitation for labels.
func JoinReversedStringSliceForK8s(slice []string) string {
	var s string
	for i := len(slice) - 1; i >= 0; i-- {
		entry := slice[i]
		if len(s)+len(entry) >= 63 {
			break
		}

		if len(s) == 0 {
			s = entry
		} else {
			s = fmt.Sprintf("%s,%s", s, entry)
		}
	}
	return s
}
