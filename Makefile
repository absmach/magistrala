# Copyright (c) Mainflux
# SPDX-License-Identifier: Apache-2.0

BUILD_DIR = build
SERVICES = users things http coap lora influxdb-writer influxdb-reader mongodb-writer \
	mongodb-reader cassandra-writer cassandra-reader postgres-writer postgres-reader cli \
	bootstrap opcua authn twins mqtt provision
DOCKERS = $(addprefix docker_,$(SERVICES))
DOCKERS_DEV = $(addprefix docker_dev_,$(SERVICES))
CGO_ENABLED ?= 0
GOARCH ?= amd64

define compile_service
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) go build -mod=vendor -ldflags "-s -w" -o ${BUILD_DIR}/mainflux-$(1) cmd/$(1)/main.go
endef

define make_docker
	$(eval svc=$(subst docker_,,$(1)))

	docker build \
		--no-cache \
		--build-arg SVC=$(svc) \
		--build-arg GOARCH=$(GOARCH) \
		--build-arg GOARM=$(GOARM) \
		--tag=mainflux/$(svc) \
		-f docker/Dockerfile .
endef

define make_docker_dev
	$(eval svc=$(subst docker_dev_,,$(1)))

	docker build \
		--no-cache \
		--build-arg SVC=$(svc) \
		--tag=mainflux/$(svc) \
		-f docker/Dockerfile.dev ./build
endef

all: $(SERVICES)

.PHONY: all $(SERVICES) dockers dockers_dev latest release

clean:
	rm -rf ${BUILD_DIR}

cleandocker:
	# Stop all containers (if running)
	docker-compose -f docker/docker-compose.yml stop
	# Remove mainflux containers
	docker ps -f name=mainflux -aq | xargs -r docker rm

	# Remove exited containers
	docker ps -f name=mainflux -f status=dead -f status=exited -aq | xargs -r docker rm -v

	# Remove unused images
	docker images "mainflux\/*" -f dangling=true -q | xargs -r docker rmi

	# Remove old mainflux images
	docker images -q mainflux\/* | xargs -r docker rmi

ifdef pv
	# Remove unused volumes
	docker volume ls -f name=mainflux -f dangling=true -q | xargs -r docker volume rm
endif

install:
	cp ${BUILD_DIR}/* $(GOBIN)

test:
	go test -mod=vendor -v -race -count 1 -tags test $(shell go list ./... | grep -v 'vendor\|cmd')

proto:
	protoc --gofast_out=plugins=grpc:. *.proto
	protoc --gofast_out=plugins=grpc:. messaging/*.proto

$(SERVICES):
	$(call compile_service,$(@))

$(DOCKERS):
	$(call make_docker,$(@),$(GOARCH))

$(DOCKERS_DEV):
	$(call make_docker_dev,$(@))

dockers: $(DOCKERS)
dockers_dev: $(DOCKERS_DEV)

define docker_push
	for svc in $(SERVICES); do \
		docker push mainflux/$$svc:$(1); \
	done
endef

changelog:
	git log $(shell git describe --tags --abbrev=0)..HEAD --pretty=format:"- %s"

latest: dockers
	$(call docker_push,latest)

release:
	$(eval version = $(shell git describe --abbrev=0 --tags))
	git checkout $(version)
	$(MAKE) dockers
	for svc in $(SERVICES); do \
		docker tag mainflux/$$svc mainflux/$$svc:$(version); \
	done
	$(call docker_push,$(version))

rundev:
	cd scripts && ./run.sh

run:
	docker-compose -f docker/docker-compose.yml up

runlora:
	docker-compose \
		-f docker/docker-compose.yml \
		-f docker/addons/influxdb-writer/docker-compose.yml \
		-f docker/addons/lora-adapter/docker-compose.yml up \

# Run all Mainflux core services except distributed tracing system - Jaeger. Recommended on gateways:
rungw:
	MF_JAEGER_URL= docker-compose -f docker/docker-compose.yml up --scale jaeger=0
