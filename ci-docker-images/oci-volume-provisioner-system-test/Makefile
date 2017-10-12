DOCKER_REPO ?= registry.oracledx.com
DOCKER_USER ?= skeppare
DOCKER_IMAGE_NAME ?= oci-volume-provisioner-system-test
DOCKER_IMAGE_TAG ?= 1.0.0

.PHONY: build
build:
	docker build -t ${DOCKER_REPO}/${DOCKER_USER}/${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG} .

.PHONY: push
push: build
	docker push ${DOCKER_REPO}/${DOCKER_USER}/${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}
