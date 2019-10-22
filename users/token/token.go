// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package token provides password recovery token generation with jwt
// Token is sent by email to user as part of recovery URL
// Token is signed by secret signature
package token

import (
	"errors"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/mainflux/mainflux/users"
)

var (

	// ErrMalformedToken malformed token
	ErrMalformedToken = errors.New("Malformed token")
	// ErrExpiredToken  password reset token has expired
	ErrExpiredToken = errors.New("Token is expired")
	// ErrWrongSignature wrong signature
	ErrWrongSignature = errors.New("Wrong token signature")
)

type tokenizer struct {
	hmacSampleSecret []byte // secret for signing token
	tokenDuration    int    // token in duration in min
}

// New creation of tokenizer.
func New(hmacSampleSecret []byte, tokenDuration int) users.Tokenizer {
	return &tokenizer{hmacSampleSecret: hmacSampleSecret, tokenDuration: tokenDuration}
}

func (t *tokenizer) Generate(email string, offset int) (string, error) {
	exp := t.tokenDuration + offset
	if exp < 0 {
		exp = 0
	}
	expires := time.Now().Add(time.Minute * time.Duration(exp))
	nbf := time.Now()

	// Create a new token object, specifying signing method and the claims
	// you would like it to contain
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": email,
		"exp":   expires.Unix(),
		"nbf":   nbf.Unix(),
	})

	// Sign and get the complete encoded token as a string using the secret
	tokenString, err := token.SignedString(t.hmacSampleSecret)
	return tokenString, err
}

// Verify verifies token validity
func (t *tokenizer) Verify(tok string) (string, error) {
	email := ""
	token, err := jwt.Parse(tok, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrWrongSignature
		}

		// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
		return t.hmacSampleSecret, nil
	})

	if err != nil {
		return email, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if claims.VerifyExpiresAt(time.Now().Unix(), false) == false {
			return "", ErrExpiredToken
		}
		email = claims["email"].(string)
	}
	return email, nil
}
