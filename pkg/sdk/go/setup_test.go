// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

	mgchannels "github.com/absmach/magistrala/channels"
	"github.com/absmach/magistrala/clients"
	mggroups "github.com/absmach/magistrala/groups"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/invitations"
	"github.com/absmach/magistrala/journal"
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

func convertClients(cs ...sdk.Client) []clients.Client {
	ccs := []clients.Client{}

	for _, c := range cs {
		ccs = append(ccs, convertClient(c))
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

func convertChannels(cs []sdk.Channel) []mgchannels.Channel {
	chs := []mgchannels.Channel{}

	for _, c := range cs {
		chs = append(chs, convertChannel(c))
	}

	return chs
}

func convertGroup(g sdk.Group) mggroups.Group {
	if g.Status == "" {
		g.Status = mggroups.EnabledStatus.String()
	}
	status, err := mggroups.ToStatus(g.Status)
	if err != nil {
		return mggroups.Group{}
	}

	return mggroups.Group{
		ID:          g.ID,
		Domain:      g.DomainID,
		Parent:      g.ParentID,
		Name:        g.Name,
		Description: g.Description,
		Metadata:    mggroups.Metadata(g.Metadata),
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

func convertClient(c sdk.Client) clients.Client {
	if c.Status == "" {
		c.Status = clients.EnabledStatus.String()
	}
	status, err := clients.ToStatus(c.Status)
	if err != nil {
		return clients.Client{}
	}
	return clients.Client{
		ID:          c.ID,
		Name:        c.Name,
		Tags:        c.Tags,
		Domain:      c.DomainID,
		Credentials: clients.Credentials(c.Credentials),
		Metadata:    clients.Metadata(c.Metadata),
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
		Status:      status,
	}
}

func convertChannel(g sdk.Channel) mgchannels.Channel {
	if g.Status == "" {
		g.Status = clients.EnabledStatus.String()
	}
	status, err := clients.ToStatus(g.Status)
	if err != nil {
		return mgchannels.Channel{}
	}
	return mgchannels.Channel{
		ID:          g.ID,
		Domain:      g.DomainID,
		ParentGroup: g.ParentID,
		Name:        g.Name,
		Metadata:    clients.Metadata(g.Metadata),
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
		FirstName: "userfirstname",
		LastName:  "userlastname",
		Email:     "useremail@example.com",
		Credentials: sdk.Credentials{
			Username: "username",
			Secret:   secret,
		},
		Tags:      []string{"tag1", "tag2"},
		Metadata:  validMetadata,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
		Status:    users.EnabledStatus.String(),
		Role:      users.UserRole.String(),
	}
}

func TestMain(m *testing.M) {
	exitCode := m.Run()
	os.Exit(exitCode)
}
