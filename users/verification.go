// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package users

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
)

const VerificationExpiryDuration = 24 * time.Hour

var (
	errFailedToCreateUserVerification = errors.New("failed to create new user verification")
	errFailedToEncodeUserVerification = errors.New("failed to encode user verification")
	errFailedToDecodeUserVerification = errors.New("failed to decode user verification")
)

// UserVerification OTP is sent to the user's email as base64 encoded with UserID, Email and OTP. It should not be exposed via API.
type UserVerification struct {
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	OTP       string    `json:"otp"`
	CreatedAt time.Time `json:"-"`
	ExpiresAt time.Time `json:"-"`
	UsedAt    time.Time `json:"-"`
}

func NewUserVerification(userID, email string) (UserVerification, error) {
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return UserVerification{}, errors.Wrap(errFailedToCreateUserVerification, err)
	}

	return UserVerification{
		UserID:    userID,
		Email:     email,
		OTP:       base64.URLEncoding.EncodeToString(randomBytes),
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().Add(VerificationExpiryDuration).UTC(),
	}, nil
}

func (u UserVerification) Encode() (string, error) {
	jsonBytes, err := json.Marshal(u)
	if err != nil {
		return "", errors.Wrap(errFailedToEncodeUserVerification, err)
	}

	return base64.URLEncoding.EncodeToString(jsonBytes), nil
}

func (u *UserVerification) Decode(data string) error {
	decodedPayload, err := base64.URLEncoding.DecodeString(data)
	if err != nil {
		return errors.Wrap(errFailedToDecodeUserVerification, err)
	}

	if err := json.Unmarshal(decodedPayload, u); err != nil {
		return errors.Wrap(errFailedToDecodeUserVerification, err)
	}

	if u.UserID == "" || u.Email == "" || u.OTP == "" {
		return svcerr.ErrInvalidUserVerification
	}

	return nil
}

func (u UserVerification) Valid() error {
	if u.UserID == "" || u.Email == "" || u.OTP == "" {
		return svcerr.ErrInvalidUserVerification
	}

	// Verification should have created time.
	if u.CreatedAt.IsZero() {
		return svcerr.ErrInvalidUserVerification
	}

	// Verification should have expiry time.
	if u.ExpiresAt.IsZero() {
		return svcerr.ErrInvalidUserVerification
	}

	// Expiry time should not be before Created time
	if u.ExpiresAt.Before(u.CreatedAt) {
		return svcerr.ErrInvalidUserVerification
	}

	// Verification should be not be Expired.
	if time.Now().After(u.ExpiresAt) {
		return svcerr.ErrUserVerificationExpired
	}

	// Verification should not be used.
	if !u.UsedAt.IsZero() {
		return svcerr.ErrUserVerificationExpired
	}

	return nil
}

func (u UserVerification) Match(ruv UserVerification) error {
	if u.UserID != ruv.UserID {
		return svcerr.ErrInvalidUserVerification
	}

	if u.Email != ruv.Email {
		return svcerr.ErrInvalidUserVerification
	}

	if u.OTP != ruv.OTP {
		return svcerr.ErrInvalidUserVerification
	}
	return nil
}
