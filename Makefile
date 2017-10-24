GO_SRC := $(shell find . -name "*.go")
VERSION := $(shell git describe --always --dirty)
DIR := dist
BIN := oci-volume-provisioner
REGISTRY ?= wcr.io
DOCKER_REGISTRY_USERNAME ?= oracle
IMAGE ?= $(REGISTRY)/$(DOCKER_REGISTRY_USERNAME)/$(BIN)

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

${DIR}/${BIN}: ${GO_SRC}
	mkdir -p ${DIR}
	GOOS=${GOOS} GOARCH=${GOARCH} CGO_ENABLED=0 go build -i -v -ldflags '-extldflags "-static"' -o $@ ./cmd/

.PHONY: image
image: ${DIR}/${BIN}
	sed "s/{{VERSION}}/$(VERSION)/g" manifests/oci-volume-provisioner.yaml > $(DIR)/oci-volume-provisioner.yaml
	docker build -t ${IMAGE}:${VERSION} .

.PHONY: push
push: image
	docker login -u '$(DOCKER_REGISTRY_USERNAME)' -p '$(DOCKER_REGISTRY_PASSWORD)' $(REGISTRY)
	docker push ${IMAGE}:${VERSION}

.PHONY: clean
clean:
	rm -rf ${DIR}
