#!/bin/bash

# Copyright The Shipwright Contributors
# 
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

GO_FLAGS=${GO_FLAGS:-""}

echo "Logging into container registry $IMAGE_HOST"
echo "$REGISTRY_PASSWORD" | ko login -u "$REGISTRY_USERNAME" --password-stdin "$IMAGE_HOST"

echo "Building container image"

echo "Adding io.shipwright.vcs-ref label with value: ${GITHUB_SHA}"

KO_DOCKER_REPO="${IMAGE_HOST}/${IMAGE_NAMESPACE}" GOFLAGS="${GO_FLAGS}" ko resolve \
  --base-import-paths \
  --tags "${TAG}" \
  --image-label "io.shipwright.vcs-ref=${GITHUB_SHA}" \
  --platform=all  -R -f deploy/ > release.yaml

KO_DOCKER_REPO="${IMAGE_HOST}/${IMAGE_NAMESPACE}" GOFLAGS="${GO_FLAGS} -tags=pprof_enabled" ko resolve \
  --base-import-paths \
  --tags "${TAG}-debug" \
  --image-label "io.shipwright.vcs-ref=${GITHUB_SHA}" \
  --platform=all  -R -f deploy/ > release-debug.yaml
