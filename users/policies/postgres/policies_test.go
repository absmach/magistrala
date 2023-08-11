// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/mainflux/mainflux/internal/testsutil"
	mfclients "github.com/mainflux/mainflux/pkg/clients"
	"github.com/mainflux/mainflux/pkg/errors"
	mfgroups "github.com/mainflux/mainflux/pkg/groups"
	gpostgres "github.com/mainflux/mainflux/pkg/groups/postgres"
	"github.com/mainflux/mainflux/pkg/uuid"
	cpostgres "github.com/mainflux/mainflux/users/clients/postgres"
	"github.com/mainflux/mainflux/users/policies"
	ppostgres "github.com/mainflux/mainflux/users/policies/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var idProvider = uuid.New()

func TestPoliciesSave(t *testing.T) {
	t.Cleanup(func() { testsutil.CleanUpDB(t, db) })
	repo := ppostgres.NewRepository(database)
	crepo := cpostgres.NewRepository(database)
	grepo := gpostgres.New(database)

	group := mfgroups.Group{
		ID:   testsutil.GenerateUUID(t, idProvider),
		Name: "policy-save",
	}
	group, err := grepo.Save(context.Background(), group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	client := mfclients.Client{
		ID:   testsutil.GenerateUUID(t, idProvider),
		Name: "policy-save@example.com",
		Credentials: mfclients.Credentials{
			Identity: "policy-save@example.com",
			Secret:   "pass",
		},
		Status: mfclients.EnabledStatus,
	}

	clients, err := crepo.Save(context.Background(), client)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	client = clients

	cases := []struct {
		desc   string
		policy policies.Policy
		err    error
	}{
		{
			desc: "add new policy successfully",
			policy: policies.Policy{
				OwnerID: client.ID,
				Subject: client.ID,
				Object:  group.ID,
				Actions: []string{"c_delete"},
			},
			err: nil,
		},
		{
			desc: "add policy with duplicate subject, object and action",
			policy: policies.Policy{
				OwnerID: client.ID,
				Subject: client.ID,
				Object:  group.ID,
				Actions: []string{"c_delete"},
			},
			err: nil,
		},
	}

	for _, tc := range cases {
		err := repo.Save(context.Background(), tc.policy)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestPoliciesEvaluate(t *testing.T) {
	t.Cleanup(func() { testsutil.CleanUpDB(t, db) })
	repo := ppostgres.NewRepository(database)
	crepo := cpostgres.NewRepository(database)
	grepo := gpostgres.New(database)

	client1 := mfclients.Client{
		ID:   testsutil.GenerateUUID(t, idProvider),
		Name: "connectedclients-clientA@example.com",
		Credentials: mfclients.Credentials{
			Identity: "connectedclients-clientA@example.com",
			Secret:   "pass",
		},
		Status: mfclients.EnabledStatus,
	}
	client2 := mfclients.Client{
		ID:   testsutil.GenerateUUID(t, idProvider),
		Name: "connectedclients-clientB@example.com",
		Credentials: mfclients.Credentials{
			Identity: "connectedclients-clientB@example.com",
			Secret:   "pass",
		},
		Status: mfclients.EnabledStatus,
	}
	group := mfgroups.Group{
		ID:   testsutil.GenerateUUID(t, idProvider),
		Name: "connecting-group@example.com",
	}

	clients1, err := crepo.Save(context.Background(), client1)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	client1 = clients1
	clients2, err := crepo.Save(context.Background(), client2)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	client2 = clients2
	group, err = grepo.Save(context.Background(), group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	policy1 := policies.Policy{
		OwnerID: client1.ID,
		Subject: client1.ID,
		Object:  group.ID,
		Actions: []string{"c_update", "c_list", "g_list", "g_update"},
	}
	policy2 := policies.Policy{
		OwnerID: client2.ID,
		Subject: client2.ID,
		Object:  group.ID,
		Actions: []string{"c_update", "g_update"},
	}
	err = repo.Save(context.Background(), policy1)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	err = repo.Save(context.Background(), policy2)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := map[string]struct {
		Subject string
		Object  string
		Action  string
		Domain  string
		err     error
	}{
		"evaluate valid client update":   {client1.ID, client2.ID, "c_update", "client", nil},
		"evaluate valid group update":    {client1.ID, group.ID, "g_update", "group", nil},
		"evaluate valid client list":     {client1.ID, client2.ID, "c_list", "client", nil},
		"evaluate valid group list":      {client1.ID, group.ID, "g_list", "group", nil},
		"evaluate invalid client delete": {client1.ID, client2.ID, "c_delete", "client", errors.ErrAuthorization},
		"evaluate invalid group delete":  {client1.ID, group.ID, "g_delete", "group", errors.ErrAuthorization},
		"evaluate invalid client update": {"unknown", "unknown", "c_update", "client", errors.ErrAuthorization},
		"evaluate invalid group update":  {"unknown", "unknown", "c_update", "group", errors.ErrAuthorization},
	}

	for desc, tc := range cases {
		aReq := policies.AccessRequest{
			Subject: tc.Subject,
			Object:  tc.Object,
			Action:  tc.Action,
		}
		var err error
		switch tc.Domain {
		case "client":
			_, err = repo.EvaluateUserAccess(context.Background(), aReq)
		case "group":
			_, err = repo.EvaluateGroupAccess(context.Background(), aReq)
		}
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestPoliciesRetrieve(t *testing.T) {
	t.Cleanup(func() { testsutil.CleanUpDB(t, db) })
	repo := ppostgres.NewRepository(database)
	crepo := cpostgres.NewRepository(database)
	grepo := gpostgres.New(database)

	group := mfgroups.Group{
		ID:   testsutil.GenerateUUID(t, idProvider),
		Name: "policy-save",
	}
	group, err := grepo.Save(context.Background(), group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	client := mfclients.Client{
		ID:   testsutil.GenerateUUID(t, idProvider),
		Name: "single-policy-retrieval@example.com",
		Credentials: mfclients.Credentials{
			Identity: "single-policy-retrieval@example.com",
			Secret:   "pass",
		},
		Status: mfclients.EnabledStatus,
	}

	clients, err := crepo.Save(context.Background(), client)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	client = clients

	policy := policies.Policy{
		OwnerID: client.ID,
		Subject: client.ID,
		Object:  group.ID,
		Actions: []string{"c_delete"},
	}

	err = repo.Save(context.Background(), policy)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := map[string]struct {
		Subject string
		Object  string
		err     error
	}{
		"retrieve existing policy":     {client.ID, group.ID, nil},
		"retrieve non-existing policy": {"unknown", "unknown", nil},
	}

	for desc, tc := range cases {
		pm := policies.Page{
			Subject: tc.Subject,
			Object:  tc.Object,
		}
		_, err := repo.RetrieveAll(context.Background(), pm)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestPoliciesUpdate(t *testing.T) {
	t.Cleanup(func() { testsutil.CleanUpDB(t, db) })
	repo := ppostgres.NewRepository(database)
	crepo := cpostgres.NewRepository(database)
	grepo := gpostgres.New(database)

	group := mfgroups.Group{
		ID:   testsutil.GenerateUUID(t, idProvider),
		Name: "policy-save",
	}
	group, err := grepo.Save(context.Background(), group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	client := mfclients.Client{
		ID:   testsutil.GenerateUUID(t, idProvider),
		Name: "policy-update@example.com",
		Credentials: mfclients.Credentials{
			Identity: "policy-update@example.com",
			Secret:   "pass",
		},
		Status: mfclients.EnabledStatus,
	}

	_, err = crepo.Save(context.Background(), client)
	require.Nil(t, err, fmt.Sprintf("unexpected error during saving client: %s", err))

	policy := policies.Policy{
		OwnerID: testsutil.GenerateUUID(t, idProvider),
		Subject: client.ID,
		Object:  group.ID,
		Actions: []string{"c_delete"},
	}
	err = repo.Save(context.Background(), policy)
	require.Nil(t, err, fmt.Sprintf("unexpected error during saving policy: %s", err))

	cases := []struct {
		desc   string
		policy policies.Policy
		resp   policies.Policy
		err    error
	}{
		{
			desc: "update policy successfully",
			policy: policies.Policy{
				Subject: client.ID,
				Object:  group.ID,
				Actions: []string{"c_update"},
			},
			resp: policies.Policy{
				OwnerID: policy.OwnerID,
				Subject: client.ID,
				Object:  group.ID,
				Actions: []string{"c_update"},
			},
			err: nil,
		},
		{
			desc: "update policy with missing subject",
			policy: policies.Policy{
				Subject: "",
				Object:  group.ID,
				Actions: []string{"c_add"},
			},
			resp: policies.Policy{
				OwnerID: policy.OwnerID,
				Subject: client.ID,
				Object:  group.ID,
				Actions: []string{"c_update"},
			},
			err: nil,
		},
		{
			desc: "update policy with missing object",
			policy: policies.Policy{
				Subject: client.ID,
				Object:  "",
				Actions: []string{"c_add"},
			},
			resp: policies.Policy{
				OwnerID: policy.OwnerID,
				Subject: client.ID,
				Object:  group.ID,
				Actions: []string{"c_update"},
			},

			err: nil,
		},
		{
			desc: "update policy with missing action",
			policy: policies.Policy{
				Subject: client.ID,
				Object:  group.ID,
				Actions: []string{""},
			},
			resp: policies.Policy{
				OwnerID: policy.OwnerID,
				Subject: client.ID,
				Object:  group.ID,
				Actions: []string{""},
			},
			err: nil,
		},
	}

	for _, tc := range cases {
		err := repo.Update(context.Background(), tc.policy)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		policPage, err := repo.RetrieveAll(context.Background(), policies.Page{
			Offset:  uint64(0),
			Limit:   uint64(10),
			Subject: tc.policy.Subject,
		})
		if err == nil {
			assert.Equal(t, tc.resp, policPage.Policies[0], fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		}
	}
}

func TestPoliciesRetrievalAll(t *testing.T) {
	t.Cleanup(func() { testsutil.CleanUpDB(t, db) })
	repo := ppostgres.NewRepository(database)
	crepo := cpostgres.NewRepository(database)
	grepo := gpostgres.New(database)

	nPolicies := uint64(10)

	clientA := mfclients.Client{
		ID:   testsutil.GenerateUUID(t, idProvider),
		Name: "policyA-retrievalall@example.com",
		Credentials: mfclients.Credentials{
			Identity: "policyA-retrievalall@example.com",
			Secret:   "pass",
		},
		Status: mfclients.EnabledStatus,
	}
	clientB := mfclients.Client{
		ID:   testsutil.GenerateUUID(t, idProvider),
		Name: "policyB-retrievalall@example.com",
		Credentials: mfclients.Credentials{
			Identity: "policyB-retrievalall@example.com",
			Secret:   "pass",
		},
		Status: mfclients.EnabledStatus,
	}

	clientsA, err := crepo.Save(context.Background(), clientA)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	clientA = clientsA
	clientsB, err := crepo.Save(context.Background(), clientB)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	clientB = clientsB

	grps := []string{}
	for i := uint64(0); i < nPolicies; i++ {
		group := mfgroups.Group{
			ID:   testsutil.GenerateUUID(t, idProvider),
			Name: fmt.Sprintf("policy-retrievalall-%d", i),
		}
		group, err := grepo.Save(context.Background(), group)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		grps = append(grps, group.ID)

		if i%2 == 0 {
			policy := policies.Policy{
				OwnerID: clientA.ID,
				Subject: clientA.ID,
				Object:  group.ID,
				Actions: []string{"c_delete"},
			}
			err = repo.Save(context.Background(), policy)
			require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		}
		policy := policies.Policy{
			OwnerID: clientB.ID,
			Subject: clientB.ID,
			Object:  group.ID,
			Actions: []string{"c_add", "c_update"},
		}
		err = repo.Save(context.Background(), policy)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	}

	cases := map[string]struct {
		size uint64
		pm   policies.Page
	}{
		"retrieve all policies with limit and offset": {
			pm: policies.Page{
				Offset: 5,
				Limit:  nPolicies,
			},
			size: 10,
		},
		"retrieve all policies by owner id": {
			pm: policies.Page{
				Offset:  0,
				Limit:   nPolicies,
				Total:   nPolicies,
				OwnerID: clientA.ID,
			},
			size: 5,
		},
		"retrieve policies by wrong owner id": {
			pm: policies.Page{
				Offset:  0,
				Limit:   nPolicies,
				Total:   nPolicies,
				OwnerID: "wrong",
			},
			size: 0,
		},
		"retrieve all policies by Subject": {
			pm: policies.Page{
				Offset:  0,
				Limit:   nPolicies,
				Total:   nPolicies,
				Subject: clientA.ID,
			},
			size: 5,
		},
		"retrieve policies by wrong Subject": {
			pm: policies.Page{
				Offset:  0,
				Limit:   nPolicies,
				Total:   nPolicies,
				Subject: "wrongSubject",
			},
			size: 0,
		},

		"retrieve all policies by Object": {
			pm: policies.Page{
				Offset: 0,
				Limit:  nPolicies,
				Total:  nPolicies,
				Object: grps[0],
			},
			size: 2,
		},
		"retrieve policies by wrong Object": {
			pm: policies.Page{
				Offset: 0,
				Limit:  nPolicies,
				Total:  nPolicies,
				Object: "TestRetrieveAll45@example.com",
			},
			size: 0,
		},
		"retrieve all policies by Action": {
			pm: policies.Page{
				Offset: 0,
				Limit:  nPolicies,
				Total:  nPolicies,
				Action: "c_delete",
			},
			size: 5,
		},
		"retrieve policies by wrong Action": {
			pm: policies.Page{
				Offset: 0,
				Limit:  nPolicies,
				Total:  nPolicies,
				Action: "wrongAction",
			},
			size: 0,
		},
		"retrieve all policies by owner id and subject": {
			pm: policies.Page{
				Offset:  0,
				Limit:   nPolicies,
				Total:   nPolicies,
				OwnerID: clientA.ID,
				Subject: clientA.ID,
			},
			size: 5,
		},
		"retrieve policies by wrong owner id and correct subject": {
			pm: policies.Page{
				Offset:  0,
				Limit:   nPolicies,
				Total:   nPolicies,
				OwnerID: clientB.ID,
				Subject: clientA.ID,
			},
			size: 0,
		},
		"retrieve policies by correct owner id and wrong subject": {
			pm: policies.Page{
				Offset:  0,
				Limit:   nPolicies,
				Total:   nPolicies,
				OwnerID: clientA.ID,
				Subject: "wrong",
			},
			size: 0,
		},
		"retrieve policies by wrong owner id and wrong subject": {
			pm: policies.Page{
				Offset:  0,
				Limit:   nPolicies,
				Total:   nPolicies,
				OwnerID: "wrong",
			},
			size: 0,
		},
		"retrieve all policies by owner id and object": {
			pm: policies.Page{
				Offset:  0,
				Limit:   nPolicies,
				Total:   nPolicies,
				OwnerID: clientA.ID,
				Object:  grps[0],
			},
			size: 1,
		},
		"retrieve policies by wrong owner id and correct object": {
			pm: policies.Page{
				Offset:  0,
				Limit:   nPolicies,
				Total:   nPolicies,
				OwnerID: "wrong",
				Object:  grps[0],
			},
			size: 0,
		},
		"retrieve policies by correct owner id and wrong object": {
			pm: policies.Page{
				Offset:  0,
				Limit:   nPolicies,
				Total:   nPolicies,
				OwnerID: clientA.ID,
				Object:  "wrong",
			},
			size: 0,
		},
		"retrieve policies by wrong owner id and wrong object": {
			pm: policies.Page{
				Offset:  0,
				Limit:   nPolicies,
				Total:   nPolicies,
				OwnerID: "wrong",
				Object:  "wrong",
			},
			size: 0,
		},
		"retrieve all policies by owner id and action": {
			pm: policies.Page{
				Offset:  0,
				Limit:   nPolicies,
				Total:   nPolicies,
				OwnerID: clientA.ID,
				Action:  "c_delete",
			},
			size: 5,
		},
		"retrieve policies by wrong owner id and correct action": {
			pm: policies.Page{
				Offset:  0,
				Limit:   nPolicies,
				Total:   nPolicies,
				OwnerID: clientB.ID,
				Action:  "c_delete",
			},
			size: 0,
		},
		"retrieve policies by correct owner id and wrong action": {
			pm: policies.Page{
				Offset:  0,
				Limit:   nPolicies,
				Total:   nPolicies,
				OwnerID: clientA.ID,
				Action:  "wrong",
			},
			size: 0,
		},
		"retrieve policies by wrong owner id and wrong action": {
			pm: policies.Page{
				Offset:  0,
				Limit:   nPolicies,
				Total:   nPolicies,
				OwnerID: clientB.ID,
				Action:  "wrong",
			},
			size: 0,
		},
	}
	for desc, tc := range cases {
		page, err := repo.RetrieveAll(context.Background(), tc.pm)
		size := uint64(len(page.Policies))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", desc, tc.size, size))
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %d\n", desc, err))
	}
}

func TestPoliciesDelete(t *testing.T) {
	t.Cleanup(func() { testsutil.CleanUpDB(t, db) })
	repo := ppostgres.NewRepository(database)
	crepo := cpostgres.NewRepository(database)
	grepo := gpostgres.New(database)

	group := mfgroups.Group{
		ID:   testsutil.GenerateUUID(t, idProvider),
		Name: "policy-save",
	}
	group, err := grepo.Save(context.Background(), group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	client := mfclients.Client{
		ID:   testsutil.GenerateUUID(t, idProvider),
		Name: "policy-delete@example.com",
		Credentials: mfclients.Credentials{
			Identity: "policy-delete@example.com",
			Secret:   "pass",
		},
		Status: mfclients.EnabledStatus,
	}

	clients, err := crepo.Save(context.Background(), client)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	client = clients

	policy := policies.Policy{
		OwnerID: client.ID,
		Subject: client.ID,
		Object:  group.ID,
		Actions: []string{"c_delete"},
	}

	err = repo.Save(context.Background(), policy)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := map[string]struct {
		Subject string
		Object  string
		err     error
	}{
		"delete non-existing policy":                      {"unknown", "unknown", nil},
		"delete non-existing policy with correct subject": {client.ID, "unknown", nil},
		"delete non-existing policy with correct object":  {"unknown", group.ID, nil},
		"delete existing policy":                          {client.ID, group.ID, nil},
	}

	for desc, tc := range cases {
		policy := policies.Policy{
			Subject: tc.Subject,
			Object:  tc.Object,
		}
		err := repo.Delete(context.Background(), policy)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
	pm := policies.Page{
		OwnerID: client.ID,
		Subject: client.ID,
		Object:  group.ID,
		Action:  "c_delete",
	}
	policyPage, err := repo.RetrieveAll(context.Background(), pm)
	assert.Equal(t, uint64(0), policyPage.Total, fmt.Sprintf("retrieve policies unexpected total %d\n", policyPage.Total))
	require.Nil(t, err, fmt.Sprintf("retrieve policies unexpected error: %s", err))
}
