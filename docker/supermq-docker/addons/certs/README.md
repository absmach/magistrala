# OpenBao Configuration for SuperMQ

This directory contains both development and production OpenBao configurations for SuperMQ certificate management.

## Overview

Two entrypoint scripts are provided:

- **`dev-entrypoint.sh`**: Development mode with in-memory storage and simple setup
- **`prod-entrypoint.sh`**: Production mode with persistent file storage and proper initialization

Both scripts use environment variables for flexible configuration, allowing you to customize OpenBao behavior without modifying the scripts directly. All configuration is centralized in the `.env` file using the `SMQ_OPENBAO_*` naming convention.

## Configuration Management

### Environment-Based Configuration
All OpenBao configuration is managed through environment variables defined in `/docker/.env`. This approach provides:

- **Consistency**: All OpenBao variables use the `SMQ_OPENBAO_*` naming pattern
- **Flexibility**: Easy customization without script modifications
- **Security**: Sensitive values (tokens, keys) can be externally managed
- **Development/Production Parity**: Same configuration approach for both environments

### Variable Organization
Variables are logically grouped by function:
- **Core**: Basic OpenBao server configuration
- **Authentication**: AppRole and token configuration  
- **PKI Engine**: Certificate authority and PKI role settings
- **PKI CA**: Certificate authority details (CN, organization, etc.)
- **Unsealing**: Production unsealing keys and tokens

## Quick Start

### Development Mode (Default)
```bash
docker compose -f docker/docker-compose.yaml -f docker/addons/certs/docker-compose.yaml up -d openbao
```

### Production Mode
To switch to production mode, edit `docker-compose.yaml` and change:
```yaml
- ./dev-entrypoint.sh:/entrypoint.sh
```
to:
```yaml
- ./prod-entrypoint.sh:/entrypoint.sh
```

Then start the service:
```bash
docker compose -f docker/docker-compose.yaml -f docker/addons/certs/docker-compose.yaml up -d openbao
```

## Development Mode Features

- **In-memory storage**: No data persistence (resets on restart)
- **Development server**: Uses `-dev` flag for simple setup
- **Hardcoded tokens**: Uses predictable root token for easy access
- **Quick setup**: Minimal configuration for development
- **No unseal process**: Automatically unsealed

### Development Access
- **Root Token**: `openbao-root-token` (or `SMQ_OPENBAO_ROOT_TOKEN` env var)
- **Web UI**: http://localhost:8200/ui
- **API**: http://localhost:8200

## Production Mode Features

- **File-based storage**: Persistent storage using file backend
- **Proper initialization**: Uses unseal keys and root token
- **Security policies**: Restricted access policies for PKI operations
- **AppRole authentication**: Service-to-service authentication
- **PKI engine**: Certificate authority for SuperMQ services
- **Automatic unsealing**: Handles unsealing on container restart

### Production Security

#### Initial Setup
- On first startup, OpenBao will be automatically initialized with 5 unseal keys and 1 root token
- The initialization data is stored in `/opt/openbao/data/init.json`
- **You must backup this file securely** - it contains the unseal keys and root token

#### Access Production Instance
To get the root token and unseal keys:
```bash
docker exec supermq-openbao cat /opt/openbao/data/init.json
```

Or to get just the root token:
```bash
docker exec supermq-openbao jq -r '.root_token' /opt/openbao/data/init.json
```

#### Manual Operations
```bash
docker exec supermq-openbao bao status

docker exec supermq-openbao bao operator unseal <unseal-key>

docker exec supermq-openbao bao operator seal
```

## Configuration Details

### Development Mode Configuration
- **Storage**: In-memory (no persistence)
- **Listener**: TCP on `0.0.0.0:8200` (TLS disabled)
- **Authentication**: Simple root token
- **PKI**: Basic setup for testing

### Production Mode Configuration
- **Storage**: File backend at `/opt/openbao/data`
- **Listener**: TCP on `0.0.0.0:8200` (TLS disabled for internal use)
- **UI**: Enabled for administration
- **Logging**: Info level
- **Initialization**: 5 unseal keys, 3 required
- **Authentication**: AppRole for services

### PKI Engine (Both Modes)
- **Path**: `/pki`
- **Root CA**: SuperMQ Root CA
- **Certificate Role**: `supermq` role for service certificates
- **Max TTL**: 720 hours (30 days) for dev, 87600 hours (10 years) for root CA in prod

### AppRole Authentication
- **Role**: `supermq`
- **Policies**: `pki-policy` (restricted PKI access)
- **Token TTL**: 1 hour (renewable up to 4 hours)

## Environment Variables

### OpenBao Core Configuration
- `SMQ_OPENBAO_HOST`: OpenBao server hostname (default: `supermq-openbao`)
- `SMQ_OPENBAO_PORT`: OpenBao server port (default: `8200`)
- `SMQ_OPENBAO_ADDR`: Full OpenBao server URL (default: `http://supermq-openbao:8200`)
- `SMQ_OPENBAO_NAMESPACE`: OpenBao namespace
- `SMQ_OPENBAO_ROOT_TOKEN`: Custom root token for development mode
- `SMQ_OPENBAO_TOKEN`: Custom token for production mode
- `SMQ_OPENBAO_UNSEAL_KEY_1`: First unseal key for production mode
- `SMQ_OPENBAO_UNSEAL_KEY_2`: Second unseal key for production mode  
- `SMQ_OPENBAO_UNSEAL_KEY_3`: Third unseal key for production mode

### OpenBao Authentication Configuration
- `SMQ_OPENBAO_APP_ROLE`: AppRole role ID for service authentication
- `SMQ_OPENBAO_APP_SECRET`: AppRole secret ID for service authentication

### OpenBao PKI Configuration
- `SMQ_OPENBAO_PKI_PATH`: PKI secrets engine path (default: `pki`)
- `SMQ_OPENBAO_PKI_ROLE`: PKI role name for certificate issuance (default: `supermq`)
- `SMQ_OPENBAO_PKI_ROLE_NAME`: PKI role name for certificate generation (default: `supermq`)

### OpenBao PKI Certificate Authority Configuration
- `SMQ_OPENBAO_PKI_CA_CN`: Certificate Authority Common Name
- `SMQ_OPENBAO_PKI_CA_OU`: Certificate Authority Organizational Unit
- `SMQ_OPENBAO_PKI_CA_O`: Certificate Authority Organization
- `SMQ_OPENBAO_PKI_CA_C`: Certificate Authority Country
- `SMQ_OPENBAO_PKI_CA_L`: Certificate Authority Locality
- `SMQ_OPENBAO_PKI_CA_ST`: Certificate Authority State/Province
- `SMQ_OPENBAO_PKI_CA_ADDR`: Certificate Authority Street Address
- `SMQ_OPENBAO_PKI_CA_PO`: Certificate Authority Postal Code

### Certs Service OpenBao Integration
For the SuperMQ certs service, the following variables are used internally:
- `SMQ_CERTS_OPENBAO_HOST`: Maps to `SMQ_OPENBAO_HOST` and `SMQ_OPENBAO_PORT`
- `SMQ_CERTS_OPENBAO_APP_ROLE`: Maps to `SMQ_OPENBAO_APP_ROLE`
- `SMQ_CERTS_OPENBAO_APP_SECRET`: Maps to `SMQ_OPENBAO_APP_SECRET`
- `SMQ_CERTS_OPENBAO_NAMESPACE`: Maps to `SMQ_OPENBAO_NAMESPACE`
- `SMQ_CERTS_OPENBAO_PKI_PATH`: Maps to `SMQ_OPENBAO_PKI_PATH`
- `SMQ_CERTS_OPENBAO_ROLE`: Maps to `SMQ_OPENBAO_PKI_ROLE`

## Switching Between Modes

### To Switch to Production Mode:
1. Edit `docker-compose.yaml`
2. Change `./dev-entrypoint.sh:/entrypoint.sh` to `./prod-entrypoint.sh:/entrypoint.sh`
3. Restart the container

### To Switch to Development Mode:
1. Edit `docker-compose.yaml`
2. Change `./prod-entrypoint.sh:/entrypoint.sh` to `./dev-entrypoint.sh:/entrypoint.sh`
3. Restart the container
