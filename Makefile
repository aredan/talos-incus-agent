REGISTRY ?= ghcr.io
IMAGE_NAME ?= aredan/talos-incus-agent
DAEMONSET_IMAGE_NAME ?= aredan/incus-agent-daemonset
TAG ?= 6.22.0
PLATFORM ?= linux/amd64

IMAGE := $(REGISTRY)/$(IMAGE_NAME):$(TAG)
DAEMONSET_IMAGE := $(REGISTRY)/$(DAEMONSET_IMAGE_NAME):$(TAG)

.PHONY: build push build-daemonset push-daemonset deploy clean

# Extension image (Talos system extension format)
build:
	docker build --platform $(PLATFORM) -t $(IMAGE) .

push: build
	docker push $(IMAGE)

# DaemonSet image (runtime container)
build-daemonset:
	docker build --platform $(PLATFORM) -f Dockerfile.daemonset -t $(DAEMONSET_IMAGE) .

push-daemonset: build-daemonset
	docker push $(DAEMONSET_IMAGE)

# Deploy DaemonSet to current kubectl context
deploy:
	kubectl apply -f deploy/daemonset.yaml

clean:
	docker rmi $(IMAGE) 2>/dev/null || true
	docker rmi $(DAEMONSET_IMAGE) 2>/dev/null || true
