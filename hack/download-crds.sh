#!/usr/bin/env bash

# Copyright The Shipwright Contributors
#
# SPDX-License-Identifier: Apache-2.0

#
# Downloads the Custom Resource Definitions direcly from the Build and Tekton's GitHub repositories,
# using `curl` to manage this task.
#

set -eu -o pipefail

# target shipwright version, by default uses the `main` revision
SHIPWRIGHT_VERSION="${SHIPWRIGHT_VERSION:-main}"
# target tekton version, by default uses the `main` revision
TEKTON_VERSION="${TEKTON_VERSION:-main}"

# diretory where the files will be stored
CRD_DIR="${CRD_DIR:-/var/tmp}"

readonly REPO_HOST="raw.githubusercontent.com"
readonly SHIPWRIGHT_REPO_PATH="shipwright-io/build/${SHIPWRIGHT_VERSION}/deploy/crds"
readonly TEKTON_REPO_PATH="tektoncd/pipeline/${TEKTON_VERSION}/config"

# list of shipwright crd files to be downloaded from the repository
declare -a SHIPWRIGHT_CRD_FILES=(
	shipwright.io_buildruns.yaml
	shipwright.io_buildstrategies.yaml
	shipwright.io_builds.yaml
	shipwright.io_clusterbuildstrategies.yaml
)

# list of tekton crd files to be downloaded from the repository
declare -a TEKTON_CRD_FILES=(
	300-clustertask.yaml
	300-customrun.yaml
	300-pipelinerun.yaml
	300-pipeline.yaml
	300-resolutionrequest.yaml
	300-resource.yaml
	300-taskrun.yaml
	300-task.yaml
	300-verificationpolicy.yaml
)

# executes curl with flags against the informed url, saves the payload on the output directory
function do_curl() {
	local URL_BASE="${1}"
	local FILENAME="${2}"
	curl \
		--location \
		--output "${CRD_DIR}/${FILENAME}" \
		--remote-header-name \
		--remote-name \
		--silent \
		"${URL_BASE}/${FILENAME}"
}

# creates teh output directory, when it does not exists
[[ -d "${CRD_DIR}" ]] || mkdir -p "${CRD_DIR}"

echo "# Shipwright '${SHIPWRIGHT_VERSION}' CRDs stored at: '${CRD_DIR}'"

for f in ${SHIPWRIGHT_CRD_FILES[@]}; do
	URL_BASE="https://${REPO_HOST}/${SHIPWRIGHT_REPO_PATH}"
	echo "# - ${URL_BASE}/${f}"
	do_curl "${URL_BASE}" "${f}"
	# The integration tests run without Conversion, therefore disable it and make beta the stored version
	goml delete -f "${CRD_DIR}/${f}" -p spec.conversion
	goml set -f "${CRD_DIR}/${f}" -p spec.versions.name:v1alpha1.storage -v false
	goml set -f "${CRD_DIR}/${f}" -p spec.versions.name:v1beta1.storage -v true
done

echo "# Tekton '${TEKTON_VERSION}' CRDs stored at: '${CRD_DIR}'"

for f in ${TEKTON_CRD_FILES[@]}; do
	URL_BASE="https://${REPO_HOST}/${TEKTON_REPO_PATH}"
	echo "# - ${URL_BASE}/${f}"
	do_curl "${URL_BASE}" "${f}"
	# The integration tests run without Conversion, therefore disable it
	goml delete -f "${CRD_DIR}/${f}" -p spec.conversion 2>/dev/null || true
done
