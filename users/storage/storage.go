// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"context"
	"io"
)

type Storage interface {
	UploadProfilePicture(ctx context.Context, file io.Reader, id string) (string, error)

	DeleteProfilePicture(ctx context.Context, imageURL string) error

	UpdateProfilePicture(ctx context.Context, file io.Reader, id string) (string, error)
}
