// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package asymmetric

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/absmach/supermq"
	"github.com/absmach/supermq/auth"
	smqjwt "github.com/absmach/supermq/auth/tokenizer/util"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

var (
	errLoadingPrivateKey      = errors.New("failed to load private key")
	errDuplicateRetiringKeyID = errors.New("retiring key ID matches active key ID")
	errInvalidKeySize         = errors.New("invalid ED25519 key size")
	errParsingPrivateKey      = errors.New("failed to parse private key")
	errInvalidKeyType         = errors.New("private key is not ED25519")
	errNoValidPublicKeys      = errors.New("no valid public keys available")
	errNoActiveKey            = errors.New("active key not loaded")
)

type keyPair struct {
	id         string
	privateKey jwk.Key
	publicKey  jwk.Key
}

// Tokenizer is safe for concurrent use. Keys are set during construction
// and never modified afterward.
type tokenizer struct {
	activeKey   *keyPair
	retiringKey *keyPair // Optional, for key rotation grace period
}

var _ auth.Tokenizer = (*tokenizer)(nil)

// NewTokenizer creates a new asymmetric tokenizer with active and optionally retiring keys.
// activeKeyPath is required. retiringKeyPath is optional (can be empty string).
// If retiringKeyPath is provided but the file doesn't exist or is invalid, a warning is logged
// but the tokenizer is still created with just the active key.
// Key IDs are derived from filenames to ensure consistency across multiple service instances.
func NewTokenizer(activeKeyPath, retiringKeyPath string, idProvider supermq.IDProvider, logger *slog.Logger) (auth.Tokenizer, error) {
	activeKID := keyIDFromPath(activeKeyPath)

	activePrivateJwk, activePublicJwk, err := loadKeyPair(activeKeyPath, activeKID)
	if err != nil {
		return nil, err
	}

	mgr := &tokenizer{
		activeKey: &keyPair{
			id:         activeKID,
			privateKey: activePrivateJwk,
			publicKey:  activePublicJwk,
		},
	}

	if retiringKeyPath != "" {
		retiringKID := keyIDFromPath(retiringKeyPath)
		if retiringKID == activeKID {
			return nil, errDuplicateRetiringKeyID
		}

		retiringPrivateJwk, retiringPublicJwk, err := loadKeyPair(retiringKeyPath, retiringKID)
		if err != nil {
			logger.Warn("failed to load retiring key, continuing without it", slog.Any("error", err))
			return mgr, nil
		}

		mgr.retiringKey = &keyPair{
			id:         retiringKID,
			privateKey: retiringPrivateJwk,
			publicKey:  retiringPublicJwk,
		}
		logger.Info("loaded retiring key for rotation grace period", slog.String("key_id", retiringKID))
	}

	return mgr, nil
}

func (tok *tokenizer) Issue(key auth.Key) (string, error) {
	if tok.activeKey == nil {
		return "", errNoActiveKey
	}

	tkn, err := smqjwt.BuildToken(key)
	if err != nil {
		return "", err
	}
	headers := jws.NewHeaders()
	if err := headers.Set(jwk.KeyIDKey, tok.activeKey.id); err != nil {
		return "", err
	}

	signedBytes, err := jwt.Sign(tkn, jwt.WithKey(jwa.EdDSA, tok.activeKey.privateKey, jws.WithProtectedHeaders(headers)))
	if err != nil {
		return "", err
	}

	return string(signedBytes), nil
}

func (tok *tokenizer) Parse(ctx context.Context, tokenString string) (auth.Key, error) {
	if len(tokenString) >= 3 && tokenString[:3] == smqjwt.PatPrefix {
		return auth.Key{Type: auth.PersonalAccessToken}, nil
	}

	set := jwk.NewSet()
	if err := set.AddKey(tok.activeKey.publicKey); err != nil {
		return auth.Key{}, err
	}
	if tok.retiringKey != nil {
		if err := set.AddKey(tok.retiringKey.publicKey); err != nil {
			return auth.Key{}, err
		}
	}

	tkn, err := jwt.Parse(
		[]byte(tokenString),
		jwt.WithValidate(true),
		jwt.WithKeySet(set, jws.WithInferAlgorithmFromKey(true)),
	)
	if err != nil {
		return auth.Key{}, err
	}

	if tkn.Issuer() != smqjwt.IssuerName {
		return auth.Key{}, smqjwt.ErrInvalidIssuer
	}

	return smqjwt.ToKey(tkn)
}

func (tok *tokenizer) RetrieveJWKS() ([]auth.PublicKeyInfo, error) {
	publicKeys := make([]auth.PublicKeyInfo, 0, 2)

	if tok.activeKey != nil {
		if pkInfo := extractPublicKeyInfo(tok.activeKey); pkInfo != nil {
			publicKeys = append(publicKeys, *pkInfo)
		}
	}

	if tok.retiringKey != nil {
		if pkInfo := extractPublicKeyInfo(tok.retiringKey); pkInfo != nil {
			publicKeys = append(publicKeys, *pkInfo)
		}
	}

	if len(publicKeys) == 0 {
		return nil, errNoValidPublicKeys
	}

	return publicKeys, nil
}

func extractPublicKeyInfo(kp *keyPair) *auth.PublicKeyInfo {
	var rawKey ed25519.PublicKey
	if err := kp.publicKey.Raw(&rawKey); err != nil {
		return nil
	}

	return &auth.PublicKeyInfo{
		KeyID:     kp.id,
		KeyType:   "OKP",
		Algorithm: "EdDSA",
		Use:       "sig",
		Curve:     "Ed25519",
		X:         base64.RawURLEncoding.EncodeToString(rawKey),
	}
}

func loadKeyPair(privateKeyPath string, kid string) (jwk.Key, jwk.Key, error) {
	privateKeyBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, nil, errors.Wrap(errLoadingPrivateKey, err)
	}

	var privateKey ed25519.PrivateKey
	block, _ := pem.Decode(privateKeyBytes)
	switch {
	case block != nil:
		parsedKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, nil, errors.Wrap(errParsingPrivateKey, err)
		}
		var ok bool
		privateKey, ok = parsedKey.(ed25519.PrivateKey)
		if !ok {
			return nil, nil, errInvalidKeyType
		}
	default:
		if len(privateKeyBytes) != ed25519.PrivateKeySize {
			return nil, nil, errInvalidKeySize
		}
		privateKey = ed25519.PrivateKey(privateKeyBytes)
	}

	publicKey := privateKey.Public().(ed25519.PublicKey)

	privateJwk, err := jwk.FromRaw(privateKey)
	if err != nil {
		return nil, nil, err
	}
	if err := privateJwk.Set(jwk.AlgorithmKey, jwa.EdDSA); err != nil {
		return nil, nil, err
	}
	if err := privateJwk.Set(jwk.KeyIDKey, kid); err != nil {
		return nil, nil, err
	}

	publicJwk, err := jwk.FromRaw(publicKey)
	if err != nil {
		return nil, nil, err
	}
	if err := publicJwk.Set(jwk.AlgorithmKey, jwa.EdDSA); err != nil {
		return nil, nil, err
	}
	if err := publicJwk.Set(jwk.KeyIDKey, kid); err != nil {
		return nil, nil, err
	}

	return privateJwk, publicJwk, nil
}

func keyIDFromPath(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}
