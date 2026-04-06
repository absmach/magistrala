// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package jwks

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	grpcAuthV1 "github.com/absmach/magistrala/api/grpc/auth/v1"
	smqauth "github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/auth/api/grpc/auth"
	smqjwt "github.com/absmach/magistrala/auth/tokenizer/util"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/grpcclient"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
	grpchealth "google.golang.org/grpc/health/grpc_health_v1"
)

const (
	issuerName   = "magistrala.auth"
	acceptHeader = "application/json"

	fetchKeyDeadline = 10 * time.Second
	cacheDuration    = 5 * time.Minute

	errorBodyBytes = 1024
)

var (
	// errJWTExpiryKey is used to check if the token is expired.
	errJWTExpiryKey = errors.New(`"exp" not satisfied`)
	// errFetchJWKS indicates an error fetching JWKS from URL.
	errFetchJWKS = errors.New("failed to fetch jwks")
	// errInvalidIssuer indicates an invalid issuer value.
	errInvalidIssuer = errors.New("invalid token issuer value")
	// ErrValidateJWTToken indicates a failure to validate JWT token.
	errValidateJWTToken = errors.New("failed to validate jwt token")
)

var _ authn.Authentication = (*authentication)(nil)

type authentication struct {
	jwksURL       string
	authSvcClient grpcAuthV1.AuthServiceClient
	httpClient    *http.Client
	cache         *jwksCache
}

type jwksCache struct {
	mu       sync.RWMutex
	jwks     jwk.Set
	cachedAt time.Time
}

func NewAuthentication(ctx context.Context, jwksURL string, cfg grpcclient.Config) (authn.Authentication, grpcclient.Handler, error) {
	client, err := grpcclient.NewHandler(cfg)
	if err != nil {
		return nil, nil, err
	}

	health := grpchealth.NewHealthClient(client.Connection())
	resp, err := health.Check(ctx, &grpchealth.HealthCheckRequest{
		Service: "auth",
	})
	if err != nil || resp.GetStatus() != grpchealth.HealthCheckResponse_SERVING {
		return nil, nil, grpcclient.ErrSvcNotServing
	}
	authSvcClient := auth.NewAuthClient(client.Connection(), cfg.Timeout)

	httpClient := &http.Client{}

	return authentication{
		jwksURL:       jwksURL,
		authSvcClient: authSvcClient,
		httpClient:    httpClient,
		cache:         &jwksCache{},
	}, client, nil
}

func (a authentication) Authenticate(ctx context.Context, token string) (authn.Session, error) {
	if strings.HasPrefix(token, authn.PatPrefix) {
		res, err := a.authSvcClient.Authenticate(ctx, &grpcAuthV1.AuthNReq{Token: token})
		if err != nil {
			return authn.Session{}, errors.Wrap(svcerr.ErrAuthentication, err)
		}
		return authn.Session{Type: authn.PersonalAccessToken, PatID: res.GetId(), UserID: res.GetUserId(), Role: authn.Role(res.GetUserRole())}, nil
	}

	jwks, err := a.fetchJWKS(ctx, false)
	if err != nil {
		return authn.Session{}, errors.Wrap(svcerr.ErrAuthentication, err)
	}

	tkn, err := validateToken(token, jwks)
	if err != nil {
		// If signature verification failed, try force with refresh JWKS (key rotation scenario)
		if isSignatureError(err) {
			jwks, fetchErr := a.fetchJWKS(ctx, true)
			if fetchErr == nil {
				tkn, err = validateToken(token, jwks)
			}
		}

		if err != nil {
			return authn.Session{}, errors.Wrap(svcerr.ErrAuthentication, err)
		}
	}

	key, err := smqjwt.ToKey(tkn)
	if err != nil {
		return authn.Session{}, errors.Wrap(svcerr.ErrAuthentication, err)
	}

	return authn.Session{
		Type:     authn.AccessToken,
		UserID:   key.Subject,
		Role:     authn.Role(key.Role),
		Verified: key.Verified,
	}, nil
}

func isSignatureError(err error) bool {
	return !errors.Contains(err, errJWTExpiryKey) &&
		!errors.Contains(err, errInvalidIssuer) &&
		!errors.Contains(err, smqauth.ErrExpiry)
}

func (a authentication) fetchJWKS(ctx context.Context, forceRefresh bool) (jwk.Set, error) {
	if !forceRefresh {
		a.cache.mu.RLock()
		if time.Since(a.cache.cachedAt) < cacheDuration && a.cache.jwks.Len() > 0 {
			cached := a.cache.jwks
			a.cache.mu.RUnlock()
			return cached, nil
		}
		a.cache.mu.RUnlock()
	}

	fetchCtx := ctx
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		fetchCtx, cancel = context.WithTimeout(ctx, fetchKeyDeadline)
		defer cancel()
	}

	// Fetch fresh JWKS from auth service
	req, err := http.NewRequestWithContext(fetchCtx, http.MethodGet, a.jwksURL, nil)
	if err != nil {
		return nil, errors.Wrap(errFetchJWKS, err)
	}
	req.Header.Set("Accept", acceptHeader)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(errFetchJWKS, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Read error body for better diagnostics
		body, _ := io.ReadAll(io.LimitReader(resp.Body, errorBodyBytes))
		return nil, errors.Wrap(errFetchJWKS, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body)))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(errFetchJWKS, err)
	}

	set, err := jwk.Parse(data)
	if err != nil {
		return nil, errors.Wrap(errFetchJWKS, err)
	}

	a.cache.mu.Lock()
	a.cache.jwks = set
	a.cache.cachedAt = time.Now()
	a.cache.mu.Unlock()

	return set, nil
}

func validateToken(token string, jwks jwk.Set) (jwt.Token, error) {
	tkn, err := jwt.Parse(
		[]byte(token),
		jwt.WithValidate(true),
		jwt.WithKeySet(jwks, jws.WithInferAlgorithmFromKey(true)),
	)
	if err != nil {
		if errors.Contains(err, errJWTExpiryKey) {
			return nil, smqauth.ErrExpiry
		}
		return nil, err
	}

	// Validate issuer
	validator := jwt.ValidatorFunc(func(_ context.Context, t jwt.Token) jwt.ValidationError {
		if t.Issuer() != issuerName {
			return jwt.NewValidationError(errInvalidIssuer)
		}
		return nil
	})
	if err := jwt.Validate(tkn, jwt.WithValidator(validator)); err != nil {
		return nil, errors.Wrap(errValidateJWTToken, err)
	}

	return tkn, nil
}
