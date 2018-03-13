BUILD_DIR=build
SERVICES=manager http normalizer coap
DOCKERS=$(addprefix docker_,$(SERVICES)) 

all: $(SERVICES)
.PHONY: all $(SERVICES) docker

define compile_service
	GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) go build -ldflags "-s -w" -o ${BUILD_DIR}/mainflux-$(1) cmd/$(1)/main.go
endef

define make_docker
	docker build --build-arg SVC_NAME=$(subst docker_,,$(1)) --tag=mainflux/$(subst docker_,,$(1)) -f docker/Dockerfile .
endef

manager:
	$(call compile_service,$(@))

http:
	$(call compile_service,$(@))

normalizer:
	$(call compile_service,$(@))

coap:
	$(call compile_service,$(@))

clean:
	rm -rf ${BUILD_DIR}

install:
	cp ${BUILD_DIR}/* $(GOBIN)

# Docker
docker_manager:
	$(call make_docker,$(@))

docker_http:
	$(call make_docker,$(@))

docker_normalizer:
	$(call make_docker,$(@))

docker_coap:
	$(call make_docker,$(@))

docker: $(DOCKERS)

