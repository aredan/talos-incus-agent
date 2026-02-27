REGISTRY ?= ghcr.io
IMAGE_NAME ?= aredan/talos-incus-agent
TAG ?= v6.22.0
PLATFORM ?= linux/amd64

IMAGE := $(REGISTRY)/$(IMAGE_NAME):$(TAG)

.PHONY: build push clean

build:
	docker build --platform $(PLATFORM) -t $(IMAGE) .

push: build
	docker push $(IMAGE)

clean:
	docker rmi $(IMAGE) 2>/dev/null || true
