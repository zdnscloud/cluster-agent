VERSION=`git describe --tags`
BUILD=`date +%FT%T%z`

LDFLAGS=-ldflags "-w -s -X main.version=${VERSION} -X main.build=${BUILD}"
GOSRC = $(shell find . -type f -name '*.go')

REGISTRY_NAME = zdnscloud
IMAGE_Name = cluster-agent
IMAGE_VERSION = v3.1.4

.PHONY: all container

all: container

container: 
	docker build -t $(REGISTRY_NAME)/$(IMAGE_Name):${IMAGE_VERSION} ./ --no-cache
	#docker build -t $(REGISTRY_NAME)/$(IMAGE_Name):$(VERSION) ./ --no-cache
