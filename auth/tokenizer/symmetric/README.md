# Symmetric Tokenizer

HMAC-based tokenizer with support for zero-downtime secret rotation.

## Features

- **Single-secret mode** - Simple setup with one active secret
- **Two-secret mode** - Active + retiring secrets for _zero-downtime rotation_
- **Multiple algorithms** - Supports HS256, HS384, and HS512

## Configuration

The tokenizer uses environment variables to specify secrets:

| Environment Variable           | Required | Description                                     |
| ------------------------------ | -------- | ----------------------------------------------- |
| `SMQ_AUTH_KEYS_ALGORITHM`      | Yes      | HMAC algorithm (HS256, HS384, or HS512)         |
| `SMQ_AUTH_ACTIVE_SECRET_KEY`   | Yes      | Active secret key for signing tokens            |
| `SMQ_AUTH_RETIRING_SECRET_KEY` | No       | Retiring secret key (for rotation grace period) |

**Important:** Never use the default value "secret" in production environments.

### Single-Secret Mode

Set only the active secret key:

```bash
export SMQ_AUTH_KEYS_ALGORITHM="HS256"
export SMQ_AUTH_ACTIVE_SECRET_KEY="your-strong-random-secret-here"
export SMQ_AUTH_RETIRING_SECRET_KEY=""
```

The tokenizer will:

- Issue new tokens signed with the active secret
- Verify tokens using the active secret

### Two-Secret Mode (Secret Rotation)

Set both active and retiring secret keys:

```bash
export SMQ_AUTH_KEYS_ALGORITHM="HS256"
export SMQ_AUTH_ACTIVE_SECRET_KEY="new-strong-random-secret"
export SMQ_AUTH_RETIRING_SECRET_KEY="old-secret-for-grace-period"
```

The tokenizer will:

- Issue new tokens signed with the active secret
- Verify tokens using both active and retiring secrets

## Secret Rotation Process

Zero-downtime secret rotation in 3 simple steps:

### 1. Generate New Secret

Generate a cryptographically secure random secret:

````bash
# Generate a 64-character random secret (recommended for HS512)
openssl rand -base64 64

# Generate a 256-bit hex secret (for HS256)
openssl rand -hex 32

# For HS384, use 48 bytes:
openssl rand -hex 48

# For HS512, use 64 bytes:
openssl rand -hex 64

**Minimum secret lengths by algorithm:**

- HS256: 32 bytes (256 bits)
- HS384: 48 bytes (384 bits)
- HS512: 64 bytes (512 bits)

### 2. Update Environment & Restart

Move the current active secret to retiring position and set the new secret as active:

```bash
# Before rotation
SMQ_AUTH_ACTIVE_SECRET_KEY="current-secret-key"
SMQ_AUTH_RETIRING_SECRET_KEY=""  # No retiring secret

# During rotation (both secrets valid for grace period)
SMQ_AUTH_ACTIVE_SECRET_KEY="new-secret-key"
SMQ_AUTH_RETIRING_SECRET_KEY="current-secret-key"

# Restart service with new config
docker-compose restart auth
````

During the grace period, tokens signed with either secret remain valid.

### 3. Clean Up After Grace Period

After the grace period expires (typically 7-30 days), remove the retiring secret:

```bash
# Remove retiring secret configuration
SMQ_AUTH_ACTIVE_SECRET_KEY="new-secret-key"
SMQ_AUTH_RETIRING_SECRET_KEY=""  # Remove retiring secret

# Restart service
docker-compose restart auth
```

## Grace Period Recommendations

**Recommended:** 168 hours (7 days)
**Minimum:** 24 hours
**Maximum:** 720 hours (30 days)

The grace period should be longer than your longest-lived access token duration.

## Security Best Practices

- **Never use default values** - The default "secret" value is not secure
- **Generate strong secrets** - Use cryptographically secure random generators
- **Minimum secret length** - Always meet or exceed the algorithm's recommended key size
- **Rotate secrets regularly:**
  - Standard environments: every 90 days
  - High-security environments: every 30 days
- **Never commit secrets to version control**
- **Use secrets management in production** (HashiCorp Vault, AWS Secrets Manager, etc.)
- **Store secrets securely** - Use environment variables or secure configuration management
- **Use HS512 for maximum security** - Provides the strongest HMAC protection

## Example: Complete Rotation

```bash
# Day 0: Normal operation
export SMQ_AUTH_KEYS_ALGORITHM="HS256"
export SMQ_AUTH_ACTIVE_SECRET_KEY="HyE2D4RUt9nnKG6v8zKEqAp6g6ka8hhZsqUpzgKvnwpXrNVQSH"
export SMQ_AUTH_RETIRING_SECRET_KEY=""

# Day 1: Start rotation - generate new secret
NEW_SECRET=$(openssl rand -base64 48)
echo "New secret: $NEW_SECRET"

# Day 1: Update config and restart
export SMQ_AUTH_ACTIVE_SECRET_KEY="$NEW_SECRET"
export SMQ_AUTH_RETIRING_SECRET_KEY="HyE2D4RUt9nnKG6v8zKEqAp6g6ka8hhZsqUpzgKvnwpXrNVQSH"
docker-compose restart auth

# Day 8: Grace period expired - remove old secret
export SMQ_AUTH_RETIRING_SECRET_KEY=""
docker-compose restart auth
```

## Troubleshooting

### Invalid or empty active secret

```bash
Error: invalid symmetric key
```

**Solution:** Ensure the active secret is not empty. Verify `SMQ_AUTH_ACTIVE_SECRET_KEY` environment variable is set correctly.

### Unsupported algorithm

```bash
Error: unsupported key algorithm
```

**Solution:** Ensure `SMQ_AUTH_KEYS_ALGORITHM` is set to one of: HS256, HS384, or HS512.

### Token verification fails during rotation

If tokens are being rejected during rotation:

1. **Verify both secrets are set correctly** - Check that the retiring secret matches the previous active secret
2. **Check token expiry** - Expired tokens will fail regardless of valid secrets
3. **Review logs** - Look for authentication errors indicating which secret is being used
