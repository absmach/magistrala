// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package token provides password recovery token generation with jwt
// Token is sent by email to user as part of recovery URL
// Token is signed by secret signature
package token

import (
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/mainflux/mainflux/errors"
	"github.com/mainflux/mainflux/users"
)

var (
	// errExpiredToken  password reset token has expired
	errExpiredToken = errors.New("Token is expired")
	// errWrongSignature wrong signature
	errWrongSignature = errors.New("Wrong token signature")
	// errValidateToken represents error when validating token
	errValidateToken = errors.New("Validate token failed")
)

type tokenizer struct {
	hmacSampleSecret []byte // secret for signing token
	tokenDuration    int    // token in duration in min
}

// New creation of tokenizer.
func New(hmacSampleSecret []byte, tokenDuration int) users.Tokenizer {
	return &tokenizer{hmacSampleSecret: hmacSampleSecret, tokenDuration: tokenDuration}
}

func (t *tokenizer) Generate(email string, offset int) (string, errors.Error) {
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
	if err != nil {
		return tokenString, errors.Wrap(users.ErrGetToken, err)
	}
	return tokenString, nil
}

// Verify verifies token validity
func (t *tokenizer) Verify(tok string) (string, errors.Error) {
	email := ""
	token, err := jwt.Parse(tok, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errWrongSignature
		}

		// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
		return t.hmacSampleSecret, nil
	})

	if err != nil {
		return email, errors.Wrap(errValidateToken, err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if claims.VerifyExpiresAt(time.Now().Unix(), false) == false {
			return "", errExpiredToken
		}
		email = claims["email"].(string)
	}
	return email, nil
}
