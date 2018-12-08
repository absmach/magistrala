BUILD_DIR = build
SERVICES = users things http normalizer ws coap lora influxdb-writer influxdb-reader mongodb-writer mongodb-reader cassandra-writer cassandra-reader cli
DOCKERS = $(addprefix docker_,$(SERVICES))
DOCKERS_DEV = $(addprefix docker_dev_,$(SERVICES))
CGO_ENABLED ?= 0
GOOS ?= linux

define compile_service
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) go build -ldflags "-s -w" -o ${BUILD_DIR}/mainflux-$(1) cmd/$(1)/main.go
endef

define make_docker
	docker build --no-cache --build-arg SVC_NAME=$(subst docker_,,$(1)) --tag=mainflux/$(subst docker_,,$(1)) -f docker/Dockerfile .
endef

define make_docker_dev
	docker build --build-arg SVC_NAME=$(subst docker_dev_,,$(1)) --tag=mainflux/$(subst docker_dev_,,$(1)) -f docker/Dockerfile.dev ./build
endef

all: $(SERVICES) mqtt

.PHONY: all $(SERVICES) dockers dockers_dev latest release mqtt

clean:
	rm -rf ${BUILD_DIR}
	rm -rf mqtt/node_modules

cleandocker: cleanghost
	# Stop all containers (if running)
	docker-compose -f docker/docker-compose.yml stop
	# Remove mainflux containers
	docker ps -f name=mainflux -aq | xargs -r docker rm
	# Remove old mainflux images
	docker images -q mainflux\/* | xargs -r docker rmi

# Clean ghost docker images
cleanghost:
	# Remove exited containers
	docker ps -f status=dead -f status=exited -aq | xargs -r docker rm -v
	# Remove unused images
	docker images -f dangling=true -q | xargs -r docker rmi
	# Remove unused volumes
	docker volume ls -f dangling=true -q | xargs -r docker volume rm

install:
	cp ${BUILD_DIR}/* $(GOBIN)

test:
	GOCACHE=off go test -v -race -tags test $(shell go list ./... | grep -v 'vendor\|cmd')

proto:
	protoc --gofast_out=plugins=grpc:. *.proto

$(SERVICES):
	$(call compile_service,$(@))

$(DOCKERS):
	$(call make_docker,$(@))

dockers: $(DOCKERS)
	docker build --tag=mainflux/dashflux -f dashflux/docker/Dockerfile dashflux
	docker build --tag=mainflux/mqtt -f mqtt/Dockerfile .

$(DOCKERS_DEV):
	$(call make_docker_dev,$(@))

dockers_dev: $(DOCKERS_DEV)

mqtt:
	cd mqtt && npm install

define docker_push
	for svc in $(SERVICES); do \
		docker push mainflux/$$svc:$(1); \
	done
	docker push mainflux/dashflux:$(1)
	docker push mainflux/mqtt:$(1)
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
	docker tag mainflux/dashflux mainflux/dashflux:$(version)
	docker tag mainflux/mqtt mainflux/mqtt:$(version)
	$(call docker_push,$(version))

rundev:
	cd scripts && ./run.sh

run:
	docker-compose -f docker/docker-compose.yml up

runlora:
	docker-compose -f docker/docker-compose.yml up -d
	docker-compose -f docker/addons/influxdb-writer/docker-compose.yml up -d
	docker-compose -f docker/addons/lora-adapter/docker-compose.yml up
