// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/absmach/magistrala/pkg/sdk"
	"github.com/stretchr/testify/assert"
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

func TestAddRule(t *testing.T) {
	cases := []struct {
		desc    string
		rule    sdk.Rule
		token   string
		handler http.HandlerFunc
		wantErr bool
		resp    sdk.Rule
	}{
		{
			desc:  "add rule successfully",
			rule:  sdk.Rule{Name: "temp-rule", InputChannel: "chan-1"},
			token: validToken,
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, fmt.Sprintf("/%s/rules", domainID), r.URL.Path)
				w.WriteHeader(http.StatusCreated)
				_ = json.NewEncoder(w).Encode(testRule)
			},
			resp: testRule,
		},
		{
			desc:  "add rule with empty token",
			rule:  sdk.Rule{Name: "temp-rule"},
			token: "",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			},
			wantErr: true,
		},
		{
			desc:  "add rule with bad request",
			rule:  sdk.Rule{},
			token: validToken,
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			defer server.Close()

			mgsdk := sdk.NewSDK(sdk.Config{RulesEngineURL: server.URL})
			result, err := mgsdk.AddRule(context.Background(), tc.rule, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
			if !tc.wantErr {
				assert.Equal(t, tc.resp, result)
			}
		})
	}
}

func TestViewRule(t *testing.T) {
	cases := []struct {
		desc    string
		id      string
		token   string
		handler http.HandlerFunc
		wantErr bool
		resp    sdk.Rule
	}{
		{
			desc:  "view rule successfully",
			id:    ruleID,
			token: validToken,
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, fmt.Sprintf("/%s/rules/%s", domainID, ruleID), r.URL.Path)
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(testRule)
			},
			resp: testRule,
		},
		{
			desc:  "view rule with empty token",
			id:    ruleID,
			token: "",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			},
			wantErr: true,
		},
		{
			desc:  "view non-existent rule",
			id:    "non-existent",
			token: validToken,
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			defer server.Close()

			mgsdk := sdk.NewSDK(sdk.Config{RulesEngineURL: server.URL})
			result, err := mgsdk.ViewRule(context.Background(), tc.id, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
			if !tc.wantErr {
				assert.Equal(t, tc.resp, result)
			}
		})
	}
}

func TestUpdateRule(t *testing.T) {
	updated := testRule
	updated.Name = "updated-rule"

	cases := []struct {
		desc    string
		rule    sdk.Rule
		token   string
		handler http.HandlerFunc
		wantErr bool
		resp    sdk.Rule
	}{
		{
			desc:  "update rule successfully",
			rule:  testRule,
			token: validToken,
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPut, r.Method)
				assert.Equal(t, fmt.Sprintf("/%s/rules/%s", domainID, ruleID), r.URL.Path)
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(updated)
			},
			resp: updated,
		},
		{
			desc:  "update rule with empty token",
			rule:  testRule,
			token: "",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			defer server.Close()

			mgsdk := sdk.NewSDK(sdk.Config{RulesEngineURL: server.URL})
			result, err := mgsdk.UpdateRule(context.Background(), tc.rule, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
			if !tc.wantErr {
				assert.Equal(t, tc.resp, result)
			}
		})
	}
}

func TestUpdateRuleTags(t *testing.T) {
	updated := testRule
	updated.Tags = []string{"new-tag"}

	cases := []struct {
		desc    string
		rule    sdk.Rule
		token   string
		handler http.HandlerFunc
		wantErr bool
		resp    sdk.Rule
	}{
		{
			desc:  "update rule tags successfully",
			rule:  sdk.Rule{ID: ruleID, Tags: []string{"new-tag"}},
			token: validToken,
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPatch, r.Method)
				assert.Equal(t, fmt.Sprintf("/%s/rules/%s/tags", domainID, ruleID), r.URL.Path)
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(updated)
			},
			resp: updated,
		},
		{
			desc:  "update rule tags with empty token",
			rule:  sdk.Rule{ID: ruleID},
			token: "",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			defer server.Close()

			mgsdk := sdk.NewSDK(sdk.Config{RulesEngineURL: server.URL})
			result, err := mgsdk.UpdateRuleTags(context.Background(), tc.rule, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
			if !tc.wantErr {
				assert.Equal(t, tc.resp, result)
			}
		})
	}
}

func TestUpdateRuleSchedule(t *testing.T) {
	updated := testRule
	updated.Schedule = map[string]any{"cron": "0 * * * *"}

	cases := []struct {
		desc    string
		rule    sdk.Rule
		token   string
		handler http.HandlerFunc
		wantErr bool
		resp    sdk.Rule
	}{
		{
			desc:  "update rule schedule successfully",
			rule:  sdk.Rule{ID: ruleID, Schedule: map[string]any{"cron": "0 * * * *"}},
			token: validToken,
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPatch, r.Method)
				assert.Equal(t, fmt.Sprintf("/%s/rules/%s/schedule", domainID, ruleID), r.URL.Path)
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(updated)
			},
			resp: updated,
		},
		{
			desc:  "update rule schedule with empty token",
			rule:  sdk.Rule{ID: ruleID},
			token: "",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			defer server.Close()

			mgsdk := sdk.NewSDK(sdk.Config{RulesEngineURL: server.URL})
			result, err := mgsdk.UpdateRuleSchedule(context.Background(), tc.rule, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
			if !tc.wantErr {
				assert.Equal(t, tc.resp, result)
			}
		})
	}
}

func TestListRules(t *testing.T) {
	rulesPage := sdk.Page{
		Total:  2,
		Offset: 0,
		Limit:  10,
		Rules:  []sdk.Rule{testRule, {ID: "rule-2", Name: "humidity-rule"}},
	}

	cases := []struct {
		desc    string
		pm      sdk.PageMetadata
		token   string
		checkQ  func(t *testing.T, r *http.Request)
		wantErr bool
		resp    sdk.Page
	}{
		{
			desc:  "list rules successfully",
			pm:    sdk.PageMetadata{Offset: 0, Limit: 10},
			token: validToken,
			checkQ: func(t *testing.T, r *http.Request) {
				assert.Equal(t, "10", r.URL.Query().Get("limit"))
			},
			resp: rulesPage,
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
			token: validToken,
			checkQ: func(t *testing.T, r *http.Request) {
				q := r.URL.Query()
				assert.Equal(t, "enabled", q.Get("status"))
				assert.Equal(t, "chan-1", q.Get("input_channel"))
				assert.Equal(t, "temperature", q.Get("tag"))
				assert.Equal(t, "temp", q.Get("name"))
				assert.Equal(t, "desc", q.Get("dir"))
				assert.Equal(t, "created_at", q.Get("order"))
			},
			resp: rulesPage,
		},
		{
			desc:  "list rules with empty metadata excludes filter params",
			pm:    sdk.PageMetadata{},
			token: validToken,
			checkQ: func(t *testing.T, r *http.Request) {
				rawQ := r.URL.RawQuery
				assert.NotContains(t, rawQ, "status=")
				assert.NotContains(t, rawQ, "input_channel=")
				assert.NotContains(t, rawQ, "tag=")
			},
			resp: sdk.Page{},
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
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, fmt.Sprintf("/%s/rules", domainID), r.URL.Path)
				if r.Header.Get("Authorization") == "" || r.Header.Get("Authorization") == "Bearer " {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				if tc.checkQ != nil {
					tc.checkQ(t, r)
				}
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(tc.resp)
			}))
			defer server.Close()

			mgsdk := sdk.NewSDK(sdk.Config{RulesEngineURL: server.URL})
			result, err := mgsdk.ListRules(context.Background(), tc.pm, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
			if !tc.wantErr {
				assert.Equal(t, tc.resp, result)
			}
		})
	}
}

func TestRemoveRule(t *testing.T) {
	cases := []struct {
		desc    string
		id      string
		token   string
		handler http.HandlerFunc
		wantErr bool
	}{
		{
			desc:  "remove rule successfully",
			id:    ruleID,
			token: validToken,
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodDelete, r.Method)
				assert.Equal(t, fmt.Sprintf("/%s/rules/%s", domainID, ruleID), r.URL.Path)
				w.WriteHeader(http.StatusNoContent)
			},
		},
		{
			desc:  "remove rule with empty token",
			id:    ruleID,
			token: "",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			},
			wantErr: true,
		},
		{
			desc:  "remove non-existent rule",
			id:    "non-existent",
			token: validToken,
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			defer server.Close()

			mgsdk := sdk.NewSDK(sdk.Config{RulesEngineURL: server.URL})
			err := mgsdk.RemoveRule(context.Background(), tc.id, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
		})
	}
}

func TestEnableRule(t *testing.T) {
	enabled := testRule
	enabled.Status = "enabled"

	cases := []struct {
		desc    string
		id      string
		token   string
		handler http.HandlerFunc
		wantErr bool
		resp    sdk.Rule
	}{
		{
			desc:  "enable rule successfully",
			id:    ruleID,
			token: validToken,
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, fmt.Sprintf("/%s/rules/%s/enable", domainID, ruleID), r.URL.Path)
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(enabled)
			},
			resp: enabled,
		},
		{
			desc:  "enable rule with empty token",
			id:    ruleID,
			token: "",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			defer server.Close()

			mgsdk := sdk.NewSDK(sdk.Config{RulesEngineURL: server.URL})
			result, err := mgsdk.EnableRule(context.Background(), tc.id, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
			if !tc.wantErr {
				assert.Equal(t, tc.resp, result)
			}
		})
	}
}

func TestDisableRule(t *testing.T) {
	disabled := testRule
	disabled.Status = "disabled"

	cases := []struct {
		desc    string
		id      string
		token   string
		handler http.HandlerFunc
		wantErr bool
		resp    sdk.Rule
	}{
		{
			desc:  "disable rule successfully",
			id:    ruleID,
			token: validToken,
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, fmt.Sprintf("/%s/rules/%s/disable", domainID, ruleID), r.URL.Path)
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(disabled)
			},
			resp: disabled,
		},
		{
			desc:  "disable rule with empty token",
			id:    ruleID,
			token: "",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			defer server.Close()

			mgsdk := sdk.NewSDK(sdk.Config{RulesEngineURL: server.URL})
			result, err := mgsdk.DisableRule(context.Background(), tc.id, domainID, tc.token)
			assert.Equal(t, tc.wantErr, err != nil)
			if !tc.wantErr {
				assert.Equal(t, tc.resp, result)
			}
		})
	}
}

