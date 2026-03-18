# Copyright (c) Abstract Machines
# SPDX-License-Identifier: Apache-2.0

SMQ_DOCKER_IMAGE_NAME_PREFIX ?= supermq
BUILD_DIR ?= build
SERVICES = auth users clients groups channels domains http coap mqtt notifications certs re postgres-writer postgres-reader timescale-writer timescale-reader cli alarms reports
TEST_API_SERVICES = journal auth certs http clients users channels groups domains
TEST_API = $(addprefix test_api_,$(TEST_API_SERVICES))
DOCKERS = $(addprefix docker_,$(SERVICES))
DOCKERS_DEV = $(addprefix docker_dev_,$(SERVICES))
CGO_ENABLED ?= 0
GOARCH ?= amd64
GOOS ?= linux
DETECTED_ARCH := $(shell uname -m)
VERSION ?= $(shell git describe --abbrev=0 --tags 2>/dev/null || echo 'unknown')
COMMIT ?= $(shell git rev-parse HEAD)
TIME ?= $(shell date +%F_%T)
USER_REPO ?= $(shell git remote get-url origin | sed -E 's@.*/([^/]+)/([^/.]+)(\.git)?@\1_\2@')
empty:=
space:= $(empty) $(empty)
# Docker compose project name should follow this guidelines: https://docs.docker.com/compose/reference/#use--p-to-specify-a-project-name
DOCKER_PROJECT ?= $(shell echo $(subst $(space),,$(USER_REPO)) | sed -E 's/[^a-zA-Z0-9]/_/g' | tr '[:upper:]' '[:lower:]')
DOCKER_COMPOSE_COMMANDS_SUPPORTED := up down config restart
DEFAULT_DOCKER_COMPOSE_COMMAND  := up
GRPC_MTLS_CERT_FILES_EXISTS = 0
MOCKERY = $(GOBIN)/mockery
MOCKERY_VERSION=3.6.4
PKG_PROTO_GEN_OUT_DIR=api/grpc
INTERNAL_PROTO_DIR=internal/proto
INTERNAL_PROTO_FILES := $(shell find $(INTERNAL_PROTO_DIR) -name "*.proto" | sed 's|$(INTERNAL_PROTO_DIR)/||')

ifneq ($(SMQ_MESSAGE_BROKER_TYPE),)
	SMQ_MESSAGE_BROKER_TYPE := $(SMQ_MESSAGE_BROKER_TYPE)
else
	SMQ_MESSAGE_BROKER_TYPE=msg_nats
endif

ifneq ($(SMQ_ES_TYPE),)
	SMQ_ES_TYPE := $(SMQ_ES_TYPE)
else
	SMQ_ES_TYPE=es_nats
endif

define compile_service
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) \
	go build -tags $(SMQ_MESSAGE_BROKER_TYPE) -tags $(SMQ_ES_TYPE) -ldflags "-s -w \
	-X 'github.com/absmach/supermq.BuildTime=$(TIME)' \
	-X 'github.com/absmach/supermq.Version=$(VERSION)' \
	-X 'github.com/absmach/supermq.Commit=$(COMMIT)'" \
	-o ${BUILD_DIR}/$(1) cmd/$(1)/main.go
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
		--tag=$(SMQ_DOCKER_IMAGE_NAME_PREFIX)/$(svc) \
		-f docker/Dockerfile .
endef

define make_docker_dev
	$(eval svc=$(subst docker_dev_,,$(1)))

	docker build \
		--no-cache \
		--build-arg SVC=$(svc) \
		--tag=$(SMQ_DOCKER_IMAGE_NAME_PREFIX)/$(svc) \
		-f docker/Dockerfile.dev ./build
endef

define run_with_arch_detection
	@echo "Detecting architecture..."
	@if [ "$(DETECTED_ARCH)" = "arm64" ] || [ "$(DETECTED_ARCH)" = "aarch64" ]; then \
		echo "ARM64 architecture detected."; \
		git checkout $(1); \
		GOARCH=arm64 $(MAKE) dockers; \
		for svc in $(SERVICES); do \
			docker tag supermq/$$svc supermq/$$svc:latest; \
			docker tag supermq/$$svc docker.io/supermq/$$svc:latest; \
		done; \
		sed -i.bak 's/^SMQ_RELEASE_TAG=.*/SMQ_RELEASE_TAG=latest/' docker/.env && rm -f docker/.env.bak; \
		docker compose -f docker/docker-compose.yaml --env-file docker/.env -p $(DOCKER_PROJECT) $(DOCKER_COMPOSE_COMMAND) $(args); \
	else \
		echo "x86_64 architecture detected."; \
		git checkout $(1); \
		sed -i.bak 's/^SMQ_RELEASE_TAG=.*/SMQ_RELEASE_TAG=$(2)/' docker/.env && rm -f docker/.env.bak; \
		docker compose -f docker/docker-compose.yaml --env-file docker/.env -p $(DOCKER_PROJECT) $(DOCKER_COMPOSE_COMMAND) $(args); \
	fi
endef

ADDON_SERVICES = journal bootstrap provision

EXTERNAL_SERVICES = prometheus

# Detect OS and architecture for cross-platform compatibility
UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

# macOS BSD sed vs GNU sed compatibility
ifeq ($(UNAME_S),Darwin)
	SED_INPLACE := sed -i ''
else
	SED_INPLACE := sed -i
endif

# Apple Silicon (arm64) Docker platform compatibility
# Pre-built images are amd64 only, so we need to use emulation on Apple Silicon
ifeq ($(UNAME_S),Darwin)
ifeq ($(UNAME_M),arm64)
	DOCKER_PLATFORM := DOCKER_DEFAULT_PLATFORM=linux/amd64
endif
endif
DOCKER_PLATFORM ?=

ifneq ($(filter run%,$(firstword $(MAKECMDGOALS))),)
  temp_args := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
  DOCKER_COMPOSE_COMMAND := $(if $(filter $(DOCKER_COMPOSE_COMMANDS_SUPPORTED),$(temp_args)), $(filter $(DOCKER_COMPOSE_COMMANDS_SUPPORTED),$(temp_args)), $(DEFAULT_DOCKER_COMPOSE_COMMAND))
  $(eval $(DOCKER_COMPOSE_COMMAND):;@)
endif

ifneq ($(filter run_addons%,$(firstword $(MAKECMDGOALS))),)
  temp_args := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
  RUN_ADDON_ARGS :=  $(if $(filter-out $(DOCKER_COMPOSE_COMMANDS_SUPPORTED),$(temp_args)), $(filter-out $(DOCKER_COMPOSE_COMMANDS_SUPPORTED),$(temp_args)),$(ADDON_SERVICES) $(EXTERNAL_SERVICES))
  $(eval $(RUN_ADDON_ARGS):;@)
endif

ifneq ("$(wildcard docker/ssl/certs/*-grpc-*)","")
GRPC_MTLS_CERT_FILES_EXISTS = 1
else
GRPC_MTLS_CERT_FILES_EXISTS = 0
endif

FILTERED_SERVICES = $(filter-out $(RUN_ADDON_ARGS), $(SERVICES))

all: $(SERVICES)

.PHONY: all $(SERVICES) dockers dockers_dev latest release run_latest run_stable run_addons grpc_mtls_certs check_mtls check_certs test_api mocks

clean:
	rm -rf ${BUILD_DIR}

cleandocker:
	# Stops containers and removes containers, networks, volumes, and images created by up
	docker compose -f docker/docker-compose.yaml -p $(DOCKER_PROJECT) down --rmi all -v --remove-orphans

ifdef pv
	# Remove unused volumes
	docker volume ls -f name=$(SMQ_DOCKER_IMAGE_NAME_PREFIX) -f dangling=true -q | xargs -r docker volume rm
endif

install:
	for file in $(BUILD_DIR)/*; do \
		cp $$file $(GOBIN)/supermq-`basename $$file`; \
	done

mocks: $(MOCKERY)
	@$(MOCKERY) --config ./tools/config/.mockery.yaml

$(MOCKERY):
	@mkdir -p $(GOBIN)
	@echo ">> installing mockery $(MOCKERY_VERSION)..."
	@go install github.com/vektra/mockery/v3@v$(MOCKERY_VERSION)

DIRS = consumers readers postgres internal
test: mocks
	mkdir -p coverage
	@for dir in $(DIRS); do \
		go test -v --race -failfast -count 1 -tags test -coverprofile=coverage/$$dir.out $$(go list ./... | grep $$dir | grep -v 'cmd'); \
	done
	go test -v --race -failfast -count 1 -tags test -coverprofile=coverage/coverage.out $$(go list ./... | grep -v 'consumers\|readers\|postgres\|internal\|cmd\|middleware')

define test_api_service
	$(eval svc=$(subst test_api_,,$(1)))
	@which uv > /dev/null || (echo "uv not found, please install it from https://github.com/astral-sh/uv" && exit 1)

	@if [ -z "$(USER_TOKEN)" ]; then \
		echo "USER_TOKEN is not set"; \
		echo "Please set it to a valid token"; \
		exit 1; \
	fi

	@if [ "$(svc)" = "http" ] && [ -z "$(CLIENT_SECRET)" ]; then \
		echo "CLIENT_SECRET is not set"; \
		echo "Please set it to a valid secret"; \
		exit 1; \
	fi

	@if [ "$(svc)" = "http" ]; then \
		uvx schemathesis run apidocs/openapi/$(svc).yaml \
		--checks all \
		--url $(2) \
		--header "Authorization: Client $(CLIENT_SECRET)" \
		--suppress-health-check=filter_too_much \
		--exclude-checks=positive_data_acceptance \
		--phases=examples,stateful; \
	else \
		uvx schemathesis run apidocs/openapi/$(svc).yaml \
		--checks all \
		--url $(2) \
		--header "Authorization: Bearer $(USER_TOKEN)" \
		--suppress-health-check=filter_too_much \
		--exclude-checks=positive_data_acceptance \
		--exclude-operation-id=requestPasswordReset \
		--phases=examples,stateful; \
	fi
endef

test_api_users: TEST_API_URL := http://localhost:9002
test_api_clients: TEST_API_URL := http://localhost:9006
test_api_domains: TEST_API_URL := http://localhost:9003
test_api_channels: TEST_API_URL := http://localhost:9005
test_api_groups: TEST_API_URL := http://localhost:9004
test_api_http: TEST_API_URL := http://localhost:8008
test_api_auth: TEST_API_URL := http://localhost:9001
test_api_certs: TEST_API_URL := http://localhost:9019
test_api_journal: TEST_API_URL := http://localhost:9021

$(TEST_API):
	$(call test_api_service,$(@),$(TEST_API_URL))

proto:
	protoc -I. --go_out=. --go_opt=paths=source_relative pkg/messaging/*.proto
	mkdir -p $(PKG_PROTO_GEN_OUT_DIR)
	protoc -I $(INTERNAL_PROTO_DIR) --go_out=$(PKG_PROTO_GEN_OUT_DIR) --go_opt=paths=source_relative --go-grpc_out=$(PKG_PROTO_GEN_OUT_DIR) --go-grpc_opt=paths=source_relative $(INTERNAL_PROTO_FILES)

$(FILTERED_SERVICES):
	$(call compile_service,$(@))

$(DOCKERS):
	$(call make_docker,$(@),$(GOARCH))

$(DOCKERS_DEV):
	$(call make_docker_dev,$(@))

dockers: $(DOCKERS)
dockers_dev: $(DOCKERS_DEV)

define docker_push
	for svc in $(SERVICES); do \
		docker push $(SMQ_DOCKER_IMAGE_NAME_PREFIX)/$$svc:$(1); \
	done
endef

changelog:
	git log $(shell git describe --tags --abbrev=0)..HEAD --pretty=format:"- %s"

latest: dockers
	$(call docker_push,latest)

publish_arch:
	$(MAKE) dockers GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM)
	for svc in $(SERVICES); do \
		docker tag $(SMQ_DOCKER_IMAGE_NAME_PREFIX)/$$svc $(SMQ_DOCKER_IMAGE_NAME_PREFIX)/$$svc:$(VERSION)-$(GOARCH); \
		docker tag $(SMQ_DOCKER_IMAGE_NAME_PREFIX)/$$svc $(SMQ_DOCKER_IMAGE_NAME_PREFIX)/$$svc:latest-$(GOARCH); \
		docker push $(SMQ_DOCKER_IMAGE_NAME_PREFIX)/$$svc:$(VERSION)-$(GOARCH); \
		docker push $(SMQ_DOCKER_IMAGE_NAME_PREFIX)/$$svc:latest-$(GOARCH); \
	done

release:
	$(eval version = $(shell git describe --abbrev=0 --tags))
	git checkout $(version)
	$(MAKE) dockers
	for svc in $(SERVICES); do \
		docker tag $(SMQ_DOCKER_IMAGE_NAME_PREFIX)/$$svc $(SMQ_DOCKER_IMAGE_NAME_PREFIX)/$$svc:$(version); \
	done
	$(call docker_push,$(version))

rundev:
	cd scripts && ./run.sh

grpc_mtls_certs:
	$(MAKE) -C docker/ssl auth_grpc_certs clients_grpc_certs

check_tls:
ifeq ($(GRPC_TLS),true)
	@echo "gRPC TLS is enabled"
	$(eval GRPC_MTLS :=)
else
	$(eval GRPC_TLS :=)
endif

check_mtls:
ifeq ($(GRPC_MTLS),true)
	@echo "gRPC MTLS is enabled"
	$(eval GRPC_TLS :=)
else
	$(eval GRPC_MTLS :=)
endif

check_certs: check_mtls check_tls
ifeq ($(GRPC_MTLS_CERT_FILES_EXISTS),0)
ifeq ($(filter true,$(GRPC_MTLS) $(GRPC_TLS)),true)
ifeq ($(filter $(DEFAULT_DOCKER_COMPOSE_COMMAND),$(DOCKER_COMPOSE_COMMAND)),$(DEFAULT_DOCKER_COMPOSE_COMMAND))
	$(MAKE) -C docker/ssl auth_grpc_certs clients_grpc_certs
endif
endif
endif

run_latest: check_certs
	git checkout main
	$(SED_INPLACE) 's/^SMQ_RELEASE_TAG=.*/SMQ_RELEASE_TAG=latest/' docker/.env
	$(DOCKER_PLATFORM) docker compose -f docker/docker-compose.yaml --env-file docker/.env -p $(DOCKER_PROJECT) $(DOCKER_COMPOSE_COMMAND) $(args)

run_stable: check_certs
	$(eval version = $(shell git describe --abbrev=0 --tags))
	git checkout $(version)
	$(SED_INPLACE) 's/^SMQ_RELEASE_TAG=.*/SMQ_RELEASE_TAG=$(version)/' docker/.env
	$(DOCKER_PLATFORM) docker compose -f docker/docker-compose.yaml --env-file docker/.env -p $(DOCKER_PROJECT) $(DOCKER_COMPOSE_COMMAND) $(args)

run_addons: check_certs
	$(foreach SVC,$(RUN_ADDON_ARGS),$(if $(filter $(SVC),$(ADDON_SERVICES) $(EXTERNAL_SERVICES)),,$(error Invalid Service $(SVC))))
	@$(DOCKER_PLATFORM) docker compose -f docker/docker-compose.yaml --env-file ./docker/.env -p $(DOCKER_PROJECT) up -d auth domains jaeger
	@for SVC in $(RUN_ADDON_ARGS); do \
		SMQ_ADDONS_CERTS_PATH_PREFIX="../" $(DOCKER_PLATFORM) docker compose -f docker/addons/$$SVC/docker-compose.yaml -p $(DOCKER_PROJECT) --env-file ./docker/.env $(DOCKER_COMPOSE_COMMAND) $(args) & \
	done

run_live: check_certs
	GOPATH=$(go env GOPATH) $(DOCKER_PLATFORM) docker compose -f docker/docker-compose.yaml -f docker/docker-compose-live.yaml --env-file docker/.env -p $(DOCKER_PROJECT) $(DOCKER_COMPOSE_COMMAND) $(args)
