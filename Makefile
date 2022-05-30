APP = triggers

IMAGE_BASE ?= ghcr.io/shipwright-io
IMAGE_TAG ?= latest

GOFLAGS ?= -v -mod=vendor -race
GOFLAGS_TEST ?= -v -cover -race

NAMESPACE ?= shipwright-build

KO_DOCKER_REPO ?= $(IMAGE_BASE)
KO_OPTS ?= --base-import-paths --tags=${IMAGE_TAG}

.EXPORT_ALL_VARIABLES:

.PHONY: $(APP)
$(APP):
	go build .

build: $(APP)

default: build

.PHONY: container-build
container-build:
	ko build --push=false ${KO_OPTS} .

.PHONY: deploy
deploy:
	helm template \
		--namespace=$(NAMESPACE) \
		--set="image.name=ko://github.com/shipwright-io/triggers" \
		./chart | \
			ko apply ${KO_OPTS} --filename -

.PHONY: test-unit
test-unit:
	go test $(GOFLAGS_TEST) ./...

.PHONY: test-e2e
test-e2e:
	echo "Not implemented"

.PHONY: verify-kind
verify-kind:
	./hack/verify-kind.sh

.PHONY: install-registry
install-registry:
	./hack/install-registry.sh

.PHONY: install-shipwright
install-shipwright:
	./hack/install-tekton.sh
	./hack/install-shipwright.sh
