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
	cpostgres "github.com/mainflux/mainflux/things/clients/postgres"
	"github.com/mainflux/mainflux/things/policies"
	ppostgres "github.com/mainflux/mainflux/things/policies/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var idProvider = uuid.New()

func TestPoliciesSave(t *testing.T) {
	t.Cleanup(func() { testsutil.CleanUpDB(t, db) })
	repo := ppostgres.NewRepository(database)
	grepo := gpostgres.New(database)

	uid := testsutil.GenerateUUID(t, idProvider)

	group := mfgroups.Group{
		ID:     uid,
		Name:   "policy-save@example.com",
		Status: mfclients.EnabledStatus,
	}

	_, err := grepo.Save(context.Background(), group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	uid = testsutil.GenerateUUID(t, idProvider)

	cases := []struct {
		desc   string
		policy policies.Policy
		err    error
	}{
		{
			desc: "add new policy successfully",
			policy: policies.Policy{
				OwnerID: testsutil.GenerateUUID(t, idProvider),
				Subject: uid,
				Object:  group.ID,
				Actions: []string{"c_delete"},
			},
			err: nil,
		},
		{
			desc: "add policy with duplicate subject, object and action",
			policy: policies.Policy{
				OwnerID: testsutil.GenerateUUID(t, idProvider),
				Subject: uid,
				Object:  group.ID,
				Actions: []string{"c_delete"},
			},
			err: nil,
		},
	}

	for _, tc := range cases {
		_, err := repo.Save(context.Background(), tc.policy)
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
			Secret:   testsutil.GenerateUUID(t, idProvider),
		},
		Status: mfclients.EnabledStatus,
	}
	client2 := mfclients.Client{
		ID:   testsutil.GenerateUUID(t, idProvider),
		Name: "connectedclients-clientB@example.com",
		Credentials: mfclients.Credentials{
			Identity: "connectedclients-clientB@example.com",
			Secret:   testsutil.GenerateUUID(t, idProvider),
		},
		Status: mfclients.EnabledStatus,
	}
	group := mfgroups.Group{
		ID:   testsutil.GenerateUUID(t, idProvider),
		Name: "connecting-group@example.com",
	}

	_, err := crepo.Save(context.Background(), client1)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	_, err = crepo.Save(context.Background(), client2)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
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
	_, err = repo.Save(context.Background(), policy1)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	_, err = repo.Save(context.Background(), policy2)
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
		p := policies.AccessRequest{
			Subject: tc.Subject,
			Object:  tc.Object,
			Action:  tc.Action,
		}
		switch tc.Domain {
		case "client":
			_, err := repo.EvaluateThingAccess(context.Background(), p)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
		case "group":
			_, err := repo.EvaluateGroupAccess(context.Background(), p)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
		}
	}
}

func TestPoliciesRetrieve(t *testing.T) {
	t.Cleanup(func() { testsutil.CleanUpDB(t, db) })
	repo := ppostgres.NewRepository(database)
	crepo := cpostgres.NewRepository(database)
	grepo := gpostgres.New(database)

	uid := testsutil.GenerateUUID(t, idProvider)

	client := mfclients.Client{
		ID:   uid,
		Name: "single-policy-retrieval@example.com",
		Credentials: mfclients.Credentials{
			Identity: "single-policy-retrieval@example.com",
			Secret:   testsutil.GenerateUUID(t, idProvider),
		},
		Status: mfclients.EnabledStatus,
	}

	_, err := crepo.Save(context.Background(), client)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	group := mfgroups.Group{
		ID:     testsutil.GenerateUUID(t, idProvider),
		Name:   "policy-save@example.com",
		Status: mfclients.EnabledStatus,
	}
	_, err = grepo.Save(context.Background(), group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	policy := policies.Policy{
		OwnerID: client.ID,
		Subject: client.ID,
		Object:  group.ID,
		Actions: []string{"c_delete"},
	}

	_, err = repo.Save(context.Background(), policy)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := map[string]struct {
		Subject string
		Object  string
		err     error
	}{
		"retrieve existing policy":     {uid, uid, nil},
		"retrieve non-existing policy": {"unknown", "unknown", nil},
	}

	for desc, tc := range cases {
		pm := policies.Page{
			Subject: tc.Subject,
			Object:  tc.Object,
		}
		_, err := repo.Retrieve(context.Background(), pm)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestPoliciesUpdate(t *testing.T) {
	t.Cleanup(func() { testsutil.CleanUpDB(t, db) })
	repo := ppostgres.NewRepository(database)
	crepo := cpostgres.NewRepository(database)
	grepo := gpostgres.New(database)

	client := mfclients.Client{
		ID:   testsutil.GenerateUUID(t, idProvider),
		Name: "policy-update@example.com",
		Credentials: mfclients.Credentials{
			Identity: "policy-update@example.com",
			Secret:   "pass",
		},
		Status: mfclients.EnabledStatus,
	}

	_, err := crepo.Save(context.Background(), client)
	require.Nil(t, err, fmt.Sprintf("unexpected error during saving client: %s", err))

	group := mfgroups.Group{
		ID:     testsutil.GenerateUUID(t, idProvider),
		Name:   "policy-save@example.com",
		Status: mfclients.EnabledStatus,
	}
	_, err = grepo.Save(context.Background(), group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	policy := policies.Policy{
		OwnerID: client.ID,
		Subject: client.ID,
		Object:  group.ID,
		Actions: []string{"c_delete"},
	}
	_, err = repo.Save(context.Background(), policy)
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
				OwnerID: client.ID,
				Subject: client.ID,
				Object:  group.ID,
				Actions: []string{"c_update"},
			},
			err: nil,
		},
		{
			desc: "update policy with missing owner id",
			policy: policies.Policy{
				OwnerID: "",
				Subject: client.ID,
				Object:  group.ID,
				Actions: []string{"c_delete"},
			},
			resp: policies.Policy{
				OwnerID: client.ID,
				Subject: client.ID,
				Object:  group.ID,
				Actions: []string{"c_delete"},
			},
			err: nil,
		},
		{
			desc: "update policy with missing subject",
			policy: policies.Policy{
				OwnerID: client.ID,
				Subject: "",
				Object:  group.ID,
				Actions: []string{"c_add"},
			},
			resp: policies.Policy{
				OwnerID: client.ID,
				Subject: client.ID,
				Object:  group.ID,
				Actions: []string{"c_delete"},
			},
			err: errors.ErrNotFound,
		},
		{
			desc: "update policy with missing object",
			policy: policies.Policy{
				OwnerID: client.ID,
				Subject: client.ID,
				Object:  "",
				Actions: []string{"c_add"},
			},
			resp: policies.Policy{
				OwnerID: client.ID,
				Subject: client.ID,
				Object:  group.ID,
				Actions: []string{"c_delete"},
			},

			err: errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		_, err := repo.Update(context.Background(), tc.policy)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		policPage, err := repo.Retrieve(context.Background(), policies.Page{
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
			Secret:   testsutil.GenerateUUID(t, idProvider),
		},
		Status: mfclients.EnabledStatus,
	}
	clientB := mfclients.Client{
		ID:   testsutil.GenerateUUID(t, idProvider),
		Name: "policyB-retrievalall@example.com",
		Credentials: mfclients.Credentials{
			Identity: "policyB-retrievalall@example.com",
			Secret:   testsutil.GenerateUUID(t, idProvider),
		},
		Status: mfclients.EnabledStatus,
	}

	_, err := crepo.Save(context.Background(), clientA)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	_, err = crepo.Save(context.Background(), clientB)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	groupID := ""
	for i := uint64(0); i < nPolicies; i++ {
		group := mfgroups.Group{
			ID:     testsutil.GenerateUUID(t, idProvider),
			Name:   fmt.Sprintf("TestRetrieveAll%d@example.com", i),
			Status: mfclients.EnabledStatus,
		}
		if i == 0 {
			groupID = group.ID
		}
		_, err = grepo.Save(context.Background(), group)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

		if i%2 == 0 {
			policy := policies.Policy{
				OwnerID: clientA.ID,
				Subject: clientA.ID,
				Object:  group.ID,
				Actions: []string{"c_delete"},
			}
			_, err = repo.Save(context.Background(), policy)
			require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		}
		policy := policies.Policy{
			Subject: clientB.ID,
			Object:  group.ID,
			Actions: []string{"c_add", "c_update"},
		}
		_, err = repo.Save(context.Background(), policy)
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
				Object: groupID,
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
				Subject: "wrongSubject",
			},
			size: 0,
		},
		"retrieve policies by wrong owner id and wrong subject": {
			pm: policies.Page{
				Offset:  0,
				Limit:   nPolicies,
				Total:   nPolicies,
				OwnerID: clientB.ID,
			},
			size: 0,
		},
		"retrieve all policies by owner id and object": {
			pm: policies.Page{
				Offset:  0,
				Limit:   nPolicies,
				Total:   nPolicies,
				OwnerID: clientA.ID,
				Object:  groupID,
			},
			size: 1,
		},
		"retrieve policies by wrong owner id and correct object": {
			pm: policies.Page{
				Offset:  0,
				Limit:   nPolicies,
				Total:   nPolicies,
				OwnerID: clientB.ID,
				Object:  groupID,
			},
			size: 0,
		},
		"retrieve policies by correct owner id and wrong object": {
			pm: policies.Page{
				Offset:  0,
				Limit:   nPolicies,
				Total:   nPolicies,
				OwnerID: clientA.ID,
				Object:  "TestRetrieveAll45@example.com",
			},
			size: 0,
		},
		"retrieve policies by wrong owner id and wrong object": {
			pm: policies.Page{
				Offset:  0,
				Limit:   nPolicies,
				Total:   nPolicies,
				OwnerID: clientB.ID,
				Object:  "TestRetrieveAll45@example.com",
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
				Action:  "wrongAction",
			},
			size: 0,
		},
		"retrieve policies by wrong owner id and wrong action": {
			pm: policies.Page{
				Offset:  0,
				Limit:   nPolicies,
				Total:   nPolicies,
				OwnerID: clientB.ID,
				Action:  "wrongAction",
			},
			size: 0,
		},
	}
	for desc, tc := range cases {
		page, err := repo.Retrieve(context.Background(), tc.pm)
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

	client := mfclients.Client{
		ID:   testsutil.GenerateUUID(t, idProvider),
		Name: "policy-delete@example.com",
		Credentials: mfclients.Credentials{
			Identity: "policy-delete@example.com",
			Secret:   testsutil.GenerateUUID(t, idProvider),
		},
		Status: mfclients.EnabledStatus,
	}

	subject, err := crepo.Save(context.Background(), client)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	group := mfgroups.Group{
		ID:     testsutil.GenerateUUID(t, idProvider),
		Name:   "policy-save@example.com",
		Status: mfclients.EnabledStatus,
	}
	_, err = grepo.Save(context.Background(), group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	policy := policies.Policy{
		OwnerID: subject[0].ID,
		Subject: subject[0].ID,
		Object:  group.ID,
		Actions: []string{"c_delete"},
	}

	_, err = repo.Save(context.Background(), policy)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := map[string]struct {
		Subject string
		Object  string
		err     error
	}{
		"delete non-existing policy":                      {"unknown", "unknown", nil},
		"delete non-existing policy with correct subject": {subject[0].ID, "unknown", nil},
		"delete non-existing policy with correct object":  {"unknown", group.ID, nil},
		"delete existing policy":                          {subject[0].ID, group.ID, nil},
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
		OwnerID: subject[0].ID,
		Subject: subject[0].ID,
		Object:  group.ID,
		Action:  "c_delete",
	}
	policyPage, err := repo.Retrieve(context.Background(), pm)
	assert.Equal(t, uint64(0), policyPage.Total, fmt.Sprintf("retrieve policies unexpected total %d\n", policyPage.Total))
	require.Nil(t, err, fmt.Sprintf("retrieve policies unexpected error: %s", err))
}
