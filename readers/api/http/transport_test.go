// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	grpcChannelsV1 "github.com/absmach/magistrala/api/grpc/channels/v1"
	grpcDevicesV1 "github.com/absmach/magistrala/api/grpc/devices/v1"
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
	e2eWorkspace = "workspace-1"
	e2eChanID    = "chan-1"
	e2eUserID    = "user-1"
	e2eSubject   = e2eWorkspace + "_" + e2eUserID
)

// fakeClientsClient satisfies grpcDevicesV1.DevicesServiceClient without implementing every
// method: only Authenticate is reachable through the request paths exercised below.
type fakeClientsClient struct {
	grpcDevicesV1.DevicesServiceClient
}

func (fakeClientsClient) Authenticate(context.Context, *grpcDevicesV1.AuthnReq, ...grpc.CallOption) (*grpcDevicesV1.AuthnRes, error) {
	return &grpcDevicesV1.AuthnRes{Authenticated: true, Id: "client-1"}, nil
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
	resolver  *fakeResolver
	endpoint  func(ctx context.Context, req any) (any, error)
}

func newTestDeps(t *testing.T) *testDeps {
	t.Helper()
	repo := readermocks.NewMessageRepository(t)
	authn := authnmocks.NewAuthentication(t)
	evaluator := policymocks.NewEvaluator(t)
	lister := policymocks.NewService(t)
	resolver := allMeters()

	authz := newReadAuthorizer(evaluator, lister, resolver)
	ep := listMessagesEndpoint(repo, authn, fakeClientsClient{}, fakeChannelsClient{authorized: true}, authz)

	return &testDeps{repo: repo, authn: authn, evaluator: evaluator, lister: lister, resolver: resolver, endpoint: ep}
}

func TestDecodeListReadsPluralFilters(t *testing.T) {
	r := httptest.NewRequest("GET", "/?publishers=pub-a&publishers=pub-b&publisher=pub-c&device_ids=meter-a&device_ids=meter-b", nil)
	r.Header.Set("Authorization", "Bearer token")

	decoded, err := decodeList(context.Background(), r)
	require.NoError(t, err)

	req, ok := decoded.(listMessagesReq)
	require.True(t, ok)
	assert.Equal(t, "pub-c", req.pageMeta.Publisher)
	assert.Equal(t, []string{"pub-a", "pub-b"}, req.pageMeta.Publishers)
	assert.Equal(t, []string{"meter-a", "meter-b"}, req.pageMeta.DeviceIDs)
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

func tokenReq(publisher string, publishers []string) listMessagesReq {
	return listMessagesReq{
		chanID:    e2eChanID,
		token:     "token",
		workspace: e2eWorkspace,
		pageMeta: readers.PageMetadata{
			Limit:      10,
			Publisher:  publisher,
			Publishers: publishers,
		},
	}
}

func deviceReq(deviceIDs []string) listMessagesReq {
	req := tokenReq("", nil)
	req.pageMeta.DeviceIDs = deviceIDs
	return req
}

func expectUser(authn *authnmocks.Authentication) {
	authn.EXPECT().Authenticate(mock.Anything, "token").Return(smqauthn.Session{
		UserID:      e2eUserID,
		WorkspaceID: e2eWorkspace,
		Role:        smqauthn.UserRole,
		Verified:    true,
	}, nil)
}

func expectSuperAdmin(authn *authnmocks.Authentication) {
	authn.EXPECT().Authenticate(mock.Anything, "token").Return(smqauthn.Session{
		UserID:      e2eUserID,
		WorkspaceID: e2eWorkspace,
		Role:        smqauthn.SuperAdminRole,
		Verified:    true,
	}, nil)
}

func expectGrants(evaluator *policymocks.Evaluator, lister *policymocks.Service, granted []string) {
	evaluator.EXPECT().CheckPolicy(mock.Anything, mock.Anything).Return(errors.New("not admin"))
	lister.EXPECT().ListAllObjects(mock.Anything, mock.MatchedBy(func(pr policies.Policy) bool {
		return pr.Subject == e2eSubject && pr.Workspace == e2eWorkspace && pr.ObjectType == policies.ClientType
	})).Return(policies.PolicyPage{Policies: granted}, nil)
}

func expectAdmin(evaluator *policymocks.Evaluator) {
	evaluator.EXPECT().CheckPolicy(mock.Anything, mock.MatchedBy(func(pr policies.Policy) bool {
		return pr.ObjectType == policies.WorkspaceType && pr.Permission == policies.AdminPermission
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

// scopeMatches asserts the query is bounded to exactly the given devices, under
// both identities they can be known by on a row.
func scopeMatches(publisherIDs, deviceIDs []string) func(readers.PageMetadata) bool {
	return func(pm readers.PageMetadata) bool {
		if pm.DeviceScope == nil {
			return false
		}
		return assert.ObjectsAreEqual(publisherIDs, pm.DeviceScope.PublisherIDs) &&
			assert.ObjectsAreEqual(deviceIDs, pm.DeviceScope.DeviceIDs)
	}
}

// Criterion 1: a user granted read on meters 1 and 3, querying a channel
// carrying all three, receives data for 1 and 3 only.
//
// This is the part-B regression: before this change the endpoint bounded the
// query by publisher alone, so meter 1's and meter 3's readings relayed by a
// gateway — which carry the gateway's publisher id and the meter's serial —
// were unreachable, while every other device's gateway-relayed row on the
// channel was returned unfiltered.
func TestListMessagesBoundsQueryToGrantedDevices(t *testing.T) {
	deps := newTestDeps(t)
	expectUser(deps.authn)
	expectGrants(deps.evaluator, deps.lister, []string{meter1UUID, meter3UUID})

	deps.repo.EXPECT().ReadAll(e2eChanID, mock.MatchedBy(
		scopeMatches([]string{meter1UUID, meter3UUID}, []string{meter1Serial, meter3Serial}),
	)).Return(readers.MessagesPage{Total: 2, Messages: []readers.Message{"m1", "m3"}}, nil)

	res, err := deps.endpoint(context.Background(), tokenReq("", nil))
	require.NoError(t, err)

	page, ok := res.(pageRes)
	require.True(t, ok)
	assert.Equal(t, uint64(2), page.Total)
}

// Criterion 2: the same user requesting meter 2 receives empty, not meter 2's data.
func TestListMessagesDeniesUnauthorizedDevice(t *testing.T) {
	deps := newTestDeps(t)
	expectUser(deps.authn)
	expectGrants(deps.evaluator, deps.lister, []string{meter1UUID, meter3UUID})
	// No ReadAll expectation: reaching the repository at all would be the bug.

	res, err := deps.endpoint(context.Background(), deviceReq([]string{meter2Serial}))
	require.NoError(t, err)

	page, ok := res.(pageRes)
	require.True(t, ok)
	assert.Empty(t, page.Messages)
	assert.Zero(t, page.Total)
}

// Criteria 3 and 7: requesting meters 1 and 2 returns meter 1 only, and nothing
// in the response distinguishes "meter 2 does not exist" from "you may not read it".
func TestListMessagesDropsUnauthorizedDeviceSilently(t *testing.T) {
	deps := newTestDeps(t)
	expectUser(deps.authn)
	expectGrants(deps.evaluator, deps.lister, []string{meter1UUID, meter3UUID})

	deps.repo.EXPECT().ReadAll(e2eChanID, mock.MatchedBy(func(pm readers.PageMetadata) bool {
		return assert.ObjectsAreEqual([]string{meter1Serial}, pm.DeviceIDs)
	})).Return(readers.MessagesPage{Total: 1, Messages: []readers.Message{"m1"}}, nil)

	res, err := deps.endpoint(context.Background(), deviceReq([]string{meter1Serial, meter2Serial}))
	require.NoError(t, err)

	page, ok := res.(pageRes)
	require.True(t, ok)
	assert.Equal(t, uint64(1), page.Total)

	// A non-existent serial is refused exactly like an existing but unauthorized
	// one: empty, no error, no hint either way.
	unknown := newTestDeps(t)
	expectUser(unknown.authn)
	expectGrants(unknown.evaluator, unknown.lister, []string{meter1UUID, meter3UUID})

	res, err = unknown.endpoint(context.Background(), deviceReq([]string{"no-such-serial"}))
	require.NoError(t, err)
	unknownPage, ok := res.(pageRes)
	require.True(t, ok)
	assert.Equal(t, page.Messages[:0], unknownPage.Messages)
	assert.Zero(t, unknownPage.Total)
}

// Criterion 4: a user with no device grants receives empty, not everything.
func TestListMessagesNoDeviceGrantsIsEmptyNotEverything(t *testing.T) {
	deps := newTestDeps(t)
	expectUser(deps.authn)
	expectGrants(deps.evaluator, deps.lister, nil)
	// No ReadAll expectation: an unfiltered read here is the full-disclosure bug.

	res, err := deps.endpoint(context.Background(), deviceReq(nil))
	require.NoError(t, err)

	page, ok := res.(pageRes)
	require.True(t, ok)
	assert.Empty(t, page.Messages)
	assert.Zero(t, page.Total)
}

// Criteria 5 and 11: a workspace admin reads all three meters, and the query is not
// bounded at all — which is also what keeps orphan rows, whose device_id matches
// no entity, readable through channel-level access.
func TestListMessagesAdminQueryIsUnbounded(t *testing.T) {
	deps := newTestDeps(t)
	expectUser(deps.authn)
	expectAdmin(deps.evaluator)

	deps.repo.EXPECT().ReadAll(e2eChanID, mock.MatchedBy(func(pm readers.PageMetadata) bool {
		return pm.DeviceScope == nil
	})).Return(readers.MessagesPage{Total: 4, Messages: []readers.Message{"m1", "m2", "m3", "orphan"}}, nil)

	res, err := deps.endpoint(context.Background(), deviceReq(nil))
	require.NoError(t, err)

	page, ok := res.(pageRes)
	require.True(t, ok)
	assert.Equal(t, uint64(4), page.Total)
}

// Criterion 11 from the customer's side: an orphan serial is not in any grant,
// so a device-scoped caller cannot name it either.
func TestListMessagesOrphanDeviceIsInvisibleToCustomer(t *testing.T) {
	deps := newTestDeps(t)
	expectUser(deps.authn)
	expectGrants(deps.evaluator, deps.lister, []string{meter1UUID})
	// No ReadAll expectation.

	res, err := deps.endpoint(context.Background(), deviceReq([]string{"orphan-serial-with-no-entity"}))
	require.NoError(t, err)

	page, ok := res.(pageRes)
	require.True(t, ok)
	assert.Empty(t, page.Messages)
}

// Criterion 9: the channel-level check still rejects a caller without subscribe,
// before any device scoping happens.
func TestListMessagesChannelCheckStillRejects(t *testing.T) {
	repo := readermocks.NewMessageRepository(t)
	authn := authnmocks.NewAuthentication(t)
	evaluator := policymocks.NewEvaluator(t)
	lister := policymocks.NewService(t)
	expectUser(authn)

	authz := newReadAuthorizer(evaluator, lister, allMeters())
	ep := listMessagesEndpoint(repo, authn, fakeClientsClient{}, fakeChannelsClient{authorized: false}, authz)

	_, err := ep(context.Background(), tokenReq("", nil))
	assert.Error(t, err, "device scoping narrows within the channel check, it does not replace it")
}

// Criterion 10 end to end: a customer granted one device reads that device's
// data, including the rows a gateway relayed for it. Those rows only carry the
// serial, so this is the assertion that fails if the UUID → external_id
// translation is skipped — the symptom otherwise looks like an authorization
// failure rather than a missing conversion.
func TestListMessagesGrantedCustomerReachesGatewayRelayedRows(t *testing.T) {
	deps := newTestDeps(t)
	expectUser(deps.authn)
	expectGrants(deps.evaluator, deps.lister, []string{meter1UUID})

	deps.repo.EXPECT().ReadAll(e2eChanID, mock.MatchedBy(func(pm readers.PageMetadata) bool {
		return pm.DeviceScope != nil && assert.ObjectsAreEqual([]string{meter1Serial}, pm.DeviceScope.DeviceIDs)
	})).Return(readers.MessagesPage{Total: 1, Messages: []readers.Message{"relayed-by-gateway"}}, nil)

	res, err := deps.endpoint(context.Background(), deviceReq(nil))
	require.NoError(t, err)

	page, ok := res.(pageRes)
	require.True(t, ok)
	assert.Equal(t, []readers.Message{"relayed-by-gateway"}, page.Messages)
}

func TestListMessagesDoesNotExposeDeviceScope(t *testing.T) {
	deps := newTestDeps(t)
	expectUser(deps.authn)
	expectGrants(deps.evaluator, deps.lister, []string{meter1UUID, meter3UUID})

	deps.repo.EXPECT().ReadAll(e2eChanID, mock.MatchedBy(
		scopeMatches([]string{meter1UUID, meter3UUID}, []string{meter1Serial, meter3Serial}),
	)).Return(readers.MessagesPage{
		PageMetadata: readers.PageMetadata{
			Limit: 10,
			DeviceScope: &readers.DeviceScope{
				PublisherIDs: []string{meter1UUID, meter3UUID},
				DeviceIDs:    []string{meter1Serial, meter3Serial},
			},
		},
		Total:    1,
		Messages: []readers.Message{"m1"},
	}, nil)

	res, err := deps.endpoint(context.Background(), deviceReq([]string{meter1Serial}))
	require.NoError(t, err)

	raw, err := json.Marshal(res)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "device_scope")
	assert.NotContains(t, string(raw), meter3UUID)
	assert.NotContains(t, string(raw), meter3Serial)
}

// A client authenticating with a secret key holds no per-device grants, so it
// keeps the channel-level access it had — unchanged from part A.
func TestListMessagesSecretKeyClientIsNotDeviceScoped(t *testing.T) {
	deps := newTestDeps(t)

	deps.repo.EXPECT().ReadAll(e2eChanID, mock.MatchedBy(func(pm readers.PageMetadata) bool {
		return pm.DeviceScope == nil
	})).Return(readers.MessagesPage{Total: 1, Messages: []readers.Message{"m1"}}, nil)

	req := tokenReq("", nil)
	req.token = ""
	req.key = "client-secret"

	res, err := deps.endpoint(context.Background(), req)
	require.NoError(t, err)

	page, ok := res.(pageRes)
	require.True(t, ok)
	assert.Equal(t, uint64(1), page.Total)
}
