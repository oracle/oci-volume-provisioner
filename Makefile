ifndef VERSION
VERSION:=${USER}-$(shell  date +%Y%m%d%H%M%S)
endif

GOOS ?= linux
BUILD_DIR:=dist
BIN_NAME:=oci-volume-provisioner
GO_SRC:=$(shell find . -name "*.go")
DOCKER_REPO ?= registry.oracledx.com
DOCKER_USER ?= skeppare
DOCKER_IMAGE_NAME ?= oci-volume-provisioner
DOCKER_IMAGE_TAG ?= ${VERSION}

.PHONY: all fmt lint vet build image push deploy clean

all: build

fmt: ${GO_SRC}
	go fmt $(shell go list ./... | grep -v /vendor/)

lint: ${GO_SRC}
	golint $(shell go list ./... | grep -v /vendor/)

vet: ${GO_SRC}
	go vet $(shell go list ./... | grep -v /vendor/)

build: ${BUILD_DIR}/${BIN_NAME}

${BUILD_DIR}/${BIN_NAME}: ${GO_SRC}
	mkdir -p ${BUILD_DIR}
	GOOS=$(GOOS) CGO_ENABLED=0 go build -i -v -ldflags '-extldflags "-static"' -o $@ ./cmd/oci-volume-provisioner/

image: ${BUILD_DIR}/${BIN_NAME}
	sed "s/{{VERSION}}/$(DOCKER_IMAGE_TAG)/g" manifests/oci-volume-provisioner.yaml > $(BUILD_DIR)/oci-volume-provisioner.yaml
	docker build -t ${DOCKER_REPO}/${DOCKER_USER}/${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG} .

push: image
	docker login -u '$(DOCKER_REGISTRY_USERNAME)' -p '$(DOCKER_REGISTRY_PASSWORD)' $(DOCKER_REPO)
	docker push ${DOCKER_REPO}/${DOCKER_USER}/${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}

deploy:
	kubectl delete pod oci-volume-provisioner -n oci || true
	kubectl create -f dist/oci-volume-provisioner.yaml

clean:
	rm -rf ${BUILD_DIR}
