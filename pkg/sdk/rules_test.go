// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/absmach/supermq/pkg/sdk"
	"github.com/absmach/supermq/re"
	"github.com/absmach/supermq/re/api"
	remocks "github.com/absmach/supermq/re/mocks"
	smqlog "github.com/absmach/supermq/logger"
	smqauthn "github.com/absmach/supermq/pkg/authn"
	authnmocks "github.com/absmach/supermq/pkg/authn/mocks"
	"github.com/absmach/supermq/pkg/roles"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const ruleID = "rule-1"

var testRule = sdk.Rule{
	ID:           ruleID,
	Name:         "temperature-rule",
	InputChannel: "chan-1",
	InputTopic:   "sensors/temperature",
	Status:       "enabled",
	Tags:         []string{"temperature", "alerts"},
}

func setupRules() (*httptest.Server, *remocks.Service, *authnmocks.Authentication) {
	rsvc := new(remocks.Service)
	log := smqlog.NewMock()
	authn := new(authnmocks.Authentication)
	am := smqauthn.NewAuthNMiddleware(authn, smqauthn.WithAllowUnverifiedUser(true))
	mux := chi.NewRouter()
	_ = api.MakeHandler(rsvc, am, mux, log, "")
	return httptest.NewServer(mux), rsvc, authn
}

func TestAddRule(t *testing.T) {
	rs, rsvc, auth := setupRules()
	defer rs.Close()

	conf := sdk.Config{
		RulesEngineURL: rs.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	svcRule := re.Rule{
		ID:           ruleID,
		Name:         "temperature-rule",
		InputChannel: "chan-1",
		Status:       re.EnabledStatus,
	}

	cases := []struct {
		desc            string
		rule            sdk.Rule
		token           string
		session         smqauthn.Session
		svcRes          re.Rule
		svcErr          error
		authenticateErr error
		wantErr         bool
	}{
		{
			desc:   "add rule successfully",
			rule:   sdk.Rule{Name: "temp-rule", InputChannel: "chan-1"},
			token:  validToken,
			svcRes: svcRule,
		},
		{
			desc:    "add rule with empty token",
			rule:    sdk.Rule{Name: "temp-rule"},
			token:   "",
			wantErr: true,
		},
		{
			desc:    "add rule with bad request",
			rule:    sdk.Rule{},
			token:   validToken,
			svcErr:  errors.New("bad request"),
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := rsvc.On("AddRule", mock.Anything, tc.session, mock.Anything).Return(tc.svcRes, []roles.RoleProvision(nil), tc.svcErr)
			result, err := mgsdk.AddRule(context.Background(), tc.rule, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
			if !tc.wantErr {
				assert.NotEmpty(t, result.ID)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestViewRule(t *testing.T) {
	rs, rsvc, auth := setupRules()
	defer rs.Close()

	conf := sdk.Config{
		RulesEngineURL: rs.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	svcRule := re.Rule{
		ID:           ruleID,
		Name:         "temperature-rule",
		InputChannel: "chan-1",
		Status:       re.EnabledStatus,
	}

	cases := []struct {
		desc            string
		id              string
		token           string
		session         smqauthn.Session
		svcRes          re.Rule
		svcErr          error
		authenticateErr error
		wantErr         bool
	}{
		{
			desc:   "view rule successfully",
			id:     ruleID,
			token:  validToken,
			svcRes: svcRule,
		},
		{
			desc:    "view rule with empty token",
			id:      ruleID,
			token:   "",
			wantErr: true,
		},
		{
			desc:    "view non-existent rule",
			id:      "non-existent",
			token:   validToken,
			svcErr:  errors.New("not found"),
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := rsvc.On("ViewRule", mock.Anything, tc.session, tc.id, mock.Anything).Return(tc.svcRes, tc.svcErr)
			result, err := mgsdk.ViewRule(context.Background(), tc.id, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
			if !tc.wantErr {
				assert.NotEmpty(t, result.ID)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestUpdateRule(t *testing.T) {
	rs, rsvc, auth := setupRules()
	defer rs.Close()

	conf := sdk.Config{
		RulesEngineURL: rs.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	updatedRule := testRule
	updatedRule.Name = "updated-rule"

	svcRule := re.Rule{
		ID:           ruleID,
		Name:         "updated-rule",
		InputChannel: "chan-1",
		InputTopic:   "sensors/temperature",
		Status:       re.EnabledStatus,
		Tags:         []string{"temperature", "alerts"},
	}

	cases := []struct {
		desc            string
		rule            sdk.Rule
		token           string
		session         smqauthn.Session
		svcRes          re.Rule
		svcErr          error
		authenticateErr error
		wantErr         bool
	}{
		{
			desc:   "update rule successfully",
			rule:   updatedRule,
			token:  validToken,
			svcRes: svcRule,
		},
		{
			desc:    "update rule with empty token",
			rule:    updatedRule,
			token:   "",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := rsvc.On("UpdateRule", mock.Anything, tc.session, mock.Anything).Return(tc.svcRes, tc.svcErr)
			result, err := mgsdk.UpdateRule(context.Background(), tc.rule, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
			if !tc.wantErr {
				assert.NotEmpty(t, result.ID)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestUpdateRuleTags(t *testing.T) {
	rs, rsvc, auth := setupRules()
	defer rs.Close()

	conf := sdk.Config{
		RulesEngineURL: rs.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	svcRule := re.Rule{
		ID:   ruleID,
		Tags: []string{"new-tag"},
	}

	cases := []struct {
		desc            string
		rule            sdk.Rule
		token           string
		session         smqauthn.Session
		svcRes          re.Rule
		svcErr          error
		authenticateErr error
		wantErr         bool
	}{
		{
			desc:   "update rule tags successfully",
			rule:   sdk.Rule{ID: ruleID, Tags: []string{"new-tag"}},
			token:  validToken,
			svcRes: svcRule,
		},
		{
			desc:    "update rule tags with empty token",
			rule:    sdk.Rule{ID: ruleID},
			token:   "",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := rsvc.On("UpdateRuleTags", mock.Anything, tc.session, mock.Anything).Return(tc.svcRes, tc.svcErr)
			result, err := mgsdk.UpdateRuleTags(context.Background(), tc.rule, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
			if !tc.wantErr {
				assert.NotEmpty(t, result.ID)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestUpdateRuleSchedule(t *testing.T) {
	rs, rsvc, auth := setupRules()
	defer rs.Close()

	conf := sdk.Config{
		RulesEngineURL: rs.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	svcRule := re.Rule{
		ID: ruleID,
	}

	cases := []struct {
		desc            string
		rule            sdk.Rule
		token           string
		session         smqauthn.Session
		svcRes          re.Rule
		svcErr          error
		authenticateErr error
		wantErr         bool
	}{
		{
			desc:   "update rule schedule successfully",
			rule:   sdk.Rule{ID: ruleID, Schedule: map[string]any{"cron": "0 * * * *"}},
			token:  validToken,
			svcRes: svcRule,
		},
		{
			desc:    "update rule schedule with empty token",
			rule:    sdk.Rule{ID: ruleID},
			token:   "",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := rsvc.On("UpdateRuleSchedule", mock.Anything, tc.session, mock.Anything).Return(tc.svcRes, tc.svcErr)
			result, err := mgsdk.UpdateRuleSchedule(context.Background(), tc.rule, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
			if !tc.wantErr {
				assert.NotEmpty(t, result.ID)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestListRules(t *testing.T) {
	rs, rsvc, auth := setupRules()
	defer rs.Close()

	conf := sdk.Config{
		RulesEngineURL: rs.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	svcPage := re.Page{}

	cases := []struct {
		desc            string
		pm              sdk.PageMetadata
		token           string
		session         smqauthn.Session
		svcRes          re.Page
		svcErr          error
		authenticateErr error
		wantErr         bool
	}{
		{
			desc:   "list rules successfully",
			pm:     sdk.PageMetadata{Offset: 0, Limit: 10},
			token:  validToken,
			svcRes: svcPage,
		},
		{
			desc: "list rules with filters",
			pm: sdk.PageMetadata{
				Limit:        5,
				Name:         "temp",
				Status:       "enabled",
				InputChannel: "chan-1",
				Tag:          "temperature",
				Dir:          "desc",
				Order:        "created_at",
			},
			token:  validToken,
			svcRes: svcPage,
		},
		{
			desc:   "list rules with empty metadata excludes filter params",
			pm:     sdk.PageMetadata{},
			token:  validToken,
			svcRes: re.Page{},
		},
		{
			desc:    "list rules with empty token",
			pm:      sdk.PageMetadata{Limit: 10},
			token:   "",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := rsvc.On("ListRules", mock.Anything, tc.session, mock.Anything).Return(tc.svcRes, tc.svcErr)
			result, err := mgsdk.ListRules(context.Background(), tc.pm, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
			if !tc.wantErr {
				assert.NotNil(t, result)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestEnableRule(t *testing.T) {
	rs, rsvc, auth := setupRules()
	defer rs.Close()

	conf := sdk.Config{
		RulesEngineURL: rs.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	svcRule := re.Rule{
		ID:     ruleID,
		Status: re.EnabledStatus,
	}

	cases := []struct {
		desc            string
		id              string
		token           string
		session         smqauthn.Session
		svcRes          re.Rule
		svcErr          error
		authenticateErr error
		wantErr         bool
	}{
		{
			desc:   "enable rule successfully",
			id:     ruleID,
			token:  validToken,
			svcRes: svcRule,
		},
		{
			desc:    "enable rule with empty token",
			id:      ruleID,
			token:   "",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := rsvc.On("EnableRule", mock.Anything, tc.session, tc.id).Return(tc.svcRes, tc.svcErr)
			result, err := mgsdk.EnableRule(context.Background(), tc.id, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
			if !tc.wantErr {
				assert.NotEmpty(t, result.ID)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestDisableRule(t *testing.T) {
	rs, rsvc, auth := setupRules()
	defer rs.Close()

	conf := sdk.Config{
		RulesEngineURL: rs.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	svcRule := re.Rule{
		ID:     ruleID,
		Status: re.DisabledStatus,
	}

	cases := []struct {
		desc            string
		id              string
		token           string
		session         smqauthn.Session
		svcRes          re.Rule
		svcErr          error
		authenticateErr error
		wantErr         bool
	}{
		{
			desc:   "disable rule successfully",
			id:     ruleID,
			token:  validToken,
			svcRes: svcRule,
		},
		{
			desc:    "disable rule with empty token",
			id:      ruleID,
			token:   "",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := rsvc.On("DisableRule", mock.Anything, tc.session, tc.id).Return(tc.svcRes, tc.svcErr)
			result, err := mgsdk.DisableRule(context.Background(), tc.id, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
			if !tc.wantErr {
				assert.NotEmpty(t, result.ID)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestRemoveRule(t *testing.T) {
	rs, rsvc, auth := setupRules()
	defer rs.Close()

	conf := sdk.Config{
		RulesEngineURL: rs.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		id              string
		token           string
		session         smqauthn.Session
		svcErr          error
		authenticateErr error
		wantErr         bool
	}{
		{
			desc:  "remove rule successfully",
			id:    ruleID,
			token: validToken,
		},
		{
			desc:    "remove rule with empty token",
			id:      ruleID,
			token:   "",
			wantErr: true,
		},
		{
			desc:    "remove non-existent rule",
			id:      "non-existent",
			token:   validToken,
			svcErr:  errors.New("not found"),
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := rsvc.On("RemoveRule", mock.Anything, tc.session, tc.id).Return(tc.svcErr)
			err := mgsdk.RemoveRule(context.Background(), tc.id, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
			svcCall.Unset()
			authCall.Unset()
		})
	}
}
