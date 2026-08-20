// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	authnmocks "github.com/absmach/magistrala/pkg/authn/mocks"
	policymocks "github.com/absmach/magistrala/pkg/policies/mocks"
	"github.com/absmach/magistrala/readers"
	readermocks "github.com/absmach/magistrala/readers/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// deviceViewDeps mirrors testDeps (transport_test.go) but wires both MG-15
// endpoints against the same mocked authn/authz stack, so the two directions
// can be exercised with the same helpers (expectUser, expectGrants, ...)
// already established for ReadMessages.
type deviceViewDeps struct {
	repo             *readermocks.MessageRepository
	authn            *authnmocks.Authentication
	evaluator        *policymocks.Evaluator
	lister           *policymocks.Service
	gatewayDevicesEP func(ctx context.Context, req any) (any, error)
	deviceGatewaysEP func(ctx context.Context, req any) (any, error)
}

func newDeviceViewDeps(t *testing.T, channelAuthorized bool) *deviceViewDeps {
	t.Helper()
	repo := readermocks.NewMessageRepository(t)
	authn := authnmocks.NewAuthentication(t)
	evaluator := policymocks.NewEvaluator(t)
	lister := policymocks.NewService(t)

	authz := newReadAuthorizer(evaluator, lister, allMeters())
	channels := fakeChannelsClient{authorized: channelAuthorized}

	return &deviceViewDeps{
		repo:             repo,
		authn:            authn,
		evaluator:        evaluator,
		lister:           lister,
		gatewayDevicesEP: listGatewayDevicesEndpoint(repo, authn, fakeClientsClient{}, channels, authz),
		deviceGatewaysEP: listDeviceGatewaysEndpoint(repo, authn, fakeClientsClient{}, channels, authz),
	}
}

func gatewayDevicesReq(publisherID string) deviceViewReq {
	return deviceViewReq{
		chanID:    e2eChanID,
		token:     "token",
		domain:    e2eDomain,
		filterVal: publisherID,
		pageMeta:  readers.PageMetadata{Limit: 10},
	}
}

func deviceGatewaysReq(deviceID string) deviceViewReq {
	return deviceViewReq{
		chanID:    e2eChanID,
		token:     "token",
		domain:    e2eDomain,
		filterVal: deviceID,
		pageMeta:  readers.PageMetadata{Limit: 10},
	}
}

// Criterion 5 (gateway->devices side): a caller need NOT hold a grant on the
// gateway client itself. A shared gateway relays for devices belonging to
// more than one customer, so the gateway named in the URL is the source of
// the roster and the caller's own DeviceScope bounds which rows come back —
// the repository is reached with that scope, never with the gateway identity
// required to be part of the grant.
func TestListGatewayDevicesSharedGatewayNarrowsToCallerScope(t *testing.T) {
	deps := newDeviceViewDeps(t, true)
	expectUser(deps.authn)
	expectGrants(deps.evaluator, deps.lister, []string{meter1UUID})

	deps.repo.EXPECT().ListGatewayDevices(e2eChanID, "shared-gateway", mock.MatchedBy(
		scopeMatches([]string{meter1UUID}, []string{meter1Serial}),
	)).Return(readers.DeviceStatsPage{
		Total: 1,
		Stats: []readers.DeviceStat{{ID: meter1Serial, LastSeen: 100, MessageCount: 5}},
	}, nil)

	res, err := deps.gatewayDevicesEP(context.Background(), gatewayDevicesReq("shared-gateway"))
	require.NoError(t, err)

	page, ok := res.(gatewayDevicesRes)
	require.True(t, ok)
	require.Len(t, page.Devices, 1)
	assert.Equal(t, meter1Serial, page.Devices[0].DeviceID)
	assert.Equal(t, float64(100), page.Devices[0].LastSeen)
	assert.Equal(t, uint64(5), page.Devices[0].MessageCount)
}

// A non-admin caller holding no per-device grant at all gets an empty roster,
// and the repository is never reached.
func TestListGatewayDevicesNoGrantIsEmptyPage(t *testing.T) {
	deps := newDeviceViewDeps(t, true)
	expectUser(deps.authn)
	expectGrants(deps.evaluator, deps.lister, nil)
	// No ListGatewayDevices expectation: reaching the repository would be the bug.

	res, err := deps.gatewayDevicesEP(context.Background(), gatewayDevicesReq("any-gateway"))
	require.NoError(t, err)

	page, ok := res.(gatewayDevicesRes)
	require.True(t, ok)
	assert.Empty(t, page.Devices)
	assert.Zero(t, page.Total)
}

// Criterion 5: a caller granted a gateway's publisher id gets its roster,
// narrowed by the caller's own DeviceScope — the roster of a gateway shared
// across customers must not spill another customer's devices into this
// caller's response.
func TestListGatewayDevicesBoundsRosterToGrantedScope(t *testing.T) {
	deps := newDeviceViewDeps(t, true)
	expectUser(deps.authn)
	expectGrants(deps.evaluator, deps.lister, []string{meter1UUID, meter3UUID})

	deps.repo.EXPECT().ListGatewayDevices(e2eChanID, meter1UUID, mock.MatchedBy(
		scopeMatches([]string{meter1UUID, meter3UUID}, []string{meter1Serial, meter3Serial}),
	)).Return(readers.DeviceStatsPage{
		Total: 1,
		Stats: []readers.DeviceStat{{ID: meter1Serial, LastSeen: 100, MessageCount: 5}},
	}, nil)

	res, err := deps.gatewayDevicesEP(context.Background(), gatewayDevicesReq(meter1UUID))
	require.NoError(t, err)

	page, ok := res.(gatewayDevicesRes)
	require.True(t, ok)
	require.Len(t, page.Devices, 1)
	assert.Equal(t, meter1Serial, page.Devices[0].DeviceID)
	assert.Equal(t, float64(100), page.Devices[0].LastSeen)
	assert.Equal(t, uint64(5), page.Devices[0].MessageCount)
}

// A super-admin's gateway roster query is not narrowed at all.
func TestListGatewayDevicesSuperAdminIsUnbounded(t *testing.T) {
	deps := newDeviceViewDeps(t, true)
	expectSuperAdmin(deps.authn)

	deps.repo.EXPECT().ListGatewayDevices(e2eChanID, "any-gateway", mock.MatchedBy(func(pm readers.PageMetadata) bool {
		return pm.DeviceScope == nil
	})).Return(readers.DeviceStatsPage{Total: 2, Stats: []readers.DeviceStat{{ID: "d1"}, {ID: "d2"}}}, nil)

	res, err := deps.gatewayDevicesEP(context.Background(), gatewayDevicesReq("any-gateway"))
	require.NoError(t, err)

	page, ok := res.(gatewayDevicesRes)
	require.True(t, ok)
	assert.Equal(t, uint64(2), page.Total)
}

// Criterion 6: the channel-level subscribe check still gates this endpoint,
// before any publisher-scoping happens.
func TestListGatewayDevicesChannelCheckStillRejects(t *testing.T) {
	deps := newDeviceViewDeps(t, false)
	expectUser(deps.authn)

	_, err := deps.gatewayDevicesEP(context.Background(), gatewayDevicesReq(meter1UUID))
	assert.Error(t, err)
}

// Criterion 5 (device->gateways side): a non-admin caller not granted the
// requested device's serial gets an empty roster.
func TestListDeviceGatewaysDeniesUnauthorizedDevice(t *testing.T) {
	deps := newDeviceViewDeps(t, true)
	expectUser(deps.authn)
	expectGrants(deps.evaluator, deps.lister, []string{meter1UUID, meter3UUID})
	// No ListDeviceGateways expectation.

	res, err := deps.deviceGatewaysEP(context.Background(), deviceGatewaysReq(meter2Serial))
	require.NoError(t, err)

	page, ok := res.(deviceGatewaysRes)
	require.True(t, ok)
	assert.Empty(t, page.Publishers)
	assert.Zero(t, page.Total)
}

// A caller granted the requested device reads its full gateway roster,
// unscoped: deviceID is itself the authorization boundary here, so every
// gateway that has relayed for it belongs in the response, whichever
// publisher owns it — narrowing by the caller's own publisher grant would
// wrongly exclude a relaying gateway the caller was never separately granted.
func TestListDeviceGatewaysDoesNotNarrowByPublisherScope(t *testing.T) {
	deps := newDeviceViewDeps(t, true)
	expectUser(deps.authn)
	expectGrants(deps.evaluator, deps.lister, []string{meter1UUID})

	deps.repo.EXPECT().ListDeviceGateways(e2eChanID, meter1Serial, mock.MatchedBy(func(pm readers.PageMetadata) bool {
		return pm.DeviceScope == nil
	})).Return(readers.DeviceStatsPage{
		Total: 2,
		Stats: []readers.DeviceStat{
			{ID: "gateway-a", LastSeen: 100, MessageCount: 3},
			{ID: "gateway-b", LastSeen: 200, MessageCount: 7},
		},
	}, nil)

	res, err := deps.deviceGatewaysEP(context.Background(), deviceGatewaysReq(meter1Serial))
	require.NoError(t, err)

	page, ok := res.(deviceGatewaysRes)
	require.True(t, ok)
	require.Len(t, page.Publishers, 2)
	assert.Equal(t, "gateway-a", page.Publishers[0].Publisher)
	assert.Equal(t, "gateway-b", page.Publishers[1].Publisher)
}

// A super-admin's device-gateways query is unrestricted, same as any other
// admin read.
func TestListDeviceGatewaysSuperAdminIsUnbounded(t *testing.T) {
	deps := newDeviceViewDeps(t, true)
	expectSuperAdmin(deps.authn)

	deps.repo.EXPECT().ListDeviceGateways(e2eChanID, "any-device", mock.Anything).
		Return(readers.DeviceStatsPage{Total: 1, Stats: []readers.DeviceStat{{ID: "gw-1"}}}, nil)

	res, err := deps.deviceGatewaysEP(context.Background(), deviceGatewaysReq("any-device"))
	require.NoError(t, err)

	page, ok := res.(deviceGatewaysRes)
	require.True(t, ok)
	assert.Equal(t, uint64(1), page.Total)
}

// Criterion 6: same channel-level gate on the inverse direction.
func TestListDeviceGatewaysChannelCheckStillRejects(t *testing.T) {
	deps := newDeviceViewDeps(t, false)
	expectUser(deps.authn)

	_, err := deps.deviceGatewaysEP(context.Background(), deviceGatewaysReq(meter1Serial))
	assert.Error(t, err)
}

// A client authenticating with a secret key holds no per-device grants, so
// both directions keep the channel-level access they had, same as ReadMessages.
func TestListGatewayDevicesSecretKeyClientIsNotDeviceScoped(t *testing.T) {
	deps := newDeviceViewDeps(t, true)

	deps.repo.EXPECT().ListGatewayDevices(e2eChanID, "gw-1", mock.MatchedBy(func(pm readers.PageMetadata) bool {
		return pm.DeviceScope == nil
	})).Return(readers.DeviceStatsPage{Total: 1, Stats: []readers.DeviceStat{{ID: "d1"}}}, nil)

	req := gatewayDevicesReq("gw-1")
	req.token = ""
	req.key = "client-secret"

	res, err := deps.gatewayDevicesEP(context.Background(), req)
	require.NoError(t, err)

	page, ok := res.(gatewayDevicesRes)
	require.True(t, ok)
	assert.Equal(t, uint64(1), page.Total)
}

func TestDeviceViewReqValidateRequiresFilterVal(t *testing.T) {
	req := gatewayDevicesReq("")
	err := req.validate()
	assert.Error(t, err)
}

func TestDeviceViewReqValidateRequiresCredentials(t *testing.T) {
	req := gatewayDevicesReq("gw-1")
	req.token = ""
	req.key = ""
	err := req.validate()
	assert.Error(t, err)
}

// The gateway->devices direction keys on a publisher id, which lives in a
// UUID column; a malformed value must surface as a request error, not a
// database error. The device->gateways direction keys on a device serial,
// which has no format constraint (MG-09), so it is left alone.
func TestDeviceViewReqValidateRejectsMalformedPublisherID(t *testing.T) {
	valid := deviceViewReq{
		chanID:            e2eChanID,
		token:             "token",
		domain:            e2eDomain,
		filterVal:         "1dcf1a0e-7a9d-4b1e-8d5f-9c2e6a3b4d01",
		filterIsPublisher: true,
		pageMeta:          readers.PageMetadata{Limit: 10},
	}
	require.NoError(t, valid.validate())

	malformed := valid
	malformed.filterVal = "not-a-uuid"
	require.Error(t, malformed.validate())

	serial := deviceViewReq{
		chanID:    e2eChanID,
		token:     "token",
		domain:    e2eDomain,
		filterVal: "Meter.A-01:X",
		pageMeta:  readers.PageMetadata{Limit: 10},
	}
	require.NoError(t, serial.validate())
}

// The unbounded form must not be the easy one to call: omitting both from
// and to bounds the query to the last 24h rather than the whole table.
func TestDecodeGatewayDevicesAppliesDefaultWindow(t *testing.T) {
	r := httptest.NewRequest("GET", "/?publisher=gw-1", nil)
	r.Header.Set("Authorization", "Bearer token")

	decoded, err := decodeGatewayDevices(context.Background(), r)
	require.NoError(t, err)

	req, ok := decoded.(deviceViewReq)
	require.True(t, ok)
	assert.Equal(t, "gw-1", req.filterVal)
	assert.True(t, req.filterIsPublisher, "the gateway->devices transport must mark filterVal as a publisher id")
	assert.NotZero(t, req.pageMeta.From)
	assert.NotZero(t, req.pageMeta.To)
	assert.InDelta(t, float64(readers.DeviceViewDefaultWindow.Nanoseconds()), req.pageMeta.To-req.pageMeta.From, 2)
}

// A caller who supplies either bound gets exactly what they asked for,
// unmodified.
func TestDecodeGatewayDevicesHonoursExplicitWindow(t *testing.T) {
	r := httptest.NewRequest("GET", "/?publisher=gw-1&from=100&to=200", nil)
	r.Header.Set("Authorization", "Bearer token")

	decoded, err := decodeGatewayDevices(context.Background(), r)
	require.NoError(t, err)

	req, ok := decoded.(deviceViewReq)
	require.True(t, ok)
	assert.Equal(t, float64(100), req.pageMeta.From)
	assert.Equal(t, float64(200), req.pageMeta.To)
}

func TestDecodeDeviceGatewaysReadsDeviceIDQueryParam(t *testing.T) {
	r := httptest.NewRequest("GET", "/?device_id=Meter.A-01%3AX", nil)
	r.Header.Set("Authorization", "Bearer token")

	decoded, err := decodeDeviceGateways(context.Background(), r)
	require.NoError(t, err)

	req, ok := decoded.(deviceViewReq)
	require.True(t, ok)
	assert.Equal(t, "Meter.A-01:X", req.filterVal)
	assert.False(t, req.filterIsPublisher, "the device->gateways transport must not treat a serial as a publisher id")
}

// A partial bound is not left dangling: the missing bound is defaulted to
// close a 24h window around the one supplied, so the query can never turn
// into an unbounded scan.
func TestDefaultTimeWindowCompletesPartialBounds(t *testing.T) {
	window := float64(readers.DeviceViewDefaultWindow.Nanoseconds())
	from, to := readers.DefaultTimeWindow(50, 0)
	assert.Equal(t, float64(50), from)
	assert.InDelta(t, window, to-from, 2)

	from, to = readers.DefaultTimeWindow(0, 50)
	assert.InDelta(t, window, to-from, 2)
	assert.Equal(t, float64(50), to)
}

func TestDefaultTimeWindowFillsInLast24hWhenBothAbsent(t *testing.T) {
	before := time.Now()
	from, to := readers.DefaultTimeWindow(0, 0)
	after := time.Now()

	assert.InDelta(t, float64(readers.DeviceViewDefaultWindow.Nanoseconds()), to-from, 2)
	assert.GreaterOrEqual(t, to, float64(before.UnixNano()))
	assert.LessOrEqual(t, to, float64(after.UnixNano()))
}
