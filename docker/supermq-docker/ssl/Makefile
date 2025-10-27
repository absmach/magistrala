# Copyright (c) Abstract Machines
# SPDX-License-Identifier: Apache-2.0

CRT_LOCATION = certs
O = SuperMQ
OU_CA = supermq_ca
OU_CRT = supermq_crt
EA = info@supermq.com
CN_CA = SuperMQ_Self_Signed_CA
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
CLIENTS_GRPC_SERVER_CONF_FILE_NAME=clients-grpc-server.conf
CLIENTS_GRPC_CLIENT_CONF_FILE_NAME=clients-grpc-client.conf
CLIENTS_GRPC_SERVER_CN=clients
CLIENTS_GRPC_CLIENT_CN=clients-client
CLIENTS_GRPC_SERVER_CRT_FILE_NAME=clients-grpc-server
CLIENTS_GRPC_CLIENT_CRT_FILE_NAME=clients-grpc-client
CHANNELS_GRPC_SERVER_CONF_FILE_NAME=channels-grpc-server.conf
CHANNELS_GRPC_CLIENT_CONF_FILE_NAME=channels-grpc-client.conf
CHANNELS_GRPC_SERVER_CN=channels
CHANNELS_GRPC_CLIENT_CN=channels-client
CHANNELS_GRPC_SERVER_CRT_FILE_NAME=channels-grpc-server
CHANNELS_GRPC_CLIENT_CRT_FILE_NAME=channels-grpc-client
COAP_DTLS_SERVER_CONF_FILE_NAME=coap-server.conf
COAP_DTLS_SERVER_CN=coap
COAP_DTLS_SERVER_CRT_FILE_NAME=coap-server

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

define ANNOUNCE_BODY
Version $(VERSION) of $(PACKAGE_NAME) has been released.

It can be downloaded from $(DOWNLOAD_URL).

etc, etc.
endef
all: clean_certs ca server_cert auth_grpc_certs domains_grpc_certs groups_grpc_certs clients_grpc_certs channels_grpc_certs coap_dtls_certs

# CA name and key is "ca".
ca:
	openssl req -newkey rsa:2048 -x509 -nodes -sha512 -days 1095 \
				-keyout $(CRT_LOCATION)/ca.key -out $(CRT_LOCATION)/ca.crt -subj "/CN=$(CN_CA)/O=$(O)/OU=$(OU_CA)/emailAddress=$(EA)"

# Server cert and key name is "supermq-server".
server_cert:
	# Create supermq server key and CSR.
	openssl req -new -sha256 -newkey rsa:4096 -nodes -keyout $(CRT_LOCATION)/supermq-server.key \
				-out $(CRT_LOCATION)/supermq-server.csr -subj "/CN=$(CN_SRV)/O=$(O)/OU=$(OU_CRT)/emailAddress=$(EA)"

	# Sign server CSR.
	openssl x509 -req -days 1000 -in $(CRT_LOCATION)/supermq-server.csr -CA $(CRT_LOCATION)/ca.crt -CAkey $(CRT_LOCATION)/ca.key -CAcreateserial -out $(CRT_LOCATION)/supermq-server.crt

	# Remove CSR.
	rm $(CRT_LOCATION)/supermq-server.csr

client_cert:
	# Create supermq server key and CSR.
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

auth_grpc_certs:
	$(call gen_grpc_cert_pair,$(AUTH_GRPC_SERVER_CRT_FILE_NAME),$(AUTH_GRPC_SERVER_CN),$(AUTH_GRPC_CLIENT_CRT_FILE_NAME),$(AUTH_GRPC_CLIENT_CN))

domains_grpc_certs:
	$(call gen_grpc_cert_pair,$(DOMAINS_GRPC_SERVER_CRT_FILE_NAME),$(DOMAINS_GRPC_SERVER_CN),$(DOMAINS_GRPC_CLIENT_CRT_FILE_NAME),$(DOMAINS_GRPC_CLIENT_CN))

groups_grpc_certs:
	$(call gen_grpc_cert_pair,$(GROUPS_GRPC_SERVER_CRT_FILE_NAME),$(GROUPS_GRPC_SERVER_CN),$(GROUPS_GRPC_CLIENT_CRT_FILE_NAME),$(GROUPS_GRPC_CLIENT_CN))

clients_grpc_certs:
	$(call gen_grpc_cert_pair,$(CLIENTS_GRPC_SERVER_CRT_FILE_NAME),$(CLIENTS_GRPC_SERVER_CN),$(CLIENTS_GRPC_CLIENT_CRT_FILE_NAME),$(CLIENTS_GRPC_CLIENT_CN))

channels_grpc_certs:
	$(call gen_grpc_cert_pair,$(CHANNELS_GRPC_SERVER_CRT_FILE_NAME),$(CHANNELS_GRPC_SERVER_CN),$(CHANNELS_GRPC_CLIENT_CRT_FILE_NAME),$(CHANNELS_GRPC_CLIENT_CN))

coap_dtls_certs:
	$(call gen_grpc_cert,$(COAP_DTLS_SERVER_CRT_FILE_NAME),$(COAP_DTLS_SERVER_CN))

clean_certs:
	rm -r $(CRT_LOCATION)/*.crt
	rm -r $(CRT_LOCATION)/*.key
