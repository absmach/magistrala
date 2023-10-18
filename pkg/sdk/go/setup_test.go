// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	mfclients "github.com/mainflux/mainflux/pkg/clients"
	mfgroups "github.com/mainflux/mainflux/pkg/groups"
	sdk "github.com/mainflux/mainflux/pkg/sdk/go"
	"github.com/mainflux/mainflux/pkg/uuid"
	"github.com/mainflux/mainflux/users/hasher"
	umocks "github.com/mainflux/mainflux/users/mocks"
	"github.com/stretchr/testify/assert"
)

const (
	invalidIdentity = "invalididentity"
	Identity        = "identity"
	secret          = "strongsecret"
	token           = "token"
	invalidToken    = "invalidtoken"
	contentType     = "application/senml+json"
)

var (
	idProvider    = uuid.New()
	phasher       = hasher.New()
	validMetadata = sdk.Metadata{"role": "client"}
	user          = sdk.User{
		Name:        "clientname",
		Tags:        []string{"tag1", "tag2"},
		Credentials: sdk.Credentials{Identity: "clientidentity", Secret: secret},
		Metadata:    validMetadata,
		Status:      mfclients.EnabledStatus.String(),
	}
	thing = sdk.Thing{
		Name:        "thingname",
		Tags:        []string{"tag1", "tag2"},
		Credentials: sdk.Credentials{Identity: "clientidentity", Secret: generateUUID(&testing.T{})},
		Metadata:    validMetadata,
		Status:      mfclients.EnabledStatus.String(),
	}
	description = "shortdescription"
	gName       = "groupname"

	limit  uint64 = 5
	offset uint64 = 0
	total  uint64 = 200

	subject   = generateUUID(&testing.T{})
	object    = generateUUID(&testing.T{})
	emailer   = umocks.NewEmailer()
	passRegex = regexp.MustCompile("^.{8,}$")
)

func generateUUID(t *testing.T) string {
	ulid, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	return ulid
}

func convertClientsPage(cp sdk.UsersPage) mfclients.ClientsPage {
	return mfclients.ClientsPage{
		Clients: convertClients(cp.Users),
	}
}

func convertThingsPage(cp sdk.ThingsPage) mfclients.ClientsPage {
	return mfclients.ClientsPage{
		Clients: convertThings(cp.Things),
	}
}

func convertClients(cs []sdk.User) []mfclients.Client {
	ccs := []mfclients.Client{}

	for _, c := range cs {
		ccs = append(ccs, convertClient(c))
	}

	return ccs
}

func convertThings(cs []sdk.Thing) []mfclients.Client {
	ccs := []mfclients.Client{}

	for _, c := range cs {
		ccs = append(ccs, convertThing(c))
	}

	return ccs
}

func convertGroups(cs []sdk.Group) []mfgroups.Group {
	cgs := []mfgroups.Group{}

	for _, c := range cs {
		cgs = append(cgs, convertGroup(c))
	}

	return cgs
}

func convertChannels(cs []sdk.Channel) []mfgroups.Group {
	cgs := []mfgroups.Group{}

	for _, c := range cs {
		cgs = append(cgs, convertChannel(c))
	}

	return cgs
}

func convertClientPage(p sdk.PageMetadata) mfclients.Page {
	if p.Status == "" {
		p.Status = mfclients.EnabledStatus.String()
	}
	status, err := mfclients.ToStatus(p.Status)
	if err != nil {
		return mfclients.Page{}
	}

	return mfclients.Page{
		Status:   status,
		Total:    p.Total,
		Offset:   p.Offset,
		Limit:    p.Limit,
		Name:     p.Name,
		Tag:      p.Tag,
		Metadata: mfclients.Metadata(p.Metadata),
	}
}

func convertGroup(g sdk.Group) mfgroups.Group {
	if g.Status == "" {
		g.Status = mfclients.EnabledStatus.String()
	}
	status, err := mfclients.ToStatus(g.Status)
	if err != nil {
		return mfgroups.Group{}
	}

	return mfgroups.Group{
		ID:          g.ID,
		Owner:       g.OwnerID,
		Parent:      g.ParentID,
		Name:        g.Name,
		Description: g.Description,
		Metadata:    mfclients.Metadata(g.Metadata),
		Level:       g.Level,
		Path:        g.Path,
		Children:    convertChildren(g.Children),
		CreatedAt:   g.CreatedAt,
		UpdatedAt:   g.UpdatedAt,
		Status:      status,
	}
}

func convertChildren(gs []*sdk.Group) []*mfgroups.Group {
	cg := []*mfgroups.Group{}

	if len(gs) == 0 {
		return cg
	}

	for _, g := range gs {
		insert := convertGroup(*g)
		cg = append(cg, &insert)
	}

	return cg
}

func convertClient(c sdk.User) mfclients.Client {
	if c.Status == "" {
		c.Status = mfclients.EnabledStatus.String()
	}
	status, err := mfclients.ToStatus(c.Status)
	if err != nil {
		return mfclients.Client{}
	}

	return mfclients.Client{
		ID:          c.ID,
		Name:        c.Name,
		Tags:        c.Tags,
		Owner:       c.Owner,
		Credentials: mfclients.Credentials(c.Credentials),
		Metadata:    mfclients.Metadata(c.Metadata),
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
		Status:      status,
	}
}

func convertThing(c sdk.Thing) mfclients.Client {
	if c.Status == "" {
		c.Status = mfclients.EnabledStatus.String()
	}
	status, err := mfclients.ToStatus(c.Status)
	if err != nil {
		return mfclients.Client{}
	}
	return mfclients.Client{
		ID:          c.ID,
		Name:        c.Name,
		Tags:        c.Tags,
		Owner:       c.Owner,
		Credentials: mfclients.Credentials(c.Credentials),
		Metadata:    mfclients.Metadata(c.Metadata),
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
		Status:      status,
	}
}

func convertChannel(g sdk.Channel) mfgroups.Group {
	if g.Status == "" {
		g.Status = mfclients.EnabledStatus.String()
	}
	status, err := mfclients.ToStatus(g.Status)
	if err != nil {
		return mfgroups.Group{}
	}
	return mfgroups.Group{
		ID:          g.ID,
		Owner:       g.OwnerID,
		Parent:      g.ParentID,
		Name:        g.Name,
		Description: g.Description,
		Metadata:    mfclients.Metadata(g.Metadata),
		Level:       g.Level,
		Path:        g.Path,
		CreatedAt:   g.CreatedAt,
		UpdatedAt:   g.UpdatedAt,
		Status:      status,
	}
}

func TestMain(m *testing.M) {
	exitCode := m.Run()
	os.Exit(exitCode)
}
