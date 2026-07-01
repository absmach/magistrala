// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package atom

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

func TestTokenVerifierVerifiesAtomJWT(t *testing.T) {
	token, jwksURL := signedAtomTokenServer(t, time.Now().Add(time.Hour))

	claims, err := NewTokenVerifier(Config{
		JWKSURL:     jwksURL,
		JWTIssuer:   "http://atom:8080",
		JWTAudience: "magistrala",
		Timeout:     time.Second,
	}).VerifyTokenClaims(context.Background(), token)
	if err != nil {
		t.Fatalf("verify token: %v", err)
	}
	if claims.SubjectID != "entity-1" || claims.SessionID != "session-1" || claims.TenantID != "tenant-1" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestTokenVerifierRejectsUnsignedPayload(t *testing.T) {
	_, jwksURL := signedAtomTokenServer(t, time.Now().Add(time.Hour))

	_, err := NewTokenVerifier(Config{JWKSURL: jwksURL, Timeout: time.Second}).VerifyTokenClaims(context.Background(), "eyJhbGciOiJub25lIn0.eyJzdWIiOiJlbnRpdHktMSJ9.")
	if err == nil {
		t.Fatal("expected unsigned token to fail")
	}
}

func TestTokenVerifierRejectsExpiredToken(t *testing.T) {
	token, jwksURL := signedAtomTokenServer(t, time.Now().Add(-time.Hour))

	_, err := NewTokenVerifier(Config{JWKSURL: jwksURL, Timeout: time.Second}).VerifyTokenClaims(context.Background(), token)
	if err == nil {
		t.Fatal("expected expired token to fail")
	}
}

func TestTokenVerifierIntrospectsAtomAccessToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != atomAuthIntrospectPath || r.Header.Get("Authorization") != "Bearer atom_test" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(IntrospectionResponse{
			Active:   true,
			EntityID: "entity-2",
			TenantID: "tenant-2",
		})
	}))
	defer srv.Close()

	claims, err := NewTokenVerifier(Config{URL: srv.URL, JWKSURL: srv.URL + "/jwks", Timeout: time.Second}).VerifyTokenClaims(context.Background(), "atom_test")
	if err != nil {
		t.Fatalf("verify access token: %v", err)
	}
	if claims.SubjectID != "entity-2" || claims.TenantID != "tenant-2" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func signedAtomTokenServer(t *testing.T, expiry time.Time) (string, string) {
	t.Helper()
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	privateJWK, err := jwk.FromRaw(privateKey)
	if err != nil {
		t.Fatalf("private jwk: %v", err)
	}
	if err := privateJWK.Set(jwk.AlgorithmKey, jwa.ES256); err != nil {
		t.Fatalf("set private alg: %v", err)
	}
	if err := privateJWK.Set(jwk.KeyIDKey, "kid-1"); err != nil {
		t.Fatalf("set private kid: %v", err)
	}
	publicJWK, err := jwk.FromRaw(privateKey.PublicKey)
	if err != nil {
		t.Fatalf("public jwk: %v", err)
	}
	if err := publicJWK.Set(jwk.AlgorithmKey, jwa.ES256); err != nil {
		t.Fatalf("set public alg: %v", err)
	}
	if err := publicJWK.Set(jwk.KeyIDKey, "kid-1"); err != nil {
		t.Fatalf("set public kid: %v", err)
	}
	set := jwk.NewSet()
	if err := set.AddKey(publicJWK); err != nil {
		t.Fatalf("add key: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(set); err != nil {
			t.Fatalf("write jwks: %v", err)
		}
	}))
	t.Cleanup(srv.Close)

	tkn, err := jwt.NewBuilder().
		Issuer("http://atom:8080").
		Audience([]string{"magistrala"}).
		Subject("entity-1").
		Claim("sid", "session-1").
		Claim("tid", "tenant-1").
		IssuedAt(time.Now()).
		Expiration(expiry).
		Build()
	if err != nil {
		t.Fatalf("build token: %v", err)
	}
	signed, err := jwt.Sign(tkn, jwt.WithKey(jwa.ES256, privateJWK))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return string(signed), srv.URL
}
