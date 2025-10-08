#!/bin/sh
# Copyright (c) Abstract Machines
# SPDX-License-Identifier: Apache-2.0

set -e

apk add --no-cache jq

# Create required directories
mkdir -p /opt/openbao/config /opt/openbao/data /opt/openbao/logs

cat > /opt/openbao/config/config.hcl << 'EOF'
storage "file" {
  path = "/opt/openbao/data"
}
listener "tcp" {
  address     = "0.0.0.0:8200"
  tls_disable = true
}
ui = true
log_level = "Info"
disable_mlock = true
# API timeout settings
default_lease_ttl = "168h"
max_lease_ttl = "720h"
EOF

export BAO_ADDR=http://127.0.0.1:8200

# Check if we have pre-configured unseal keys and root token
if [ -n "$AM_CERTS_OPENBAO_UNSEAL_KEY_1" ] && [ -n "$AM_CERTS_OPENBAO_UNSEAL_KEY_2" ] && [ -n "$AM_CERTS_OPENBAO_UNSEAL_KEY_3" ] && [ -n "$AM_CERTS_OPENBAO_ROOT_TOKEN" ]; then
  echo "Using pre-configured unseal keys and root token..."
  bao server -config=/opt/openbao/config/config.hcl > /opt/openbao/logs/server.log 2>&1 &
  BAO_PID=$!
  sleep 5
  
  bao operator unseal "$AM_CERTS_OPENBAO_UNSEAL_KEY_1"
  bao operator unseal "$AM_CERTS_OPENBAO_UNSEAL_KEY_2"
  bao operator unseal "$AM_CERTS_OPENBAO_UNSEAL_KEY_3"
  
  export BAO_TOKEN=$AM_CERTS_OPENBAO_ROOT_TOKEN
else
  # Initialize OpenBao if not already done
  if [ ! -f /opt/openbao/data/init.json ]; then
    echo "Initializing OpenBao for the first time..."
    bao server -config=/opt/openbao/config/config.hcl > /opt/openbao/logs/server.log 2>&1 &
    BAO_PID=$!
    sleep 5

    # Initialize with 5 key shares and threshold of 3
    bao operator init -key-shares=5 -key-threshold=3 -format=json > /opt/openbao/data/init.json

    # Extract unseal keys and root token
    UNSEAL_KEY_1=$(cat /opt/openbao/data/init.json | jq -r '.unseal_keys_b64[0]')
    UNSEAL_KEY_2=$(cat /opt/openbao/data/init.json | jq -r '.unseal_keys_b64[1]')
    UNSEAL_KEY_3=$(cat /opt/openbao/data/init.json | jq -r '.unseal_keys_b64[2]')
    ROOT_TOKEN=$(cat /opt/openbao/data/init.json | jq -r '.root_token')

    # Unseal OpenBao
    bao operator unseal "$UNSEAL_KEY_1"
    bao operator unseal "$UNSEAL_KEY_2"
    bao operator unseal "$UNSEAL_KEY_3"

    export BAO_TOKEN=$ROOT_TOKEN
    echo "OpenBao initialized successfully!"
  else
    echo "OpenBao already initialized, starting server..."
    bao server -config=/opt/openbao/config/config.hcl > /opt/openbao/logs/server.log 2>&1 &
    BAO_PID=$!
    sleep 5

    # Check if OpenBao is sealed and unseal if necessary
    if bao status -format=json | jq -e '.sealed == true' >/dev/null; then
      echo "OpenBao is sealed, unsealing..."
      UNSEAL_KEY_1=$(cat /opt/openbao/data/init.json | jq -r '.unseal_keys_b64[0]')
      UNSEAL_KEY_2=$(cat /opt/openbao/data/init.json | jq -r '.unseal_keys_b64[1]')
      UNSEAL_KEY_3=$(cat /opt/openbao/data/init.json | jq -r '.unseal_keys_b64[2]')

      bao operator unseal "$UNSEAL_KEY_1"
      bao operator unseal "$UNSEAL_KEY_2"
      bao operator unseal "$UNSEAL_KEY_3"
      echo "OpenBao unsealed successfully!"
    else
      echo "OpenBao is already unsealed!"
    fi

    ROOT_TOKEN=$(cat /opt/openbao/data/init.json | jq -r '.root_token')
    export BAO_TOKEN=$ROOT_TOKEN
  fi
fi

# Configure OpenBao PKI and AppRole if not already configured
if [ ! -f /opt/openbao/data/configured ]; then
  echo "Configuring OpenBao PKI and AppRole..."
  
  # Create namespace if specified
  if [ -n "$AM_CERTS_OPENBAO_NAMESPACE" ]; then
    if bao namespace create "$AM_CERTS_OPENBAO_NAMESPACE" 2>/tmp/ns_error; then
      export BAO_NAMESPACE="$AM_CERTS_OPENBAO_NAMESPACE"
      echo "$AM_CERTS_OPENBAO_NAMESPACE" > /opt/openbao/data/namespace
      echo "Created namespace: $AM_CERTS_OPENBAO_NAMESPACE"
    else
      if grep -q "namespace already exists" /tmp/ns_error; then
        export BAO_NAMESPACE="$AM_CERTS_OPENBAO_NAMESPACE"
        echo "$AM_CERTS_OPENBAO_NAMESPACE" > /opt/openbao/data/namespace
        echo "Using existing namespace: $AM_CERTS_OPENBAO_NAMESPACE"
      else
        echo "ERROR: Failed to create namespace $AM_CERTS_OPENBAO_NAMESPACE:" >&2
        cat /tmp/ns_error >&2
        exit 1
      fi
    fi
    rm -f /tmp/ns_error
  fi

  # Enable authentication methods and secrets engines
  if ! bao auth enable approle > /tmp/auth_success 2>/tmp/auth_error; then
    if ! grep -q "already in use" /tmp/auth_error; then
      echo "ERROR: Failed to enable AppRole auth method:" >&2
      cat /tmp/auth_error >&2
      exit 1
    fi
    echo "AppRole already enabled"
  fi
  rm -f /tmp/auth_error /tmp/auth_success

  # Enable PKI secrets engine
  if ! bao secrets enable -path=pki pki > /tmp/pki_success 2>/tmp/pki_error; then
    # If the failure wasn’t because the mount already exists, abort
    if ! grep -q "already in use" /tmp/pki_error; then
      echo "ERROR: Failed to enable PKI secrets engine:" >&2
      cat /tmp/pki_error >&2
      exit 1
    fi
    echo "PKI already enabled"
  fi
  rm -f /tmp/pki_error /tmp/pki_success

  # Configure PKI engine
  bao secrets tune -max-lease-ttl=87600h pki > /dev/null

  # Validate required CA environment variables
  for var in AM_CERTS_OPENBAO_PKI_CA_CN AM_CERTS_OPENBAO_PKI_CA_O AM_CERTS_OPENBAO_PKI_CA_C; do
    eval "value=\$var"
    if [ -z "$value" ]; then
      echo "ERROR: Required environment variable $var is not set" >&2
      exit 1
    fi
  done

  PKI_CMD="bao write -field=certificate pki/root/generate/internal \
    common_name=\"$AM_CERTS_OPENBAO_PKI_CA_CN\" \
    organization=\"$AM_CERTS_OPENBAO_PKI_CA_O\" \
    country=\"$AM_CERTS_OPENBAO_PKI_CA_C\" \
    ttl=87600h \
    key_bits=2048 \
    exclude_cn_from_sans=false"

  [ -n "$AM_CERTS_OPENBAO_PKI_CA_OU" ] && PKI_CMD="$PKI_CMD ou=\"$AM_CERTS_OPENBAO_PKI_CA_OU\""
  [ -n "$AM_CERTS_OPENBAO_PKI_CA_L" ] && PKI_CMD="$PKI_CMD locality=\"$AM_CERTS_OPENBAO_PKI_CA_L\""
  [ -n "$AM_CERTS_OPENBAO_PKI_CA_ST" ] && PKI_CMD="$PKI_CMD province=\"$AM_CERTS_OPENBAO_PKI_CA_ST\""
  [ -n "$AM_CERTS_OPENBAO_PKI_CA_ADDR" ] && PKI_CMD="$PKI_CMD street_address=\"$AM_CERTS_OPENBAO_PKI_CA_ADDR\""
  [ -n "$AM_CERTS_OPENBAO_PKI_CA_PO" ] && PKI_CMD="$PKI_CMD postal_code=\"$AM_CERTS_OPENBAO_PKI_CA_PO\""
  
  [ -n "$AM_CERTS_OPENBAO_PKI_CA_DNS_NAMES" ] && PKI_CMD="$PKI_CMD alt_names=\"$AM_CERTS_OPENBAO_PKI_CA_DNS_NAMES\""
  [ -n "$AM_CERTS_OPENBAO_PKI_CA_IP_ADDRESSES" ] && PKI_CMD="$PKI_CMD ip_sans=\"$AM_CERTS_OPENBAO_PKI_CA_IP_ADDRESSES\""
  [ -n "$AM_CERTS_OPENBAO_PKI_CA_URI_SANS" ] && PKI_CMD="$PKI_CMD uri_sans=\"$AM_CERTS_OPENBAO_PKI_CA_URI_SANS\""
  [ -n "$AM_CERTS_OPENBAO_PKI_CA_EMAIL_ADDRESSES" ] && PKI_CMD="$PKI_CMD email_sans=\"$AM_CERTS_OPENBAO_PKI_CA_EMAIL_ADDRESSES\""

  eval $PKI_CMD > /dev/null

  if [ $? -eq 0 ]; then
    echo "OpenBao root CA certificate generated successfully!"
  else
    echo "ERROR: Failed to generate OpenBao root CA certificate" >&2
    exit 1
  fi

  if ! bao secrets enable -path=pki_int pki > /tmp/pki_int_success 2>/tmp/pki_int_error; then
    if ! grep -q "already in use" /tmp/pki_int_error; then
      echo "ERROR: Failed to enable intermediate PKI secrets engine:" >&2
      cat /tmp/pki_int_error >&2
      exit 1
    fi
    echo "Intermediate PKI already enabled"
  fi
  rm -f /tmp/pki_int_error /tmp/pki_int_success

  bao secrets tune -max-lease-ttl=8760h pki_int > /dev/null

  INTERMEDIATE_CN="${AM_CERTS_OPENBAO_PKI_CA_CN} Intermediate"
  INTERMEDIATE_CSR_CMD="bao write -field=csr pki_int/intermediate/generate/internal \
    common_name=\"$INTERMEDIATE_CN\" \
    organization=\"$AM_CERTS_OPENBAO_PKI_CA_O\" \
    country=\"$AM_CERTS_OPENBAO_PKI_CA_C\" \
    ttl=8760h \
    key_bits=2048"

  [ -n "$AM_CERTS_OPENBAO_PKI_CA_OU" ] && INTERMEDIATE_CSR_CMD="$INTERMEDIATE_CSR_CMD ou=\"$AM_CERTS_OPENBAO_PKI_CA_OU\""
  [ -n "$AM_CERTS_OPENBAO_PKI_CA_L" ] && INTERMEDIATE_CSR_CMD="$INTERMEDIATE_CSR_CMD locality=\"$AM_CERTS_OPENBAO_PKI_CA_L\""
  [ -n "$AM_CERTS_OPENBAO_PKI_CA_ST" ] && INTERMEDIATE_CSR_CMD="$INTERMEDIATE_CSR_CMD province=\"$AM_CERTS_OPENBAO_PKI_CA_ST\""
  [ -n "$AM_CERTS_OPENBAO_PKI_CA_ADDR" ] && INTERMEDIATE_CSR_CMD="$INTERMEDIATE_CSR_CMD street_address=\"$AM_CERTS_OPENBAO_PKI_CA_ADDR\""
  [ -n "$AM_CERTS_OPENBAO_PKI_CA_PO" ] && INTERMEDIATE_CSR_CMD="$INTERMEDIATE_CSR_CMD postal_code=\"$AM_CERTS_OPENBAO_PKI_CA_PO\""
  
  [ -n "$AM_CERTS_OPENBAO_PKI_CA_DNS_NAMES" ] && INTERMEDIATE_CSR_CMD="$INTERMEDIATE_CSR_CMD alt_names=\"$AM_CERTS_OPENBAO_PKI_CA_DNS_NAMES\""
  [ -n "$AM_CERTS_OPENBAO_PKI_CA_IP_ADDRESSES" ] && INTERMEDIATE_CSR_CMD="$INTERMEDIATE_CSR_CMD ip_sans=\"$AM_CERTS_OPENBAO_PKI_CA_IP_ADDRESSES\""
  [ -n "$AM_CERTS_OPENBAO_PKI_CA_URI_SANS" ] && INTERMEDIATE_CSR_CMD="$INTERMEDIATE_CSR_CMD uri_sans=\"$AM_CERTS_OPENBAO_PKI_CA_URI_SANS\""
  [ -n "$AM_CERTS_OPENBAO_PKI_CA_EMAIL_ADDRESSES" ] && INTERMEDIATE_CSR_CMD="$INTERMEDIATE_CSR_CMD email_sans=\"$AM_CERTS_OPENBAO_PKI_CA_EMAIL_ADDRESSES\""

  INTERMEDIATE_CSR=$(eval $INTERMEDIATE_CSR_CMD)

  if [ $? -ne 0 ] || [ -z "$INTERMEDIATE_CSR" ]; then
    echo "ERROR: Failed to generate intermediate CA CSR" >&2
    exit 1
  fi

  echo "Intermediate CA CSR generated successfully!"

  INTERMEDIATE_CERT=$(bao write -field=certificate pki/root/sign-intermediate \
    csr="$INTERMEDIATE_CSR" \
    format=pem_bundle \
    ttl=8760h \
    use_csr_values=true)

  if [ $? -ne 0 ] || [ -z "$INTERMEDIATE_CERT" ]; then
    echo "ERROR: Failed to sign intermediate CA certificate" >&2
    exit 1
  fi

  echo "Intermediate CA certificate signed successfully!"

  bao write pki_int/intermediate/set-signed certificate="$INTERMEDIATE_CERT" > /dev/null

  if [ $? -eq 0 ]; then
    echo "Intermediate CA setup completed successfully!"
  else
    echo "ERROR: Failed to set signed intermediate certificate" >&2
    exit 1
  fi

  echo "$INTERMEDIATE_CERT" > /opt/openbao/data/intermediate_ca.pem

  bao write pki/config/urls \
    issuing_certificates='http://127.0.0.1:8200/v1/pki/ca' \
    crl_distribution_points='http://127.0.0.1:8200/v1/pki/crl' \
    ocsp_servers='http://127.0.0.1:8200/v1/pki/ocsp' > /dev/null

  bao write pki_int/config/urls \
    issuing_certificates='http://127.0.0.1:8200/v1/pki_int/ca' \
    crl_distribution_points='http://127.0.0.1:8200/v1/pki_int/crl' \
    ocsp_servers='http://127.0.0.1:8200/v1/pki_int/ocsp' > /dev/null

  ROLE_CMD="bao write pki_int/roles/${AM_CERTS_OPENBAO_PKI_ROLE} \
    allow_any_name=true \
    enforce_hostnames=false \
    allow_ip_sans=true \
    allow_localhost=true \
    allow_bare_domains=true \
    allow_subdomains=true \
    allow_glob_domains=true \
    allowed_domains=\"*\" \
    allowed_uri_sans=\"*\" \
    allowed_other_sans=\"*\" \
    server_flag=true \
    client_flag=true \
    code_signing_flag=false \
    email_protection_flag=false \
    key_type=rsa \
    key_bits=2048 \
    key_usage=\"DigitalSignature,KeyEncipherment,KeyAgreement\" \
    ext_key_usage=\"ServerAuth,ClientAuth,OCSPSigning\" \
    use_csr_common_name=true \
    use_csr_sans=true \
    copy_extensions=true \
    allowed_extensions=\"*\" \
    basic_constraints_valid_for_non_ca=true \
    max_ttl=720h \
    ttl=720h"

  eval "$ROLE_CMD" > /dev/null

  # Create PKI policy
  cat > /opt/openbao/config/pki-policy.hcl << EOF
path "pki_int/issue/${AM_CERTS_OPENBAO_PKI_ROLE}" {
  capabilities = ["create", "update"]
}
path "pki_int/sign/${AM_CERTS_OPENBAO_PKI_ROLE}" {
  capabilities = ["create", "update"]
}
path "pki_int/sign-verbatim/${AM_CERTS_OPENBAO_PKI_ROLE}" {
  capabilities = ["create", "update"]
}
path "pki_int/certs" {
  capabilities = ["list"]
}
path "pki_int/cert/*" {
  capabilities = ["read"]
}
path "pki_int/revoke" {
  capabilities = ["create", "update"]
}
path "pki_int/ca" {
  capabilities = ["read"]
}
path "pki_int/ca_chain" {
  capabilities = ["read"]
}
path "pki_int/crl" {
  capabilities = ["read"]
}
path "pki/ca" {
  capabilities = ["read"]
}
path "pki/ca_chain" {
  capabilities = ["read"]
}
path "pki/crl" {
  capabilities = ["read"]
}
# Token management
path "auth/token/renew-self" {
  capabilities = ["update"]
}
path "auth/token/lookup-self" {
  capabilities = ["read"]
}
# System lease renewal
path "sys/renew/*" {
  capabilities = ["update"]
}
EOF

  bao policy write pki-policy /opt/openbao/config/pki-policy.hcl > /dev/null

  # Create AppRole
  SECRET_ID_TTL="${AM_CERTS_OPENBAO_SECRET_ID_TTL}"
  bao write auth/approle/role/"${AM_CERTS_OPENBAO_PKI_ROLE}" \
    token_policies=pki-policy \
    token_ttl=1h \
    token_max_ttl=4h \
    bind_secret_id=true \
    secret_id_ttl="$SECRET_ID_TTL" > /dev/null

  # Set custom role ID if provided
  if [ -n "$AM_CERTS_OPENBAO_APP_ROLE" ]; then
    bao write auth/approle/role/"${AM_CERTS_OPENBAO_PKI_ROLE}"/role-id role_id="$AM_CERTS_OPENBAO_APP_ROLE" > /dev/null
  fi

  # Set custom secret ID if provided
  if [ -n "$AM_CERTS_OPENBAO_APP_SECRET" ]; then
    bao write auth/approle/role/"${AM_CERTS_OPENBAO_PKI_ROLE}"/custom-secret-id secret_id="$AM_CERTS_OPENBAO_APP_SECRET" > /dev/null
  fi

  # Generate service token for additional access
  SERVICE_TOKEN=$(bao write -field=token auth/token/create \
    policies=pki-policy \
    ttl=24h \
    renewable=true \
    display_name="certs-service" 2>/dev/null)

  echo "SERVICE_TOKEN=$SERVICE_TOKEN" > /opt/openbao/data/service_token
  
  # Mark configuration as complete
  touch /opt/openbao/data/configured
  echo "OpenBao configuration completed successfully!"
else
  echo "OpenBao already configured, skipping setup..."
  
  # Restore namespace if it exists
  if [ -f /opt/openbao/data/namespace ] && [ -n "$AM_CERTS_OPENBAO_NAMESPACE" ]; then
    SAVED_NAMESPACE=$(cat /opt/openbao/data/namespace)
    if [ "$SAVED_NAMESPACE" = "$AM_CERTS_OPENBAO_NAMESPACE" ]; then
      export BAO_NAMESPACE="$AM_CERTS_OPENBAO_NAMESPACE"
    fi
  fi
  
  if [ -n "$AM_CERTS_OPENBAO_APP_SECRET" ]; then
    echo "Verifying existing secret ID validity..."
    if ! bao write -field=client_token auth/approle/login role_id="$AM_CERTS_OPENBAO_APP_ROLE" secret_id="$AM_CERTS_OPENBAO_APP_SECRET" > /dev/null 2>&1; then
      echo "================================"
      echo "ERROR: Secret ID has expired!"
      echo "Please regenerate AM_CERTS_OPENBAO_APP_SECRET"
      echo "and update your environment configuration"
      echo "================================"
    else
      echo "Existing secret ID is valid"
    fi
  fi
fi

echo "================================"
echo "OpenBao Production Setup Complete"
echo "================================"
echo "OpenBao Address: http://localhost:8200"
echo "UI Available at: http://localhost:8200/ui"
echo "================================"
echo "IMPORTANT: Store the init.json file securely!"
echo "It contains unseal keys and root token!"
echo "================================"

echo "OpenBao is ready for certs service on port 8200"

if [ -n "$BAO_PID" ]; then
  wait $BAO_PID
else
  echo "ERROR: OpenBao server process ID not available" >&2
  exit 1
fi
