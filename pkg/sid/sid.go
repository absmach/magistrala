// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package ulid provides a ULID identity provider.
package sid

import (
	"encoding/binary"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/gofrs/uuid/v5"
	"github.com/sqids/sqids-go"
)

// ErrGeneratingID indicates error in generating ULID.
var (
	ErrInitializingShortID = errors.New("failed to initialize short id provider")
	ErrGeneratingID        = errors.New("generating id failed")
	ErrEncodeID            = errors.New("encoding id failed")
)
var _ magistrala.IDProvider = (*sidProvider)(nil)

type sidProvider struct {
	sidEncoder *sqids.Sqids
}

// New instantiates a short ID provider.
func New() (magistrala.IDProvider, error) {
	sidEncoder, err := sqids.New(sqids.Options{
		Alphabet: "FxnXM1kBN6cuhsAvjW3Co7l2RePyY8DwaU04Tzt9fHQrqSVKdpimLGIJOgb5ZE",
	})
	if err != nil {
		return nil, errors.Wrap(ErrInitializingShortID, err)
	}
	return &sidProvider{sidEncoder}, nil
}

func (s *sidProvider) ID() (string, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return "", errors.Wrap(ErrGeneratingID, err)
	}
	idBytes := id.Bytes()

	sid, err := s.sidEncoder.Encode([]uint64{
		binary.BigEndian.Uint64(idBytes[:8]),
		binary.BigEndian.Uint64(idBytes[8:]),
	})
	if err != nil {
		return "", errors.Wrap(ErrEncodeID, err)
	}

	return sid, nil
}
