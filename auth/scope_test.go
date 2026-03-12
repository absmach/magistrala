// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package auth_test

import (
	"testing"
	"time"

	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/auth"
	channelsOps "github.com/absmach/supermq/channels/operations"
	clientsOps "github.com/absmach/supermq/clients/operations"
	groupsOps "github.com/absmach/supermq/groups/operations"
	"github.com/stretchr/testify/assert"
)

func TestScopeAuthorized(t *testing.T) {
	cases := []struct {
		desc       string
		scope      *auth.Scope
		entityType auth.EntityType
		domainID   string
		operation  string
		entityID   string
		expected   bool
	}{
		{
			desc: "Authorized with matching entity type, domain, operation and entity ID",
			scope: &auth.Scope{
				EntityType: auth.EntityType("groups"),
				DomainID:   "domain1",
				Operation:  "view",
				EntityID:   "entity1",
			},
			entityType: auth.EntityType("groups"),
			domainID:   "domain1",
			operation:  "view",
			entityID:   "entity1",
			expected:   true,
		},
		{
			desc: "Authorized with wildcard entity ID",
			scope: &auth.Scope{
				EntityType: auth.EntityType("groups"),
				DomainID:   "domain1",
				Operation:  "view",
				EntityID:   "*",
			},
			entityType: auth.EntityType("groups"),
			domainID:   "domain1",
			operation:  "view",
			entityID:   "any-entity",
			expected:   true,
		},
		{
			desc: "Authorized without domain ID",
			scope: &auth.Scope{
				EntityType: auth.EntityType("clients"),
				DomainID:   "",
				Operation:  "view",
				EntityID:   "client1",
			},
			entityType: auth.EntityType("clients"),
			domainID:   "domain1",
			operation:  "view",
			entityID:   "client1",
			expected:   true,
		},
		{
			desc: "Not authorized with different entity type",
			scope: &auth.Scope{
				EntityType: auth.EntityType("groups"),
				DomainID:   "domain1",
				Operation:  "view",
				EntityID:   "entity1",
			},
			entityType: auth.EntityType("channels"),
			domainID:   "domain1",
			operation:  "view",
			entityID:   "entity1",
			expected:   false,
		},
		{
			desc: "Not authorized with different domain ID",
			scope: &auth.Scope{
				EntityType: auth.EntityType("groups"),
				DomainID:   "domain1",
				Operation:  "view",
				EntityID:   "entity1",
			},
			entityType: auth.EntityType("groups"),
			domainID:   "domain2",
			operation:  "view",
			entityID:   "entity1",
			expected:   false,
		},
		{
			desc: "Not authorized with different operation",
			scope: &auth.Scope{
				EntityType: auth.EntityType("groups"),
				DomainID:   "domain1",
				Operation:  "view",
				EntityID:   "entity1",
			},
			entityType: auth.EntityType("groups"),
			domainID:   "domain1",
			operation:  "delete",
			entityID:   "entity1",
			expected:   false,
		},
		{
			desc: "Not authorized with different entity ID",
			scope: &auth.Scope{
				EntityType: auth.EntityType("groups"),
				DomainID:   "domain1",
				Operation:  "view",
				EntityID:   "entity1",
			},
			entityType: auth.EntityType("groups"),
			domainID:   "domain1",
			operation:  "view",
			entityID:   "entity2",
			expected:   false,
		},
		{
			desc:       "Not authorized with nil scope",
			scope:      nil,
			entityType: auth.EntityType("groups"),
			domainID:   "domain1",
			operation:  "view",
			entityID:   "entity1",
			expected:   false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			result := tc.scope.Authorized(tc.entityType, tc.domainID, tc.operation, tc.entityID)
			assert.Equal(t, tc.expected, result, "Authorized() = %v, expected %v", result, tc.expected)
		})
	}
}

func TestScopeValidate(t *testing.T) {
	cases := []struct {
		desc  string
		scope *auth.Scope
		err   error
	}{
		{
			desc: "Valid scope for groups with domain ID",
			scope: &auth.Scope{
				EntityType: auth.EntityType("groups"),
				DomainID:   "domain1",
				Operation:  "view",
				EntityID:   "entity1",
			},
			err: nil,
		},
		{
			desc: "Valid scope for channels with domain ID",
			scope: &auth.Scope{
				EntityType: auth.EntityType("channels"),
				DomainID:   "domain1",
				Operation:  "view",
				EntityID:   "channel1",
			},
			err: nil,
		},
		{
			desc: "Valid scope for clients with domain ID",
			scope: &auth.Scope{
				EntityType: auth.EntityType("clients"),
				DomainID:   "domain1",
				Operation:  "update",
				EntityID:   "client1",
			},
			err: nil,
		},
		{
			desc: "Valid scope for messages with domain ID",
			scope: &auth.Scope{
				EntityType: auth.EntityType("messages"),
				DomainID:   "domain1",
				Operation:  "message_publish",
				EntityID:   "message1",
			},
			err: nil,
		},
		{
			desc: "Valid scope for dashboard with domain ID",
			scope: &auth.Scope{
				EntityType: auth.EntityType("dashboards"),
				DomainID:   "domain1",
				Operation:  "dashboard_share",
				EntityID:   "dashboard1",
			},
			err: nil,
		},
		{
			desc: "Valid scope with wildcard entity ID",
			scope: &auth.Scope{
				EntityType: auth.EntityType("groups"),
				DomainID:   "domain1",
				Operation:  "view",
				EntityID:   "*",
			},
			err: nil,
		},
		{
			desc:  "Invalid nil scope",
			scope: nil,
			err:   assert.AnError, // Will be checked with Contains
		},
		{
			desc: "Invalid scope without entity ID",
			scope: &auth.Scope{
				EntityType: auth.EntityType("groups"),
				DomainID:   "domain1",
				Operation:  groupsOps.OperationDetails()[groupsOps.OpViewGroup].Name,
				EntityID:   "",
			},
			err: apiutil.ErrMissingEntityID,
		},
		{
			desc: "Invalid scope for groups without domain ID",
			scope: &auth.Scope{
				EntityType: auth.EntityType("groups"),
				DomainID:   "",
				Operation:  groupsOps.OperationDetails()[groupsOps.OpViewGroup].Name,
				EntityID:   "entity1",
			},
			err: apiutil.ErrMissingDomainID,
		},
		{
			desc: "Invalid scope for channels without domain ID",
			scope: &auth.Scope{
				EntityType: auth.EntityType("channels"),
				DomainID:   "",
				Operation:  channelsOps.OperationDetails()[channelsOps.OpViewChannel].Name,
				EntityID:   "channel1",
			},
			err: apiutil.ErrMissingDomainID,
		},
		{
			desc: "Invalid scope for clients without domain ID",
			scope: &auth.Scope{
				EntityType: auth.EntityType("clients"),
				DomainID:   "",
				Operation:  clientsOps.OperationDetails()[clientsOps.OpViewClient].Name,
				EntityID:   "client1",
			},
			err: apiutil.ErrMissingDomainID,
		},
		{
			desc: "Invalid scope for dashboard without domain ID",
			scope: &auth.Scope{
				EntityType: auth.EntityType("dashboards"),
				DomainID:   "",
				Operation:  "dashboard_share",
				EntityID:   "dashboard1",
			},
			err: apiutil.ErrMissingDomainID,
		},
		{
			desc: "Invalid scope for messages without domain ID",
			scope: &auth.Scope{
				EntityType: auth.EntityType("messages"),
				DomainID:   "",
				Operation:  "message_publish",
				EntityID:   "message1",
			},
			err: apiutil.ErrMissingDomainID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.scope.Validate()
			if tc.err != nil {
				assert.Error(t, err, "Validate() should return error")
				if tc.err != assert.AnError {
					assert.Equal(t, tc.err, err, "Validate() error = %v, expected %v", err, tc.err)
				}
			} else {
				assert.NoError(t, err, "Validate() should not return error")
			}
		})
	}
}

func TestPATValidate(t *testing.T) {
	cases := []struct {
		desc string
		pat  *auth.PAT
		err  bool
	}{
		{
			desc: "Valid PAT",
			pat: &auth.PAT{
				ID:          "pat-id",
				User:        "user-id",
				Name:        "test-pat",
				Description: "test description",
			},
			err: false,
		},
		{
			desc: "Invalid nil PAT",
			pat:  nil,
			err:  true,
		},
		{
			desc: "Invalid PAT without name",
			pat: &auth.PAT{
				ID:          "pat-id",
				User:        "user-id",
				Name:        "",
				Description: "test description",
			},
			err: true,
		},
		{
			desc: "Invalid PAT without user",
			pat: &auth.PAT{
				ID:          "pat-id",
				User:        "",
				Name:        "test-pat",
				Description: "test description",
			},
			err: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.pat.Validate()
			if tc.err {
				assert.Error(t, err, "Validate() should return error")
			} else {
				assert.NoError(t, err, "Validate() should not return error")
			}
		})
	}
}

func TestPATMarshalUnmarshalBinary(t *testing.T) {
	pat := auth.PAT{
		ID:          "pat-id",
		User:        "user-id",
		Name:        "test-pat",
		Description: "test description",
		Secret:      "secret",
		IssuedAt:    time.Now().UTC().Round(time.Second),
		ExpiresAt:   time.Now().UTC().Add(24 * time.Hour).Round(time.Second),
		Status:      auth.ActiveStatus,
	}

	// Marshal
	data, err := pat.MarshalBinary()
	assert.NoError(t, err, "MarshalBinary() should not return error")
	assert.NotNil(t, data, "MarshalBinary() should return data")

	// Unmarshal
	var newPAT auth.PAT
	err = newPAT.UnmarshalBinary(data)
	assert.NoError(t, err, "UnmarshalBinary() should not return error")

	assert.Equal(t, pat.ID, newPAT.ID, "ID mismatch")
	assert.Equal(t, pat.User, newPAT.User, "User mismatch")
	assert.Equal(t, pat.Name, newPAT.Name, "Name mismatch")
	assert.Equal(t, pat.Description, newPAT.Description, "Description mismatch")
	assert.Equal(t, pat.Secret, newPAT.Secret, "Secret mismatch")
	assert.Equal(t, pat.Status, newPAT.Status, "Status mismatch")
}
