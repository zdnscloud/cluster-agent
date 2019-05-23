REGISTRY_NAME = zdnscloud
IMAGE_Name = cluster-agent
IMAGE_VERSION = v1.0

.PHONY: all container

all: container

container: 
	docker build -t $(REGISTRY_NAME)/$(IMAGE_Name):${IMAGE_VERSION} ./ --no-cache
