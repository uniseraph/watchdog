SHELL = /bin/bash

GOLANG = golang:1.7.5

PROJECT = github.com/omega/watchdog

VERSION = $(shell cat VERSION)
GITCOMMIT = $(shell git log -1 --pretty=format:%h)
BUILD_TIME = $(shell date --rfc-3339 ns 2>/dev/null | sed -e 's/ /T/')

IMAGE_NAME = omega-reg/watchdog
REGISTRY = registry.cn-hangzhou.aliyuncs.com

build:
	docker run -v $(shell pwd):/go/src/${PROJECT} -w /go/src/${PROJECT} --rm ${GOLANG} make local

binary: build

local:
	@rm -rf bundles/${VERSION}
	mkdir -p bundles/${VERSION}/binary
	CGO_ENABLED=0 go build -v -ldflags "-X main.Version=${VERSION} -X main.GitCommit=${GITCOMMIT} -X main.BuildTime=${BUILD_TIME}" -o bundles/${VERSION}/binary/watchdog ${PROJECT}

image:
	docker build -t ${IMAGE_NAME}:${VERSION} .
	docker tag ${IMAGE_NAME}:${VERSION} ${IMAGE_NAME}:${VERSION}-${GITCOMMIT}
	docker tag ${IMAGE_NAME}:${VERSION}-${GITCOMMIT} ${IMAGE_NAME}:${VERSION}
	docker tag ${IMAGE_NAME}:${VERSION}-${GITCOMMIT} ${IMAGE_NAME}
	docker tag ${IMAGE_NAME}:${VERSION}-${GITCOMMIT} ${REGISTRY}/${IMAGE_NAME}:${VERSION}
	docker tag ${IMAGE_NAME}:${VERSION}-${GITCOMMIT} ${REGISTRY}/${IMAGE_NAME}:${VERSION}-${GITCOMMIT}

release:
	docker push ${IMAGE_NAME}:${VERSION}-${GITCOMMIT}
	docker push ${IMAGE_NAME}:${VERSION}
	docker push ${IMAGE_NAME}

run:
	docker run -ti --rm  --net=host -v /var/run/docker.sock:/var/run/docker.sock  omega-reg/watchdog:0.1.0 --log-level=info -H unix:///var/run/docker.sock consul://10.15.232.5:8500

.PHONY: build binary build-local image release
