# Copyright (c) Abstract Machines
# SPDX-License-Identifier: Apache-2.0

CRT_LOCATION = certs
O = Magistrala
OU_CA = magistrala_ca
OU_CRT = magistrala_crt
EA = info@magistrala.com
CN_CA = Magistrala_Self_Signed_CA
CN_SRV ?= localhost
CLIENT_SECRET ?= "" # e.g. 8f65ed04-0770-4ce4-a291-6d1bf2000f4d
CRT_FILE_NAME = client
AUTH_GRPC_SERVER_CONF_FILE_NAME=auth-grpc-server.conf
AUTH_GRPC_CLIENT_CONF_FILE_NAME=auth-grpc-client.conf
AUTH_GRPC_SERVER_CN=auth
AUTH_GRPC_CLIENT_CN=auth-client
AUTH_GRPC_SERVER_CRT_FILE_NAME=auth-grpc-server
AUTH_GRPC_CLIENT_CRT_FILE_NAME=auth-grpc-client
DOMAINS_GRPC_SERVER_CONF_FILE_NAME=domains-grpc-server.conf
DOMAINS_GRPC_CLIENT_CONF_FILE_NAME=domains-grpc-client.conf
DOMAINS_GRPC_SERVER_CN=domains
DOMAINS_GRPC_CLIENT_CN=domains-client
DOMAINS_GRPC_SERVER_CRT_FILE_NAME=domains-grpc-server
DOMAINS_GRPC_CLIENT_CRT_FILE_NAME=domains-grpc-client
GROUPS_GRPC_SERVER_CONF_FILE_NAME=groups-grpc-server.conf
GROUPS_GRPC_CLIENT_CONF_FILE_NAME=groups-grpc-client.conf
GROUPS_GRPC_SERVER_CN=groups
GROUPS_GRPC_CLIENT_CN=groups-client
GROUPS_GRPC_SERVER_CRT_FILE_NAME=groups-grpc-server
GROUPS_GRPC_CLIENT_CRT_FILE_NAME=groups-grpc-client
DEVICES_GRPC_SERVER_CONF_FILE_NAME=devices-grpc-server.conf
DEVICES_GRPC_CLIENT_CONF_FILE_NAME=devices-grpc-client.conf
DEVICES_GRPC_SERVER_CN=devices
DEVICES_GRPC_CLIENT_CN=devices-client
DEVICES_GRPC_SERVER_CRT_FILE_NAME=devices-grpc-server
DEVICES_GRPC_CLIENT_CRT_FILE_NAME=devices-grpc-client
CHANNELS_GRPC_SERVER_CONF_FILE_NAME=channels-grpc-server.conf
CHANNELS_GRPC_CLIENT_CONF_FILE_NAME=channels-grpc-client.conf
CHANNELS_GRPC_SERVER_CN=channels
CHANNELS_GRPC_CLIENT_CN=channels-client
CHANNELS_GRPC_SERVER_CRT_FILE_NAME=channels-grpc-server
CHANNELS_GRPC_CLIENT_CRT_FILE_NAME=channels-grpc-client
COAP_DTLS_SERVER_CONF_FILE_NAME=coap-server.conf
COAP_DTLS_SERVER_CN=coap
COAP_DTLS_SERVER_CRT_FILE_NAME=coap-server
# FluxMQ service listener. The broker identifies a first-party service by the
# URI SAN of the certificate it presents, matching it against a principal
# declared in the broker's own configuration, so the SAN below must stay in
# step with docker/fluxmq/node*.yaml.
FLUXMQ_SERVICE_SERVER_CRT_FILE_NAME=fluxmq-service-server
RE_FLUXMQ_CLIENT_CN=rules-engine
RE_FLUXMQ_CLIENT_CRT_FILE_NAME=re-fluxmq-client
RE_FLUXMQ_CLIENT_URI_SAN=spiffe://absmach/magistrala/rules-engine
TIMESCALE_WRITER_FLUXMQ_CLIENT_CN=timescale-writer
TIMESCALE_WRITER_FLUXMQ_CLIENT_CRT_FILE_NAME=timescale-writer-fluxmq-client
TIMESCALE_WRITER_FLUXMQ_CLIENT_URI_SAN=spiffe://absmach/magistrala/timescale-writer
POSTGRES_WRITER_FLUXMQ_CLIENT_CN=postgres-writer
POSTGRES_WRITER_FLUXMQ_CLIENT_CRT_FILE_NAME=postgres-writer-fluxmq-client
POSTGRES_WRITER_FLUXMQ_CLIENT_URI_SAN=spiffe://absmach/magistrala/postgres-writer
FLUXMQ_AUTH_FLUXMQ_CLIENT_CN=fluxmq-auth
FLUXMQ_AUTH_FLUXMQ_CLIENT_CRT_FILE_NAME=fluxmq-auth-fluxmq-client
FLUXMQ_AUTH_FLUXMQ_CLIENT_URI_SAN=spiffe://absmach/magistrala/fluxmq-auth

define GRPC_CERT_CONFIG
[req]
req_extensions = v3_req
distinguished_name = dn
prompt = no

[dn]
CN = mg.svc
C  = RS
ST = RS
L  = BELGRADE
O  = SUPERMQ
OU = SUPERMQ

[v3_req]
subjectAltName = @alt_names

[alt_names]
DNS.1 = <<SERVICE_NAME>>
endef

define FLUXMQ_SERVICE_SERVER_CERT_CONFIG
[req]
req_extensions = v3_req
distinguished_name = dn
prompt = no

[dn]
CN = mg.svc
C  = RS
ST = RS
L  = BELGRADE
O  = SUPERMQ
OU = SUPERMQ

[v3_req]
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

# The Rules Engine verifies this certificate against the host in
# MG_RE_BROKER_URL, so every name a deployment may dial the service listener by
# has to appear here: the per-node names this Compose stack uses, and the plain
# "fluxmq" of a single-node deployment.
[alt_names]
DNS.1 = fluxmq
DNS.2 = fluxmq-node1
DNS.3 = fluxmq-node2
DNS.4 = fluxmq-node3
endef

define URI_SAN_CERT_CONFIG
[req]
req_extensions = v3_req
distinguished_name = dn
prompt = no

[dn]
CN = <<SERVICE_NAME>>
C  = RS
ST = RS
L  = BELGRADE
O  = SUPERMQ
OU = SUPERMQ

[v3_req]
extendedKeyUsage = clientAuth
subjectAltName = @alt_names

[alt_names]
URI.1 = <<URI_SAN>>
endef

define ANNOUNCE_BODY
Version $(VERSION) of $(PACKAGE_NAME) has been released.

It can be downloaded from $(DOWNLOAD_URL).

etc, etc.
endef
all: clean_certs ca server_cert auth_grpc_certs domains_grpc_certs groups_grpc_certs devices_grpc_certs channels_grpc_certs coap_dtls_certs fluxmq_service_certs fluxmq_service_secret timescale_writer_fluxmq_service_secret postgres_writer_fluxmq_service_secret fluxmq_auth_fluxmq_service_secret re_trace_key

# CA name and key is "ca".
ca:
	openssl req -newkey rsa:2048 -x509 -nodes -sha512 -days 1095 \
				-keyout $(CRT_LOCATION)/ca.key -out $(CRT_LOCATION)/ca.crt -subj "/CN=$(CN_CA)/O=$(O)/OU=$(OU_CA)/emailAddress=$(EA)"

# Server cert and key name is "magistrala-server".
server_cert:
	# Create magistrala server key and CSR.
	openssl req -new -sha256 -newkey rsa:4096 -nodes -keyout $(CRT_LOCATION)/magistrala-server.key \
				-out $(CRT_LOCATION)/magistrala-server.csr -subj "/CN=$(CN_SRV)/O=$(O)/OU=$(OU_CRT)/emailAddress=$(EA)"

	# Sign server CSR with SANs for container hostnames.
	printf '[v3_req]\nsubjectAltName=DNS:localhost,DNS:nginx,DNS:%s' "$(CN_SRV)" > $(CRT_LOCATION)/magistrala-server.san
	openssl x509 -req -days 1000 -in $(CRT_LOCATION)/magistrala-server.csr \
		-CA $(CRT_LOCATION)/ca.crt -CAkey $(CRT_LOCATION)/ca.key -CAcreateserial \
		-out $(CRT_LOCATION)/magistrala-server.crt \
		-extfile $(CRT_LOCATION)/magistrala-server.san -extensions v3_req

	# Remove CSR and SAN config.
	rm $(CRT_LOCATION)/magistrala-server.csr $(CRT_LOCATION)/magistrala-server.san

client_cert:
	# Create magistrala server key and CSR.
	openssl req -new -sha256 -newkey rsa:4096 -nodes -keyout $(CRT_LOCATION)/$(CRT_FILE_NAME).key \
				-out $(CRT_LOCATION)/$(CRT_FILE_NAME).csr -subj "/CN=$(CLIENT_SECRET)/O=$(O)/OU=$(OU_CRT)/emailAddress=$(EA)"

	# Sign client CSR.
	openssl x509 -req -days 730 -in $(CRT_LOCATION)/$(CRT_FILE_NAME).csr -CA $(CRT_LOCATION)/ca.crt -CAkey $(CRT_LOCATION)/ca.key -CAcreateserial -out $(CRT_LOCATION)/$(CRT_FILE_NAME).crt

	# Remove CSR.
	rm $(CRT_LOCATION)/$(CRT_FILE_NAME).csr

# Function to generate gRPC certificates (server or client)
# Usage: $(call gen_grpc_cert,cert_file_name,common_name)
define gen_grpc_cert
	$(file > $(CRT_LOCATION)/$(1).conf,$(subst <<SERVICE_NAME>>,$(2),$(GRPC_CERT_CONFIG)))

	openssl req -new -sha256 -newkey rsa:4096 -nodes \
		-keyout $(CRT_LOCATION)/$(1).key \
		-out $(CRT_LOCATION)/$(1).csr \
		-config $(CRT_LOCATION)/$(1).conf \
		-extensions v3_req

	openssl x509 -req -sha256 \
		-in $(CRT_LOCATION)/$(1).csr \
		-CA $(CRT_LOCATION)/ca.crt \
		-CAkey $(CRT_LOCATION)/ca.key \
		-CAcreateserial \
		-out $(CRT_LOCATION)/$(1).crt \
		-days 365 \
		-extfile $(CRT_LOCATION)/$(1).conf \
		-extensions v3_req

	rm -rf $(CRT_LOCATION)/$(1).csr $(CRT_LOCATION)/$(1).conf
endef

# Alternative: Single function that generates both server and client certs
# Usage: $(call gen_grpc_cert_pair,server_cert_name,server_cn,client_cert_name,client_cn)
define gen_grpc_cert_pair
	# Server certificate
	$(call gen_grpc_cert,$(1),$(2))
	# Client certificate
	$(call gen_grpc_cert,$(3),$(4))
endef

# Usage: $(call gen_fluxmq_server_cert,cert_file_name)
define gen_fluxmq_server_cert
	$(file > $(CRT_LOCATION)/$(1).conf,$(FLUXMQ_SERVICE_SERVER_CERT_CONFIG))

	openssl req -new -sha256 -newkey rsa:4096 -nodes \
		-keyout $(CRT_LOCATION)/$(1).key \
		-out $(CRT_LOCATION)/$(1).csr \
		-config $(CRT_LOCATION)/$(1).conf \
		-extensions v3_req

	openssl x509 -req -sha256 \
		-in $(CRT_LOCATION)/$(1).csr \
		-CA $(CRT_LOCATION)/ca.crt \
		-CAkey $(CRT_LOCATION)/ca.key \
		-CAcreateserial \
		-out $(CRT_LOCATION)/$(1).crt \
		-days 365 \
		-extfile $(CRT_LOCATION)/$(1).conf \
		-extensions v3_req

	rm -rf $(CRT_LOCATION)/$(1).csr $(CRT_LOCATION)/$(1).conf
endef

# Usage: $(call gen_uri_san_cert,cert_file_name,common_name,uri_san)
define gen_uri_san_cert
	$(file > $(CRT_LOCATION)/$(1).conf,$(subst <<URI_SAN>>,$(3),$(subst <<SERVICE_NAME>>,$(2),$(URI_SAN_CERT_CONFIG))))

	openssl req -new -sha256 -newkey rsa:4096 -nodes \
		-keyout $(CRT_LOCATION)/$(1).key \
		-out $(CRT_LOCATION)/$(1).csr \
		-config $(CRT_LOCATION)/$(1).conf \
		-extensions v3_req

	openssl x509 -req -sha256 \
		-in $(CRT_LOCATION)/$(1).csr \
		-CA $(CRT_LOCATION)/ca.crt \
		-CAkey $(CRT_LOCATION)/ca.key \
		-CAcreateserial \
		-out $(CRT_LOCATION)/$(1).crt \
		-days 365 \
		-extfile $(CRT_LOCATION)/$(1).conf \
		-extensions v3_req

	rm -rf $(CRT_LOCATION)/$(1).csr $(CRT_LOCATION)/$(1).conf
endef

# Certificates for the FluxMQ service listener: a server certificate for the
# broker and a client certificate carrying each local principal's URI SAN. The
# listener requires mTLS, so both sides are issued by the same CA.
fluxmq_service_certs:
	$(call gen_fluxmq_server_cert,$(FLUXMQ_SERVICE_SERVER_CRT_FILE_NAME))
	$(call gen_uri_san_cert,$(RE_FLUXMQ_CLIENT_CRT_FILE_NAME),$(RE_FLUXMQ_CLIENT_CN),$(RE_FLUXMQ_CLIENT_URI_SAN))
	$(call gen_uri_san_cert,$(TIMESCALE_WRITER_FLUXMQ_CLIENT_CRT_FILE_NAME),$(TIMESCALE_WRITER_FLUXMQ_CLIENT_CN),$(TIMESCALE_WRITER_FLUXMQ_CLIENT_URI_SAN))
	$(call gen_uri_san_cert,$(POSTGRES_WRITER_FLUXMQ_CLIENT_CRT_FILE_NAME),$(POSTGRES_WRITER_FLUXMQ_CLIENT_CN),$(POSTGRES_WRITER_FLUXMQ_CLIENT_URI_SAN))
	$(call gen_uri_san_cert,$(FLUXMQ_AUTH_FLUXMQ_CLIENT_CRT_FILE_NAME),$(FLUXMQ_AUTH_FLUXMQ_CLIENT_CN),$(FLUXMQ_AUTH_FLUXMQ_CLIENT_URI_SAN))

# Generate a single principal's certificate when adding it to an existing
# deployment, without rotating the broker certificate or the other principals.
timescale_writer_fluxmq_client_cert:
	$(call gen_uri_san_cert,$(TIMESCALE_WRITER_FLUXMQ_CLIENT_CRT_FILE_NAME),$(TIMESCALE_WRITER_FLUXMQ_CLIENT_CN),$(TIMESCALE_WRITER_FLUXMQ_CLIENT_URI_SAN))

postgres_writer_fluxmq_client_cert:
	$(call gen_uri_san_cert,$(POSTGRES_WRITER_FLUXMQ_CLIENT_CRT_FILE_NAME),$(POSTGRES_WRITER_FLUXMQ_CLIENT_CN),$(POSTGRES_WRITER_FLUXMQ_CLIENT_URI_SAN))

fluxmq_auth_fluxmq_client_cert:
	$(call gen_uri_san_cert,$(FLUXMQ_AUTH_FLUXMQ_CLIENT_CRT_FILE_NAME),$(FLUXMQ_AUTH_FLUXMQ_CLIENT_CN),$(FLUXMQ_AUTH_FLUXMQ_CLIENT_URI_SAN))

# The SASL secret a local principal presents alongside its certificate. FluxMQ
# reads it from a file and the service sends it in its broker URL, so both sides
# must see the same value: .env is the single source and these targets derive
# the file from it. FluxMQ requires at least 32 bytes.
#
# Usage: $(call gen_fluxmq_service_secret,env_variable_name,secret_file_name)
define gen_fluxmq_service_secret
	@mkdir -p ../fluxmq/secrets
	@secret="$${$(1)}"; \
	if [ -z "$$secret" ]; then \
		secret=$$(grep -E "^$(1)=" ../.env | cut -d= -f2-); \
	fi; \
	if [ -z "$$secret" ]; then \
		echo "$(1) is not set in docker/.env"; exit 1; \
	fi; \
	if [ $${#secret} -lt 32 ]; then \
		echo "$(1) must be at least 32 characters"; exit 1; \
	fi; \
	printf '%s' "$$secret" > ../fluxmq/secrets/$(2); \
	chmod 600 ../fluxmq/secrets/$(2); \
	echo "wrote docker/fluxmq/secrets/$(2)"
endef

fluxmq_service_secret:
	$(call gen_fluxmq_service_secret,MG_RE_BROKER_SECRET,re-current)

timescale_writer_fluxmq_service_secret:
	$(call gen_fluxmq_service_secret,MG_TIMESCALE_WRITER_BROKER_SECRET,timescale-writer-current)

postgres_writer_fluxmq_service_secret:
	$(call gen_fluxmq_service_secret,MG_POSTGRES_WRITER_BROKER_SECRET,postgres-writer-current)

fluxmq_auth_fluxmq_service_secret:
	$(call gen_fluxmq_service_secret,MG_FLUXMQ_BROKER_SECRET,fluxmq-auth-current)

# Persistent HMAC key shared by Rules Engine replicas. Do not replace an
# existing key: doing so while old messages are in flight would invalidate
# their loop traces. The containing directory is excluded from git.
re_trace_key:
	@mkdir -p ../re/secrets
	@if [ ! -s ../re/secrets/trace.key ]; then \
		umask 077; openssl rand -hex 32 > ../re/secrets/trace.key; \
		echo "wrote docker/re/secrets/trace.key"; \
	else \
		echo "preserving docker/re/secrets/trace.key"; \
	fi

auth_grpc_certs:
	$(call gen_grpc_cert_pair,$(AUTH_GRPC_SERVER_CRT_FILE_NAME),$(AUTH_GRPC_SERVER_CN),$(AUTH_GRPC_CLIENT_CRT_FILE_NAME),$(AUTH_GRPC_CLIENT_CN))

domains_grpc_certs:
	$(call gen_grpc_cert_pair,$(DOMAINS_GRPC_SERVER_CRT_FILE_NAME),$(DOMAINS_GRPC_SERVER_CN),$(DOMAINS_GRPC_CLIENT_CRT_FILE_NAME),$(DOMAINS_GRPC_CLIENT_CN))

groups_grpc_certs:
	$(call gen_grpc_cert_pair,$(GROUPS_GRPC_SERVER_CRT_FILE_NAME),$(GROUPS_GRPC_SERVER_CN),$(GROUPS_GRPC_CLIENT_CRT_FILE_NAME),$(GROUPS_GRPC_CLIENT_CN))

devices_grpc_certs:
	$(call gen_grpc_cert_pair,$(DEVICES_GRPC_SERVER_CRT_FILE_NAME),$(DEVICES_GRPC_SERVER_CN),$(DEVICES_GRPC_CLIENT_CRT_FILE_NAME),$(DEVICES_GRPC_CLIENT_CN))

channels_grpc_certs:
	$(call gen_grpc_cert_pair,$(CHANNELS_GRPC_SERVER_CRT_FILE_NAME),$(CHANNELS_GRPC_SERVER_CN),$(CHANNELS_GRPC_CLIENT_CRT_FILE_NAME),$(CHANNELS_GRPC_CLIENT_CN))

coap_dtls_certs:
	$(call gen_grpc_cert,$(COAP_DTLS_SERVER_CRT_FILE_NAME),$(COAP_DTLS_SERVER_CN))

clean_certs:
	rm -r $(CRT_LOCATION)/*.crt
	rm -r $(CRT_LOCATION)/*.key
