DOCKER_REGISTRY ?= wcr.io
DOCKER_IMAGE_NAME ?= oci-volume-provisioner-system-test
DOCKER_IMAGE_TAG ?= 1.0.2

.PHONY: build
build:
	docker build --build-arg http_proxy=${http_proxy} \
		         --build-arg https_proxy=${https_proxy} \
				 -t ${DOCKER_REGISTRY}/${DOCKER_REGISTRY_USERNAME}/${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG} .

.PHONY: push
push: build
	docker login -u '$(DOCKER_REGISTRY_USERNAME)' -p '$(DOCKER_REGISTRY_PASSWORD)' $(DOCKER_REGISTRY)
	docker push ${DOCKER_REGISTRY}/${DOCKER_REGISTRY_USERNAME}/${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}
