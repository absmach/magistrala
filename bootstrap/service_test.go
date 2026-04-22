// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package bootstrap_test

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"sort"
	"testing"

	"github.com/absmach/magistrala/bootstrap"
	mocks "github.com/absmach/magistrala/bootstrap/mocks"
	"github.com/absmach/magistrala/internal/testsutil"
	smqauthn "github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	policysvc "github.com/absmach/magistrala/pkg/policies"
	policymocks "github.com/absmach/magistrala/pkg/policies/mocks"
	mgsdk "github.com/absmach/magistrala/pkg/sdk"
	sdkmocks "github.com/absmach/magistrala/pkg/sdk/mocks"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	validToken      = "validToken"
	invalidToken    = "invalid"
	invalidDomainID = "invalid"
	email           = "test@example.com"
	unknown         = "unknown"
	channelsNum     = 3
	instanceID      = "5de9b29a-feb9-11ed-be56-0242ac120002"
	validID         = "d4ebb847-5d0e-4e46-bdd9-b6aceaaa3a22"
)

var (
	encKey    = []byte("1234567891011121")
	connTypes = []string{"Publish", "Subscribe"}
	domainID  = testsutil.GenerateUUID(&testing.T{})
	channel   = bootstrap.Channel{
		ID:       testsutil.GenerateUUID(&testing.T{}),
		Name:     "name",
		Metadata: map[string]any{"name": "value"},
	}

	config = bootstrap.Config{
		ClientID:     testsutil.GenerateUUID(&testing.T{}),
		ClientSecret: testsutil.GenerateUUID(&testing.T{}),
		ExternalID:   testsutil.GenerateUUID(&testing.T{}),
		ExternalKey:  testsutil.GenerateUUID(&testing.T{}),
		Channels:     []bootstrap.Channel{channel},
		Content:      "config",
	}
)

var (
	boot     *mocks.ConfigRepository
	policies *policymocks.Service
	sdk      *sdkmocks.SDK
)

func newService() bootstrap.Service {
	boot = new(mocks.ConfigRepository)
	policies = new(policymocks.Service)
	sdk = new(sdkmocks.SDK)
	idp := uuid.NewMock()
	return bootstrap.New(policies, boot, sdk, encKey, idp)
}

func enc(in []byte) ([]byte, error) {
	block, err := aes.NewCipher(encKey)
	if err != nil {
		return nil, err
	}
	ciphertext := make([]byte, aes.BlockSize+len(in))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], in)
	return ciphertext, nil
}

func TestAdd(t *testing.T) {
	svc := newService()

	neID := config
	neID.ClientID = "non-existent"

	wrongChannels := config
	ch := channel
	ch.ID = "invalid"
	wrongChannels.Channels = append(wrongChannels.Channels, ch)

	cases := []struct {
		desc            string
		config          bootstrap.Config
		token           string
		session         smqauthn.Session
		userID          string
		domainID        string
		clientErr       error
		createClientErr error
		channelErr      error
		connectErr      error
		listExistingErr error
		saveErr         error
		deleteClientErr error
		err             error
	}{
		{
			desc:     "add a new config",
			config:   config,
			token:    validToken,
			userID:   validID,
			domainID: domainID,
			err:      nil,
		},
		{
			desc:      "add a config with an invalid ID",
			config:    neID,
			token:     validToken,
			userID:    validID,
			domainID:  domainID,
			clientErr: errors.NewSDKError(svcerr.ErrNotFound),
			err:       svcerr.ErrNotFound,
		},
		{
			desc:            "add a config with invalid list of channels",
			config:          wrongChannels,
			token:           validToken,
			userID:          validID,
			domainID:        domainID,
			listExistingErr: svcerr.ErrMalformedEntity,
			err:             svcerr.ErrMalformedEntity,
		},
		{
			desc:       "add a config with failed client connection",
			config:     config,
			token:      validToken,
			userID:     validID,
			domainID:   domainID,
			connectErr: bootstrap.ErrClients,
			err:        bootstrap.ErrClients,
		},
		{
			desc:     "add empty config",
			config:   bootstrap.Config{},
			token:    validToken,
			userID:   validID,
			domainID: domainID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			tc.session = smqauthn.Session{UserID: tc.userID, DomainID: tc.domainID, DomainUserID: validID}
			repoCall := sdk.On("Client", mock.Anything, tc.config.ClientID, mock.Anything, tc.token).Return(mgsdk.Client{ID: tc.config.ClientID, Credentials: mgsdk.ClientCredentials{Secret: tc.config.ClientSecret}}, tc.clientErr)
			repoCall1 := sdk.On("CreateClient", mock.Anything, mock.Anything, tc.domainID, tc.token).Return(mgsdk.Client{}, tc.createClientErr)
			repoCall2 := sdk.On("DeleteClient", mock.Anything, tc.config.ClientID, tc.domainID, tc.token).Return(tc.deleteClientErr)
			repoCall3 := boot.On("ListExisting", context.Background(), tc.domainID, mock.Anything).Return(tc.config.Channels, tc.listExistingErr)
			repoCall4 := boot.On("Save", context.Background(), mock.Anything, mock.Anything).Return(mock.Anything, tc.saveErr)
			sdkCall := sdk.On("ConnectClients", mock.Anything, mock.Anything, mock.Anything, connTypes, tc.domainID, tc.token).Return(errors.NewSDKError(tc.connectErr))
			_, err := svc.Add(context.Background(), tc.session, tc.token, tc.config)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			repoCall.Unset()
			repoCall1.Unset()
			repoCall2.Unset()
			repoCall3.Unset()
			repoCall4.Unset()
			sdkCall.Unset()
		})
	}
}

func TestAddSkipsConnectedChannels(t *testing.T) {
	svc := newService()

	ch1 := channel
	ch1.ID = testsutil.GenerateUUID(t)
	ch1.DomainID = domainID
	ch2 := channel
	ch2.ID = testsutil.GenerateUUID(t)
	ch2.DomainID = domainID
	cfg := config
	cfg.Channels = []bootstrap.Channel{ch1, ch2}
	session := smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: validID}

	clientCall := sdk.On("Client", mock.Anything, cfg.ClientID, mock.Anything, validToken).Return(mgsdk.Client{
		ID:       cfg.ClientID,
		DomainID: domainID,
		Credentials: mgsdk.ClientCredentials{
			Secret: cfg.ClientSecret,
		},
	}, nil)
	createClientCall := sdk.On("CreateClient", mock.Anything, mock.Anything, domainID, validToken).Return(mgsdk.Client{}, nil)
	deleteClientCall := sdk.On("DeleteClient", mock.Anything, cfg.ClientID, domainID, validToken).Return(nil)
	listExistingCall := boot.On("ListExisting", context.Background(), domainID, []string{ch1.ID, ch2.ID}).Return([]bootstrap.Channel{}, nil)
	channelCall1 := sdk.On("Channel", mock.Anything, ch1.ID, domainID, validToken).Return(mgsdk.Channel{
		ID:       ch1.ID,
		Name:     ch1.Name,
		Metadata: ch1.Metadata,
		DomainID: ch1.DomainID,
	}, nil)
	channelCall2 := sdk.On("Channel", mock.Anything, ch2.ID, domainID, validToken).Return(mgsdk.Channel{
		ID:       ch2.ID,
		Name:     ch2.Name,
		Metadata: ch2.Metadata,
		DomainID: ch2.DomainID,
	}, nil)
	connectCall := sdk.On("ConnectClients", mock.Anything, mock.Anything, mock.Anything, connTypes, domainID, validToken).Return(errors.NewSDKError(svcerr.ErrConflict))
	saveCall := boot.On("Save", context.Background(), mock.MatchedBy(func(saved bootstrap.Config) bool {
		return saved.State == bootstrap.Active
	}), mock.Anything).Return(cfg.ClientID, nil)

	saved, err := svc.Add(context.Background(), session, validToken, cfg)
	assert.Nil(t, err, fmt.Sprintf("expected add to skip existing channel connection: %s", err))
	assert.Equal(t, bootstrap.Active, saved.State)

	_ = clientCall
	_ = createClientCall
	_ = deleteClientCall
	_ = listExistingCall
	_ = channelCall1
	_ = channelCall2
	_ = connectCall
	_ = saveCall
}

func TestView(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc         string
		configID     string
		userID       string
		domain       string
		clientDomain string
		token        string
		session      smqauthn.Session
		retrieveErr  error
		clientErr    error
		channelErr   error
		err          error
	}{
		{
			desc:         "view an existing config",
			configID:     config.ClientID,
			userID:       validID,
			clientDomain: domainID,
			domain:       domainID,
			token:        validToken,
			err:          nil,
		},
		{
			desc:         "view a non-existing config",
			configID:     unknown,
			userID:       validID,
			clientDomain: domainID,
			domain:       domainID,
			token:        validToken,
			retrieveErr:  svcerr.ErrNotFound,
			err:          svcerr.ErrNotFound,
		},
		{
			desc:         "view a config with invalid domain",
			configID:     config.ClientID,
			userID:       validID,
			clientDomain: invalidDomainID,
			domain:       invalidDomainID,
			token:        validToken,
			retrieveErr:  svcerr.ErrNotFound,
			err:          svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			tc.session = smqauthn.Session{UserID: tc.userID, DomainID: tc.domain, DomainUserID: validID}
			repoCall := boot.On("RetrieveByID", context.Background(), tc.clientDomain, tc.configID).Return(config, tc.retrieveErr)
			_, err := svc.View(context.Background(), tc.session, tc.configID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			repoCall.Unset()
		})
	}
}

func TestUpdate(t *testing.T) {
	svc := newService()

	c := config
	ch := channel
	ch.ID = "2"
	c.Channels = append(c.Channels, ch)

	modifiedCreated := c
	modifiedCreated.Content = "new-config"
	modifiedCreated.Name = "new name"

	nonExisting := c
	nonExisting.ClientID = unknown

	cases := []struct {
		desc      string
		config    bootstrap.Config
		token     string
		session   smqauthn.Session
		userID    string
		domainID  string
		updateErr error
		err       error
	}{
		{
			desc:     "update a config with state Created",
			config:   modifiedCreated,
			token:    validToken,
			userID:   validID,
			domainID: domainID,
			err:      nil,
		},
		{
			desc:      "update a non-existing config",
			config:    nonExisting,
			token:     validToken,
			userID:    validID,
			domainID:  domainID,
			updateErr: svcerr.ErrNotFound,
			err:       svcerr.ErrNotFound,
		},
		{
			desc:      "update a config with update error",
			config:    c,
			token:     validToken,
			userID:    validID,
			domainID:  domainID,
			updateErr: svcerr.ErrUpdateEntity,
			err:       svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			tc.session = smqauthn.Session{UserID: tc.userID, DomainID: tc.domainID, DomainUserID: validID}
			repoCall := boot.On("Update", context.Background(), mock.Anything).Return(tc.updateErr)
			err := svc.Update(context.Background(), tc.session, tc.config)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			repoCall.Unset()
		})
	}
}

func TestUpdateCert(t *testing.T) {
	svc := newService()

	c := config
	ch := channel
	ch.ID = "2"
	c.Channels = append(c.Channels, ch)

	cases := []struct {
		desc            string
		token           string
		session         smqauthn.Session
		userID          string
		domainID        string
		clientID        string
		clientCert      string
		clientKey       string
		caCert          string
		expectedConfig  bootstrap.Config
		authorizeErr    error
		authenticateErr error
		updateErr       error
		err             error
	}{
		{
			desc:       "update certs for the valid config",
			userID:     validID,
			domainID:   domainID,
			clientID:   c.ClientID,
			clientCert: "newCert",
			clientKey:  "newKey",
			caCert:     "newCert",
			token:      validToken,
			expectedConfig: bootstrap.Config{
				Name:         c.Name,
				ClientSecret: c.ClientSecret,
				Channels:     c.Channels,
				ExternalID:   c.ExternalID,
				ExternalKey:  c.ExternalKey,
				Content:      c.Content,
				State:        c.State,
				DomainID:     c.DomainID,
				ClientID:     c.ClientID,
				ClientCert:   "newCert",
				CACert:       "newCert",
				ClientKey:    "newKey",
			},
			err: nil,
		},
		{
			desc:           "update cert for a non-existing config",
			userID:         validID,
			domainID:       domainID,
			clientID:       "empty",
			clientCert:     "newCert",
			clientKey:      "newKey",
			caCert:         "newCert",
			token:          validToken,
			expectedConfig: bootstrap.Config{},
			updateErr:      svcerr.ErrNotFound,
			err:            svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			tc.session = smqauthn.Session{UserID: tc.userID, DomainID: tc.domainID, DomainUserID: validID}
			repoCall := boot.On("UpdateCert", context.Background(), mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tc.expectedConfig, tc.updateErr)
			cfg, err := svc.UpdateCert(context.Background(), tc.session, tc.clientID, tc.clientCert, tc.clientKey, tc.caCert)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			sort.Slice(cfg.Channels, func(i, j int) bool {
				return cfg.Channels[i].ID < cfg.Channels[j].ID
			})
			sort.Slice(tc.expectedConfig.Channels, func(i, j int) bool {
				return tc.expectedConfig.Channels[i].ID < tc.expectedConfig.Channels[j].ID
			})
			assert.Equal(t, tc.expectedConfig, cfg, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.expectedConfig, cfg))
			repoCall.Unset()
		})
	}
}

func TestUpdateConnections(t *testing.T) {
	svc := newService()

	c := config
	c.State = bootstrap.Inactive

	activeConf := config
	activeConf.State = bootstrap.Active

	activeConfEmpty := config
	activeConfEmpty.State = bootstrap.Active
	activeConfEmpty.Channels = []bootstrap.Channel{}

	ch := channel

	cases := []struct {
		desc           string
		config         bootstrap.Config
		token          string
		session        smqauthn.Session
		id             string
		state          bootstrap.State
		userID         string
		domainID       string
		connections   []string
		updateErr     error
		clientErr     error
		channelErr     error
		connectErr     error
		disconnectErr  error
		retrieveErr    error
		listErr        error
		err            error
	}{
		{
			desc:        "update connections for config with state Inactive",
			config:      c,
			token:       validToken,
			userID:      validID,
			domainID:    domainID,
			id:          c.ClientID,
			state:       c.State,
			connections: []string{ch.ID},
			err:         nil,
		},
		{
			desc:        "update connections for config with state Active",
			config:      activeConf,
			token:       validToken,
			userID:      validID,
			domainID:    domainID,
			id:          activeConf.ClientID,
			state:       activeConf.State,
			connections: []string{ch.ID},
			err:         nil,
		},
		{
			desc:        "update connections with invalid channels",
			config:      c,
			token:       validToken,
			userID:      validID,
			domainID:    domainID,
			id:          c.ClientID,
			connections: []string{"wrong"},
			channelErr:  errors.NewSDKError(svcerr.ErrNotFound),
			err:         svcerr.ErrNotFound,
		},
		{
			desc:        "update connections with failed connect",
			config:      activeConfEmpty,
			token:       validToken,
			userID:      validID,
			domainID:    domainID,
			id:          activeConfEmpty.ClientID,
			connections: []string{ch.ID},
			connectErr:  bootstrap.ErrClients,
			err:         bootstrap.ErrClients,
		},
		{
			desc:          "update connections with failed disconnect",
			config:        activeConf,
			token:         validToken,
			userID:        validID,
			domainID:      domainID,
			id:            activeConf.ClientID,
			connections:   []string{},
			disconnectErr: bootstrap.ErrClients,
			err:           bootstrap.ErrClients,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			tc.session = smqauthn.Session{UserID: tc.userID, DomainID: tc.domainID, DomainUserID: validID}
			sdkCall := sdk.On("Channel", mock.Anything, mock.Anything, tc.domainID, tc.token).Return(mgsdk.Channel{}, tc.channelErr)
			repoCall := boot.On("RetrieveByID", context.Background(), tc.domainID, tc.id).Return(tc.config, tc.retrieveErr)
			repoCall1 := boot.On("ListExisting", context.Background(), mock.Anything, mock.Anything, mock.Anything).Return(tc.config.Channels, tc.listErr)
			repoCall2 := boot.On("UpdateConnections", context.Background(), mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tc.updateErr)
			connectCall := sdk.On("Connect", mock.Anything, mock.Anything, tc.domainID, tc.token).Return(errors.NewSDKError(tc.connectErr))
			disconnectCall := sdk.On("Disconnect", mock.Anything, mock.Anything, tc.domainID, tc.token).Return(errors.NewSDKError(tc.disconnectErr))
			err := svc.UpdateConnections(context.Background(), tc.session, tc.token, tc.id, tc.connections)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			sdkCall.Unset()
			repoCall.Unset()
			repoCall1.Unset()
			repoCall2.Unset()
			connectCall.Unset()
			disconnectCall.Unset()
		})
	}
}

func TestUpdateConnectionsInactiveConfigOnlyUpdatesDB(t *testing.T) {
	svc := newService()

	c := config
	c.State = bootstrap.Inactive
	c.Channels = []bootstrap.Channel{}
	ch := channel
	ch.DomainID = domainID
	session := smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: validID}
	connections := []string{ch.ID}

	repoCall := boot.On("RetrieveByID", context.Background(), domainID, c.ClientID).Return(c, nil)
	repoCall1 := boot.On("ListExisting", context.Background(), domainID, connections).Return([]bootstrap.Channel{}, nil)
	sdkCall := sdk.On("Channel", mock.Anything, ch.ID, domainID, validToken).Return(mgsdk.Channel{
		ID:       ch.ID,
		Name:     ch.Name,
		Metadata: ch.Metadata,
		DomainID: ch.DomainID,
	}, nil)
	repoCall2 := boot.On("UpdateConnections", context.Background(), domainID, c.ClientID, mock.Anything, connections).Return(nil)

	err := svc.UpdateConnections(context.Background(), session, validToken, c.ClientID, connections)
	assert.Nil(t, err, fmt.Sprintf("expected update connections for inactive config to succeed: %s", err))
	sdk.AssertNotCalled(t, "Connect")
	sdk.AssertNotCalled(t, "ChangeState")

	sdkCall.Unset()
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()
}

func TestUpdateConnectionsDisconnectsActiveConfig(t *testing.T) {
	svc := newService()

	c := config
	c.State = bootstrap.Active
	ch := channel
	ch.DomainID = domainID
	c.Channels = []bootstrap.Channel{ch}
	session := smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: validID}
	// Empty connections: all existing channels should be removed.
	connections := []string{}

	repoCall := boot.On("RetrieveByID", context.Background(), domainID, c.ClientID).Return(c, nil)
	repoCall1 := boot.On("ListExisting", context.Background(), domainID, connections).Return(c.Channels, nil)
	disconnectCall := sdk.On("Disconnect", mock.Anything, mgsdk.Connection{
		ChannelIDs: []string{ch.ID},
		ClientIDs:  []string{c.ClientID},
		Types:      connTypes,
	}, domainID, validToken).Return(nil)
	repoCall2 := boot.On("UpdateConnections", context.Background(), domainID, c.ClientID, mock.Anything, connections).Return(nil)

	err := svc.UpdateConnections(context.Background(), session, validToken, c.ClientID, connections)
	assert.Nil(t, err, fmt.Sprintf("expected update connections to disconnect active config: %s", err))
	sdk.AssertCalled(t, "Disconnect", mock.Anything, mgsdk.Connection{
		ChannelIDs: []string{ch.ID},
		ClientIDs:  []string{c.ClientID},
		Types:      connTypes,
	}, domainID, validToken)

	repoCall.Unset()
	repoCall1.Unset()
	disconnectCall.Unset()
	repoCall2.Unset()
}

func TestList(t *testing.T) {
	svc := newService()

	numClients := 101
	var saved []bootstrap.Config
	for i := 0; i < numClients; i++ {
		c := config
		c.ExternalID = testsutil.GenerateUUID(t)
		c.ExternalKey = testsutil.GenerateUUID(t)
		c.Name = fmt.Sprintf("%s-%d", config.Name, i)
		if i == 41 {
			c.State = bootstrap.Active
		}
		saved = append(saved, c)
	}
	cases := []struct {
		desc                string
		config              bootstrap.ConfigsPage
		filter              bootstrap.Filter
		offset              uint64
		limit               uint64
		token               string
		session             smqauthn.Session
		userID              string
		domainID            string
		listObjectsResponse policysvc.PolicyPage
		listObjectsErr      error
		retrieveErr         error
		err                 error
	}{
		{
			desc: "list configs successfully as super admin",
			config: bootstrap.ConfigsPage{
				Total:   uint64(len(saved)),
				Offset:  0,
				Limit:   10,
				Configs: saved[0:10],
			},
			filter:   bootstrap.Filter{},
			token:    validToken,
			session:  smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: validID, SuperAdmin: true},
			userID:   validID,
			domainID: domainID,
			offset:   0,
			limit:    10,
			err:      nil,
		},
		{
			desc:                "list configs with failed super admin check",
			config:              bootstrap.ConfigsPage{},
			filter:              bootstrap.Filter{},
			token:               validID,
			session:             smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: validID},
			userID:              validID,
			domainID:            domainID,
			listObjectsResponse: policysvc.PolicyPage{},
			offset:              0,
			limit:               10,
			err:                 nil,
		},
		{
			desc: "list configs successfully as domain admin",
			config: bootstrap.ConfigsPage{
				Total:   uint64(len(saved)),
				Offset:  0,
				Limit:   10,
				Configs: saved[0:10],
			},
			filter:              bootstrap.Filter{},
			token:               validToken,
			userID:              validID,
			domainID:            domainID,
			session:             smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: validID, SuperAdmin: true},
			listObjectsResponse: policysvc.PolicyPage{},
			offset:              0,
			limit:               10,
			err:                 nil,
		},
		{
			desc: "list configs successfully as non admin",
			config: bootstrap.ConfigsPage{
				Total:   uint64(len(saved)),
				Offset:  0,
				Limit:   10,
				Configs: saved[0:10],
			},
			filter:              bootstrap.Filter{},
			token:               validToken,
			userID:              validID,
			domainID:            domainID,
			session:             smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: validID},
			listObjectsResponse: policysvc.PolicyPage{Policies: []string{"test", "test"}},
			offset:              0,
			limit:               10,
			err:                 nil,
		},
		{
			desc: "list configs with specified name as super admin",
			config: bootstrap.ConfigsPage{
				Total:   1,
				Offset:  0,
				Limit:   100,
				Configs: saved[95:96],
			},
			filter:   bootstrap.Filter{PartialMatch: map[string]string{"name": "95"}},
			token:    validToken,
			session:  smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: validID, SuperAdmin: true},
			userID:   validID,
			domainID: domainID,
			offset:   0,
			limit:    100,
			err:      nil,
		},
		{
			desc: "list configs with specified name as domain admin",
			config: bootstrap.ConfigsPage{
				Total:   1,
				Offset:  0,
				Limit:   100,
				Configs: saved[95:96],
			},
			filter:   bootstrap.Filter{PartialMatch: map[string]string{"name": "95"}},
			token:    validToken,
			userID:   validID,
			domainID: domainID,
			session:  smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: validID, SuperAdmin: true},
			offset:   0,
			limit:    100,
			err:      nil,
		},
		{
			desc: "list configs with specified name as non admin",
			config: bootstrap.ConfigsPage{
				Total:   1,
				Offset:  0,
				Limit:   100,
				Configs: saved[95:96],
			},
			filter:              bootstrap.Filter{PartialMatch: map[string]string{"name": "95"}},
			token:               validToken,
			userID:              validID,
			domainID:            domainID,
			session:             smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: validID},
			listObjectsResponse: policysvc.PolicyPage{Policies: []string{"test", "test"}},
			offset:              0,
			limit:               100,
			err:                 nil,
		},
		{
			desc: "list last page as super admin",
			config: bootstrap.ConfigsPage{
				Total:   uint64(len(saved)),
				Offset:  95,
				Limit:   10,
				Configs: saved[95:],
			},
			filter:   bootstrap.Filter{},
			token:    validToken,
			userID:   validID,
			domainID: domainID,
			session:  smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: validID, SuperAdmin: true},
			offset:   95,
			limit:    10,
			err:      nil,
		},
		{
			desc: "list last page as domain admin",
			config: bootstrap.ConfigsPage{
				Total:   uint64(len(saved)),
				Offset:  95,
				Limit:   10,
				Configs: saved[95:],
			},
			filter:   bootstrap.Filter{},
			token:    validToken,
			userID:   validID,
			domainID: domainID,
			session:  smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: validID, SuperAdmin: true},
			offset:   95,
			limit:    10,
			err:      nil,
		},
		{
			desc: "list last page as non admin",
			config: bootstrap.ConfigsPage{
				Total:   uint64(len(saved)),
				Offset:  95,
				Limit:   10,
				Configs: saved[95:],
			},
			filter:              bootstrap.Filter{},
			token:               validToken,
			userID:              validID,
			domainID:            domainID,
			session:             smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: validID},
			listObjectsResponse: policysvc.PolicyPage{Policies: []string{"test", "test"}},
			offset:              95,
			limit:               10,
			err:                 nil,
		},
		{
			desc: "list configs with Active state as super admin",
			config: bootstrap.ConfigsPage{
				Total:   1,
				Offset:  35,
				Limit:   20,
				Configs: []bootstrap.Config{saved[41]},
			},
			filter:   bootstrap.Filter{FullMatch: map[string]string{"state": bootstrap.Active.String()}},
			token:    validToken,
			userID:   validID,
			domainID: domainID,
			session:  smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: validID, SuperAdmin: true},
			offset:   35,
			limit:    20,
			err:      nil,
		},
		{
			desc: "list configs with Active state as domain admin",
			config: bootstrap.ConfigsPage{
				Total:   1,
				Offset:  35,
				Limit:   20,
				Configs: []bootstrap.Config{saved[41]},
			},
			filter:   bootstrap.Filter{FullMatch: map[string]string{"state": bootstrap.Active.String()}},
			token:    validToken,
			userID:   validID,
			domainID: domainID,
			session:  smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: validID, SuperAdmin: true},
			offset:   35,
			limit:    20,
			err:      nil,
		},
		{
			desc: "list configs with Active state as non admin",
			config: bootstrap.ConfigsPage{
				Total:   1,
				Offset:  35,
				Limit:   20,
				Configs: []bootstrap.Config{saved[41]},
			},
			filter:              bootstrap.Filter{FullMatch: map[string]string{"state": bootstrap.Active.String()}},
			token:               validToken,
			userID:              validID,
			domainID:            domainID,
			session:             smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: validID},
			listObjectsResponse: policysvc.PolicyPage{Policies: []string{"test", "test"}},
			offset:              35,
			limit:               20,
			err:                 nil,
		},
		{
			desc:                "list configs with failed to list objects",
			config:              bootstrap.ConfigsPage{},
			filter:              bootstrap.Filter{},
			offset:              0,
			limit:               10,
			token:               validToken,
			userID:              validID,
			domainID:            domainID,
			session:             smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: validID},
			listObjectsResponse: policysvc.PolicyPage{},
			listObjectsErr:      svcerr.ErrNotFound,
			err:                 svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			policyCall := policies.On("ListAllObjects", mock.Anything, policysvc.Policy{
				SubjectType: policysvc.UserType,
				Subject:     tc.userID,
				Permission:  policysvc.ViewPermission,
				ObjectType:  policysvc.ClientType,
			}).Return(tc.listObjectsResponse, tc.listObjectsErr)
			repoCall := boot.On("RetrieveAll", context.Background(), mock.Anything, mock.Anything, tc.filter, tc.offset, tc.limit).Return(tc.config, tc.retrieveErr)

			result, err := svc.List(context.Background(), tc.session, tc.filter, tc.offset, tc.limit)
			assert.ElementsMatch(t, tc.config.Configs, result.Configs, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.config.Configs, result.Configs))
			assert.Equal(t, tc.config.Total, result.Total, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.config.Total, result.Total))
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			policyCall.Unset()
			repoCall.Unset()
		})
	}
}

func TestRemove(t *testing.T) {
	svc := newService()

	c := config
	cases := []struct {
		desc      string
		id        string
		token     string
		session   smqauthn.Session
		userID    string
		domainID  string
		removeErr error
		err       error
	}{
		{
			desc:     "remove an existing config",
			id:       c.ClientID,
			token:    validToken,
			userID:   validID,
			domainID: domainID,
			err:      nil,
		},
		{
			desc:     "remove removed config",
			id:       c.ClientID,
			token:    validToken,
			userID:   validID,
			domainID: domainID,
			err:      nil,
		},
		{
			desc:      "remove a config with failed remove",
			id:        c.ClientID,
			token:     validToken,
			userID:    validID,
			domainID:  domainID,
			removeErr: svcerr.ErrRemoveEntity,
			err:       svcerr.ErrRemoveEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			tc.session = smqauthn.Session{UserID: tc.userID, DomainID: tc.domainID, DomainUserID: validID}
			repoCall := boot.On("Remove", context.Background(), mock.Anything, mock.Anything).Return(tc.removeErr)
			err := svc.Remove(context.Background(), tc.session, tc.id)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			repoCall.Unset()
		})
	}
}

func TestBootstrap(t *testing.T) {
	svc := newService()

	c := config
	e, err := enc([]byte(c.ExternalKey))
	assert.Nil(t, err, fmt.Sprintf("Encrypting external key expected to succeed: %s.\n", err))

	cases := []struct {
		desc        string
		config      bootstrap.Config
		externalKey string
		externalID  string
		userID      string
		domainID    string
		err         error
		encrypted   bool
	}{
		{
			desc:        "bootstrap using invalid external id",
			config:      bootstrap.Config{},
			externalID:  "invalid",
			externalKey: c.ExternalKey,
			userID:      validID,
			domainID:    invalidDomainID,
			err:         svcerr.ErrNotFound,
			encrypted:   false,
		},
		{
			desc:        "bootstrap using invalid external key",
			config:      bootstrap.Config{},
			externalID:  c.ExternalID,
			externalKey: "invalid",
			userID:      validID,
			domainID:    domainID,
			err:         bootstrap.ErrExternalKey,
			encrypted:   false,
		},
		{
			desc:        "bootstrap an existing config",
			config:      c,
			externalID:  c.ExternalID,
			externalKey: c.ExternalKey,
			userID:      validID,
			domainID:    domainID,
			err:         nil,
			encrypted:   false,
		},
		{
			desc:        "bootstrap encrypted",
			config:      c,
			externalID:  c.ExternalID,
			externalKey: hex.EncodeToString(e),
			userID:      validID,
			domainID:    domainID,
			err:         nil,
			encrypted:   true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := boot.On("RetrieveByExternalID", context.Background(), mock.Anything).Return(tc.config, tc.err)
			config, err := svc.Bootstrap(context.Background(), tc.externalKey, tc.externalID, tc.encrypted)
			assert.Equal(t, tc.config, config, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.config, config))
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			repoCall.Unset()
		})
	}
}

func TestChangeState(t *testing.T) {
	svc := newService()

	c := config
	activeConfig := config
	activeConfig.State = bootstrap.Active
	cases := []struct {
		desc          string
		config        bootstrap.Config
		state         bootstrap.State
		id            string
		token         string
		session       smqauthn.Session
		userID        string
		domainID      string
		retrieveErr   error
		connectErr    errors.SDKError
		disconnectErr error
		stateErr      error
		err           error
	}{
		{
			desc:        "change state of non-existing config",
			config:      c,
			state:       bootstrap.Active,
			id:          unknown,
			token:       validToken,
			userID:      validID,
			domainID:    domainID,
			retrieveErr: svcerr.ErrNotFound,
			err:         svcerr.ErrNotFound,
		},
		{
			desc:     "change state to Active",
			config:   c,
			state:    bootstrap.Active,
			id:       c.ClientID,
			token:    validToken,
			userID:   validID,
			domainID: domainID,
			err:      nil,
		},
		{
			desc:     "change state to current state",
			config:   activeConfig,
			state:    bootstrap.Active,
			id:       c.ClientID,
			token:    validToken,
			userID:   validID,
			domainID: domainID,
			err:      nil,
		},
		{
			desc:     "change state to Inactive",
			config:   activeConfig,
			state:    bootstrap.Inactive,
			id:       c.ClientID,
			token:    validToken,
			userID:   validID,
			domainID: domainID,
			err:      nil,
		},
		{
			desc:          "change state with failed Disconnect",
			config:        activeConfig,
			state:         bootstrap.Inactive,
			id:            c.ClientID,
			token:         validToken,
			userID:        validID,
			domainID:      domainID,
			disconnectErr: bootstrap.ErrClients,
			err:           bootstrap.ErrClients,
		},
		{
			desc:       "change state with failed Connect",
			config:     c,
			state:      bootstrap.Active,
			id:         c.ClientID,
			token:      validToken,
			userID:     validID,
			domainID:   domainID,
			connectErr: errors.NewSDKError(bootstrap.ErrClients),
			err:        bootstrap.ErrClients,
		},
		{
			desc:     "change state with invalid state",
			config:   c,
			state:    bootstrap.State(2),
			id:       c.ClientID,
			token:    validToken,
			userID:   validID,
			domainID: domainID,
			stateErr: svcerr.ErrMalformedEntity,
			err:      svcerr.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			tc.session = smqauthn.Session{UserID: tc.userID, DomainID: tc.domainID, DomainUserID: validID}
			repoCall := boot.On("RetrieveByID", context.Background(), tc.domainID, tc.id).Return(tc.config, tc.retrieveErr)
			sdkCall := sdk.On("Connect", mock.Anything, mock.Anything, mock.Anything, tc.token).Return(tc.connectErr)
			sdkCall1 := sdk.On("Disconnect", mock.Anything, mock.Anything, mock.Anything, tc.token).Return(errors.NewSDKError(tc.disconnectErr))
			repoCall1 := boot.On("ChangeState", context.Background(), mock.Anything, mock.Anything, mock.Anything).Return(tc.stateErr)
			err := svc.ChangeState(context.Background(), tc.session, tc.token, tc.id, tc.state)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			sdkCall.Unset()
			sdkCall1.Unset()
			repoCall.Unset()
			repoCall1.Unset()
		})
	}
}

func TestUpdateChannelHandler(t *testing.T) {
	svc := newService()

	ch := bootstrap.Channel{
		ID:       channel.ID,
		Name:     "new name",
		Metadata: map[string]any{"meta": "new"},
	}

	cases := []struct {
		desc    string
		channel bootstrap.Channel
		err     error
	}{
		{
			desc:    "update an existing channel",
			channel: ch,
			err:     nil,
		},
		{
			desc:    "update a non-existing channel",
			channel: bootstrap.Channel{ID: ""},
			err:     nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := boot.On("UpdateChannel", context.Background(), mock.Anything).Return(tc.err)
			err := svc.UpdateChannelHandler(context.Background(), tc.channel)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			repoCall.Unset()
		})
	}
}

func TestRemoveChannelHandler(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc string
		id   string
		err  error
	}{
		{
			desc: "remove an existing channel",
			id:   config.Channels[0].ID,
			err:  nil,
		},
		{
			desc: "remove a non-existing channel",
			id:   "unknown",
			err:  nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := boot.On("RemoveChannel", context.Background(), mock.Anything).Return(tc.err)
			err := svc.RemoveChannelHandler(context.Background(), tc.id)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			repoCall.Unset()
		})
	}
}

func TestRemoveConfigHandler(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc string
		id   string
		err  error
	}{
		{
			desc: "remove an existing config",
			id:   config.ClientID,
			err:  nil,
		},
		{
			desc: "remove a non-existing channel",
			id:   "unknown",
			err:  nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := boot.On("RemoveClient", context.Background(), mock.Anything).Return(tc.err)
			err := svc.RemoveConfigHandler(context.Background(), tc.id)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			repoCall.Unset()
		})
	}
}

func TestConnectClientHandler(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc      string
		clientID  string
		channelID string
		err       error
	}{
		{
			desc:      "connect",
			channelID: channel.ID,
			clientID:  config.ClientID,
			err:       nil,
		},
		{
			desc:      "connect connected",
			channelID: channel.ID,
			clientID:  config.ClientID,
			err:       svcerr.ErrAddPolicies,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := boot.On("ConnectClient", context.Background(), mock.Anything, mock.Anything).Return(tc.err)
			err := svc.ConnectClientHandler(context.Background(), tc.channelID, tc.clientID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			repoCall.Unset()
		})
	}
}

func TestDisconnectClientsHandler(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc      string
		clientID  string
		channelID string
		err       error
	}{
		{
			desc:      "disconnect",
			channelID: channel.ID,
			clientID:  config.ClientID,
			err:       nil,
		},
		{
			desc:      "disconnect disconnected",
			channelID: channel.ID,
			clientID:  config.ClientID,
			err:       nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := boot.On("DisconnectClient", context.Background(), mock.Anything, mock.Anything).Return(tc.err)
			err := svc.DisconnectClientHandler(context.Background(), tc.channelID, tc.clientID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			repoCall.Unset()
		})
	}
}
