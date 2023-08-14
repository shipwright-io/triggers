#!/bin/bash

# Copyright The Shipwright Contributors
#
# SPDX-License-Identifier: Apache-2.0

# Verifies if a developer has forgot to run the
# `make generate` so that all the changes in the
# clientset and CRDs should also be pushed

if [[ -n "$(git status --porcelain)" ]]; then
  echo "There are changes:"
  git --no-pager diff --name-only
  echo
  echo "Run make generate, then commit those changes!"
  exit 1
fi
