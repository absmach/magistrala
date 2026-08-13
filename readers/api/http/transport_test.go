// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	grpcChannelsV1 "github.com/absmach/magistrala/api/grpc/channels/v1"
	grpcClientsV1 "github.com/absmach/magistrala/api/grpc/clients/v1"
	smqauthn "github.com/absmach/magistrala/pkg/authn"
	authnmocks "github.com/absmach/magistrala/pkg/authn/mocks"
	"github.com/absmach/magistrala/pkg/policies"
	policymocks "github.com/absmach/magistrala/pkg/policies/mocks"
	"github.com/absmach/magistrala/readers"
	readermocks "github.com/absmach/magistrala/readers/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

const (
	e2eDomain  = "domain-1"
	e2eChanID  = "chan-1"
	e2eUserID  = "user-1"
	e2eSubject = e2eDomain + "_" + e2eUserID
)

// fakeClientsClient satisfies grpcClientsV1.ClientsServiceClient without implementing every
// method: only Authenticate is reachable through the request paths exercised below.
type fakeClientsClient struct {
	grpcClientsV1.ClientsServiceClient
}

// fakeChannelsClient stubs the channel-level subscribe check that this change does not alter.
type fakeChannelsClient struct {
	grpcChannelsV1.ChannelsServiceClient
	authorized bool
}

func (f fakeChannelsClient) Authorize(context.Context, *grpcChannelsV1.AuthzReq, ...grpc.CallOption) (*grpcChannelsV1.AuthzRes, error) {
	return &grpcChannelsV1.AuthzRes{Authorized: f.authorized}, nil
}

type testDeps struct {
	repo      *readermocks.MessageRepository
	authn     *authnmocks.Authentication
	evaluator *policymocks.Evaluator
	lister    *policymocks.Service
	endpoint  func(ctx context.Context, req any) (any, error)
}

func newTestDeps(t *testing.T) *testDeps {
	t.Helper()
	repo := readermocks.NewMessageRepository(t)
	authn := authnmocks.NewAuthentication(t)
	evaluator := policymocks.NewEvaluator(t)
	lister := policymocks.NewService(t)

	authz := newPublisherAuthorizer(evaluator, lister)
	ep := listMessagesEndpoint(repo, authn, fakeClientsClient{}, fakeChannelsClient{authorized: true}, authz)

	return &testDeps{repo: repo, authn: authn, evaluator: evaluator, lister: lister, endpoint: ep}
}

func TestDecodeListReadsPluralPublisherFilters(t *testing.T) {
	r := httptest.NewRequest("GET", "/?publishers=pub-a&publishers=pub-b&publisher=pub-c", nil)
	r.Header.Set("Authorization", "Bearer token")

	decoded, err := decodeList(context.Background(), r)
	require.NoError(t, err)

	req, ok := decoded.(listMessagesReq)
	require.True(t, ok)
	assert.Equal(t, "pub-c", req.pageMeta.Publisher)
	assert.Equal(t, []string{"pub-a", "pub-b"}, req.pageMeta.Publishers)
}

func TestDecodeListReadsDeviceIDFilter(t *testing.T) {
	r := httptest.NewRequest("GET", "/?device_ids=Meter.A-01%3AX&device_ids=meter%2Fb%2C02&device_ids=", nil)
	r.Header.Set("Authorization", "Bearer token")

	decoded, err := decodeList(context.Background(), r)
	require.NoError(t, err)

	req, ok := decoded.(listMessagesReq)
	require.True(t, ok)
	// Serials carry no format constraint (MG-09), so values are taken verbatim:
	// no splitting on a separator that a real serial may contain, and empty
	// entries dropped rather than turned into a filter for "".
	assert.Equal(t, []string{"Meter.A-01:X", "meter/b,02"}, req.pageMeta.DeviceIDs)
}

func TestDecodeListWithoutDeviceIDFilterLeavesItUnset(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer token")

	decoded, err := decodeList(context.Background(), r)
	require.NoError(t, err)

	req, ok := decoded.(listMessagesReq)
	require.True(t, ok)
	assert.Nil(t, req.pageMeta.DeviceIDs)
	assert.Nil(t, req.pageMeta.Publishers)
}

// device_ids is a convenience filter here, not a boundary — MG-08 part B makes it
// one. Until then it must reach the repository exactly as asked for.
func TestListMessagesPassesDeviceIDFilterThrough(t *testing.T) {
	d := newTestDeps(t)
	expectUser(d.authn)
	d.evaluator.EXPECT().CheckPolicy(mock.Anything, mock.Anything).Return(errors.New("not admin"))
	d.lister.EXPECT().ListAllObjects(mock.Anything, mock.Anything).Return(policies.PolicyPage{Policies: []string{"pub-a"}}, nil)
	d.repo.EXPECT().ReadAll(e2eChanID, mock.MatchedBy(func(pm readers.PageMetadata) bool {
		return assert.ObjectsAreEqual([]string{"meter-1", "meter-3"}, pm.DeviceIDs)
	})).Return(readers.MessagesPage{}, nil)

	req := tokenReq("", []string{"pub-a"})
	req.pageMeta.DeviceIDs = []string{"meter-1", "meter-3"}

	_, err := d.endpoint(context.Background(), req)
	require.NoError(t, err)
}

func tokenReq(publisher string, publishers []string) listMessagesReq {
	return listMessagesReq{
		chanID: e2eChanID,
		token:  "token",
		domain: e2eDomain,
		pageMeta: readers.PageMetadata{
			Limit:      10,
			Publisher:  publisher,
			Publishers: publishers,
		},
	}
}

func expectUser(authn *authnmocks.Authentication) {
	authn.EXPECT().Authenticate(mock.Anything, "token").Return(smqauthn.Session{
		UserID:   e2eUserID,
		DomainID: e2eDomain,
		Role:     smqauthn.UserRole,
		Verified: true,
	}, nil)
}

func expectSuperAdmin(authn *authnmocks.Authentication) {
	authn.EXPECT().Authenticate(mock.Anything, "token").Return(smqauthn.Session{
		UserID:   e2eUserID,
		DomainID: e2eDomain,
		Role:     smqauthn.SuperAdminRole,
		Verified: true,
	}, nil)
}

func expectGrants(evaluator *policymocks.Evaluator, lister *policymocks.Service, granted []string) {
	evaluator.EXPECT().CheckPolicy(mock.Anything, mock.Anything).Return(errors.New("not admin"))
	lister.EXPECT().ListAllObjects(mock.Anything, mock.MatchedBy(func(pr policies.Policy) bool {
		return pr.Subject == e2eSubject && pr.Domain == e2eDomain && pr.ObjectType == policies.ClientType
	})).Return(policies.PolicyPage{Policies: granted}, nil)
}

func expectAdmin(evaluator *policymocks.Evaluator) {
	evaluator.EXPECT().CheckPolicy(mock.Anything, mock.MatchedBy(func(pr policies.Policy) bool {
		return pr.ObjectType == policies.DomainType && pr.Permission == policies.AdminPermission
	})).Return(nil)
}

// TestListMessagesDeniesUnauthorizedPublisher guards against a caller who can read the channel
// getting another publisher's messages just by naming it in the query. Against a transport that
// only applies the channel-level check, this fails because the repository is invoked with the
// caller-supplied filter unchecked.
func TestListMessagesDeniesUnauthorizedPublisher(t *testing.T) {
	deps := newTestDeps(t)
	expectUser(deps.authn)
	expectGrants(deps.evaluator, deps.lister, []string{"pub-granted"})
	// ReadAll must never be called: no expectation is registered for it, so the mock
	// framework fails the test if the endpoint tries to reach the repository.

	res, err := deps.endpoint(context.Background(), tokenReq("pub-not-granted", nil))
	require.NoError(t, err)

	page, ok := res.(pageRes)
	require.True(t, ok)
	assert.Empty(t, page.Messages)
	assert.Zero(t, page.Total)
}

func TestListMessagesReturnsExactlyRequestedGrantedPublishers(t *testing.T) {
	deps := newTestDeps(t)
	expectUser(deps.authn)
	expectGrants(deps.evaluator, deps.lister, []string{"pub-a", "pub-b", "pub-c"})

	deps.repo.EXPECT().ReadAll(e2eChanID, mock.MatchedBy(func(pm readers.PageMetadata) bool {
		return len(pm.Publishers) == 1 && pm.Publishers[0] == "pub-a" && pm.Publisher == ""
	})).Return(readers.MessagesPage{Total: 1, Messages: []readers.Message{"msg-from-pub-a"}}, nil)

	res, err := deps.endpoint(context.Background(), tokenReq("pub-a", nil))
	require.NoError(t, err)

	page, ok := res.(pageRes)
	require.True(t, ok)
	assert.Equal(t, uint64(1), page.Total)
	assert.Equal(t, []readers.Message{"msg-from-pub-a"}, page.Messages)
}

// TestListMessagesSuperAdminBypassesPublisherGrants checks that a super-admin token keeps
// unrestricted access after the channel-level authorization succeeds.
func TestListMessagesSuperAdminBypassesPublisherGrants(t *testing.T) {
	deps := newTestDeps(t)
	expectSuperAdmin(deps.authn)
	// No evaluator or lister expectations: super-admin status is carried from authentication.

	deps.repo.EXPECT().ReadAll(e2eChanID, mock.MatchedBy(func(pm readers.PageMetadata) bool {
		return pm.Publisher == "pub-anything" && len(pm.Publishers) == 0
	})).Return(readers.MessagesPage{Total: 3, Messages: []readers.Message{"m1", "m2", "m3"}}, nil)

	res, err := deps.endpoint(context.Background(), tokenReq("pub-anything", nil))
	require.NoError(t, err)

	page, ok := res.(pageRes)
	require.True(t, ok)
	assert.Equal(t, uint64(3), page.Total)
}

// TestListMessagesAdminSeesEverythingRegardlessOfFilter checks that the admin bypass is
// capability-driven (evaluator.CheckPolicy allowed), not inferred from the session role, which
// here is the ordinary UserRole.
func TestListMessagesAdminSeesEverythingRegardlessOfFilter(t *testing.T) {
	deps := newTestDeps(t)
	expectUser(deps.authn)
	expectAdmin(deps.evaluator)
	// No ListAllObjects expectation: the admin path must not depend on it.

	deps.repo.EXPECT().ReadAll(e2eChanID, mock.MatchedBy(func(pm readers.PageMetadata) bool {
		return len(pm.Publishers) == 1 && pm.Publishers[0] == "pub-anything" && pm.Publisher == ""
	})).Return(readers.MessagesPage{Total: 3, Messages: []readers.Message{"m1", "m2", "m3"}}, nil)

	res, err := deps.endpoint(context.Background(), tokenReq("pub-anything", nil))
	require.NoError(t, err)

	page, ok := res.(pageRes)
	require.True(t, ok)
	assert.Equal(t, uint64(3), page.Total)
}

// TestListMessagesNoGrantsAtAllIsEmptyNotEverything guards the empty-set-inversion failure mode:
// a caller with zero publisher grants must get zero rows, never the whole channel.
func TestListMessagesNoGrantsAtAllIsEmptyNotEverything(t *testing.T) {
	deps := newTestDeps(t)
	expectUser(deps.authn)
	expectGrants(deps.evaluator, deps.lister, nil)
	// No publisher filter requested and nothing granted: ReadAll must not be called.

	res, err := deps.endpoint(context.Background(), tokenReq("", nil))
	require.NoError(t, err)

	page, ok := res.(pageRes)
	require.True(t, ok)
	assert.Empty(t, page.Messages)
	assert.Zero(t, page.Total)
}

// TestListMessagesNonAdminWithLargeGrantSetIsStillBounded pairs a non-admin holding many grants
// against an admin holding none, to show the bypass tracks a capability, not grant-list size.
func TestListMessagesNonAdminWithLargeGrantSetIsStillBounded(t *testing.T) {
	granted := make([]string, 200)
	for i := range granted {
		granted[i] = "pub-" + string(rune('a'+i%26)) + string(rune('0'+i/26))
	}

	deps := newTestDeps(t)
	expectUser(deps.authn)
	expectGrants(deps.evaluator, deps.lister, granted)
	// No expectation for ReadAll: "pub-outside-grant-set" is not in the 200 granted IDs.

	res, err := deps.endpoint(context.Background(), tokenReq("pub-outside-grant-set", nil))
	require.NoError(t, err)

	page, ok := res.(pageRes)
	require.True(t, ok)
	assert.Empty(t, page.Messages)
}
