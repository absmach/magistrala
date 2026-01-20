# Asymmetric Tokenizer

EdDSA (Ed25519) tokenizer with support for zero-downtime key rotation.

## Features

- **Single-key mode** - Simple setup with one active key
- **Two-key mode** - Active + retiring keys for _zero-downtime rotation_
- **JWKS endpoint** - Publishes all valid public keys for token verification

## Configuration

The tokenizer uses environment variables to specify key file paths:

| Environment Variable              | Required | Description                                      |
| --------------------------------- | -------- | ------------------------------------------------ |
| `SMQ_AUTH_KEYS_ACTIVE_KEY_PATH`   | Yes      | Path to active private key file                  |
| `SMQ_AUTH_KEYS_RETIRING_KEY_PATH` | No       | Path to retiring private key file (for rotation) |

Please note that key names are used as **key IDs (kid)**.

### Single-Key Mode

Set only the active key path:

```bash
export SMQ_AUTH_KEYS_ACTIVE_KEY_PATH="./keys/private.key"
```

The tokenizer will:

- Issue new tokens signed with the active key
- Verify tokens using the active key
- Return one public key in JWKS endpoint

### Two-Key Mode (Key Rotation)

Set both active and retiring key paths:

```bash
export SMQ_AUTH_KEYS_ACTIVE_KEY_PATH="./keys/active.key"
export SMQ_AUTH_KEYS_RETIRING_KEY_PATH="./keys/retiring.key"
```

The tokenizer will:

- Issue new tokens signed with the active key
- Verify tokens using both active and retiring keys
- Return both public keys in JWKS endpoint

## Key Rotation Process

Zero-downtime key rotation in 3 simple steps:

### 1. Generate New Key

```bash
openssl genpkey -algorithm Ed25519 -out keys/new.key
```

### 2. Update Environment & Restart

Move the current active key to retiring position and set the new key as active:

```bash
# Before rotation
SMQ_AUTH_KEYS_ACTIVE_KEY_PATH="./keys/current.key"
SMQ_AUTH_KEYS_RETIRING_KEY_PATH=""  # No retiring key

# During rotation (both keys active for grace period)
SMQ_AUTH_KEYS_ACTIVE_KEY_PATH="./keys/new.key"
SMQ_AUTH_KEYS_RETIRING_KEY_PATH="./keys/current.key"

# After rotation (restart service with new config)
docker-compose restart auth
```

During the grace period, tokens signed with either key remain valid.

### 3. Clean Up After Grace Period

After the grace period expires (typically 7-30 days), remove the retiring key:

```bash
# Remove retiring key configuration
SMQ_AUTH_KEYS_ACTIVE_KEY_PATH="./keys/new.key"
SMQ_AUTH_KEYS_RETIRING_KEY_PATH=""  # Remove retiring key

# Restart service
docker-compose restart auth

# Delete old key file
rm keys/current.key
```

## Grace Period Recommendations

**Recommended:** 168 hours (7 days)
**Minimum:** 24 hours
**Maximum:** 720 hours (30 days)

The grace period should be longer than your longest-lived access token duration.

## Security Best Practices

- Store private keys with `0600` permissions
- Use cryptographically secure key generation:

  ```bash
  openssl genpkey -algorithm Ed25519 -out private.key
  chmod 600 private.key
  ```

- Rotate keys regularly:
  - Standard environments: every 90 days
  - High-security environments: every 30 days
- Never commit keys to version control
- Use secrets management in production (HashiCorp Vault, AWS Secrets Manager, etc.)

## Example: Complete Rotation

```bash
# Day 0: Normal operation
export SMQ_AUTH_KEYS_ACTIVE_KEY_PATH="./keys/key-2024.pem"
export SMQ_AUTH_KEYS_RETIRING_KEY_PATH=""

# Day 1: Start rotation - generate new key
openssl genpkey -algorithm Ed25519 -out ./keys/key-2025.pem
chmod 600 ./keys/key-2025.pem

# Day 1: Update config and restart
export SMQ_AUTH_KEYS_ACTIVE_KEY_PATH="./keys/key-2025.pem"
export SMQ_AUTH_KEYS_RETIRING_KEY_PATH="./keys/key-2024.pem"
docker-compose restart auth

# Day 8: Grace period expired - remove old key
export SMQ_AUTH_KEYS_RETIRING_KEY_PATH=""
docker-compose restart auth
rm ./keys/key-2024.pem
```

## Troubleshooting

### Active key not found

```bash
Error: active key file not found: ./keys/active.key
```

**Solution:** Ensure the file exists and path is correct. Verify `SMQ_AUTH_KEYS_ACTIVE_KEY_PATH` environment variable.

### Retiring key warning

If the retiring key path is set but the file is missing or invalid, the tokenizer logs a warning but continues with only the active key:

```bash
WARN: failed to load retiring key, continuing without it
```

This is by design - a missing retiring key won't prevent startup.

### Invalid key format

```bash
Error: failed to parse private key
```

**Solution:** Ensure you're using Ed25519 keys in PEM format (PKCS8).
