// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

type storageClient struct {
	client *storage.Client
}

func NewStorageClient(ctx context.Context) (Storage, error) {
	client, err := storage.NewClient(context.Background(), option.WithCredentialsFile(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")))
	if err != nil {
		return nil, fmt.Errorf("failed to create cloud storage client: %v", err)
	}
	return &storageClient{client: client}, nil
}

// UploadProfilePicture uploads a profile picture to Google Cloud Storage and returns the file URL.
func (s *storageClient) UploadProfilePicture(ctx context.Context, file io.Reader, id string) (string, error) {
	fileName := fmt.Sprintf("%s.jpg", id)

	bucketName := os.Getenv("BUCKET_NAME")
	bucket := s.client.Bucket(bucketName)
	object := bucket.Object(fileName)

	writer := object.NewWriter(ctx)
	if _, err := io.Copy(writer, file); err != nil {
		return "", fmt.Errorf("failed to write file to bucket: %v", err)
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("failed to close writer: %v", err)
	}

	url := fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucketName, fileName)
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

	bucketName := os.Getenv("BUCKET_NAME")
	bucket := s.client.Bucket(bucketName)
	object := bucket.Object(fileName)

	if err := object.Delete(ctx); err != nil {
		return fmt.Errorf("failed to delete object from bucket: %v", err)
	}

	return nil
}

// UpdateProfilePicture replaces the existing profile picture with a new one and returns the new URL.
func (s *storageClient) UpdateProfilePicture(ctx context.Context, file io.Reader, id string) (string, error) {
	fileName := fmt.Sprintf("%s.jpg", id)

	bucketName := os.Getenv("BUCKET_NAME")
	bucket := s.client.Bucket(bucketName)
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

	url := fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucketName, fileName)
	return url, nil
}
