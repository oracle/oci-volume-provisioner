# Copyright (c) 2018 Oracle and/or its affiliates. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

GO_SRC := $(shell find . -name "*.go")
# Allow overriding for release versions
# Else just equal the build (git hash)
BUILD := $(shell git describe --tags --dirty --always)
ifeq ($(DEV_BUILD), true)
	# If DEV_BUILD is set, use the dev format.
	VERSION ?= ${BUILD}-${USER}-dev
else
	VERSION ?= ${BUILD}
endif
DIR := dist
BIN := oci-volume-provisioner
REGISTRY ?= iad.ocir.io
DOCKER_REGISTRY_TENANCY ?= spinnaker
DOCKER_REGISTRY_USERNAME ?= spinnaker/ioana-madalina.patrichi@oracle.com
IMAGE ?= $(REGISTRY)/$(DOCKER_REGISTRY_TENANCY)/$(BIN)
TEST_IMAGE ?= $(REGISTRY)/$(DOCKER_REGISTRY_TENANCY)/$(BIN)-test

OCI_SHORT_REGION ?= iad

GOOS ?= linux
GOARCH ?= amd64
SRC_DIRS := cmd pkg # directories which hold app source (not vendored)

.PHONY: all
all: gofmt golint govet test build

.PHONY: gofmt
gofmt:
	@./hack/check-gofmt.sh ${SRC_DIRS}

.PHONY: golint
golint:
	@./hack/check-golint.sh ${SRC_DIRS}

.PHONY: govet
govet:
	@./hack/check-govet.sh ${SRC_DIRS}

.PHONY: test
test:
	@./hack/test.sh $(SRC_DIRS)

.PHONY: build
build: ${DIR}/${BIN}
	sed 's#@VERSION@#${VERSION}#g; s#@IMAGE@#${IMAGE}#g' \
	 manifests/oci-volume-provisioner.yaml > $(DIR)/oci-volume-provisioner.yaml
	sed 's#@VERSION@#${VERSION}#g; s#@IMAGE@#${IMAGE}#g' \
	 manifests/oci-volume-provisioner-fss.yaml > $(DIR)/oci-volume-provisioner-fss.yaml
	cp manifests/storage-class.yaml $(DIR)/storage-class.yaml
	cp manifests/storage-class-ext3.yaml $(DIR)/storage-class-ext3.yaml
	cp manifests/oci-volume-provisioner-rbac.yaml $(DIR)/oci-volume-provisioner-rbac.yaml


${DIR}/${BIN}: ${GO_SRC}
	mkdir -p ${DIR}
	GOOS=${GOOS} GOARCH=${GOARCH} CGO_ENABLED=0 go build -i -v -ldflags '-extldflags "-static"' -o $@ ./cmd/

.PHONY: image
image: build
	docker build -t ${IMAGE}:${VERSION} -f Dockerfile .
	docker build -t ${TEST_IMAGE}:${VERSION} -f Dockerfile.test .

.PHONY: push
push: image
	docker login -u '$(DOCKER_REGISTRY_USERNAME)' -p '$(DOCKER_REGISTRY_PASSWORD)' $(REGISTRY)
	docker push ${IMAGE}:${VERSION}
	docker push ${TEST_IMAGE}:${VERSION}

.PHONY:system-test-config
system-test-config:
ifndef KUBECONFIG
ifndef KUBECONFIG_VAR
	$(error "KUBECONFIG or KUBECONFIG_VAR must be defined")
else
	$(eval KUBECONFIG:=/tmp/kubeconfig)
	$(eval export KUBECONFIG)
	$(shell echo "$${KUBECONFIG_VAR}" | openssl enc -base64 -d -A > $(KUBECONFIG))
endif
endif
ifndef OCICONFIG
ifdef OCICONFIG_VAR
	$(eval OCICONFIG:=/tmp/ociconfig)
	$(eval export OCICONFIG)
	$(shell echo "$${OCICONFIG_VAR}" | openssl enc -base64 -d -A > $(OCICONFIG))
	$(eval DOCKER_OPTIONS+=-e OCICONFIG=$(OCICONFIG) -v $(OCICONFIG):$(OCICONFIG))
	$(eval export DOCKER_OPTIONS)
endif
endif

.PHONY: system-test
system-test: system-test-config
	docker run -it ${DOCKER_OPTIONS} \
        -e KUBECONFIG=$(KUBECONFIG) \
        -v $(KUBECONFIG):$(KUBECONFIG) \
        -e HTTPS_PROXY=$$HTTPS_PROXY \
        ${TEST_IMAGE}:${VERSION} ${TEST_IMAGE_ARGS}

.PHONY: clean
clean:
	rm -rf ${DIR}

.PHONY: version
version:
	@echo ${VERSION}

.PHONY: run-dev
run-dev:
	NODE_NAME=$(shell hostname) OCI_SHORT_REGION=$(OCI_SHORT_REGION) go run cmd/main.go \
		--kubeconfig=${KUBECONFIG} \
		-v=4
