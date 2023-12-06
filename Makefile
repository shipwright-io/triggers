APP = triggers

# temporary directory to store auxiliary tools
LOCAL_BIN ?= $(shell pwd)/bin

# full path to the application executable
BIN ?= $(LOCAL_BIN)/$(APP)

# container image prefix, the final part and tag are appended afterwards
IMAGE_BASE ?= ghcr.io/shipwright-io
IMAGE_TAG ?= latest

# golang flags and settings
GOFLAGS ?= -v -a
GOFLAGS_TEST ?= -v -race -cover
CGO_ENABLED ?= 0

# deployment target namespace, same default than Shipwright Build project
NAMESPACE ?= shipwright-build

# ko base image repository and options
IMAGE_HOST ?= ghcr.io
IMAGE_NAMESPACE ?= shipwright-io/triggers
KO_DOCKER_REPO ?= $(IMAGE_HOST)/$(IMAGE_NAMESPACE)
KO_OPTS ?= --base-import-paths --tags=${IMAGE_TAG}

# controller-gen version and full path to the executable
CONTROLLER_TOOLS_VERSION ?= v0.10.0
CONTROLLER_GEN ?= $(LOCAL_BIN)/controller-gen

# envtest version and full path to the executable
ENVTEST_K8S_VERSION ?= 1.25
ENVTEST ?= $(LOCAL_BIN)/setup-envtest

# chart base directory and path to the "templates" folder
CHART_DIR ?= ./chart
MANIFEST_DIR ?= $(CHART_DIR)/generated

# shipwright and tekton target versions to download upstream crd resources
SHIPWRIGHT_VERSION ?= v0.12.0
TEKTON_VERSION ?= v0.53.2

# full path to the directory where the crds are downloaded
CRD_DIR ?= $(LOCAL_BIN)/crds

# generic arguments used on certain targets
ARGS ?=

.EXPORT_ALL_VARIABLES:

# uses helm to render the resource templates to the stdout
define render-template
    @helm template \
		--namespace=$(NAMESPACE) \
		--set="image.name=ko://github.com/shipwright-io/triggers" \
		shipwright-triggers \
		$(CHART_DIR)
endef

default: build

# ensure that the local "bin" directory exists
$(LOCAL_BIN):
	@mkdir -p $(LOCAL_BIN) || true

# builds the primary application executable
.PHONY: $(BIN)
build: $(BIN)
$(BIN): $(LOCAL_BIN)
	go build -o $(BIN) .

# downloads shipwright crds from upstream repository
download-crds: $(CRD_DIR)
$(CRD_DIR):
	./hack/download-crds.sh

# installs controller-gen in the local bin folder
.PHONY: controller-gen
controller-gen: GOBIN=$(LOCAL_BIN)
controller-gen: $(CONTROLLER_GEN)
$(CONTROLLER_GEN):
	go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

# generates all Kubernetes releated resources in the project
.PHONY: manifests
manifests: controller-gen
	$(CONTROLLER_GEN) \
		rbac:roleName=shipwright-trigger crd paths="./..." \
		output:dir=$(MANIFEST_DIR)

# runs the manager from your host
.PHONY: run
run: manifests
	go run ./main.go $(ARGS)

# builds the container image with ko without push to registry
.PHONY: container-build
container-build: CGO_ENABLED=0
container-build:
	ko build --push=false $(KO_OPTS) $(ARGS) .

# uses helm to render kubernetes manifests and ko for the container image
.PHONY: deploy
deploy: CGO_ENABLED=0
deploy:
	$(call render-template) | \
			ko apply $(KO_OPTS) $(ARGS) --filename -

# uses helm to create a package and ko for the container image
.PHONY: release
release: CGO_ENABLED=0
release:
	$(call render-template) | \
			ko resolve $(KO_OPTS) $(ARGS) --platform=all -R -f - > release.yaml

# runs the unit tests, with optional arguments
.PHONY: test-unit
test-unit: CGO_ENABLED=1
test-unit:
	go test $(GOFLAGS_TEST) $(ARGS) ./pkg/... ./controllers/...

# installs latest envtest-setup, if necessary
.PHONY: envtest
envtest: GOBIN=$(LOCAL_BIN)
envtest: $(ENVTEST)
$(ENVTEST):
	go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

# run integration tests, with optional arguments
.PHONY: test-integration
test-integration: CGO_ENABLED=1
test-integration: KUBEBUILDER_ATTACH_CONTROL_PLANE_OUTPUT=true
test-integration: download-crds manifests envtest
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" \
		go test $(GOFLAGS_TEST) ./test/integration/... \
			-coverprofile=integration.out -ginkgo.v $(ARGS)

# run end-to-end tests
.PHONY: test-e2e
test-e2e:
	echo "Not implemented"

# runs act, with optional arguments
.PHONY: act
act:
	@act --secret="GITHUB_TOKEN=${GITHUB_TOKEN}" $(ARGS)

# removes the output directory
.PHONY: clean
clean:
	rm -rf "$(LOCAL_BIN)" || true

# calls everything that generates files
.PHONY: generate
generate: manifests
	./hack/generate-copyright.sh

.PHONY: verify-generate
verify-generate: generate
	./hack/verify-generate.sh

GO_LINT=$(shell which golangci-lint 2> /dev/null || echo '')
GO_LINT_URI=github.com/golangci/golangci-lint/cmd/golangci-lint@latest

GO_SEC=$(shell which gosec 2> /dev/null || echo '')
GO_SEC_URI=github.com/securego/gosec/v2/cmd/gosec@latest

GO_VULNCHECK=$(shell which govulncheck 2> /dev/null || echo '')
GO_VULNCHECK_URI=golang.org/x/vuln/cmd/govulncheck@latest

.PHONY: lint
golint:
	$(if $(GO_LINT), ,go install $(GO_LINT_URI))
	@echo "##### Running golangci-lint #####"
	golangci-lint run -v

.PHONY: sec
gosec:
	$(if $(GO_SEC), ,go install $(GO_SEC_URI))
	@echo "##### Running gosec #####"
	gosec ./...

.PHONY: govulncheck
govulncheck:
	$(if $(GO_VULNCHECK), ,go install $(GO_VULNCHECK_URI))
	@echo "##### Running govulncheck #####"
	govulncheck ./...

.PHONY: verify
verify: verify-generate gosec govulncheck golint
