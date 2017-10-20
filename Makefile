GO_SRC := $(shell find . -name "*.go")
VERSION := $(shell git describe --always --dirty)
DIR := dist
BIN := oci-volume-provisioner
REGISTRY ?= wcr.io
DOCKER_REGISTRY_USERNAME ?= oracle
IMAGE ?= $(REGISTRY)/$(DOCKER_REGISTRY_USERNAME)/$(BIN)

GOOS ?= linux
GOARCH ?= amd64
SRC_DIRS := cmd # directories which hold app source (not vendored)

.PHONY: all gofmt golint govet build image push deploy clean

all: build

gofmt:
	@./hack/check-gofmt.sh ${SRC_DIRS}

golint:
	@./hack/check-golint.sh ${SRC_DIRS}

govet:
	@./hack/check-govet.sh ${SRC_DIRS}

build: ${DIR}/${BIN}

${DIR}/${BIN}: ${GO_SRC}
	mkdir -p ${DIR}
	GOOS=${GOOS} GOARCH=${GOARCH} CGO_ENABLED=0 go build -i -v -ldflags '-extldflags "-static"' -o $@ ./cmd/oci-volume-provisioner/

image: ${DIR}/${BIN}
	sed "s/{{VERSION}}/$(VERSION)/g" manifests/oci-volume-provisioner.yaml > $(DIR)/oci-volume-provisioner.yaml
	docker build -t ${IMAGE}:${VERSION} .

push: image
	docker login -u '$(DOCKER_REGISTRY_USERNAME)' -p '$(DOCKER_REGISTRY_PASSWORD)' $(REGISTRY)
	docker push ${IMAGE}:${VERSION}

clean:
	rm -rf ${DIR}
