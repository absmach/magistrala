// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

// Storage is an interface that specifies the methods for uploading, deleting, and updating profile pictures.
//
//go:generate mockery --name Storage --output=../mocks --filename storage.go --quiet --note "Copyright (c) Abstract Machines"
type Storage interface {
	UploadProfilePicture(ctx context.Context, file io.Reader, id string) (string, error)
	DeleteProfilePicture(ctx context.Context, imageURL string) error
	UpdateProfilePicture(ctx context.Context, file io.Reader, id string) (string, error)
}

type storageClient struct {
	client     *storage.Client
	bucketName string
}

func NewStorageClient(ctx context.Context, credentials, bucketName string) (Storage, error) {
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(credentials))
	if err != nil {
		return nil, fmt.Errorf("failed to create cloud storage client: %v", err)
	}

	return &storageClient{
		client:     client,
		bucketName: bucketName,
	}, nil
}

// UploadProfilePicture uploads a profile picture to Google Cloud Storage and returns the file URL.
func (s *storageClient) UploadProfilePicture(ctx context.Context, file io.Reader, id string) (string, error) {
	fileName := fmt.Sprintf("%s.jpg", id)

	bucket := s.client.Bucket(s.bucketName)
	object := bucket.Object(fileName)

	writer := object.NewWriter(ctx)
	if _, err := io.Copy(writer, file); err != nil {
		return "", fmt.Errorf("failed to write file to bucket: %v", err)
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("failed to close writer: %v", err)
	}

	url := fmt.Sprintf("https://storage.googleapis.com/%s/%s", s.bucketName, fileName)
	return url, nil
}

// DeleteProfilePicture deletes a profile picture from Google Cloud Storage.
func (s *storageClient) DeleteProfilePicture(ctx context.Context, imageURL string) error {
	u, err := url.Parse(imageURL)
	if err != nil {
		return fmt.Errorf("failed to parse image URL: %v", err)
	}

	pathSegments := strings.Split(u.Path, "/")
	if len(pathSegments) == 0 {
		return fmt.Errorf("invalid image URL format")
	}
	fileName := pathSegments[len(pathSegments)-1]

	bucket := s.client.Bucket(s.bucketName)
	object := bucket.Object(fileName)

	if err := object.Delete(ctx); err != nil {
		if err != storage.ErrObjectNotExist {
			return fmt.Errorf("failed to delete object from bucket: %v", err)
		}
	}

	return nil
}

// UpdateProfilePicture replaces the existing profile picture with a new one and returns the new URL.
func (s *storageClient) UpdateProfilePicture(ctx context.Context, file io.Reader, id string) (string, error) {
	fileName := fmt.Sprintf("%s.jpg", id)

	bucket := s.client.Bucket(s.bucketName)
	object := bucket.Object(fileName)

	// Delete existing profile picture if it exists
	if err := object.Delete(ctx); err != nil && err != storage.ErrObjectNotExist {
		return "", fmt.Errorf("failed to delete existing profile picture: %v", err)
	}

	writer := object.NewWriter(ctx)
	if _, err := io.Copy(writer, file); err != nil {
		return "", fmt.Errorf("failed to write file to bucket: %v", err)
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("failed to close writer: %v", err)
	}

	url := fmt.Sprintf("https://storage.googleapis.com/%s/%s", s.bucketName, fileName)
	return url, nil
}
