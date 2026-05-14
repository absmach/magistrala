// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package atom

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

var ErrInvalidBearerToken = errors.New("invalid bearer token")

const jwksCacheDuration = 5 * time.Minute

type TokenVerifier struct {
	jwksURL    string
	issuer     string
	audience   string
	httpClient *http.Client
	client     *Client
	cache      jwk.Set
	cachedAt   time.Time
	mu         sync.RWMutex
}

func NewTokenVerifier(cfg Config) *TokenVerifier {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}
	return &TokenVerifier{
		jwksURL:  cfg.JWKSURL,
		issuer:   cfg.JWTIssuer,
		audience: cfg.JWTAudience,
		client:   NewClient(cfg),
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (v *TokenVerifier) VerifyTokenClaims(ctx context.Context, token string) (TokenClaims, error) {
	if strings.HasPrefix(token, "atom_") {
		res, err := v.client.Introspect(ctx, token)
		if err != nil || !res.Active || res.EntityID == "" {
			return TokenClaims{}, ErrInvalidBearerToken
		}
		return TokenClaims{
			SubjectID: res.EntityID,
			SessionID: res.SessionID,
			TenantID:  res.TenantID,
		}, nil
	}

	set, err := v.fetchJWKS(ctx, false)
	if err != nil {
		return TokenClaims{}, err
	}
	tkn, err := v.parseVerifiedToken(token, set)
	if err != nil {
		set, refreshErr := v.fetchJWKS(ctx, true)
		if refreshErr == nil {
			tkn, err = v.parseVerifiedToken(token, set)
		}
	}
	if err != nil {
		return TokenClaims{}, ErrInvalidBearerToken
	}
	claims := TokenClaims{
		SubjectID: tkn.Subject(),
		ExpiresAt: tkn.Expiration().Unix(),
		IssuedAt:  tkn.IssuedAt().Unix(),
	}
	if claims.SubjectID == "" {
		return TokenClaims{}, ErrInvalidBearerToken
	}
	if sid, ok := stringClaim(tkn, "sid"); ok {
		claims.SessionID = sid
	}
	if tid, ok := stringClaim(tkn, "tid"); ok {
		claims.TenantID = tid
	}
	return claims, nil
}

func (v *TokenVerifier) fetchJWKS(ctx context.Context, force bool) (jwk.Set, error) {
	if !force {
		v.mu.RLock()
		if v.cache != nil && time.Since(v.cachedAt) < jwksCacheDuration {
			set := v.cache
			v.mu.RUnlock()
			return set, nil
		}
		v.mu.RUnlock()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	res, err := v.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 1024))
		return nil, fmt.Errorf("fetch atom jwks: status=%d body=%s", res.StatusCode, string(body))
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	set, err := jwk.Parse(body)
	if err != nil {
		return nil, err
	}
	v.mu.Lock()
	v.cache = set
	v.cachedAt = time.Now()
	v.mu.Unlock()
	return set, nil
}

func (v *TokenVerifier) parseVerifiedToken(token string, set jwk.Set) (jwt.Token, error) {
	options := []jwt.ParseOption{
		jwt.WithValidate(true),
		jwt.WithKeySet(set, jws.WithInferAlgorithmFromKey(true)),
	}
	if v.issuer != "" {
		options = append(options, jwt.WithIssuer(v.issuer))
	}
	if v.audience != "" {
		options = append(options, jwt.WithAudience(v.audience))
	}
	return jwt.Parse(
		[]byte(token),
		options...,
	)
}

func stringClaim(tkn jwt.Token, name string) (string, bool) {
	value, ok := tkn.Get(name)
	if !ok {
		return "", false
	}
	str, ok := value.(string)
	return str, ok
}
