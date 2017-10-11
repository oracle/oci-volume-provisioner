DOCKER_REPO ?= registry.oracledx.com
DOCKER_USER ?= skeppare
DOCKER_IMAGE_NAME ?= bristol-oci-volume-provisioner-system-test

ifdef VERSION
DOCKER_IMAGE_TAG := ${VERSION}
else
DOCKER_IMAGE_TAG := 1.0.0
endif

.PHONY: build
build:
	docker build --build-arg=http_proxy --build-arg=https_proxy -t ${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}  -f Dockerfile .
	docker tag ${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG} ${DOCKER_REPO}/${DOCKER_USER}/${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}

.PHONY: push
push: build
	docker push ${DOCKER_REPO}/${DOCKER_USER}/${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}
