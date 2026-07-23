// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	smqauthn "github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
)

const secureEnvelopeVersion = "v2"

const secureCredentialTTL = 5 * time.Minute

func (bs bootstrapService) GenerateSecureCredential(ctx context.Context, session smqauthn.Session, configID string) (SecureBootstrapCredential, error) {
	if bs.transportKeys == nil || bs.dbCipher == nil {
		return SecureBootstrapCredential{}, errors.Wrap(errGenerateSecureCredential, errors.New("domain transport key support not configured"))
	}
	cfg, err := bs.configs.RetrieveByID(ctx, session.DomainID, configID)
	if err != nil {
		return SecureBootstrapCredential{}, errors.Wrap(errGenerateSecureCredential, err)
	}
	cfg, err = bs.decryptConfigExternalKeyForManagement(cfg, errGenerateSecureCredential)
	if err != nil {
		return SecureBootstrapCredential{}, err
	}
	if cfg.ExternalKey == "" {
		return SecureBootstrapCredential{}, errors.Wrap(errGenerateSecureCredential, ErrExternalKeyUnavailable)
	}
	if cfg.Status == DisabledStatus {
		return SecureBootstrapCredential{}, errors.Wrap(errGenerateSecureCredential, ErrBootstrapDisabled)
	}

	key, err := bs.transportKeys.RetrieveCurrent(ctx, session.DomainID)
	if err != nil {
		return SecureBootstrapCredential{}, errors.Wrap(errGenerateSecureCredential, err)
	}
	secret, err := bs.dbCipher.open("domain-transport-key", key.EncryptedSecret, transportSecretAAD(session.DomainID, key.KeyID))
	if err != nil {
		return SecureBootstrapCredential{}, errors.Wrap(errGenerateSecureCredential, err)
	}
	aead, err := newTransportAEAD(secret)
	if err != nil {
		return SecureBootstrapCredential{}, errors.Wrap(errGenerateSecureCredential, err)
	}
	requestID, err := bs.idProvider.ID()
	if err != nil {
		return SecureBootstrapCredential{}, errors.Wrap(errGenerateSecureCredential, err)
	}
	now := bs.now().UTC()
	request := SecureBootstrapRequest{
		ExternalID: cfg.ExternalID, ExternalKey: cfg.ExternalKey,
		IssuedAt: now.Unix(), RequestID: requestID,
	}
	plain, err := json.Marshal(request)
	if err != nil {
		return SecureBootstrapCredential{}, errors.Wrap(errGenerateSecureCredential, err)
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return SecureBootstrapCredential{}, errors.Wrap(errGenerateSecureCredential, err)
	}
	ciphertext := aead.Seal(nil, nonce, plain, []byte(secureRequestAAD(session.DomainID, cfg.ExternalID, key.KeyID)))
	payload := append(nonce, ciphertext...)
	encryptedKey := secureEnvelopeVersion + "." + key.KeyID + "." + base64.RawURLEncoding.EncodeToString(payload)

	return SecureBootstrapCredential{
		ExternalID: cfg.ExternalID, KeyID: key.KeyID,
		EncryptedExternalKey: encryptedKey,
		Authorization:        "Client " + encryptedKey,
		RequestID:            requestID, ExpiresAt: now.Add(secureCredentialTTL),
		BootstrapURL: "/clients/bootstrap/secure/" + cfg.ExternalID,
	}, nil
}

func (bs bootstrapService) decryptSecureRequest(ctx context.Context, cfg *Config, envelope string) (string, error) {
	if bs.transportKeys == nil || bs.dbCipher == nil {
		return "", errors.New("domain transport key support not configured")
	}
	parts := strings.Split(envelope, ".")
	if len(parts) != 3 || parts[0] != secureEnvelopeVersion || parts[1] == "" {
		return "", errors.New("invalid secure bootstrap envelope")
	}
	keyID := parts[1]
	key, err := bs.transportKeys.Retrieve(ctx, cfg.DomainID, keyID)
	if err != nil {
		return "", err
	}
	now := bs.now().UTC()
	if key.Status != TransportKeyActive {
		if key.Status != TransportKeyRetiring || key.RetireAt == nil || !now.Before(*key.RetireAt) {
			return "", errors.New("domain transport key is no longer valid")
		}
	}
	secret, err := bs.dbCipher.open("domain-transport-key", key.EncryptedSecret, transportSecretAAD(cfg.DomainID, keyID))
	if err != nil {
		return "", err
	}
	aead, err := newTransportAEAD(secret)
	if err != nil {
		return "", err
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || len(payload) < aead.NonceSize()+aead.Overhead() {
		return "", errors.New("invalid secure bootstrap envelope")
	}
	nonce, ciphertext := payload[:aead.NonceSize()], payload[aead.NonceSize():]
	plain, err := aead.Open(nil, nonce, ciphertext, []byte(secureRequestAAD(cfg.DomainID, cfg.ExternalID, keyID)))
	if err != nil {
		return "", errors.New("failed to decrypt secure bootstrap request")
	}
	var request SecureBootstrapRequest
	if err := json.Unmarshal(plain, &request); err != nil {
		return "", errors.New("invalid secure bootstrap request")
	}
	if request.ExternalID != cfg.ExternalID || request.ExternalKey == "" || request.RequestID == "" {
		return "", errors.New("secure bootstrap request does not match the requested enrollment")
	}
	issuedAt := time.Unix(request.IssuedAt, 0)
	if issuedAt.Before(now.Add(-5*time.Minute)) || issuedAt.After(now.Add(time.Minute)) {
		return "", errors.New("secure bootstrap request has expired")
	}
	if err := bs.transportKeys.ConsumeRequestID(ctx, cfg.DomainID, keyID, request.RequestID, now.Add(5*time.Minute)); err != nil {
		if errors.Contains(err, repoerr.ErrConflict) {
			return "", errors.New("secure bootstrap request was already used")
		}
		return "", err
	}
	cfg.SecureTransportKey = append([]byte(nil), secret...)
	cfg.SecureTransportKeyID = keyID
	cfg.SecureRequestID = request.RequestID
	return request.ExternalKey, nil
}

func encryptSecureResponse(cfg Config, plain []byte) ([]byte, error) {
	aead, err := newTransportAEAD(cfg.SecureTransportKey)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	ciphertext := aead.Seal(nil, nonce, plain, []byte(secureResponseAAD(cfg.DomainID, cfg.ExternalID, cfg.SecureTransportKeyID, cfg.SecureRequestID)))
	return append(nonce, ciphertext...), nil
}

func secureRequestAAD(domainID, externalID, keyID string) string {
	return fmt.Sprintf("bootstrap-request:%s:%s:%s", domainID, externalID, keyID)
}

func secureResponseAAD(domainID, externalID, keyID, requestID string) string {
	return fmt.Sprintf("bootstrap-response:%s:%s:%s:%s", domainID, externalID, keyID, requestID)
}
