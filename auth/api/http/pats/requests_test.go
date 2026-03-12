// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package pats

import (
	"encoding/json"
	"testing"
	"time"

	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/auth"
	"github.com/stretchr/testify/assert"
)

var valid = "valid"

func TestCreatePatReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  createPatReq
		err  error
	}{
		{
			desc: "valid request",
			req: createPatReq{
				token:       valid,
				Name:        "test-pat",
				Description: "test description",
				Duration:    24 * time.Hour,
			},
			err: nil,
		},
		{
			desc: "empty token",
			req: createPatReq{
				token:       "",
				Name:        "test-pat",
				Description: "test description",
				Duration:    24 * time.Hour,
			},
			err: apiutil.ErrBearerToken,
		},
		{
			desc: "empty name",
			req: createPatReq{
				token:       valid,
				Name:        "",
				Description: "test description",
				Duration:    24 * time.Hour,
			},
			err: apiutil.ErrMissingName,
		},
		{
			desc: "whitespace only name",
			req: createPatReq{
				token:       valid,
				Name:        "   ",
				Description: "test description",
				Duration:    24 * time.Hour,
			},
			err: apiutil.ErrMissingName,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.req.validate()
			assert.Equal(t, tc.err, err, "validate() error = %v, expected %v", err, tc.err)
		})
	}
}

func TestCreatePatReqUnmarshalJSON(t *testing.T) {
	cases := []struct {
		desc     string
		data     string
		expected createPatReq
		err      bool
	}{
		{
			desc: "valid JSON with duration",
			data: `{"name":"test-pat","description":"test desc","duration":"24h"}`,
			expected: createPatReq{
				Name:        "test-pat",
				Description: "test desc",
				Duration:    24 * time.Hour,
			},
			err: false,
		},
		{
			desc: "invalid duration format",
			data: `{"name":"test-pat","description":"test desc","duration":"invalid"}`,
			err:  true,
		},
		{
			desc: "invalid JSON",
			data: `{invalid json}`,
			err:  true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			var req createPatReq
			err := json.Unmarshal([]byte(tc.data), &req)
			if tc.err {
				assert.Error(t, err, "UnmarshalJSON() should return error")
			} else {
				assert.NoError(t, err, "UnmarshalJSON() should not return error")
				assert.Equal(t, tc.expected.Name, req.Name)
				assert.Equal(t, tc.expected.Description, req.Description)
				assert.Equal(t, tc.expected.Duration, req.Duration)
			}
		})
	}
}

func TestRetrievePatReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  retrievePatReq
		err  error
	}{
		{
			desc: "valid request",
			req: retrievePatReq{
				token: valid,
				id:    "pat-id",
			},
			err: nil,
		},
		{
			desc: "empty token",
			req: retrievePatReq{
				token: "",
				id:    "pat-id",
			},
			err: apiutil.ErrBearerToken,
		},
		{
			desc: "empty id",
			req: retrievePatReq{
				token: valid,
				id:    "",
			},
			err: apiutil.ErrMissingPATID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.req.validate()
			assert.Equal(t, tc.err, err, "validate() error = %v, expected %v", err, tc.err)
		})
	}
}

func TestUpdatePatNameReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  updatePatNameReq
		err  error
	}{
		{
			desc: "valid request",
			req: updatePatNameReq{
				token: valid,
				id:    "pat-id",
				Name:  "new-name",
			},
			err: nil,
		},
		{
			desc: "empty token",
			req: updatePatNameReq{
				token: "",
				id:    "pat-id",
				Name:  "new-name",
			},
			err: apiutil.ErrBearerToken,
		},
		{
			desc: "empty id",
			req: updatePatNameReq{
				token: valid,
				id:    "",
				Name:  "new-name",
			},
			err: apiutil.ErrMissingPATID,
		},
		{
			desc: "empty name",
			req: updatePatNameReq{
				token: valid,
				id:    "pat-id",
				Name:  "",
			},
			err: apiutil.ErrMissingName,
		},
		{
			desc: "whitespace only name",
			req: updatePatNameReq{
				token: valid,
				id:    "pat-id",
				Name:  "   ",
			},
			err: apiutil.ErrMissingName,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.req.validate()
			assert.Equal(t, tc.err, err, "validate() error = %v, expected %v", err, tc.err)
		})
	}
}

func TestUpdatePatDescriptionReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  updatePatDescriptionReq
		err  error
	}{
		{
			desc: "valid request",
			req: updatePatDescriptionReq{
				token:       valid,
				id:          "pat-id",
				Description: "new description",
			},
			err: nil,
		},
		{
			desc: "empty token",
			req: updatePatDescriptionReq{
				token:       "",
				id:          "pat-id",
				Description: "new description",
			},
			err: apiutil.ErrBearerToken,
		},
		{
			desc: "empty id",
			req: updatePatDescriptionReq{
				token:       valid,
				id:          "",
				Description: "new description",
			},
			err: apiutil.ErrMissingPATID,
		},
		{
			desc: "empty description",
			req: updatePatDescriptionReq{
				token:       valid,
				id:          "pat-id",
				Description: "",
			},
			err: apiutil.ErrMissingDescription,
		},
		{
			desc: "whitespace only description",
			req: updatePatDescriptionReq{
				token:       valid,
				id:          "pat-id",
				Description: "   ",
			},
			err: apiutil.ErrMissingDescription,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.req.validate()
			assert.Equal(t, tc.err, err, "validate() error = %v, expected %v", err, tc.err)
		})
	}
}

func TestListPatsReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  listPatsReq
		err  error
	}{
		{
			desc: "valid request",
			req: listPatsReq{
				token:  valid,
				offset: 0,
				limit:  10,
			},
			err: nil,
		},
		{
			desc: "empty token",
			req: listPatsReq{
				token:  "",
				offset: 0,
				limit:  10,
			},
			err: apiutil.ErrBearerToken,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.req.validate()
			assert.Equal(t, tc.err, err, "validate() error = %v, expected %v", err, tc.err)
		})
	}
}

func TestDeletePatReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  deletePatReq
		err  error
	}{
		{
			desc: "valid request",
			req: deletePatReq{
				token: valid,
				id:    "pat-id",
			},
			err: nil,
		},
		{
			desc: "empty token",
			req: deletePatReq{
				token: "",
				id:    "pat-id",
			},
			err: apiutil.ErrBearerToken,
		},
		{
			desc: "empty id",
			req: deletePatReq{
				token: valid,
				id:    "",
			},
			err: apiutil.ErrMissingPATID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.req.validate()
			assert.Equal(t, tc.err, err, "validate() error = %v, expected %v", err, tc.err)
		})
	}
}

func TestResetPatSecretReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  resetPatSecretReq
		err  error
	}{
		{
			desc: "valid request",
			req: resetPatSecretReq{
				token:    valid,
				id:       "pat-id",
				Duration: 24 * time.Hour,
			},
			err: nil,
		},
		{
			desc: "empty token",
			req: resetPatSecretReq{
				token:    "",
				id:       "pat-id",
				Duration: 24 * time.Hour,
			},
			err: apiutil.ErrBearerToken,
		},
		{
			desc: "empty id",
			req: resetPatSecretReq{
				token:    valid,
				id:       "",
				Duration: 24 * time.Hour,
			},
			err: apiutil.ErrMissingPATID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.req.validate()
			assert.Equal(t, tc.err, err, "validate() error = %v, expected %v", err, tc.err)
		})
	}
}

func TestResetPatSecretReqUnmarshalJSON(t *testing.T) {
	cases := []struct {
		desc     string
		data     string
		expected resetPatSecretReq
		err      bool
	}{
		{
			desc: "valid JSON with duration",
			data: `{"duration":"48h"}`,
			expected: resetPatSecretReq{
				Duration: 48 * time.Hour,
			},
			err: false,
		},
		{
			desc: "invalid duration format",
			data: `{"duration":"invalid"}`,
			err:  true,
		},
		{
			desc: "invalid JSON",
			data: `{invalid}`,
			err:  true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			var req resetPatSecretReq
			err := json.Unmarshal([]byte(tc.data), &req)
			if tc.err {
				assert.Error(t, err, "UnmarshalJSON() should return error")
			} else {
				assert.NoError(t, err, "UnmarshalJSON() should not return error")
				assert.Equal(t, tc.expected.Duration, req.Duration)
			}
		})
	}
}

func TestRevokePatSecretReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  revokePatSecretReq
		err  error
	}{
		{
			desc: "valid request",
			req: revokePatSecretReq{
				token: valid,
				id:    "pat-id",
			},
			err: nil,
		},
		{
			desc: "empty token",
			req: revokePatSecretReq{
				token: "",
				id:    "pat-id",
			},
			err: apiutil.ErrBearerToken,
		},
		{
			desc: "empty id",
			req: revokePatSecretReq{
				token: valid,
				id:    "",
			},
			err: apiutil.ErrMissingPATID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.req.validate()
			assert.Equal(t, tc.err, err, "validate() error = %v, expected %v", err, tc.err)
		})
	}
}

func TestClearAllPATReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  clearAllPATReq
		err  error
	}{
		{
			desc: "valid request",
			req: clearAllPATReq{
				token: valid,
			},
			err: nil,
		},
		{
			desc: "empty token",
			req: clearAllPATReq{
				token: "",
			},
			err: apiutil.ErrBearerToken,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.req.validate()
			assert.Equal(t, tc.err, err, "validate() error = %v, expected %v", err, tc.err)
		})
	}
}

func TestAddScopeReqValidate(t *testing.T) {
	validScope := auth.Scope{
		DomainID:   "domain1",
		EntityType: auth.EntityType("groups"),
		EntityID:   "entity1",
		Operation:  "create_groups",
	}

	invalidScope := auth.Scope{
		DomainID:   "",
		EntityType: auth.EntityType("groups"),
		EntityID:   "",
		Operation:  "view",
	}

	cases := []struct {
		desc string
		req  addScopeReq
		err  error
	}{
		{
			desc: "valid request",
			req: addScopeReq{
				token:  valid,
				id:     "pat-id",
				Scopes: []auth.Scope{validScope},
			},
			err: nil,
		},
		{
			desc: "empty token",
			req: addScopeReq{
				token:  "",
				id:     "pat-id",
				Scopes: []auth.Scope{validScope},
			},
			err: apiutil.ErrBearerToken,
		},
		{
			desc: "empty id",
			req: addScopeReq{
				token:  valid,
				id:     "",
				Scopes: []auth.Scope{validScope},
			},
			err: apiutil.ErrMissingPATID,
		},
		{
			desc: "empty scopes",
			req: addScopeReq{
				token:  valid,
				id:     "pat-id",
				Scopes: []auth.Scope{},
			},
			err: apiutil.ErrValidation,
		},
		{
			desc: "invalid scope",
			req: addScopeReq{
				token:  valid,
				id:     "pat-id",
				Scopes: []auth.Scope{invalidScope},
			},
			err: apiutil.ErrValidation,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.req.validate()
			if tc.err != nil {
				assert.Error(t, err, "validate() should return error")
			} else {
				assert.NoError(t, err, "validate() should not return error")
			}
		})
	}
}

func TestRemoveScopeReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  removeScopeReq
		err  error
	}{
		{
			desc: "valid request",
			req: removeScopeReq{
				token:    valid,
				id:       "pat-id",
				ScopesID: []string{"scope1", "scope2"},
			},
			err: nil,
		},
		{
			desc: "empty token",
			req: removeScopeReq{
				token:    "",
				id:       "pat-id",
				ScopesID: []string{"scope1"},
			},
			err: apiutil.ErrBearerToken,
		},
		{
			desc: "empty id",
			req: removeScopeReq{
				token:    valid,
				id:       "",
				ScopesID: []string{"scope1"},
			},
			err: apiutil.ErrMissingPATID,
		},
		{
			desc: "empty scopes list",
			req: removeScopeReq{
				token:    valid,
				id:       "pat-id",
				ScopesID: []string{},
			},
			err: apiutil.ErrValidation,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.req.validate()
			assert.Equal(t, tc.err, err, "validate() error = %v, expected %v", err, tc.err)
		})
	}
}

func TestClearAllScopeReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  clearAllScopeReq
		err  error
	}{
		{
			desc: "valid request",
			req: clearAllScopeReq{
				token: valid,
				id:    "pat-id",
			},
			err: nil,
		},
		{
			desc: "empty token",
			req: clearAllScopeReq{
				token: "",
				id:    "pat-id",
			},
			err: apiutil.ErrBearerToken,
		},
		{
			desc: "empty id",
			req: clearAllScopeReq{
				token: valid,
				id:    "",
			},
			err: apiutil.ErrMissingPATID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.req.validate()
			assert.Equal(t, tc.err, err, "validate() error = %v, expected %v", err, tc.err)
		})
	}
}

func TestListScopesReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  listScopesReq
		err  error
	}{
		{
			desc: "valid request",
			req: listScopesReq{
				token:  valid,
				offset: 0,
				limit:  10,
				patID:  "pat-id",
			},
			err: nil,
		},
		{
			desc: "empty token",
			req: listScopesReq{
				token:  "",
				offset: 0,
				limit:  10,
				patID:  "pat-id",
			},
			err: apiutil.ErrBearerToken,
		},
		{
			desc: "empty patID",
			req: listScopesReq{
				token:  valid,
				offset: 0,
				limit:  10,
				patID:  "",
			},
			err: apiutil.ErrMissingPATID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.req.validate()
			assert.Equal(t, tc.err, err, "validate() error = %v, expected %v", err, tc.err)
		})
	}
}
