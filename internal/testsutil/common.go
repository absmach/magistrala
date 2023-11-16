// Copyright (c) Magistrala
// SPDX-License-Identifier: Apache-2.0

package testsutil

import (
	"context"
	"fmt"
	"testing"

	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/absmach/magistrala/users"
	cmocks "github.com/absmach/magistrala/users/mocks"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func GenerateUUID(t *testing.T) string {
	idProvider := uuid.New()
	ulid, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	return ulid
}

func GenerateValidToken(t *testing.T, clientID string, svc users.Service, cRepo *cmocks.Repository, phasher users.Hasher, domainID string) string {
	client := mgclients.Client{
		ID:   clientID,
		Name: "validtoken",
		Credentials: mgclients.Credentials{
			Identity: "validtoken",
			Secret:   "secret",
		},
		Status: mgclients.EnabledStatus,
	}
	rClient := client
	rClient.Credentials.Secret, _ = phasher.Hash(client.Credentials.Secret)

	repoCall := cRepo.On("RetrieveByIdentity", context.Background(), client.Credentials.Identity).Return(rClient, nil)
	token, err := svc.IssueToken(context.Background(), client.Credentials.Identity, client.Credentials.Secret, domainID)
	assert.True(t, errors.Contains(err, nil), fmt.Sprintf("Create token expected nil got %s\n", err))
	ok := repoCall.Parent.AssertCalled(t, "RetrieveByIdentity", context.Background(), client.Credentials.Identity)
	assert.True(t, ok, "RetrieveByIdentity was not called on creating token")
	repoCall.Unset()
	return token.AccessToken
}

func CleanUpDB(t *testing.T, db *sqlx.DB) {
	_, err := db.Exec("DELETE FROM groups")
	require.Nil(t, err, fmt.Sprintf("clean groups unexpected error: %s", err))
	_, err = db.Exec("DELETE FROM clients")
	require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
}
