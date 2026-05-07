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
	bootstraphasher "github.com/absmach/magistrala/bootstrap/hasher"
	mocks "github.com/absmach/magistrala/bootstrap/mocks"
	"github.com/absmach/magistrala/internal/testsutil"
	smqauthn "github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	sdkmocks "github.com/absmach/magistrala/pkg/sdk/mocks"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	validToken      = "validToken"
	invalidDomainID = "invalid"
	unknown         = "unknown"
	validID         = "d4ebb847-5d0e-4e46-bdd9-b6aceaaa3a22"
)

var (
	encKey   = []byte("1234567891011121")
	domainID = testsutil.GenerateUUID(&testing.T{})

	config = bootstrap.Config{
		ID:          testsutil.GenerateUUID(&testing.T{}),
		ExternalID:  testsutil.GenerateUUID(&testing.T{}),
		ExternalKey: testsutil.GenerateUUID(&testing.T{}),
		Content:     "config",
	}
)

var (
	boot         *mocks.ConfigRepository
	sdk          *sdkmocks.SDK
	profileRepo  *mocks.ProfileRepository
	bindingStore *mocks.BindingStore
	resolver     *mocks.BindingResolver
	renderer     *mocks.Renderer
)

func newService() bootstrap.Service {
	boot = new(mocks.ConfigRepository)
	sdk = new(sdkmocks.SDK)
	profileRepo = new(mocks.ProfileRepository)
	bindingStore = new(mocks.BindingStore)
	resolver = new(mocks.BindingResolver)
	renderer = new(mocks.Renderer)
	idp := uuid.NewMock()
	return bootstrap.New(boot, profileRepo, bindingStore, resolver, renderer, sdk, bootstraphasher.New(), encKey, idp)
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
	neID.ID = "non-existent"

	cases := []struct {
		desc     string
		config   bootstrap.Config
		token    string
		session  smqauthn.Session
		userID   string
		domainID string
		saveErr  error
		err      error
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
			desc:     "add a config with an invalid ID",
			config:   neID,
			token:    validToken,
			userID:   validID,
			domainID: domainID,
			err:      nil,
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
			repoCall3 := boot.On("Save", context.Background(), mock.Anything).Return(mock.Anything, tc.saveErr)
			_, err := svc.Add(context.Background(), tc.session, tc.token, tc.config)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
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
			configID:     config.ID,
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
			configID:     config.ID,
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
	nonExisting.ID = unknown

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
			desc:     "update a config with status Created",
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
		configID        string
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
			configID:   c.ID,
			clientCert: "newCert",
			clientKey:  "newKey",
			caCert:     "newCert",
			token:      validToken,
			expectedConfig: bootstrap.Config{
				Name:        c.Name,
				ExternalID:  c.ExternalID,
				ExternalKey: c.ExternalKey,
				Content:     c.Content,
				Status:      c.Status,
				DomainID:    c.DomainID,
				ID:          c.ID,
				ClientCert:  "newCert",
				CACert:      "newCert",
				ClientKey:   "newKey",
			},
			err: nil,
		},
		{
			desc:           "update cert for a non-existing config",
			userID:         validID,
			domainID:       domainID,
			configID:       "empty",
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
			cfg, err := svc.UpdateCert(context.Background(), tc.session, tc.configID, tc.clientCert, tc.clientKey, tc.caCert)
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
			c.Status = bootstrap.Active
		}
		saved = append(saved, c)
	}
	cases := []struct {
		desc        string
		config      bootstrap.ConfigsPage
		filter      bootstrap.Filter
		offset      uint64
		limit       uint64
		token       string
		session     smqauthn.Session
		userID      string
		domainID    string
		retrieveErr error
		err         error
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
			desc:     "list configs with failed super admin check",
			config:   bootstrap.ConfigsPage{},
			filter:   bootstrap.Filter{},
			token:    validID,
			session:  smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: validID},
			userID:   validID,
			domainID: domainID,
			offset:   0,
			limit:    10,
			err:      nil,
		},
		{
			desc: "list configs successfully as domain admin",
			config: bootstrap.ConfigsPage{
				Total:   uint64(len(saved)),
				Offset:  0,
				Limit:   10,
				Configs: saved[0:10],
			},
			filter:   bootstrap.Filter{},
			token:    validToken,
			userID:   validID,
			domainID: domainID,
			session:  smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: validID, SuperAdmin: true},
			offset:   0,
			limit:    10,
			err:      nil,
		},
		{
			desc: "list configs successfully as non admin",
			config: bootstrap.ConfigsPage{
				Total:   uint64(len(saved)),
				Offset:  0,
				Limit:   10,
				Configs: saved[0:10],
			},
			filter:   bootstrap.Filter{},
			token:    validToken,
			userID:   validID,
			domainID: domainID,
			session:  smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: validID},
			offset:   0,
			limit:    10,
			err:      nil,
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
			filter:   bootstrap.Filter{PartialMatch: map[string]string{"name": "95"}},
			token:    validToken,
			userID:   validID,
			domainID: domainID,
			session:  smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: validID},
			offset:   0,
			limit:    100,
			err:      nil,
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
			filter:   bootstrap.Filter{},
			token:    validToken,
			userID:   validID,
			domainID: domainID,
			session:  smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: validID},
			offset:   95,
			limit:    10,
			err:      nil,
		},
		{
			desc: "list configs with Active status as super admin",
			config: bootstrap.ConfigsPage{
				Total:   1,
				Offset:  35,
				Limit:   20,
				Configs: []bootstrap.Config{saved[41]},
			},
			filter:   bootstrap.Filter{FullMatch: map[string]string{"status": bootstrap.Active.String()}},
			token:    validToken,
			userID:   validID,
			domainID: domainID,
			session:  smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: validID, SuperAdmin: true},
			offset:   35,
			limit:    20,
			err:      nil,
		},
		{
			desc: "list configs with Active status as domain admin",
			config: bootstrap.ConfigsPage{
				Total:   1,
				Offset:  35,
				Limit:   20,
				Configs: []bootstrap.Config{saved[41]},
			},
			filter:   bootstrap.Filter{FullMatch: map[string]string{"status": bootstrap.Active.String()}},
			token:    validToken,
			userID:   validID,
			domainID: domainID,
			session:  smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: validID, SuperAdmin: true},
			offset:   35,
			limit:    20,
			err:      nil,
		},
		{
			desc: "list configs with Active status as non admin",
			config: bootstrap.ConfigsPage{
				Total:   1,
				Offset:  35,
				Limit:   20,
				Configs: []bootstrap.Config{saved[41]},
			},
			filter:   bootstrap.Filter{FullMatch: map[string]string{"status": bootstrap.Active.String()}},
			token:    validToken,
			userID:   validID,
			domainID: domainID,
			session:  smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: validID},
			offset:   35,
			limit:    20,
			err:      nil,
		},
		{
			desc:     "list configs with empty result",
			config:   bootstrap.ConfigsPage{},
			filter:   bootstrap.Filter{},
			offset:   0,
			limit:    10,
			token:    validToken,
			userID:   validID,
			domainID: domainID,
			session:  smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: validID},
			err:      nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := boot.On("RetrieveAll", context.Background(), mock.Anything, tc.filter, tc.offset, tc.limit).Return(tc.config, tc.retrieveErr)

			result, err := svc.List(context.Background(), tc.session, tc.filter, tc.offset, tc.limit)
			assert.ElementsMatch(t, tc.config.Configs, result.Configs, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.config.Configs, result.Configs))
			assert.Equal(t, tc.config.Total, result.Total, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.config.Total, result.Total))
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
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
			id:       c.ID,
			token:    validToken,
			userID:   validID,
			domainID: domainID,
			err:      nil,
		},
		{
			desc:     "remove removed config",
			id:       c.ID,
			token:    validToken,
			userID:   validID,
			domainID: domainID,
			err:      nil,
		},
		{
			desc:      "remove a config with failed remove",
			id:        c.ID,
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
	c.Status = bootstrap.Active
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

func TestBootstrapRender(t *testing.T) {
	profile := bootstrap.Profile{
		ID:              testsutil.GenerateUUID(&testing.T{}),
		DomainID:        domainID,
		Name:            "gateway-profile",
		TemplateFormat:  bootstrap.TemplateFormatGoTemplate,
		ContentTemplate: `{"mode":"profile"}`,
	}
	bindings := []bootstrap.BindingSnapshot{
		{
			ConfigID:   config.ID,
			Slot:       "mqtt_client",
			Type:       "client",
			ResourceID: config.ID,
			Snapshot: map[string]any{
				"id": config.ID,
			},
		},
	}

	cases := []struct {
		desc        string
		cfg         bootstrap.Config
		rendererOut []byte
		rendererErr error
		rendered    string
		err         error
	}{
		{
			desc: "bootstrap renders assigned profile content",
			cfg: func() bootstrap.Config {
				cfg := config
				cfg.DomainID = domainID
				cfg.ProfileID = profile.ID
				cfg.Status = bootstrap.Active
				cfg.Content = "legacy"
				return cfg
			}(),
			rendererOut: []byte(`{"mode":"profile"}`),
			rendered:    `{"mode":"profile"}`,
		},
		{
			desc: "bootstrap falls back to legacy content when no profile is assigned",
			cfg: func() bootstrap.Config {
				cfg := config
				cfg.DomainID = domainID
				cfg.Status = bootstrap.Active
				cfg.Content = "legacy"
				return cfg
			}(),
			rendered: "legacy",
		},
		{
			desc: "bootstrap fails when renderer fails",
			cfg: func() bootstrap.Config {
				cfg := config
				cfg.DomainID = domainID
				cfg.ProfileID = profile.ID
				cfg.Status = bootstrap.Active
				return cfg
			}(),
			rendererErr: errors.New("render failed"),
			err:         errors.New("render failed"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svc := newService()
			repoCall := boot.On("RetrieveByExternalID", context.Background(), tc.cfg.ExternalID).Return(tc.cfg, nil)

			var prCall, bsCall, rndCall *mock.Call
			if tc.cfg.ProfileID != "" {
				prCall = profileRepo.On("RetrieveByID", context.Background(), tc.cfg.DomainID, tc.cfg.ProfileID).Return(profile, nil)
				bsCall = bindingStore.On("Retrieve", context.Background(), tc.cfg.ID).Return(bindings, nil)
				rndCall = renderer.On("Render", mock.Anything, mock.Anything, mock.Anything).Return(tc.rendererOut, tc.rendererErr)
			}

			res, err := svc.Bootstrap(context.Background(), tc.cfg.ExternalKey, tc.cfg.ExternalID, false)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.err, err))
			if tc.err == nil {
				assert.Equal(t, tc.rendered, res.Content, fmt.Sprintf("%s: expected rendered content %q got %q\n", tc.desc, tc.rendered, res.Content))
			}

			repoCall.Unset()
			if prCall != nil {
				prCall.Unset()
			}
			if bsCall != nil {
				bsCall.Unset()
			}
			if rndCall != nil {
				rndCall.Unset()
			}
		})
	}
}

func TestEnableConfig(t *testing.T) {
	svc := newService()

	c := config
	activeConfig := config
	activeConfig.Status = bootstrap.Active
	inactiveConfig := config
	inactiveConfig.Status = bootstrap.Inactive

	cases := []struct {
		desc        string
		config      bootstrap.Config
		id          string
		session     smqauthn.Session
		userID      string
		domainID    string
		retrieveErr error
		statusErr   error
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
			config:   inactiveConfig,
			id:       c.ID,
			userID:   validID,
			domainID: domainID,
			err:      nil,
		},
		{
			desc:     "enable already active config",
			config:   activeConfig,
			id:       c.ID,
			userID:   validID,
			domainID: domainID,
			err:      nil,
		},
		{
			desc:      "enable with repo error",
			config:    inactiveConfig,
			id:        c.ID,
			userID:    validID,
			domainID:  domainID,
			statusErr: svcerr.ErrUpdateEntity,
			err:       svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			tc.session = smqauthn.Session{UserID: tc.userID, DomainID: tc.domainID, DomainUserID: validID}
			repoCall := boot.On("RetrieveByID", context.Background(), tc.domainID, tc.id).Return(tc.config, tc.retrieveErr)
			repoCall1 := boot.On("ChangeStatus", context.Background(), mock.Anything, mock.Anything, mock.Anything).Return(tc.statusErr)
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
	activeConfig.Status = bootstrap.Active
	inactiveConfig := config
	inactiveConfig.Status = bootstrap.Inactive

	cases := []struct {
		desc        string
		config      bootstrap.Config
		id          string
		session     smqauthn.Session
		userID      string
		domainID    string
		retrieveErr error
		statusErr   error
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
			id:       c.ID,
			userID:   validID,
			domainID: domainID,
			err:      nil,
		},
		{
			desc:     "disable already inactive config",
			config:   inactiveConfig,
			id:       c.ID,
			userID:   validID,
			domainID: domainID,
			err:      nil,
		},
		{
			desc:      "disable with repo error",
			config:    activeConfig,
			id:        c.ID,
			userID:    validID,
			domainID:  domainID,
			statusErr: svcerr.ErrUpdateEntity,
			err:       svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			tc.session = smqauthn.Session{UserID: tc.userID, DomainID: tc.domainID, DomainUserID: validID}
			repoCall := boot.On("RetrieveByID", context.Background(), tc.domainID, tc.id).Return(tc.config, tc.retrieveErr)
			repoCall1 := boot.On("ChangeStatus", context.Background(), mock.Anything, mock.Anything, mock.Anything).Return(tc.statusErr)
			_, err := svc.DisableConfig(context.Background(), tc.session, tc.id)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			repoCall.Unset()
			repoCall1.Unset()
		})
	}
}

func TestAssignProfile(t *testing.T) {
	profile := bootstrap.Profile{
		ID:             testsutil.GenerateUUID(t),
		DomainID:       domainID,
		Name:           "gateway-profile",
		TemplateFormat: bootstrap.TemplateFormatGoTemplate,
		Version:        1,
	}

	cases := []struct {
		desc         string
		configID     string
		profileID    string
		retrieveErr  error
		assignErr    error
		expectedErr  error
		expectAssign bool
	}{
		{
			desc:         "assign profile to enrollment",
			configID:     config.ID,
			profileID:    profile.ID,
			expectAssign: true,
		},
		{
			desc:        "assign profile with missing profile",
			configID:    config.ID,
			profileID:   profile.ID,
			retrieveErr: svcerr.ErrNotFound,
			expectedErr: svcerr.ErrNotFound,
		},
		{
			desc:         "assign profile with repository error",
			configID:     config.ID,
			profileID:    profile.ID,
			assignErr:    svcerr.ErrUpdateEntity,
			expectedErr:  svcerr.ErrUpdateEntity,
			expectAssign: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svc := newService()
			session := smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: validID}

			prCall := profileRepo.On("RetrieveByID", context.Background(), domainID, tc.profileID).Return(profile, tc.retrieveErr)

			var assignCall *mock.Call
			if tc.expectAssign {
				assignCall = boot.On("AssignProfile", context.Background(), domainID, tc.configID, tc.profileID).Return(tc.assignErr)
			}

			err := svc.AssignProfile(context.Background(), session, tc.configID, tc.profileID)
			assert.True(t, errors.Contains(err, tc.expectedErr), fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.expectedErr, err))

			prCall.Unset()
			if assignCall != nil {
				assignCall.Unset()
			}
		})
	}
}

func TestCreateProfile(t *testing.T) {
	session := smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: validID}

	validProfile := bootstrap.Profile{
		Name:           "test-profile",
		TemplateFormat: bootstrap.TemplateFormatGoTemplate,
	}

	cases := []struct {
		desc    string
		profile bootstrap.Profile
		saveErr error
		err     error
	}{
		{
			desc:    "create profile successfully",
			profile: validProfile,
		},
		{
			desc:    "create profile defaults to go-template format",
			profile: bootstrap.Profile{Name: "no-format"},
		},
		{
			desc: "create profile with invalid slot: empty name",
			profile: bootstrap.Profile{
				Name:         "test",
				BindingSlots: []bootstrap.BindingSlot{{Name: "", Type: "client"}},
			},
			err: errors.New("invalid binding slot: slot name is required"),
		},
		{
			desc: "create profile with invalid slot: empty type",
			profile: bootstrap.Profile{
				Name:         "test",
				BindingSlots: []bootstrap.BindingSlot{{Name: "mqtt", Type: ""}},
			},
			err: errors.New("invalid binding slot: slot \"mqtt\" type is required"),
		},
		{
			desc: "create profile with duplicate slot names",
			profile: bootstrap.Profile{
				Name: "test",
				BindingSlots: []bootstrap.BindingSlot{
					{Name: "mqtt", Type: "client"},
					{Name: "mqtt", Type: "channel"},
				},
			},
			err: errors.New("invalid binding slot: duplicate slot \"mqtt\""),
		},
		{
			desc: "create profile with invalid template syntax",
			profile: bootstrap.Profile{
				Name:            "test",
				ContentTemplate: `{{ index .Vars \"mqtt_url\" }}`,
			},
			err: bootstrap.ErrRenderFailed,
		},
		{
			desc:    "create profile with repository save error",
			profile: validProfile,
			saveErr: svcerr.ErrCreateEntity,
			err:     svcerr.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svc := newService()
			saveCall := profileRepo.EXPECT().Save(mock.Anything, mock.Anything).RunAndReturn(
				func(_ context.Context, p bootstrap.Profile) (bootstrap.Profile, error) {
					return p, tc.saveErr
				})
			saved, err := svc.CreateProfile(context.Background(), session, tc.profile)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.err, err))
			if tc.err == nil {
				assert.NotEmpty(t, saved.ID, fmt.Sprintf("%s: expected non-empty profile ID\n", tc.desc))
				assert.Equal(t, domainID, saved.DomainID, fmt.Sprintf("%s: expected domain ID %s got %s\n", tc.desc, domainID, saved.DomainID))
				assert.Equal(t, bootstrap.TemplateFormatGoTemplate, saved.TemplateFormat, fmt.Sprintf("%s: expected go-template format\n", tc.desc))
				assert.Equal(t, 1, saved.Version, fmt.Sprintf("%s: expected version 1\n", tc.desc))
			}
			saveCall.Unset()
		})
	}
}

func TestViewProfile(t *testing.T) {
	session := smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: validID}

	profile := bootstrap.Profile{
		ID:             testsutil.GenerateUUID(t),
		DomainID:       domainID,
		Name:           "view-profile",
		TemplateFormat: bootstrap.TemplateFormatGoTemplate,
		Version:        1,
	}

	cases := []struct {
		desc        string
		profileID   string
		retrieveErr error
		err         error
	}{
		{
			desc:      "view profile successfully",
			profileID: profile.ID,
		},
		{
			desc:        "view non-existing profile",
			profileID:   unknown,
			retrieveErr: svcerr.ErrNotFound,
			err:         svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svc := newService()
			prCall := profileRepo.On("RetrieveByID", context.Background(), domainID, tc.profileID).Return(profile, tc.retrieveErr)
			got, err := svc.ViewProfile(context.Background(), session, tc.profileID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.err, err))
			if tc.err == nil {
				assert.Equal(t, profile, got, fmt.Sprintf("%s: expected profile %v got %v\n", tc.desc, profile, got))
			}
			prCall.Unset()
		})
	}
}

func TestUpdateProfile(t *testing.T) {
	session := smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: validID}

	validProfile := bootstrap.Profile{
		ID:             testsutil.GenerateUUID(t),
		DomainID:       domainID,
		Name:           "updated-profile",
		TemplateFormat: bootstrap.TemplateFormatGoTemplate,
	}

	cases := []struct {
		desc      string
		profile   bootstrap.Profile
		updateErr error
		err       error
	}{
		{
			desc:    "update profile successfully",
			profile: validProfile,
		},
		{
			desc:    "update profile defaults to go-template format",
			profile: bootstrap.Profile{ID: validProfile.ID, Name: "no-format"},
		},
		{
			desc: "update profile with invalid slot: empty type",
			profile: bootstrap.Profile{
				ID:           validProfile.ID,
				Name:         "test",
				BindingSlots: []bootstrap.BindingSlot{{Name: "mqtt", Type: ""}},
			},
			err: errors.New("invalid binding slot: slot \"mqtt\" type is required"),
		},
		{
			desc: "update profile with duplicate slot names",
			profile: bootstrap.Profile{
				ID:   validProfile.ID,
				Name: "test",
				BindingSlots: []bootstrap.BindingSlot{
					{Name: "slot1", Type: "client"},
					{Name: "slot1", Type: "channel"},
				},
			},
			err: errors.New("invalid binding slot: duplicate slot \"slot1\""),
		},
		{
			desc: "update profile with invalid template syntax",
			profile: bootstrap.Profile{
				ID:              validProfile.ID,
				Name:            "test",
				ContentTemplate: `{{ index .Vars \"mqtt_url\" }}`,
			},
			err: bootstrap.ErrRenderFailed,
		},
		{
			desc:      "update profile with repository error",
			profile:   validProfile,
			updateErr: svcerr.ErrUpdateEntity,
			err:       svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svc := newService()
			updateCall := profileRepo.On("Update", context.Background(), mock.Anything).Return(tc.updateErr)
			err := svc.UpdateProfile(context.Background(), session, tc.profile)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.err, err))
			updateCall.Unset()
		})
	}
}

func TestListProfiles(t *testing.T) {
	session := smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: validID}

	profiles := []bootstrap.Profile{
		{ID: testsutil.GenerateUUID(t), DomainID: domainID, Name: "p1", TemplateFormat: bootstrap.TemplateFormatGoTemplate, Version: 1},
		{ID: testsutil.GenerateUUID(t), DomainID: domainID, Name: "p2", TemplateFormat: bootstrap.TemplateFormatGoTemplate, Version: 1},
	}
	page := bootstrap.ProfilesPage{Total: 2, Offset: 0, Limit: 10, Profiles: profiles}

	cases := []struct {
		desc    string
		offset  uint64
		limit   uint64
		page    bootstrap.ProfilesPage
		listErr error
		err     error
	}{
		{
			desc:  "list profiles successfully",
			limit: 10,
			page:  page,
		},
		{
			desc:    "list profiles with repository error",
			limit:   10,
			listErr: svcerr.ErrViewEntity,
			err:     svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svc := newService()
			listCall := profileRepo.On("RetrieveAll", context.Background(), domainID, tc.offset, tc.limit).Return(tc.page, tc.listErr)
			got, err := svc.ListProfiles(context.Background(), session, tc.offset, tc.limit)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.err, err))
			if tc.err == nil {
				assert.Equal(t, tc.page, got, fmt.Sprintf("%s: expected page %v got %v\n", tc.desc, tc.page, got))
			}
			listCall.Unset()
		})
	}
}

func TestDeleteProfile(t *testing.T) {
	session := smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: validID}
	profileID := testsutil.GenerateUUID(t)

	cases := []struct {
		desc      string
		profileID string
		deleteErr error
		err       error
	}{
		{
			desc:      "delete profile successfully",
			profileID: profileID,
		},
		{
			desc:      "delete profile with repository error",
			profileID: profileID,
			deleteErr: svcerr.ErrRemoveEntity,
			err:       svcerr.ErrRemoveEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svc := newService()
			deleteCall := profileRepo.On("Delete", context.Background(), domainID, tc.profileID).Return(tc.deleteErr)
			err := svc.DeleteProfile(context.Background(), session, tc.profileID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.err, err))
			deleteCall.Unset()
		})
	}
}

func TestBindResources(t *testing.T) {
	session := smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: validID}

	profile := bootstrap.Profile{
		ID:       testsutil.GenerateUUID(t),
		DomainID: domainID,
		Name:     "bind-profile",
		BindingSlots: []bootstrap.BindingSlot{
			{Name: "mqtt", Type: "client", Required: true},
		},
	}

	channelProfile := bootstrap.Profile{
		ID:       testsutil.GenerateUUID(t),
		DomainID: domainID,
		Name:     "channel-profile",
		BindingSlots: []bootstrap.BindingSlot{
			{Name: "data", Type: "channel", Required: true},
		},
	}

	cfg := bootstrap.Config{
		ID:        config.ID,
		DomainID:  domainID,
		ProfileID: profile.ID,
	}

	channelCfg := bootstrap.Config{
		ID:        config.ID,
		DomainID:  domainID,
		ProfileID: channelProfile.ID,
	}

	snapshot := bootstrap.BindingSnapshot{
		ConfigID:   config.ID,
		Slot:       "mqtt",
		Type:       "client",
		ResourceID: validID,
		Snapshot:   map[string]any{"id": validID},
	}

	channelSnapshot := bootstrap.BindingSnapshot{
		ConfigID:   config.ID,
		Slot:       "data",
		Type:       "channel",
		ResourceID: validID,
		Snapshot:   map[string]any{"id": validID},
	}

	requested := []bootstrap.BindingRequest{
		{Slot: "mqtt", Type: "client", ResourceID: validID},
	}

	channelRequested := []bootstrap.BindingRequest{
		{Slot: "data", Type: "channel", ResourceID: validID},
	}

	cases := []struct {
		desc        string
		configID    string
		bindings    []bootstrap.BindingRequest
		cfgErr      error
		prErr       error
		resolveErr  error
		retrieveErr error
		saveErr     error
		snapshots   []bootstrap.BindingSnapshot
		useChannel  bool
		err         error
	}{
		{
			desc:     "bind resources with config not found",
			configID: config.ID,
			bindings: requested,
			cfgErr:   svcerr.ErrNotFound,
			err:      svcerr.ErrNotFound,
		},
		{
			desc:     "bind resources with profile not found",
			configID: config.ID,
			bindings: requested,
			prErr:    svcerr.ErrNotFound,
			err:      svcerr.ErrNotFound,
		},
		{
			desc:     "bind resources with unknown slot",
			configID: config.ID,
			bindings: []bootstrap.BindingRequest{{Slot: "unknown", Type: "client", ResourceID: validID}},
			err:      errors.New("invalid binding slot: unknown slot \"unknown\""),
		},
		{
			desc:     "bind resources with wrong slot type",
			configID: config.ID,
			bindings: []bootstrap.BindingRequest{{Slot: "mqtt", Type: "channel", ResourceID: validID}},
			err:      errors.New("invalid binding slot: slot \"mqtt\" expects \"client\", got \"channel\""),
		},
		{
			desc:       "bind resources with resolver error",
			configID:   config.ID,
			bindings:   requested,
			resolveErr: errors.New("resolve failed"),
			err:        errors.New("resolve failed"),
		},
		{
			desc:        "bind resources with binding store retrieve error",
			configID:    config.ID,
			bindings:    requested,
			snapshots:   []bootstrap.BindingSnapshot{snapshot},
			retrieveErr: svcerr.ErrViewEntity,
			err:         svcerr.ErrViewEntity,
		},
		{
			desc:     "bind resources with required slot not satisfied",
			configID: config.ID,
			bindings: requested,
			snapshots: []bootstrap.BindingSnapshot{
				{ConfigID: config.ID, Slot: "mqtt", Type: "channel", ResourceID: validID},
			},
			err: errors.New("invalid binding slot: slot \"mqtt\" expects \"client\", got \"channel\""),
		},
		{
			desc:      "bind resources with save error",
			configID:  config.ID,
			bindings:  requested,
			snapshots: []bootstrap.BindingSnapshot{snapshot},
			saveErr:   svcerr.ErrCreateEntity,
			err:       svcerr.ErrCreateEntity,
		},
		{
			desc:      "bind resources successfully",
			configID:  config.ID,
			bindings:  requested,
			snapshots: []bootstrap.BindingSnapshot{snapshot},
		},
		{
			desc:       "bind channel resource successfully",
			configID:   config.ID,
			bindings:   channelRequested,
			snapshots:  []bootstrap.BindingSnapshot{channelSnapshot},
			useChannel: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svc := newService()
			activeCfg, activeProfile := cfg, profile
			if tc.useChannel {
				activeCfg = channelCfg
				activeProfile = channelProfile
			}
			boot.On("RetrieveByID", context.Background(), domainID, tc.configID).Return(activeCfg, tc.cfgErr)
			profileRepo.On("RetrieveByID", context.Background(), domainID, activeProfile.ID).Return(activeProfile, tc.prErr)
			resolver.On("Resolve", context.Background(), mock.Anything).Return(tc.snapshots, tc.resolveErr)
			bindingStore.On("Retrieve", context.Background(), tc.configID).Return([]bootstrap.BindingSnapshot{}, tc.retrieveErr)
			bindingStore.On("Save", context.Background(), tc.configID, mock.Anything).Return(tc.saveErr)

			err := svc.BindResources(context.Background(), session, validToken, tc.configID, tc.bindings)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.err, err))
		})
	}
}

func TestListBindings(t *testing.T) {
	session := smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: validID}

	snapshots := []bootstrap.BindingSnapshot{
		{ConfigID: config.ID, Slot: "mqtt", Type: "client", ResourceID: validID, Snapshot: map[string]any{"id": validID}},
	}

	cases := []struct {
		desc        string
		configID    string
		cfgErr      error
		bindings    []bootstrap.BindingSnapshot
		retrieveErr error
		err         error
	}{
		{
			desc:     "list bindings with config not found",
			configID: config.ID,
			cfgErr:   svcerr.ErrNotFound,
			err:      svcerr.ErrNotFound,
		},
		{
			desc:        "list bindings with retrieve error",
			configID:    config.ID,
			retrieveErr: svcerr.ErrViewEntity,
			err:         svcerr.ErrViewEntity,
		},
		{
			desc:     "list bindings successfully",
			configID: config.ID,
			bindings: snapshots,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svc := newService()
			boot.On("RetrieveByID", context.Background(), domainID, tc.configID).Return(config, tc.cfgErr)
			bindingStore.On("Retrieve", context.Background(), tc.configID).Return(tc.bindings, tc.retrieveErr)

			got, err := svc.ListBindings(context.Background(), session, tc.configID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.err, err))
			if tc.err == nil {
				assert.Len(t, got, len(tc.bindings), fmt.Sprintf("%s: expected %d bindings got %d\n", tc.desc, len(tc.bindings), len(got)))
			}
		})
	}
}

func TestRefreshBindings(t *testing.T) {
	session := smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: validID}

	profile := bootstrap.Profile{
		ID:       testsutil.GenerateUUID(t),
		DomainID: domainID,
		Name:     "refresh-profile",
		BindingSlots: []bootstrap.BindingSlot{
			{Name: "mqtt", Type: "client", Required: true},
		},
	}

	cfg := bootstrap.Config{
		ID:        config.ID,
		DomainID:  domainID,
		ProfileID: profile.ID,
	}

	existing := []bootstrap.BindingSnapshot{
		{ConfigID: config.ID, Slot: "mqtt", Type: "client", ResourceID: validID},
	}

	refreshed := []bootstrap.BindingSnapshot{
		{ConfigID: config.ID, Slot: "mqtt", Type: "client", ResourceID: validID, Snapshot: map[string]any{"id": validID}},
	}

	cases := []struct {
		desc        string
		configID    string
		cfgErr      error
		prErr       error
		existing    []bootstrap.BindingSnapshot
		retrieveErr error
		snapshots   []bootstrap.BindingSnapshot
		resolveErr  error
		saveErr     error
		err         error
	}{
		{
			desc:     "refresh bindings with config not found",
			configID: config.ID,
			cfgErr:   svcerr.ErrNotFound,
			err:      svcerr.ErrNotFound,
		},
		{
			desc:     "refresh bindings with profile not found",
			configID: config.ID,
			prErr:    svcerr.ErrNotFound,
			err:      svcerr.ErrNotFound,
		},
		{
			desc:        "refresh bindings with retrieve error",
			configID:    config.ID,
			retrieveErr: svcerr.ErrViewEntity,
			err:         svcerr.ErrViewEntity,
		},
		{
			desc:     "refresh bindings with no existing bindings is a no-op",
			configID: config.ID,
		},
		{
			desc:       "refresh bindings with resolver error",
			configID:   config.ID,
			existing:   existing,
			resolveErr: errors.New("resolve failed"),
			err:        errors.New("resolve failed"),
		},
		{
			desc:     "refresh bindings with required binding missing after refresh",
			configID: config.ID,
			existing: existing,
			snapshots: []bootstrap.BindingSnapshot{
				{ConfigID: config.ID, Slot: "mqtt", Type: "channel", ResourceID: validID},
			},
			err: errors.New("invalid binding slot: slot \"mqtt\" expects \"client\", got \"channel\""),
		},
		{
			desc:      "refresh bindings with save error",
			configID:  config.ID,
			existing:  existing,
			snapshots: refreshed,
			saveErr:   svcerr.ErrCreateEntity,
			err:       svcerr.ErrCreateEntity,
		},
		{
			desc:      "refresh bindings successfully",
			configID:  config.ID,
			existing:  existing,
			snapshots: refreshed,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svc := newService()
			boot.On("RetrieveByID", context.Background(), domainID, tc.configID).Return(cfg, tc.cfgErr)
			profileRepo.On("RetrieveByID", context.Background(), domainID, profile.ID).Return(profile, tc.prErr)
			bindingStore.On("Retrieve", context.Background(), tc.configID).Return(tc.existing, tc.retrieveErr)
			resolver.On("Resolve", context.Background(), mock.Anything).Return(tc.snapshots, tc.resolveErr)
			bindingStore.On("Save", context.Background(), tc.configID, mock.Anything).Return(tc.saveErr)

			err := svc.RefreshBindings(context.Background(), session, validToken, tc.configID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.err, err))
		})
	}
}
