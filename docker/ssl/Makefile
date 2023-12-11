# Copyright (c) Abstract Machines
# SPDX-License-Identifier: Apache-2.0

CRT_LOCATION = certs
O = Magistrala
OU_CA = magistrala_ca
OU_CRT = magistrala_crt
EA = info@magistrala.com
CN_CA = Magistrala_Self_Signed_CA
CN_SRV = localhost
THING_SECRET = <THING_SECRET> # e.g. 8f65ed04-0770-4ce4-a291-6d1bf2000f4d
CRT_FILE_NAME = thing
THINGS_GRPC_SERVER_CONF_FILE_NAME=thing-grpc-server.conf
THINGS_GRPC_CLIENT_CONF_FILE_NAME=thing-grpc-client.conf
THINGS_GRPC_SERVER_CN=things
THINGS_GRPC_CLIENT_CN=things-client
THINGS_GRPC_SERVER_CRT_FILE_NAME=things-grpc-server
THINGS_GRPC_CLIENT_CRT_FILE_NAME=things-grpc-client
AUTH_GRPC_SERVER_CONF_FILE_NAME=auth-grpc-server.conf
AUTH_GRPC_CLIENT_CONF_FILE_NAME=auth-grpc-client.conf
AUTH_GRPC_SERVER_CN=auth
AUTH_GRPC_CLIENT_CN=auth-client
AUTH_GRPC_SERVER_CRT_FILE_NAME=auth-grpc-server
AUTH_GRPC_CLIENT_CRT_FILE_NAME=auth-grpc-client

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
O  = MAGISTRALA
OU = MAGISTRALA

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
all: clean_certs ca server_cert things_grpc_certs auth_grpc_certs

# CA name and key is "ca".
ca:
	openssl req -newkey rsa:2048 -x509 -nodes -sha512 -days 1095 \
				-keyout $(CRT_LOCATION)/ca.key -out $(CRT_LOCATION)/ca.crt -subj "/CN=$(CN_CA)/O=$(O)/OU=$(OU_CA)/emailAddress=$(EA)"

# Server cert and key name is "magistrala-server".
server_cert:
	# Create magistrala server key and CSR.
	openssl req -new -sha256 -newkey rsa:4096 -nodes -keyout $(CRT_LOCATION)/magistrala-server.key \
				-out $(CRT_LOCATION)/magistrala-server.csr -subj "/CN=$(CN_SRV)/O=$(O)/OU=$(OU_CRT)/emailAddress=$(EA)"

	# Sign server CSR.
	openssl x509 -req -days 1000 -in $(CRT_LOCATION)/magistrala-server.csr -CA $(CRT_LOCATION)/ca.crt -CAkey $(CRT_LOCATION)/ca.key -CAcreateserial -out $(CRT_LOCATION)/magistrala-server.crt

	# Remove CSR.
	rm $(CRT_LOCATION)/magistrala-server.csr

thing_cert:
	# Create magistrala server key and CSR.
	openssl req -new -sha256 -newkey rsa:4096 -nodes -keyout $(CRT_LOCATION)/$(CRT_FILE_NAME).key \
				-out $(CRT_LOCATION)/$(CRT_FILE_NAME).csr -subj "/CN=$(THING_SECRET)/O=$(O)/OU=$(OU_CRT)/emailAddress=$(EA)"

	# Sign client CSR.
	openssl x509 -req -days 730 -in $(CRT_LOCATION)/$(CRT_FILE_NAME).csr -CA $(CRT_LOCATION)/ca.crt -CAkey $(CRT_LOCATION)/ca.key -CAcreateserial -out $(CRT_LOCATION)/$(CRT_FILE_NAME).crt

	# Remove CSR.
	rm $(CRT_LOCATION)/$(CRT_FILE_NAME).csr

things_grpc_certs:
	# Things server grpc certificates
	$(file > $(CRT_LOCATION)/$(THINGS_GRPC_SERVER_CRT_FILE_NAME).conf,$(subst <<SERVICE_NAME>>,$(THINGS_GRPC_SERVER_CN),$(GRPC_CERT_CONFIG)) )

	openssl req -new -sha256  -newkey rsa:4096 -nodes \
				-keyout $(CRT_LOCATION)/$(THINGS_GRPC_SERVER_CRT_FILE_NAME).key  \
				-out $(CRT_LOCATION)/$(THINGS_GRPC_SERVER_CRT_FILE_NAME).csr \
				-config $(CRT_LOCATION)/$(THINGS_GRPC_SERVER_CRT_FILE_NAME).conf \
				-extensions v3_req

	openssl x509 -req -sha256 \
			-in $(CRT_LOCATION)/$(THINGS_GRPC_SERVER_CRT_FILE_NAME).csr \
			-CA $(CRT_LOCATION)/ca.crt \
			-CAkey $(CRT_LOCATION)/ca.key \
			-CAcreateserial \
			-out $(CRT_LOCATION)/$(THINGS_GRPC_SERVER_CRT_FILE_NAME).crt \
			-days 365 \
			-extfile $(CRT_LOCATION)/$(THINGS_GRPC_SERVER_CRT_FILE_NAME).conf \
			-extensions v3_req

	rm -rf  $(CRT_LOCATION)/$(THINGS_GRPC_SERVER_CRT_FILE_NAME).csr $(CRT_LOCATION)/$(THINGS_GRPC_SERVER_CRT_FILE_NAME).conf
	# Things client grpc certificates
	$(file > $(CRT_LOCATION)/$(THINGS_GRPC_CLIENT_CRT_FILE_NAME).conf,$(subst <<SERVICE_NAME>>,$(THINGS_GRPC_CLIENT_CN),$(GRPC_CERT_CONFIG)) )

	openssl req -new -sha256 -newkey rsa:4096 -nodes \
				-keyout $(CRT_LOCATION)/$(THINGS_GRPC_CLIENT_CRT_FILE_NAME).key  \
				-out $(CRT_LOCATION)/$(THINGS_GRPC_CLIENT_CRT_FILE_NAME).csr \
				-config $(CRT_LOCATION)/$(THINGS_GRPC_CLIENT_CRT_FILE_NAME).conf \
				-extensions v3_req

	openssl x509 -req -sha256 \
			-in $(CRT_LOCATION)/$(THINGS_GRPC_CLIENT_CRT_FILE_NAME).csr \
			-CA $(CRT_LOCATION)/ca.crt \
			-CAkey $(CRT_LOCATION)/ca.key \
			-CAcreateserial \
			-out $(CRT_LOCATION)/$(THINGS_GRPC_CLIENT_CRT_FILE_NAME).crt \
			-days 365 \
			-extfile $(CRT_LOCATION)/$(THINGS_GRPC_CLIENT_CRT_FILE_NAME).conf \
			-extensions v3_req

	rm -rf  $(CRT_LOCATION)/$(THINGS_GRPC_CLIENT_CRT_FILE_NAME).csr $(CRT_LOCATION)/$(THINGS_GRPC_CLIENT_CRT_FILE_NAME).conf

auth_grpc_certs:
	# Auth gRPC server certificate
	$(file > $(CRT_LOCATION)/$(AUTH_GRPC_SERVER_CRT_FILE_NAME).conf,$(subst <<SERVICE_NAME>>,$(AUTH_GRPC_SERVER_CN),$(GRPC_CERT_CONFIG)) )

	openssl req -new -sha256  -newkey rsa:4096 -nodes \
				-keyout $(CRT_LOCATION)/$(AUTH_GRPC_SERVER_CRT_FILE_NAME).key  \
				-out $(CRT_LOCATION)/$(AUTH_GRPC_SERVER_CRT_FILE_NAME).csr \
				-config $(CRT_LOCATION)/$(AUTH_GRPC_SERVER_CRT_FILE_NAME).conf \
				-extensions v3_req

	openssl x509 -req -sha256 \
			-in $(CRT_LOCATION)/$(AUTH_GRPC_SERVER_CRT_FILE_NAME).csr \
			-CA $(CRT_LOCATION)/ca.crt \
			-CAkey $(CRT_LOCATION)/ca.key \
			-CAcreateserial \
			-out $(CRT_LOCATION)/$(AUTH_GRPC_SERVER_CRT_FILE_NAME).crt \
			-days 365 \
			-extfile $(CRT_LOCATION)/$(AUTH_GRPC_SERVER_CRT_FILE_NAME).conf \
			-extensions v3_req

	rm -rf  $(CRT_LOCATION)/$(AUTH_GRPC_SERVER_CRT_FILE_NAME).csr $(CRT_LOCATION)/$(AUTH_GRPC_SERVER_CRT_FILE_NAME).conf
	# Auth gRPC client certificate
	$(file > $(CRT_LOCATION)/$(AUTH_GRPC_CLIENT_CRT_FILE_NAME).conf,$(subst <<SERVICE_NAME>>,$(AUTH_GRPC_CLIENT_CN),$(GRPC_CERT_CONFIG)) )

	openssl req -new -sha256 -newkey rsa:4096 -nodes \
				-keyout $(CRT_LOCATION)/$(AUTH_GRPC_CLIENT_CRT_FILE_NAME).key  \
				-out $(CRT_LOCATION)/$(AUTH_GRPC_CLIENT_CRT_FILE_NAME).csr \
				-config $(CRT_LOCATION)/$(AUTH_GRPC_CLIENT_CRT_FILE_NAME).conf \
				-extensions v3_req

	openssl x509 -req -sha256 \
			-in $(CRT_LOCATION)/$(AUTH_GRPC_CLIENT_CRT_FILE_NAME).csr \
			-CA $(CRT_LOCATION)/ca.crt \
			-CAkey $(CRT_LOCATION)/ca.key \
			-CAcreateserial \
			-out $(CRT_LOCATION)/$(AUTH_GRPC_CLIENT_CRT_FILE_NAME).crt \
			-days 365 \
			-extfile $(CRT_LOCATION)/$(AUTH_GRPC_CLIENT_CRT_FILE_NAME).conf \
			-extensions v3_req

	rm -rf  $(CRT_LOCATION)/$(AUTH_GRPC_CLIENT_CRT_FILE_NAME).csr $(CRT_LOCATION)/$(AUTH_GRPC_CLIENT_CRT_FILE_NAME).conf
	
clean_certs:
	rm -r $(CRT_LOCATION)/*.crt
	rm -r $(CRT_LOCATION)/*.key
