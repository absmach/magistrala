BUILD_DIR = build
SERVICES = manager http normalizer coap
DOCKERS = $(addprefix docker_,$(SERVICES))
CGO_ENABLED ?= 0
GOOS ?= linux

all: $(SERVICES)
.PHONY: all $(SERVICES) dockers

define compile_service
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) go build -ldflags "-s -w" -o ${BUILD_DIR}/mainflux-$(1) cmd/$(1)/main.go
endef

define make_docker
	docker build --build-arg SVC_NAME=$(subst docker_,,$(1)) --tag=mainflux/$(subst docker_,,$(1)) -f docker/Dockerfile .
endef

proto:
	protoc --go_out=. *.proto

$(SERVICES): proto
	$(call compile_service,$(@))

clean:
	rm -rf ${BUILD_DIR}

install:
	cp ${BUILD_DIR}/* $(GOBIN)

$(DOCKERS):
	$(call make_docker,$(@))

dockers: $(DOCKERS)
