DOCKER_REPO ?= wcr.io
DOCKER_USER ?= oracle
DOCKER_IMAGE_NAME ?= oci-volume-provisioner-system-test
DOCKER_IMAGE_TAG ?= 1.0.0

.PHONY: build
build:
	docker build --build-arg PROXY=${http_proxy} -t ${DOCKER_REPO}/${DOCKER_USER}/${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG} .

.PHONY: push
push: build
	docker login -u '$(DOCKER_REGISTRY_USERNAME)' -p '$(DOCKER_REGISTRY_PASSWORD)' $(DOCKER_REPO)
	docker push ${DOCKER_REPO}/${DOCKER_USER}/${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}
