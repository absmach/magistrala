BUILD_DIR = build
SERVICES = users things http normalizer ws influxdb mongodb cassandra
DOCKERS = $(addprefix docker_,$(SERVICES))
CGO_ENABLED ?= 0
GOOS ?= linux

define compile_service
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) go build -ldflags "-s -w" -o ${BUILD_DIR}/mainflux-$(1) cmd/$(1)/main.go
endef

define make_docker
	docker build --build-arg SVC_NAME=$(subst docker_,,$(1)) --tag=mainflux/$(subst docker_,,$(1)) -f docker/Dockerfile .
endef

all: $(SERVICES)

.PHONY: all $(SERVICES) dockers latest release

clean:
	rm -rf ${BUILD_DIR}

install:
	cp ${BUILD_DIR}/* $(GOBIN)

proto:
	protoc --go_out=plugins=grpc:. *.proto

$(SERVICES): proto
	$(call compile_service,$(@))

$(DOCKERS):
	$(call make_docker,$(@))

dockers: $(DOCKERS)
	docker build --tag=mainflux/dashflux -f dashflux/docker/Dockerfile dashflux
	docker build --tag=mainflux/mqtt -f mqtt/Dockerfile .

latest: dockers
	for svc in $(SERVICES); do \
		docker push mainflux/$$svc; \
	done
	docker push mainflux/dashflux
	docker push mainflux/mqtt

release:
	$(eval version = $(shell git describe --abbrev=0 --tags))
	git checkout $(version)
	$(MAKE) dockers
	for svc in $(SERVICES); do \
		docker tag mainflux/$$svc mainflux/$$svc:$(version); \
		docker push mainflux/$$svc:$(version); \
	done
	docker tag mainflux/dashflux mainflux/dashflux:$(version)
	docker push mainflux/dashflux:$(version)
	docker tag mainflux/mqtt mainflux/mqtt:$(version)
	docker push mainflux/mqtt:$(version)
