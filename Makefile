# Copyright (c) Mainflux
# SPDX-License-Identifier: Apache-2.0

MF_DOCKER_IMAGE_NAME_PREFIX ?= mainflux
BUILD_DIR = build
SERVICES = users things http coap lora influxdb-writer influxdb-reader mongodb-writer \
	mongodb-reader cassandra-writer cassandra-reader postgres-writer postgres-reader timescale-writer timescale-reader cli \
	bootstrap opcua auth twins mqtt provision certs smtp-notifier smpp-notifier
DOCKERS = $(addprefix docker_,$(SERVICES))
DOCKERS_DEV = $(addprefix docker_dev_,$(SERVICES))
CGO_ENABLED ?= 0
GOARCH ?= amd64
VERSION ?= $(shell git describe --abbrev=0 --tags)
COMMIT ?= $(shell git rev-parse HEAD)
TIME ?= $(shell date +%F_%T)

define compile_service
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) \
	go build -mod=vendor -ldflags "-s -w \
	-X 'github.com/mainflux/mainflux.BuildTime=$(TIME)' \
	-X 'github.com/mainflux/mainflux.Version=$(VERSION)' \
	-X 'github.com/mainflux/mainflux.Commit=$(COMMIT)'" \
	-o ${BUILD_DIR}/mainflux-$(1) cmd/$(1)/main.go
endef

define make_docker
	$(eval svc=$(subst docker_,,$(1)))

	docker build \
		--no-cache \
		--build-arg SVC=$(svc) \
		--build-arg GOARCH=$(GOARCH) \
		--build-arg GOARM=$(GOARM) \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg TIME=$(TIME) \
		--tag=$(MF_DOCKER_IMAGE_NAME_PREFIX)/$(svc) \
		-f docker/Dockerfile .
endef

define make_docker_dev
	$(eval svc=$(subst docker_dev_,,$(1)))

	docker build \
		--no-cache \
		--build-arg SVC=$(svc) \
		--tag=$(MF_DOCKER_IMAGE_NAME_PREFIX)/$(svc) \
		-f docker/Dockerfile.dev ./build
endef

all: $(SERVICES)

.PHONY: all $(SERVICES) dockers dockers_dev latest release

clean:
	rm -rf ${BUILD_DIR}

cleandocker:
	# Stops containers and removes containers, networks, volumes, and images created by up
	docker-compose -f docker/docker-compose.yml down --rmi all -v --remove-orphans

ifdef pv
	# Remove unused volumes
	docker volume ls -f name=$(MF_DOCKER_IMAGE_NAME_PREFIX) -f dangling=true -q | xargs -r docker volume rm
endif

install:
	cp ${BUILD_DIR}/* $(GOBIN)

test:
	go test -mod=vendor -v -race -count 1 -tags test $(shell go list ./... | grep -v 'vendor\|cmd')

proto:
	protoc --gofast_out=plugins=grpc:. *.proto
	protoc --gofast_out=plugins=grpc:. pkg/messaging/*.proto

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
		docker push $(MF_DOCKER_IMAGE_NAME_PREFIX)/$$svc:$(1); \
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
		docker tag $(MF_DOCKER_IMAGE_NAME_PREFIX)/$$svc $(MF_DOCKER_IMAGE_NAME_PREFIX)/$$svc:$(version); \
	done
	$(call docker_push,$(version))

rundev:
	cd scripts && ./run.sh

run:
	docker-compose -f docker/docker-compose.yml up
