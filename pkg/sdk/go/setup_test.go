package sdk_test

// import (
// 	"context"
// 	"fmt"
// 	"os"
// 	"regexp"
// 	"testing"
// 	"time"

// 	mfclients "github.com/mainflux/mainflux/pkg/clients"
// 	"github.com/mainflux/mainflux/pkg/errors"
// 	mfgroups "github.com/mainflux/mainflux/pkg/groups"
// 	sdk "github.com/mainflux/mainflux/pkg/sdk/go"
// 	"github.com/mainflux/mainflux/pkg/uuid"
// 	tpolicies "github.com/mainflux/mainflux/things/policies"
// 	"github.com/mainflux/mainflux/users/clients"
// 	umocks "github.com/mainflux/mainflux/users/clients/mocks"
// 	"github.com/mainflux/mainflux/users/hasher"
// 	upolicies "github.com/mainflux/mainflux/users/policies"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/mock"
// )

// const (
// 	invalidIdentity = "invalididentity"
// 	Identity        = "identity"
// 	secret          = "strongsecret"
// 	token           = "token"
// 	invalidToken    = "invalidtoken"
// 	contentType     = "application/senml+json"
// )

// var (
// 	idProvider    = uuid.New()
// 	phasher       = hasher.New()
// 	validMetadata = sdk.Metadata{"role": "client"}
// 	user          = sdk.User{
// 		Name:        "clientname",
// 		Tags:        []string{"tag1", "tag2"},
// 		Credentials: sdk.Credentials{Identity: "clientidentity", Secret: secret},
// 		Metadata:    validMetadata,
// 		Status:      mfclients.EnabledStatus.String(),
// 	}
// 	thing = sdk.Thing{
// 		Name:        "thingname",
// 		Tags:        []string{"tag1", "tag2"},
// 		Credentials: sdk.Credentials{Identity: "clientidentity", Secret: generateUUID(&testing.T{})},
// 		Metadata:    validMetadata,
// 		Status:      mfclients.EnabledStatus.String(),
// 	}
// 	description = "shortdescription"
// 	gName       = "groupname"

// 	limit  uint64 = 5
// 	offset uint64 = 0
// 	total  uint64 = 200

// 	authoritiesObj  = "authorities"
// 	subject         = generateUUID(&testing.T{})
// 	object          = generateUUID(&testing.T{})
// 	emailer         = umocks.NewEmailer()
// 	passRegex       = regexp.MustCompile("^.{8,}$")
// 	accessDuration  = time.Minute * 1
// 	refreshDuration = time.Minute * 10
// )

// func generateValidToken(t *testing.T, svc clients.Service, cRepo *umocks.Repository) string {
// 	client := mfclients.Client{
// 		ID:   generateUUID(t),
// 		Name: "validtoken",
// 		Credentials: mfclients.Credentials{
// 			Identity: "validtoken",
// 			Secret:   secret,
// 		},
// 		Role:   mfclients.AdminRole,
// 		Status: mfclients.EnabledStatus,
// 	}
// 	rclient := client
// 	rclient.Credentials.Secret, _ = phasher.Hash(client.Credentials.Secret)

// 	repoCall := cRepo.On("RetrieveByIdentity", context.Background(), mock.Anything).Return(rclient, nil)
// 	token, err := svc.IssueToken(context.Background(), client.Credentials.Identity, client.Credentials.Secret)
// 	assert.True(t, errors.Contains(err, nil), fmt.Sprintf("Create token expected nil got %s\n", err))
// 	repoCall.Unset()

// 	return token.AccessToken
// }

// func generateUUID(t *testing.T) string {
// 	ulid, err := idProvider.ID()
// 	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

// 	return ulid
// }

// func convertClientsPage(cp sdk.UsersPage) mfclients.ClientsPage {
// 	return mfclients.ClientsPage{
// 		Clients: convertClients(cp.Users),
// 	}
// }

// func convertThingsPage(cp sdk.ThingsPage) mfclients.ClientsPage {
// 	return mfclients.ClientsPage{
// 		Clients: convertThings(cp.Things),
// 	}
// }

// func convertClients(cs []sdk.User) []mfclients.Client {
// 	ccs := []mfclients.Client{}

// 	for _, c := range cs {
// 		ccs = append(ccs, convertClient(c))
// 	}

// 	return ccs
// }

// func convertThings(cs []sdk.Thing) []mfclients.Client {
// 	ccs := []mfclients.Client{}

// 	for _, c := range cs {
// 		ccs = append(ccs, convertThing(c))
// 	}

// 	return ccs
// }

// func convertGroups(cs []sdk.Group) []mfgroups.Group {
// 	cgs := []mfgroups.Group{}

// 	for _, c := range cs {
// 		cgs = append(cgs, convertGroup(c))
// 	}

// 	return cgs
// }

// func convertChannels(cs []sdk.Channel) []mfgroups.Group {
// 	cgs := []mfgroups.Group{}

// 	for _, c := range cs {
// 		cgs = append(cgs, convertChannel(c))
// 	}

// 	return cgs
// }

// func convertUserPolicies(cs []sdk.Policy) []upolicies.Policy {
// 	ccs := []upolicies.Policy{}

// 	for _, c := range cs {
// 		ccs = append(ccs, convertUserPolicy(c))
// 	}

// 	return ccs
// }

// func convertUserPolicy(sp sdk.Policy) upolicies.Policy {
// 	return upolicies.Policy{
// 		OwnerID:   sp.OwnerID,
// 		Subject:   sp.Subject,
// 		Object:    sp.Object,
// 		Actions:   sp.Actions,
// 		CreatedAt: sp.CreatedAt,
// 		UpdatedAt: sp.UpdatedAt,
// 	}
// }

// func convertThingPolicy(sp sdk.Policy) tpolicies.Policy {
// 	return tpolicies.Policy{
// 		OwnerID:   sp.OwnerID,
// 		Subject:   sp.Subject,
// 		Object:    sp.Object,
// 		Actions:   sp.Actions,
// 		CreatedAt: sp.CreatedAt,
// 		UpdatedAt: sp.UpdatedAt,
// 	}
// }

// func convertMembershipsPage(m sdk.MembershipsPage) mfgroups.Memberships {
// 	return mfgroups.Memberships{
// 		PageMeta: mfgroups.PageMeta{
// 			Limit:  m.Limit,
// 			Total:  m.Total,
// 			Offset: m.Offset,
// 		},
// 		Groups: convertMemberships(m.Memberships),
// 	}
// }

// func convertChannelsMembershipPage(m sdk.ChannelsPage) mfgroups.Memberships {
// 	return mfgroups.Memberships{
// 		PageMeta: mfgroups.PageMeta{
// 			Limit:  m.Limit,
// 			Total:  m.Total,
// 			Offset: m.Offset,
// 		},
// 		Groups: convertChannels(m.Channels),
// 	}
// }

// func convertClientPage(p sdk.PageMetadata) mfclients.Page {
// 	if p.Status == "" {
// 		p.Status = mfclients.EnabledStatus.String()
// 	}
// 	status, err := mfclients.ToStatus(p.Status)
// 	if err != nil {
// 		return mfclients.Page{}
// 	}

// 	return mfclients.Page{
// 		Status:   status,
// 		Total:    p.Total,
// 		Offset:   p.Offset,
// 		Limit:    p.Limit,
// 		Name:     p.Name,
// 		Action:   p.Action,
// 		Tag:      p.Tag,
// 		Metadata: mfclients.Metadata(p.Metadata),
// 	}
// }

// func convertMemberships(gs []sdk.Group) []mfgroups.Group {
// 	cg := []mfgroups.Group{}
// 	for _, g := range gs {
// 		cg = append(cg, convertGroup(g))
// 	}

// 	return cg
// }

// func convertGroup(g sdk.Group) mfgroups.Group {
// 	if g.Status == "" {
// 		g.Status = mfclients.EnabledStatus.String()
// 	}
// 	status, err := mfclients.ToStatus(g.Status)
// 	if err != nil {
// 		return mfgroups.Group{}
// 	}

// 	return mfgroups.Group{
// 		ID:          g.ID,
// 		Owner:       g.OwnerID,
// 		Parent:      g.ParentID,
// 		Name:        g.Name,
// 		Description: g.Description,
// 		Metadata:    mfclients.Metadata(g.Metadata),
// 		Level:       g.Level,
// 		Path:        g.Path,
// 		Children:    convertChildren(g.Children),
// 		CreatedAt:   g.CreatedAt,
// 		UpdatedAt:   g.UpdatedAt,
// 		Status:      status,
// 	}
// }

// func convertChildren(gs []*sdk.Group) []*mfgroups.Group {
// 	cg := []*mfgroups.Group{}

// 	if len(gs) == 0 {
// 		return cg
// 	}

// 	for _, g := range gs {
// 		insert := convertGroup(*g)
// 		cg = append(cg, &insert)
// 	}

// 	return cg
// }

// func convertClient(c sdk.User) mfclients.Client {
// 	if c.Status == "" {
// 		c.Status = mfclients.EnabledStatus.String()
// 	}
// 	status, err := mfclients.ToStatus(c.Status)
// 	if err != nil {
// 		return mfclients.Client{}
// 	}

// 	return mfclients.Client{
// 		ID:          c.ID,
// 		Name:        c.Name,
// 		Tags:        c.Tags,
// 		Owner:       c.Owner,
// 		Credentials: mfclients.Credentials(c.Credentials),
// 		Metadata:    mfclients.Metadata(c.Metadata),
// 		CreatedAt:   c.CreatedAt,
// 		UpdatedAt:   c.UpdatedAt,
// 		Status:      status,
// 	}
// }

// func convertThing(c sdk.Thing) mfclients.Client {
// 	if c.Status == "" {
// 		c.Status = mfclients.EnabledStatus.String()
// 	}
// 	status, err := mfclients.ToStatus(c.Status)
// 	if err != nil {
// 		return mfclients.Client{}
// 	}
// 	return mfclients.Client{
// 		ID:          c.ID,
// 		Name:        c.Name,
// 		Tags:        c.Tags,
// 		Owner:       c.Owner,
// 		Credentials: mfclients.Credentials(c.Credentials),
// 		Metadata:    mfclients.Metadata(c.Metadata),
// 		CreatedAt:   c.CreatedAt,
// 		UpdatedAt:   c.UpdatedAt,
// 		Status:      status,
// 	}
// }

// func convertChannel(g sdk.Channel) mfgroups.Group {
// 	if g.Status == "" {
// 		g.Status = mfclients.EnabledStatus.String()
// 	}
// 	status, err := mfclients.ToStatus(g.Status)
// 	if err != nil {
// 		return mfgroups.Group{}
// 	}
// 	return mfgroups.Group{
// 		ID:          g.ID,
// 		Owner:       g.OwnerID,
// 		Parent:      g.ParentID,
// 		Name:        g.Name,
// 		Description: g.Description,
// 		Metadata:    mfclients.Metadata(g.Metadata),
// 		Level:       g.Level,
// 		Path:        g.Path,
// 		CreatedAt:   g.CreatedAt,
// 		UpdatedAt:   g.UpdatedAt,
// 		Status:      status,
// 	}
// }

// func convertUserPolicyPage(pp sdk.PolicyPage) upolicies.PolicyPage {
// 	return upolicies.PolicyPage{
// 		Page: upolicies.Page{
// 			Limit:  pp.Limit,
// 			Total:  pp.Total,
// 			Offset: pp.Offset,
// 		},
// 		Policies: convertUserPolicies(pp.Policies),
// 	}
// }

// func TestMain(m *testing.M) {
// 	exitCode := m.Run()
// 	os.Exit(exitCode)
// }
