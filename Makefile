NAME:=oci-provisioner

ifdef GITLAB_CI
BUILD_DIR:=${CI_PROJECT_DIR}/dist

ifdef CI_COMMIT_TAG
VERSION:=${CI_COMMIT_TAG}
else
VERSION=${CI_COMMIT_SHA}
endif

else
# localbuild user timestamp it
BUILD_DIR:=dist
VERSION:=${USER}-$(shell  date +%Y%m%d%H%M%S)
endif

BIN_DIR:=${BUILD_DIR}/bin
GOOS ?= linux

DOCKER_REPO ?= registry.oracledx.com
DOCKER_USER ?= skeppare
DOCKER_IMAGE_NAME ?= bristol-oci-volume-provisioner
DOCKER_IMAGE_TAG ?= ${VERSION}

BIN_NAME:=${NAME}

GO_SRC:=$(shell find . -name "*.go")

.PHONY: image vet deploy all build clean

all: build

deploy: push
	kubectl delete pod oci-provisioner -n oci || true
	kubectl create -f manifests/oci-provisioner.yaml

image: ${BIN_DIR}/${BIN_NAME}
	sed "s/{{VERSION}}/$(DOCKER_IMAGE_TAG)/g" manifests/oci-volume-provisioner.yaml > \
		$(BUILD_DIR)/oci-volume-provisioner.yaml
	cp manifests/namespace.yaml $(BUILD_DIR)
	cp manifests/storage-class.yaml $(BUILD_DIR)
	cp manifests/example-claim.yaml $(BUILD_DIR)
	cp manifests/example-pod.yaml $(BUILD_DIR)
	docker build --build-arg=http_proxy --build-arg=https_proxy -t ${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}  -f Dockerfile.scratch .
	docker tag ${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG} ${DOCKER_REPO}/${DOCKER_USER}/${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}

push: image
	docker login -u '$(DOCKER_REGISTRY_USERNAME)' -p '$(DOCKER_REGISTRY_PASSWORD)' $(DOCKER_REPO)
	docker push ${DOCKER_REPO}/${DOCKER_USER}/${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}

vet: ${GO_SRC}
	echo go vet $(shell go list ./... | grep -v /vendor/)

build: ${BIN_DIR}/${BIN_NAME}

${BIN_DIR}/${BIN_NAME}: ${GO_SRC}
	mkdir -p ${BIN_DIR}
	GOOS=$(GOOS) CGO_ENABLED=0 go build -i -v -ldflags '-extldflags "-static"' -o $@ ./cmd/oci-volume-provisioner/

clean:
	rm -rf ${BUILD_DIR}
