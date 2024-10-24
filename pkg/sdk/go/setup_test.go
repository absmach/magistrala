// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/invitations"
	"github.com/absmach/magistrala/journal"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	mggroups "github.com/absmach/magistrala/pkg/groups"
	sdk "github.com/absmach/magistrala/pkg/sdk/go"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/absmach/magistrala/users"
	"github.com/stretchr/testify/assert"
)

const (
	invalidIdentity = "invalididentity"
	Identity        = "identity"
	Email           = "email"
	InvalidEmail    = "invalidemail"
	secret          = "strongsecret"
	invalidToken    = "invalid"
	contentType     = "application/senml+json"
	invalid         = "invalid"
	wrongID         = "wrongID"
)

var (
	idProvider           = uuid.New()
	validMetadata        = sdk.Metadata{"role": "client"}
	user                 = generateTestUser(&testing.T{})
	description          = "shortdescription"
	gName                = "groupname"
	validToken           = "valid"
	limit         uint64 = 5
	offset        uint64 = 0
	total         uint64 = 200
	passRegex            = regexp.MustCompile("^.{8,}$")
	validID              = testsutil.GenerateUUID(&testing.T{})
)

func generateUUID(t *testing.T) string {
	ulid, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	return ulid
}

func convertUsers(cs []sdk.User) []users.User {
	ccs := []users.User{}

	for _, c := range cs {
		ccs = append(ccs, convertUser(c))
	}

	return ccs
}

func convertThings(cs ...sdk.Thing) []mgclients.Client {
	ccs := []mgclients.Client{}

	for _, c := range cs {
		ccs = append(ccs, convertThing(c))
	}

	return ccs
}

func convertGroups(cs []sdk.Group) []mggroups.Group {
	cgs := []mggroups.Group{}

	for _, c := range cs {
		cgs = append(cgs, convertGroup(c))
	}

	return cgs
}

func convertChannels(cs []sdk.Channel) []mggroups.Group {
	cgs := []mggroups.Group{}

	for _, c := range cs {
		cgs = append(cgs, convertChannel(c))
	}

	return cgs
}

func convertGroup(g sdk.Group) mggroups.Group {
	if g.Status == "" {
		g.Status = mgclients.EnabledStatus.String()
	}
	status, err := mgclients.ToStatus(g.Status)
	if err != nil {
		return mggroups.Group{}
	}

	return mggroups.Group{
		ID:          g.ID,
		Domain:      g.DomainID,
		Parent:      g.ParentID,
		Name:        g.Name,
		Description: g.Description,
		Metadata:    mgclients.Metadata(g.Metadata),
		Level:       g.Level,
		Path:        g.Path,
		Children:    convertChildren(g.Children),
		CreatedAt:   g.CreatedAt,
		UpdatedAt:   g.UpdatedAt,
		Status:      status,
	}
}

func convertChildren(gs []*sdk.Group) []*mggroups.Group {
	cg := []*mggroups.Group{}

	if len(gs) == 0 {
		return cg
	}

	for _, g := range gs {
		insert := convertGroup(*g)
		cg = append(cg, &insert)
	}

	return cg
}

func convertUser(c sdk.User) users.User {
	if c.Status == "" {
		c.Status = users.EnabledStatus.String()
	}
	status, err := users.ToStatus(c.Status)
	if err != nil {
		return users.User{}
	}
	role, err := users.ToRole(c.Role)
	if err != nil {
		return users.User{}
	}
	return users.User{
		ID:             c.ID,
		FirstName:      c.FirstName,
		LastName:       c.LastName,
		Tags:           c.Tags,
		Email:          c.Email,
		Credentials:    users.Credentials(c.Credentials),
		Metadata:       users.Metadata(c.Metadata),
		CreatedAt:      c.CreatedAt,
		UpdatedAt:      c.UpdatedAt,
		Status:         status,
		Role:           role,
		ProfilePicture: c.ProfilePicture,
	}
}

func convertThing(c sdk.Thing) mgclients.Client {
	if c.Status == "" {
		c.Status = mgclients.EnabledStatus.String()
	}
	status, err := mgclients.ToStatus(c.Status)
	if err != nil {
		return mgclients.Client{}
	}
	return mgclients.Client{
		ID:          c.ID,
		Name:        c.Name,
		Tags:        c.Tags,
		Domain:      c.DomainID,
		Credentials: mgclients.Credentials(c.Credentials),
		Metadata:    mgclients.Metadata(c.Metadata),
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
		Status:      status,
	}
}

func convertChannel(g sdk.Channel) mggroups.Group {
	if g.Status == "" {
		g.Status = mgclients.EnabledStatus.String()
	}
	status, err := mgclients.ToStatus(g.Status)
	if err != nil {
		return mggroups.Group{}
	}
	return mggroups.Group{
		ID:          g.ID,
		Domain:      g.DomainID,
		Parent:      g.ParentID,
		Name:        g.Name,
		Description: g.Description,
		Metadata:    mgclients.Metadata(g.Metadata),
		Level:       g.Level,
		Path:        g.Path,
		CreatedAt:   g.CreatedAt,
		UpdatedAt:   g.UpdatedAt,
		Status:      status,
	}
}

func convertInvitation(i sdk.Invitation) invitations.Invitation {
	return invitations.Invitation{
		InvitedBy:   i.InvitedBy,
		UserID:      i.UserID,
		DomainID:    i.DomainID,
		Token:       i.Token,
		Relation:    i.Relation,
		CreatedAt:   i.CreatedAt,
		UpdatedAt:   i.UpdatedAt,
		ConfirmedAt: i.ConfirmedAt,
		Resend:      i.Resend,
	}
}

func convertJournal(j sdk.Journal) journal.Journal {
	return journal.Journal{
		ID:         j.ID,
		Operation:  j.Operation,
		OccurredAt: j.OccurredAt,
		Attributes: j.Attributes,
		Metadata:   j.Metadata,
	}
}

func generateTestUser(t *testing.T) sdk.User {
	createdAt, err := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")
	assert.Nil(t, err, fmt.Sprintf("Unexpected error parsing time: %v", err))
	return sdk.User{
		ID:        generateUUID(t),
		FirstName: "clientname",
		LastName:  "clientlastname",
		Email:     "clientemail@example.com",
		Credentials: sdk.Credentials{
			Username: "clientusername",
			Secret:   secret,
		},
		Tags:      []string{"tag1", "tag2"},
		Metadata:  validMetadata,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
		Status:    users.EnabledStatus.String(),
	}
}

func TestMain(m *testing.M) {
	exitCode := m.Run()
	os.Exit(exitCode)
}
