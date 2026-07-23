// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/absmach/magistrala/pkg/authn"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/stretchr/testify/require"
)

func TestEncryptedExternalKeyAndDomainSecureBootstrap(t *testing.T) {
	configs := &memoryConfigRepository{}
	keys := &memoryTransportKeyRepository{requests: make(map[string]struct{})}
	idp := &sequenceIDProvider{ids: []string{"config-id", "transport-key-id", "request-01", "request-02"}}
	svc := NewWithTransportKeys(
		configs, nil, nil, keys, nil, nil, nil,
		[]byte("12345678910111213141516171819202"), "test-master", idp,
	)
	fixedNow := time.Date(2026, 7, 22, 10, 0, 0, 0, time.UTC)
	svc.(*bootstrapService).now = func() time.Time { return fixedNow }
	session := authn.Session{DomainID: "domain-id"}

	created, err := svc.Add(context.Background(), session, "token", Config{
		ExternalID: "device-01", ExternalKey: "device-secret", Content: "configuration",
	})
	require.NoError(t, err)
	require.Equal(t, "device-secret", created.ExternalKey)
	require.NotEqual(t, "device-secret", configs.config.ExternalKey)
	require.True(t, strings.HasPrefix(configs.config.ExternalKey, "dbv1.test-master."))

	viewed, err := svc.View(context.Background(), session, "config-id")
	require.NoError(t, err)
	require.Equal(t, "device-secret", viewed.ExternalKey)
	listed, err := svc.List(context.Background(), session, Filter{}, 0, 10)
	require.NoError(t, err)
	require.Len(t, listed.Configs, 1)
	require.Equal(t, "device-secret", listed.Configs[0].ExternalKey)

	err = svc.Update(context.Background(), session, Config{
		ID: "config-id", Name: created.Name, Content: created.Content,
		ExternalKey: "rotated-device-secret",
	})
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(configs.config.ExternalKey, "dbv1.test-master."))
	require.NotContains(t, configs.config.ExternalKey, "rotated-device-secret")
	viewed, err = svc.View(context.Background(), session, "config-id")
	require.NoError(t, err)
	require.Equal(t, "rotated-device-secret", viewed.ExternalKey)
	_, err = svc.Bootstrap(context.Background(), "device-secret", "device-01", false)
	require.ErrorIs(t, err, ErrExternalKey)
	_, err = svc.Bootstrap(context.Background(), "rotated-device-secret", "device-01", false)
	require.NoError(t, err)

	transportKey, err := svc.CreateDomainTransportKey(context.Background(), session)
	require.NoError(t, err)
	require.Equal(t, TransportKeyActive, transportKey.Status)
	require.NotEmpty(t, transportKey.Secret)
	require.NotEqual(t, transportKey.Secret, keys.key.EncryptedSecret)

	secret, err := base64.RawURLEncoding.DecodeString(transportKey.Secret)
	require.NoError(t, err)
	credential, err := svc.GenerateSecureCredential(context.Background(), session, "config-id")
	require.NoError(t, err)
	require.Equal(t, "device-01", credential.ExternalID)
	require.Equal(t, transportKey.KeyID, credential.KeyID)
	require.Equal(t, "request-01", credential.RequestID)
	require.Equal(t, fixedNow.Add(secureCredentialTTL), credential.ExpiresAt)
	require.Equal(t, "Client "+credential.EncryptedExternalKey, credential.Authorization)
	require.Equal(t, "/clients/bootstrap/secure/device-01", credential.BootstrapURL)
	require.True(t, strings.HasPrefix(credential.EncryptedExternalKey, "v2.transport-key-id."))

	bootstrapped, err := svc.Bootstrap(context.Background(), credential.EncryptedExternalKey, credential.ExternalID, true)
	require.NoError(t, err)
	require.Equal(t, "configuration", bootstrapped.Content)
	require.Equal(t, transportKey.KeyID, bootstrapped.SecureTransportKeyID)

	reader := NewConfigReader(nil)
	read, err := reader.ReadConfig(bootstrapped, true)
	require.NoError(t, err)
	encrypted := read.(SecureConfigPayload)
	aead, err := newTransportAEAD(secret)
	require.NoError(t, err)
	nonce, ciphertext := encrypted.Payload[:aead.NonceSize()], encrypted.Payload[aead.NonceSize():]
	plain, err := aead.Open(nil, nonce, ciphertext, []byte(secureResponseAAD(session.DomainID, credential.ExternalID, transportKey.KeyID, credential.RequestID)))
	require.NoError(t, err)
	var response bootstrapRes
	require.NoError(t, json.Unmarshal(plain, &response))
	require.Equal(t, "configuration", response.Content)

	_, err = svc.Bootstrap(context.Background(), credential.EncryptedExternalKey, credential.ExternalID, true)
	require.ErrorContains(t, err, "already used")
	svc.(*bootstrapService).now = func() time.Time { return fixedNow.Add(secureCredentialTTL + time.Second) }
	_, err = svc.Bootstrap(context.Background(), credential.EncryptedExternalKey, credential.ExternalID, true)
	require.ErrorContains(t, err, "expired")

	_, err = svc.DisableConfig(context.Background(), session, "config-id")
	require.NoError(t, err)
	require.Equal(t, DisabledStatus, configs.config.Status)
	_, err = svc.GenerateSecureCredential(context.Background(), session, "config-id")
	require.ErrorContains(t, err, "bootstrap configuration is disabled")
}

func TestLegacyHashedExternalKeyDoesNotBreakManagementReads(t *testing.T) {
	configs := &memoryConfigRepository{config: Config{
		ID:          "config-id",
		DomainID:    "domain-id",
		ExternalID:  "device-01",
		ExternalKey: "$2a$10$legacy-bcrypt-hash",
		Status:      EnabledStatus,
	}}
	svc := NewWithTransportKeys(
		configs, nil, nil, &memoryTransportKeyRepository{}, nil, nil, nil,
		[]byte("12345678910111213141516171819202"), "test-master", &sequenceIDProvider{},
	)
	session := authn.Session{DomainID: "domain-id"}

	viewed, err := svc.View(context.Background(), session, "config-id")
	require.NoError(t, err)
	require.Empty(t, viewed.ExternalKey)

	listed, err := svc.List(context.Background(), session, Filter{}, 0, 10)
	require.NoError(t, err)
	require.Len(t, listed.Configs, 1)
	require.Empty(t, listed.Configs[0].ExternalKey)

	_, err = svc.GenerateSecureCredential(context.Background(), session, "config-id")
	require.ErrorContains(t, err, "external key is not available")

	updated, err := svc.DisableConfig(context.Background(), session, "config-id")
	require.NoError(t, err)
	require.Empty(t, updated.ExternalKey)

	_, err = svc.GenerateSecureCredential(context.Background(), session, "config-id")
	require.ErrorContains(t, err, ErrExternalKeyUnavailable.Error())
}

func makeSecureRequestEnvelope(t *testing.T, secret []byte, keyID, domainID string, request SecureBootstrapRequest) string {
	t.Helper()
	aead, err := newTransportAEAD(secret)
	require.NoError(t, err)
	plain, err := json.Marshal(request)
	require.NoError(t, err)
	nonce := make([]byte, aead.NonceSize())
	_, err = rand.Read(nonce)
	require.NoError(t, err)
	ciphertext := aead.Seal(nil, nonce, plain, []byte(secureRequestAAD(domainID, request.ExternalID, keyID)))
	payload := append(nonce, ciphertext...)
	return secureEnvelopeVersion + "." + keyID + "." + base64.RawURLEncoding.EncodeToString(payload)
}

type sequenceIDProvider struct {
	ids []string
}

func (p *sequenceIDProvider) ID() (string, error) {
	id := p.ids[0]
	p.ids = p.ids[1:]
	return id, nil
}

type memoryConfigRepository struct {
	config Config
}

func (r *memoryConfigRepository) Save(_ context.Context, cfg Config) (string, error) {
	r.config = cfg
	return cfg.ID, nil
}

func (r *memoryConfigRepository) RetrieveByID(_ context.Context, domainID, id string) (Config, error) {
	if r.config.DomainID != domainID || r.config.ID != id {
		return Config{}, repoerr.ErrNotFound
	}
	return r.config, nil
}

func (r *memoryConfigRepository) RetrieveAll(_ context.Context, domainID string, _ Filter, offset, limit uint64) ConfigsPage {
	return ConfigsPage{Total: 1, Offset: offset, Limit: limit, Configs: []Config{r.config}}
}

func (r *memoryConfigRepository) RetrieveByExternalID(_ context.Context, externalID string) (Config, error) {
	if r.config.ExternalID != externalID {
		return Config{}, repoerr.ErrNotFound
	}
	return r.config, nil
}

func (r *memoryConfigRepository) Update(_ context.Context, cfg Config) error {
	if r.config.DomainID != cfg.DomainID || r.config.ID != cfg.ID {
		return repoerr.ErrNotFound
	}
	r.config.Name = cfg.Name
	r.config.Content = cfg.Content
	r.config.RenderContext = cfg.RenderContext
	if cfg.ExternalKey != "" {
		r.config.ExternalKey = cfg.ExternalKey
	}
	return nil
}
func (*memoryConfigRepository) AssignProfile(context.Context, string, string, string) error {
	return nil
}
func (*memoryConfigRepository) UpdateCert(context.Context, string, string, string, string, string) (Config, error) {
	return Config{}, nil
}
func (*memoryConfigRepository) Remove(context.Context, string, string) error { return nil }

func (r *memoryConfigRepository) ChangeStatus(_ context.Context, domainID, id string, status Status) error {
	if r.config.DomainID != domainID || r.config.ID != id {
		return repoerr.ErrNotFound
	}
	r.config.Status = status
	return nil
}

type memoryTransportKeyRepository struct {
	key      DomainTransportKey
	requests map[string]struct{}
}

func (r *memoryTransportKeyRepository) Create(_ context.Context, key DomainTransportKey) error {
	if r.key.KeyID != "" {
		return repoerr.ErrConflict
	}
	r.key = key
	return nil
}

func (r *memoryTransportKeyRepository) RetrieveCurrent(context.Context, string) (DomainTransportKey, error) {
	if r.key.KeyID == "" {
		return DomainTransportKey{}, repoerr.ErrNotFound
	}
	return r.key, nil
}

func (r *memoryTransportKeyRepository) Retrieve(_ context.Context, domainID, keyID string) (DomainTransportKey, error) {
	if r.key.DomainID != domainID || r.key.KeyID != keyID {
		return DomainTransportKey{}, repoerr.ErrNotFound
	}
	return r.key, nil
}

func (r *memoryTransportKeyRepository) Rotate(_ context.Context, _ string, next DomainTransportKey, _ time.Time) error {
	r.key = next
	return nil
}

func (r *memoryTransportKeyRepository) ConsumeRequestID(_ context.Context, domainID, keyID, requestID string, _ time.Time) error {
	id := domainID + ":" + keyID + ":" + requestID
	if _, ok := r.requests[id]; ok {
		return repoerr.ErrConflict
	}
	r.requests[id] = struct{}{}
	return nil
}
