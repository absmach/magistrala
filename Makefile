# Copyright (c) Abstract Machines
# SPDX-License-Identifier: Apache-2.0

override MG_DOCKER_IMAGE_NAME_PREFIX := ghcr.io/absmach/magistrala
MG_DOCKER_VOLUME_NAME_PREFIX ?= magistrala
BUILD_DIR ?= build
SERVICES = atom-bootstrap bootstrap certs postgres-writer postgres-reader timescale-writer timescale-reader fluxmq
CLI = cli
TEST_API_SERVICES = certs clients users channels groups workspaces
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
ATOM_TOKENS_ENV ?= docker/.env.tokens
REQUIRED_ATOM_TOKEN_ENVS := MG_ATOM_TOKEN_FLUXMQ_AUTH MG_ATOM_TOKEN_FLUXMQ_NODE1 MG_ATOM_TOKEN_FLUXMQ_NODE2 MG_ATOM_TOKEN_FLUXMQ_NODE3 MG_ATOM_TOKEN_TIMESCALE_READER MG_ATOM_TOKEN_RE MG_ATOM_TOKEN_ALARMS MG_ATOM_TOKEN_REPORTS MG_ATOM_TOKEN_POSTGRES_READER MG_ATOM_TOKEN_BOOTSTRAP
PROVISION_ATOM_TOKENS ?= false
PROVISION_ATOM_TOKEN_GOALS := provision-atom-tokens
DOCKER_BASE_ENV_FILES := --env-file docker/.env
DOCKER_ENV_FILES = $(if $(filter down,$(DOCKER_COMPOSE_COMMAND)),$(DOCKER_BASE_ENV_FILES),$(DOCKER_BASE_ENV_FILES) --env-file $(ATOM_TOKENS_ENV))
DOCKER_PROVISION_ENV_FILES = $(DOCKER_BASE_ENV_FILES) $(if $(wildcard $(ATOM_TOKENS_ENV)),--env-file $(ATOM_TOKENS_ENV))
HOST_UID := $(shell id -u)
HOST_GID := $(shell id -g)
GRPC_MTLS_CERT_FILES_EXISTS = 0
MOCKERY = $(GOBIN)/mockery
MOCKERY_VERSION=3.7.4
PKG_PROTO_GEN_OUT_DIR=api/grpc
INTERNAL_PROTO_DIR=internal/proto
INTERNAL_PROTO_FILES := $(shell find $(INTERNAL_PROTO_DIR) -name "*.proto" | sed 's|$(INTERNAL_PROTO_DIR)/||')

ifneq ($(MG_MESSAGE_BROKER_TYPE),)
	MG_MESSAGE_BROKER_TYPE := $(MG_MESSAGE_BROKER_TYPE)
else
	MG_MESSAGE_BROKER_TYPE=msg_fluxmq
endif

ifneq ($(MG_ES_TYPE),)
	MG_ES_TYPE := $(MG_ES_TYPE)
else
	MG_ES_TYPE=es_fluxmq
endif

BUILD_TAGS := $(strip $(MG_MESSAGE_BROKER_TYPE) $(MG_ES_TYPE))

define compile_service
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) \
	go build -tags "$(BUILD_TAGS)" -ldflags "-s -w \
	-X 'github.com/absmach/magistrala.BuildTime=$(TIME)' \
	-X 'github.com/absmach/magistrala.Version=$(VERSION)' \
	-X 'github.com/absmach/magistrala.Commit=$(COMMIT)'" \
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
		--build-arg BUILD_TAGS="$(BUILD_TAGS)" \
		--tag=$(MG_DOCKER_IMAGE_NAME_PREFIX)/$(svc) \
		-f docker/Dockerfile .
endef

define make_docker_dev
	$(eval svc=$(subst docker_dev_,,$(1)))

	docker build \
		--no-cache \
		--build-arg SVC=$(svc) \
		--tag=$(MG_DOCKER_IMAGE_NAME_PREFIX)/$(svc) \
		-f docker/Dockerfile.dev ./build
endef

define require_atom_tokens_env
	@if [ -z "$(filter down,$(DOCKER_COMPOSE_COMMAND))" ]; then \
		if [ ! -f "$(ATOM_TOKENS_ENV)" ]; then \
			echo "Missing $(ATOM_TOKENS_ENV). Run 'make provision_atom_tokens' before starting the Docker Compose stack."; \
			exit 2; \
		fi; \
		missing=""; \
		for env_name in $(REQUIRED_ATOM_TOKEN_ENVS); do \
			if ! grep -q "^$${env_name}=" "$(ATOM_TOKENS_ENV)"; then \
				missing="$${missing} $${env_name}"; \
			fi; \
		done; \
		if [ -n "$${missing}" ]; then \
			echo "Missing Atom service token(s) in $(ATOM_TOKENS_ENV):$${missing}. Run 'make provision_atom_tokens' before starting the Docker Compose stack."; \
			exit 2; \
		fi; \
	fi
endef

define ensure_atom_tokens_env
	@if [ "$(PROVISION_ATOM_TOKENS)" = "true" ] && [ -z "$(filter down,$(DOCKER_COMPOSE_COMMAND))" ]; then \
		$(MAKE) provision_atom_tokens; \
	elif [ -z "$(filter down,$(DOCKER_COMPOSE_COMMAND))" ]; then \
		if [ ! -f "$(ATOM_TOKENS_ENV)" ]; then \
			echo "Missing $(ATOM_TOKENS_ENV). Run 'make provision_atom_tokens' before starting the Docker Compose stack."; \
			exit 2; \
		fi; \
		missing=""; \
		for env_name in $(REQUIRED_ATOM_TOKEN_ENVS); do \
			if ! grep -q "^$${env_name}=" "$(ATOM_TOKENS_ENV)"; then \
				missing="$${missing} $${env_name}"; \
			fi; \
		done; \
		if [ -n "$${missing}" ]; then \
			echo "Missing Atom service token(s) in $(ATOM_TOKENS_ENV):$${missing}. Run 'make provision_atom_tokens' before starting the Docker Compose stack."; \
			exit 2; \
		fi; \
	fi
endef

# Atom hard-requires a working connection to ATOM_EVENTS_AMQP_URL at every
# boot once that variable is set (see absmach/atom src/main.rs) -- but nginx
# (the AMQP proxy in front of FluxMQ, MG_NGINX_AMQP_PORT) and the FluxMQ
# auth/broker chain behind it depend on atom-bootstrap completing, which
# depends on atom itself being reachable. A single `docker compose up` for
# the whole stack deadlocks: atom can never reach nginx, because nginx can
# never start, because the chain behind it can never finish bootstrapping
# against an atom that keeps failing to connect to nginx.
#
# Break the deadlock by bringing the bootstrap chain all the way up to nginx
# first, with event publishing disabled for this one invocation -- an empty
# ATOM_EVENTS_AMQP_URL override takes precedence over docker/.env's value --
# so atom boots cleanly and atom-bootstrap, fluxmq-auth and the FluxMQ nodes
# behind it can all finish starting. Targeting nginx (not just fluxmq-auth)
# matters: the regular `up` that follows recreates atom with the real value,
# and separately -- since docker compose's `up` restarts *any* non-running
# service it manages, including atom-bootstrap after its clean exit(0) --
# re-runs atom-bootstrap too. Both need the AMQP proxy nginx fronts to
# already be genuinely functional at that point, not merely "started", or
# the same deadlock reopens inside the second invocation. Targeting nginx
# pulls in the full chain (atom-db -> atom -> atom-bootstrap -> fluxmq-auth
# -> fluxmq-node1/2/3 -> nginx) via the compose file's own depends_on
# conditions, which is the correct place to express "wait for this to
# actually finish", not a guessed sleep.
define bootstrap_atom_events_amqp
	@if [ -z "$(filter down,$(DOCKER_COMPOSE_COMMAND))" ]; then \
		ATOM_EVENTS_AMQP_URL= $(DOCKER_PLATFORM) docker compose -f docker/docker-compose.yaml $(DOCKER_ENV_FILES) -p $(DOCKER_PROJECT) up -d nginx; \
	fi
endef

define run_with_arch_detection
	$(call require_atom_tokens_env)
	@echo "Detecting architecture..."
	@if [ "$(DETECTED_ARCH)" = "arm64" ] || [ "$(DETECTED_ARCH)" = "aarch64" ]; then \
		echo "ARM64 architecture detected."; \
		git checkout $(1); \
		GOARCH=arm64 $(MAKE) dockers; \
		for svc in $(SERVICES); do \
				docker tag $(MG_DOCKER_IMAGE_NAME_PREFIX)/$$svc $(MG_DOCKER_IMAGE_NAME_PREFIX)/$$svc:latest; \
		done; \
		sed -i.bak 's/^MG_RELEASE_TAG=.*/MG_RELEASE_TAG=latest/' docker/.env && rm -f docker/.env.bak; \
		docker compose -f docker/docker-compose.yaml $(DOCKER_ENV_FILES) -p $(DOCKER_PROJECT) $(DOCKER_COMPOSE_COMMAND) $(args); \
	else \
		echo "x86_64 architecture detected."; \
		git checkout $(1); \
		sed -i.bak 's/^MG_RELEASE_TAG=.*/MG_RELEASE_TAG=$(2)/' docker/.env && rm -f docker/.env.bak; \
		docker compose -f docker/docker-compose.yaml $(DOCKER_ENV_FILES) -p $(DOCKER_PROJECT) $(DOCKER_COMPOSE_COMMAND) $(args); \
	fi
endef

ADDON_SERVICES = bootstrap provision postgres-writer postgres-reader

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
  ifneq ($(filter $(PROVISION_ATOM_TOKEN_GOALS),$(temp_args)),)
    override PROVISION_ATOM_TOKENS := true
  endif
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

all: $(SERVICES) $(CLI)

.PHONY: all help $(SERVICES) $(CLI) dockers dockers_dev latest release provision_atom_tokens provision-atom-tokens migrate_atom run_latest run_latest_ci run_tls run_stable run_addons grpc_mtls_certs check_mtls check_certs check_fluxmq_service_certs check_re_trace_key test_api mocks

help:
	@printf 'Usage:\n  make <target> [VARIABLE=value ...]\n\nAvailable targets:\n'
	@$(MAKE) -qpRr : 2>/dev/null | \
		awk -F: '/^[[:alnum:]_][^$$#\/\t=]*:([^=]|$$)/ { split($$1, targets, /[[:space:]]+/); for (i in targets) if (targets[i] != "") print targets[i] }' | \
		LC_ALL=C sort -u | \
		awk '$$0 != "Makefile" { printf "  make %s\n", $$0 }'

clean:
	rm -rf ${BUILD_DIR}

cleandocker:
	# Stops containers and removes containers, networks, volumes, and images created by up
	docker compose -f docker/docker-compose.yaml -p $(DOCKER_PROJECT) down --rmi all -v --remove-orphans

ifdef pv
	# Remove unused volumes
	docker volume ls -f name=$(MG_DOCKER_VOLUME_NAME_PREFIX) -f dangling=true -q | xargs -r docker volume rm
endif

install:
	for file in $(BUILD_DIR)/*; do \
		cp $$file $(GOBIN)/magistrala-`basename $$file`; \
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

	@uvx schemathesis run apidocs/openapi/$(svc).yaml \
	--checks all \
	--url $(2) \
	--header "Authorization: Bearer $(USER_TOKEN)" \
	--suppress-health-check=filter_too_much \
	--exclude-checks=positive_data_acceptance \
	--exclude-operation-id=requestPasswordReset \
	--phases=examples,stateful
endef

test_api_users: TEST_API_URL := http://localhost:9000
test_api_clients: TEST_API_URL := http://localhost:9000
test_api_workspaces: TEST_API_URL := http://localhost:9000
test_api_channels: TEST_API_URL := http://localhost:9000
test_api_groups: TEST_API_URL := http://localhost:9000
test_api_certs: TEST_API_URL := http://localhost:9019

$(TEST_API):
	$(call test_api_service,$(@),$(TEST_API_URL))

proto:
	protoc -I. --go_out=. --go_opt=paths=source_relative pkg/messaging/*.proto
	mkdir -p $(PKG_PROTO_GEN_OUT_DIR)
	protoc -I $(INTERNAL_PROTO_DIR) --go_out=$(PKG_PROTO_GEN_OUT_DIR) --go_opt=paths=source_relative --go-grpc_out=$(PKG_PROTO_GEN_OUT_DIR) --go-grpc_opt=paths=source_relative $(INTERNAL_PROTO_FILES)

$(FILTERED_SERVICES):
	$(call compile_service,$(@))

$(CLI):
	$(call compile_service,$(@))

$(DOCKERS):
	$(call make_docker,$(@),$(GOARCH))

$(DOCKERS_DEV):
	$(call make_docker_dev,$(@))

dockers: $(DOCKERS)
dockers_dev: $(DOCKERS_DEV)

define docker_push
	for svc in $(SERVICES); do \
		docker push $(MG_DOCKER_IMAGE_NAME_PREFIX)/$$svc:$(1); \
	done
endef

changelog:
	git log $(shell git describe --tags --abbrev=0)..HEAD --pretty=format:"- %s"

latest: dockers
	$(call docker_push,latest)

publish_arch:
	$(MAKE) dockers GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM)
	for svc in $(SERVICES); do \
		docker tag $(MG_DOCKER_IMAGE_NAME_PREFIX)/$$svc $(MG_DOCKER_IMAGE_NAME_PREFIX)/$$svc:$(VERSION)-$(GOARCH); \
		docker tag $(MG_DOCKER_IMAGE_NAME_PREFIX)/$$svc $(MG_DOCKER_IMAGE_NAME_PREFIX)/$$svc:latest-$(GOARCH); \
		docker push $(MG_DOCKER_IMAGE_NAME_PREFIX)/$$svc:$(VERSION)-$(GOARCH); \
		docker push $(MG_DOCKER_IMAGE_NAME_PREFIX)/$$svc:latest-$(GOARCH); \
	done

release:
	$(eval version = $(shell git describe --abbrev=0 --tags))
	git checkout $(version)
	$(MAKE) dockers
	for svc in $(SERVICES); do \
		docker tag $(MG_DOCKER_IMAGE_NAME_PREFIX)/$$svc $(MG_DOCKER_IMAGE_NAME_PREFIX)/$$svc:$(version); \
	done
	$(call docker_push,$(version))

rundev:
	cd scripts && ./run.sh

grpc_mtls_certs:
	$(MAKE) -C docker/ssl clients_grpc_certs

provision_atom_tokens:
	# This target brings up only atom-db and atom -- nginx and the FluxMQ
	# chain are out of scope here entirely, so atom must not be made to wait
	# on its AMQP proxy this early; see bootstrap_atom_events_amqp's comment
	# for why. Without this override, atom never becomes healthy within the
	# wait timeout and this step fails outright on every run, since
	# docker/.env always carries the real ATOM_EVENTS_AMQP_URL.
	ATOM_EVENTS_AMQP_URL= $(DOCKER_PLATFORM) docker compose -f docker/docker-compose.yaml $(DOCKER_PROVISION_ENV_FILES) -p $(DOCKER_PROJECT) up -d --wait --wait-timeout 120 atom
	$(MAKE) docker_atom-bootstrap
	$(DOCKER_PLATFORM) docker compose -f docker/docker-compose.yaml $(DOCKER_PROVISION_ENV_FILES) -p $(DOCKER_PROJECT) run --rm --no-deps --user "$(HOST_UID):$(HOST_GID)" -v "$(PWD)/docker:/host/docker" atom-bootstrap provision-tokens --output /host/docker/.env.tokens

provision-atom-tokens:
	@:

# Migrate an old Magistrala (v0.30.0 / pre-Atom) deployment into Atom. Runs an
# isolated, collision-free stack, seeds the Atom schema into the run_latest Atom
# volume and loads the data. Default is a dry-run; pass args="--apply" to load,
# args="--verify" to reconcile afterwards.
#   make migrate_atom                 # dry-run
#   make migrate_atom args="--apply"  # perform the migration
#   make migrate_atom args="--apply --fresh-atom"  # rebuild Atom schema first
migrate_atom:
	DOCKER_PROJECT="$(DOCKER_PROJECT)" tools/atom-migration/migrate.sh $(args)

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

# Internal services reach FluxMQ on the mTLS local listener, and Compose mounts
# both sides of those connections. Certificates and principal secrets are
# generated rather than committed, so make them before anything binds them.
check_fluxmq_service_certs:
ifeq ("$(wildcard docker/ssl/certs/re-fluxmq-client.crt)","")
	$(MAKE) -C docker/ssl fluxmq_service_certs
endif
ifeq ("$(wildcard docker/ssl/certs/timescale-writer-fluxmq-client.crt)","")
	$(MAKE) -C docker/ssl timescale_writer_fluxmq_client_cert
endif
ifeq ("$(wildcard docker/ssl/certs/postgres-writer-fluxmq-client.crt)","")
	$(MAKE) -C docker/ssl postgres_writer_fluxmq_client_cert
endif
ifeq ("$(wildcard docker/ssl/certs/fluxmq-auth-fluxmq-client.crt)","")
	$(MAKE) -C docker/ssl fluxmq_auth_fluxmq_client_cert
endif
ifeq ("$(wildcard docker/fluxmq/secrets/re-current)","")
	$(MAKE) -C docker/ssl fluxmq_service_secret
endif
ifeq ("$(wildcard docker/fluxmq/secrets/timescale-writer-current)","")
	$(MAKE) -C docker/ssl timescale_writer_fluxmq_service_secret
endif
ifeq ("$(wildcard docker/fluxmq/secrets/postgres-writer-current)","")
	$(MAKE) -C docker/ssl postgres_writer_fluxmq_service_secret
endif
ifeq ("$(wildcard docker/fluxmq/secrets/fluxmq-auth-current)","")
	$(MAKE) -C docker/ssl fluxmq_auth_fluxmq_service_secret
endif

check_re_trace_key:
ifeq ("$(wildcard docker/re/secrets/trace.key)","")
	$(MAKE) -C docker/ssl re_trace_key
endif

check_certs: check_mtls check_tls check_fluxmq_service_certs check_re_trace_key
ifeq ($(GRPC_MTLS_CERT_FILES_EXISTS),0)
ifeq ($(filter true,$(GRPC_MTLS) $(GRPC_TLS)),true)
ifeq ($(filter $(DEFAULT_DOCKER_COMPOSE_COMMAND),$(DOCKER_COMPOSE_COMMAND)),$(DEFAULT_DOCKER_COMPOSE_COMMAND))
	$(MAKE) -C docker/ssl clients_grpc_certs
endif
endif
endif

run_latest: check_certs
	$(SED_INPLACE) 's/^MG_RELEASE_TAG=.*/MG_RELEASE_TAG=latest/' docker/.env
	$(call ensure_atom_tokens_env)
	$(call bootstrap_atom_events_amqp)
	$(DOCKER_PLATFORM) docker compose -f docker/docker-compose.yaml $(DOCKER_ENV_FILES) -p $(DOCKER_PROJECT) $(DOCKER_COMPOSE_COMMAND) $(args)

run_latest_ci: check_certs
	$(call require_atom_tokens_env)
	$(SED_INPLACE) 's/^MG_RELEASE_TAG=.*/MG_RELEASE_TAG=latest/' docker/.env
	$(call bootstrap_atom_events_amqp)
	$(DOCKER_PLATFORM) docker compose -f docker/docker-compose.yaml -f docker/docker-compose-ci.yaml $(DOCKER_ENV_FILES) -p $(DOCKER_PROJECT) $(DOCKER_COMPOSE_COMMAND) $(args)

run_tls: check_certs
	@test -n "$(host)" || (echo "Usage: make run_tls host=example.com [email=admin@example.com] [letsencrypt=false] [staging=true] [force=true]" && exit 2)
	@if [ "$(or $(letsencrypt),true)" != "false" ] && [ -z "$(email)" ]; then echo "Usage: make run_tls host=example.com email=admin@example.com [letsencrypt=false] [staging=true] [force=true]"; exit 2; fi
	MG_PUBLIC_HOST="$(host)" \
	MG_LETSENCRYPT_ENABLED="$(or $(letsencrypt),true)" \
	MG_LETSENCRYPT_EMAIL="$(email)" \
	MG_LETSENCRYPT_STAGING="$(or $(staging),false)" \
	MG_LETSENCRYPT_FORCE_RENEWAL="$(or $(force),false)" \
	DOCKER_PROJECT="$(DOCKER_PROJECT)" \
	./docker/setup-tls.sh

run_stable: check_certs
	$(call require_atom_tokens_env)
	$(eval version = $(shell git describe --abbrev=0 --tags))
	git checkout $(version)
	$(SED_INPLACE) 's/^MG_RELEASE_TAG=.*/MG_RELEASE_TAG=$(version)/' docker/.env
	$(DOCKER_PLATFORM) docker compose -f docker/docker-compose.yaml $(DOCKER_ENV_FILES) -p $(DOCKER_PROJECT) $(DOCKER_COMPOSE_COMMAND) $(args)

run_addons: check_certs
	$(call require_atom_tokens_env)
	$(foreach SVC,$(RUN_ADDON_ARGS),$(if $(filter $(SVC),$(ADDON_SERVICES) $(EXTERNAL_SERVICES)),,$(error Invalid Service $(SVC))))
	@$(DOCKER_PLATFORM) docker compose -f docker/docker-compose.yaml $(DOCKER_ENV_FILES) -p $(DOCKER_PROJECT) up -d atom atom-bootstrap jaeger
	@for SVC in $(RUN_ADDON_ARGS); do \
		MG_ADDONS_CERTS_PATH_PREFIX="../" $(DOCKER_PLATFORM) docker compose -f docker/addons/$$SVC/docker-compose.yaml -p $(DOCKER_PROJECT) $(DOCKER_ENV_FILES) $(DOCKER_COMPOSE_COMMAND) $(args) & \
	done

run_live: check_certs
	$(call require_atom_tokens_env)
	GOPATH=$(go env GOPATH) $(DOCKER_PLATFORM) docker compose -f docker/docker-compose.yaml -f docker/docker-compose-live.yaml $(DOCKER_ENV_FILES) -p $(DOCKER_PROJECT) $(DOCKER_COMPOSE_COMMAND) $(args)
