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
	email      = "test@example.com"
	unknown    = "unknown"
	instanceID = "5de9b29a-feb9-11ed-be56-0242ac120002"
	validID         = "d4ebb847-5d0e-4e46-bdd9-b6aceaaa3a22"
)

var (
	encKey   = []byte("1234567891011121")
	domainID = testsutil.GenerateUUID(&testing.T{})

	config = bootstrap.Config{
		ClientID:     testsutil.GenerateUUID(&testing.T{}),
		ClientSecret: testsutil.GenerateUUID(&testing.T{}),
		ExternalID:   testsutil.GenerateUUID(&testing.T{}),
		ExternalKey:  testsutil.GenerateUUID(&testing.T{}),
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

	cases := []struct {
		desc            string
		config          bootstrap.Config
		token           string
		session         smqauthn.Session
		userID          string
		domainID        string
		clientErr       error
		createClientErr error
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
			repoCall3 := boot.On("Save", context.Background(), mock.Anything).Return(mock.Anything, tc.saveErr)
			_, err := svc.Add(context.Background(), tc.session, tc.token, tc.config)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			repoCall.Unset()
			repoCall1.Unset()
			repoCall2.Unset()
			repoCall3.Unset()
		})
	}
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
			assert.Equal(t, tc.expectedConfig, cfg, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.expectedConfig, cfg))
			repoCall.Unset()
		})
	}
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

func TestEnableConfig(t *testing.T) {
	svc := newService()

	c := config
	activeConfig := config
	activeConfig.State = bootstrap.Active

	cases := []struct {
		desc        string
		config      bootstrap.Config
		id          string
		session     smqauthn.Session
		userID      string
		domainID    string
		retrieveErr error
		stateErr    error
		err         error
	}{
		{
			desc:        "enable non-existing config",
			config:      c,
			id:          unknown,
			userID:      validID,
			domainID:    domainID,
			retrieveErr: svcerr.ErrNotFound,
			err:         svcerr.ErrNotFound,
		},
		{
			desc:     "enable inactive config",
			config:   c,
			id:       c.ClientID,
			userID:   validID,
			domainID: domainID,
			err:      nil,
		},
		{
			desc:     "enable already active config",
			config:   activeConfig,
			id:       c.ClientID,
			userID:   validID,
			domainID: domainID,
			err:      svcerr.ErrStatusAlreadyAssigned,
		},
		{
			desc:     "enable with repo error",
			config:   c,
			id:       c.ClientID,
			userID:   validID,
			domainID: domainID,
			stateErr: svcerr.ErrUpdateEntity,
			err:      svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			tc.session = smqauthn.Session{UserID: tc.userID, DomainID: tc.domainID, DomainUserID: validID}
			repoCall := boot.On("RetrieveByID", context.Background(), tc.domainID, tc.id).Return(tc.config, tc.retrieveErr)
			repoCall1 := boot.On("ChangeState", context.Background(), mock.Anything, mock.Anything, mock.Anything).Return(tc.stateErr)
			_, err := svc.EnableConfig(context.Background(), tc.session, tc.id)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			repoCall.Unset()
			repoCall1.Unset()
		})
	}
}

func TestDisableConfig(t *testing.T) {
	svc := newService()

	c := config
	activeConfig := config
	activeConfig.State = bootstrap.Active

	cases := []struct {
		desc        string
		config      bootstrap.Config
		id          string
		session     smqauthn.Session
		userID      string
		domainID    string
		retrieveErr error
		stateErr    error
		err         error
	}{
		{
			desc:        "disable non-existing config",
			config:      c,
			id:          unknown,
			userID:      validID,
			domainID:    domainID,
			retrieveErr: svcerr.ErrNotFound,
			err:         svcerr.ErrNotFound,
		},
		{
			desc:     "disable active config",
			config:   activeConfig,
			id:       c.ClientID,
			userID:   validID,
			domainID: domainID,
			err:      nil,
		},
		{
			desc:     "disable already inactive config",
			config:   c,
			id:       c.ClientID,
			userID:   validID,
			domainID: domainID,
			err:      svcerr.ErrStatusAlreadyAssigned,
		},
		{
			desc:     "disable with repo error",
			config:   activeConfig,
			id:       c.ClientID,
			userID:   validID,
			domainID: domainID,
			stateErr: svcerr.ErrUpdateEntity,
			err:      svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			tc.session = smqauthn.Session{UserID: tc.userID, DomainID: tc.domainID, DomainUserID: validID}
			repoCall := boot.On("RetrieveByID", context.Background(), tc.domainID, tc.id).Return(tc.config, tc.retrieveErr)
			repoCall1 := boot.On("ChangeState", context.Background(), mock.Anything, mock.Anything, mock.Anything).Return(tc.stateErr)
			_, err := svc.DisableConfig(context.Background(), tc.session, tc.id)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			repoCall.Unset()
			repoCall1.Unset()
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

