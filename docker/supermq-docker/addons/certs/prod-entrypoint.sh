#!/bin/sh
# Copyright (c) Abstract Machines
# SPDX-License-Identifier: Apache-2.0

set -e

apk add --no-cache jq

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

if [ -n "$SMQ_OPENBAO_UNSEAL_KEY_1" ] && [ -n "$SMQ_OPENBAO_UNSEAL_KEY_2" ] && [ -n "$SMQ_OPENBAO_UNSEAL_KEY_3" ] && [ -n "$SMQ_OPENBAO_ROOT_TOKEN" ]; then
  bao server -config=/opt/openbao/config/config.hcl > /opt/openbao/logs/server.log 2>&1 &
  BAO_PID=$!
  sleep 5
  
  bao operator unseal "$SMQ_OPENBAO_UNSEAL_KEY_1"
  bao operator unseal "$SMQ_OPENBAO_UNSEAL_KEY_2"
  bao operator unseal "$SMQ_OPENBAO_UNSEAL_KEY_3"
  
  export BAO_TOKEN=$SMQ_OPENBAO_ROOT_TOKEN
else
  if [ ! -f /opt/openbao/data/init.json ]; then
    bao server -config=/opt/openbao/config/config.hcl > /opt/openbao/logs/server.log 2>&1 &
    BAO_PID=$!
    sleep 5

    bao operator init -key-shares=5 -key-threshold=3 -format=json > /opt/openbao/data/init.json

    UNSEAL_KEY_1=$(cat /opt/openbao/data/init.json | jq -r '.unseal_keys_b64[0]')
    UNSEAL_KEY_2=$(cat /opt/openbao/data/init.json | jq -r '.unseal_keys_b64[1]')
    UNSEAL_KEY_3=$(cat /opt/openbao/data/init.json | jq -r '.unseal_keys_b64[2]')
    ROOT_TOKEN=$(cat /opt/openbao/data/init.json | jq -r '.root_token')

    bao operator unseal "$UNSEAL_KEY_1"
    bao operator unseal "$UNSEAL_KEY_2"
    bao operator unseal "$UNSEAL_KEY_3"

    export BAO_TOKEN=$ROOT_TOKEN
  else
    bao server -config=/opt/openbao/config/config.hcl > /opt/openbao/logs/server.log 2>&1 &
    BAO_PID=$!
    sleep 5

    if bao status | grep -q "Sealed.*true"; then
      UNSEAL_KEY_1=$(cat /opt/openbao/data/init.json | jq -r '.unseal_keys_b64[0]')
      UNSEAL_KEY_2=$(cat /opt/openbao/data/init.json | jq -r '.unseal_keys_b64[1]')
      UNSEAL_KEY_3=$(cat /opt/openbao/data/init.json | jq -r '.unseal_keys_b64[2]')

      bao operator unseal "$UNSEAL_KEY_1"
      bao operator unseal "$UNSEAL_KEY_2"
      bao operator unseal "$UNSEAL_KEY_3"
    fi

    ROOT_TOKEN=$(cat /opt/openbao/data/init.json | jq -r '.root_token')
    export BAO_TOKEN=$ROOT_TOKEN
  fi
fi

if [ ! -f /opt/openbao/data/configured ]; then
  if bao namespace create "$SMQ_OPENBAO_NAMESPACE" 2>/dev/null; then
    export BAO_NAMESPACE="$SMQ_OPENBAO_NAMESPACE"
    echo "$SMQ_OPENBAO_NAMESPACE" > /opt/openbao/data/namespace
  fi

  bao auth enable approle || echo "AppRole already enabled"
  bao secrets enable -path=pki pki || echo "PKI already enabled"
  bao secrets tune -max-lease-ttl=87600h pki

  bao write -field=certificate pki/root/generate/internal \
    common_name="${SMQ_OPENBAO_PKI_CA_CN}" \
    organization="${SMQ_OPENBAO_PKI_CA_O}" \
    ou="${SMQ_OPENBAO_PKI_CA_OU}" \
    country="${SMQ_OPENBAO_PKI_CA_C}" \
    locality="${SMQ_OPENBAO_PKI_CA_L}" \
    province="${SMQ_OPENBAO_PKI_CA_ST}" \
    street_address="${SMQ_OPENBAO_PKI_CA_ADDR}" \
    postal_code="${SMQ_OPENBAO_PKI_CA_PO}" \
    ttl=87600h \
    key_bits=2048 \
    exclude_cn_from_sans=true

  bao write pki/config/urls \
    issuing_certificates='http://127.0.0.1:8200/v1/pki/ca' \
    crl_distribution_points='http://127.0.0.1:8200/v1/pki/crl'

  bao write pki/roles/"${SMQ_OPENBAO_PKI_ROLE_NAME:-supermq}" \
    allow_any_name=true \
    enforce_hostnames=false \
    allow_ip_sans=true \
    allow_localhost=true \
    max_ttl=720h \
    ttl=720h \
    key_bits=2048

  cat > /opt/openbao/config/pki-policy.hcl << EOF
path "pki/issue/${SMQ_OPENBAO_PKI_ROLE_NAME:-supermq}" {
  capabilities = ["create", "update"]
}
path "pki/certs" {
  capabilities = ["list"]
}
path "pki/cert/*" {
  capabilities = ["read"]
}
path "pki/revoke" {
  capabilities = ["create", "update"]
}
path "auth/token/renew-self" {
  capabilities = ["update"]
}
path "auth/token/lookup-self" {
  capabilities = ["read"]
}
EOF

  bao policy write pki-policy /opt/openbao/config/pki-policy.hcl

  bao write auth/approle/role/"${SMQ_OPENBAO_PKI_ROLE_NAME:-supermq}" \
    token_policies=pki-policy \
    token_ttl=1h \
    token_max_ttl=4h \
    bind_secret_id=true \
    secret_id_ttl=24h

  if [ -n "$SMQ_OPENBAO_APP_ROLE" ]; then
    bao write auth/approle/role/"${SMQ_OPENBAO_PKI_ROLE_NAME:-supermq}"/role-id role_id="$SMQ_OPENBAO_APP_ROLE"
  fi

  if [ -n "$SMQ_OPENBAO_APP_SECRET" ]; then
    bao write auth/approle/role/"${SMQ_OPENBAO_PKI_ROLE_NAME:-supermq}"/custom-secret-id secret_id="$SMQ_OPENBAO_APP_SECRET"
  fi

  SERVICE_TOKEN=$(bao write -field=token auth/token/create \
    policies=pki-policy \
    ttl=24h \
    renewable=true \
    display_name="supermq-service")

  echo "SERVICE_TOKEN=$SERVICE_TOKEN" > /opt/openbao/data/service_token
  touch /opt/openbao/data/configured
  echo "OpenBao configuration completed successfully!"
else
  echo "OpenBao already configured, skipping setup..."
  if [ -f /opt/openbao/data/namespace ] && [ -n "$SMQ_OPENBAO_NAMESPACE" ]; then
    SAVED_NAMESPACE=$(cat /opt/openbao/data/namespace)
    if [ "$SAVED_NAMESPACE" = "$SMQ_OPENBAO_NAMESPACE" ]; then
      export BAO_NAMESPACE="$SMQ_OPENBAO_NAMESPACE"
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

echo "OpenBao is ready for SuperMQ on port 8200"
wait $BAO_PID
