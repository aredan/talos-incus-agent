REGISTRY ?= ghcr.io
IMAGE_NAME ?= aredan/incus-agent-daemonset
TAG ?= 6.22.1
PLATFORM ?= linux/amd64

IMAGE := $(REGISTRY)/$(IMAGE_NAME):$(TAG)

.PHONY: build push deploy clean

build:
	docker build --platform $(PLATFORM) -t $(IMAGE) .

push: build
	docker push $(IMAGE)

deploy:
	kubectl apply -f deploy/daemonset.yaml

clean:
	docker rmi $(IMAGE) 2>/dev/null || true
