// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/auth/postgres"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/absmach/magistrala/pkg/policies"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	inValid = "invalid"
)

var (
	domainID = testsutil.GenerateUUID(&testing.T{})
	userID   = testsutil.GenerateUUID(&testing.T{})
)

func TestAddPolicyCopy(t *testing.T) {
	repo := postgres.NewDomainRepository(database)
	cases := []struct {
		desc string
		pc   auth.Policy
		err  error
	}{
		{
			desc: "add a  policy copy",
			pc: auth.Policy{
				SubjectType: "unknown",
				SubjectID:   "unknown",
				Relation:    "unknown",
				ObjectType:  "unknown",
				ObjectID:    "unknown",
			},
			err: nil,
		},
		{
			desc: "add again same policy copy",
			pc: auth.Policy{
				SubjectType: "unknown",
				SubjectID:   "unknown",
				Relation:    "unknown",
				ObjectType:  "unknown",
				ObjectID:    "unknown",
			},
			err: repoerr.ErrConflict,
		},
	}

	for _, tc := range cases {
		err := repo.SavePolicies(context.Background(), tc.pc)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.err, err))
	}
}

func TestDeletePolicyCopy(t *testing.T) {
	repo := postgres.NewDomainRepository(database)
	cases := []struct {
		desc string
		pc   auth.Policy
		err  error
	}{
		{
			desc: "delete a  policy copy",
			pc: auth.Policy{
				SubjectType: "unknown",
				SubjectID:   "unknown",
				Relation:    "unknown",
				ObjectType:  "unknown",
				ObjectID:    "unknown",
			},
			err: nil,
		},
		{
			desc: "delete a  policy with empty relation",
			pc: auth.Policy{
				SubjectType: "unknown",
				SubjectID:   "unknown",
				Relation:    "",
				ObjectType:  "unknown",
				ObjectID:    "unknown",
			},
			err: nil,
		},
	}

	for _, tc := range cases {
		err := repo.DeletePolicies(context.Background(), tc.pc)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.err, err))
	}
}

func TestSave(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM domains")
		require.Nil(t, err, fmt.Sprintf("clean domains unexpected error: %s", err))
	})

	repo := postgres.NewDomainRepository(database)

	cases := []struct {
		desc   string
		domain auth.Domain
		err    error
	}{
		{
			desc: "add new domain with all fields successfully",
			domain: auth.Domain{
				ID:    domainID,
				Name:  "test",
				Alias: "test",
				Tags:  []string{"test"},
				Metadata: map[string]interface{}{
					"test": "test",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				CreatedBy: userID,
				UpdatedBy: userID,
				Status:    auth.EnabledStatus,
			},
			err: nil,
		},
		{
			desc: "add the same domain again",
			domain: auth.Domain{
				ID:    domainID,
				Name:  "test",
				Alias: "test",
				Tags:  []string{"test"},
				Metadata: map[string]interface{}{
					"test": "test",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				CreatedBy: userID,
				UpdatedBy: userID,
				Status:    auth.EnabledStatus,
			},
			err: repoerr.ErrConflict,
		},
		{
			desc: "add domain with empty ID",
			domain: auth.Domain{
				ID:    "",
				Name:  "test1",
				Alias: "test1",
				Tags:  []string{"test"},
				Metadata: map[string]interface{}{
					"test": "test",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				CreatedBy: userID,
				UpdatedBy: userID,
				Status:    auth.EnabledStatus,
			},
			err: nil,
		},
		{
			desc: "add domain with empty alias",
			domain: auth.Domain{
				ID:    testsutil.GenerateUUID(&testing.T{}),
				Name:  "test1",
				Alias: "",
				Tags:  []string{"test"},
				Metadata: map[string]interface{}{
					"test": "test",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				CreatedBy: userID,
				UpdatedBy: userID,
				Status:    auth.EnabledStatus,
			},
			err: repoerr.ErrCreateEntity,
		},
		{
			desc: "add domain with malformed metadata",
			domain: auth.Domain{
				ID:    domainID,
				Name:  "test1",
				Alias: "test1",
				Tags:  []string{"test"},
				Metadata: map[string]interface{}{
					"key": make(chan int),
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				CreatedBy: userID,
				UpdatedBy: userID,
				Status:    auth.EnabledStatus,
			},
			err: repoerr.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		_, err := repo.Save(context.Background(), tc.domain)
		{
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		}
	}
}

func TestRetrieveByID(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM domains")
		require.Nil(t, err, fmt.Sprintf("clean domains unexpected error: %s", err))
	})

	repo := postgres.NewDomainRepository(database)

	domain := auth.Domain{
		ID:    domainID,
		Name:  "test",
		Alias: "test",
		Tags:  []string{"test"},
		Metadata: map[string]interface{}{
			"test": "test",
		},
		CreatedBy: userID,
		UpdatedBy: userID,
		Status:    auth.EnabledStatus,
	}

	_, err := repo.Save(context.Background(), domain)
	require.Nil(t, err, fmt.Sprintf("failed to save client %s", domain.ID))

	cases := []struct {
		desc     string
		domainID string
		response auth.Domain
		err      error
	}{
		{
			desc:     "retrieve existing client",
			domainID: domain.ID,
			response: domain,
			err:      nil,
		},
		{
			desc:     "retrieve non-existing client",
			domainID: inValid,
			response: auth.Domain{},
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "retrieve with empty client id",
			domainID: "",
			response: auth.Domain{},
			err:      repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		d, err := repo.RetrieveByID(context.Background(), tc.domainID)
		assert.Equal(t, tc.response, d, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, d))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.err, err))
	}
}

func TestRetreivePermissions(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM domains")
		require.Nil(t, err, fmt.Sprintf("clean domains unexpected error: %s", err))
		_, err = db.Exec("DELETE FROM policies")
		require.Nil(t, err, fmt.Sprintf("clean policies unexpected error: %s", err))
	})

	repo := postgres.NewDomainRepository(database)

	domain := auth.Domain{
		ID:    domainID,
		Name:  "test",
		Alias: "test",
		Tags:  []string{"test"},
		Metadata: map[string]interface{}{
			"test": "test",
		},
		CreatedBy:  userID,
		UpdatedBy:  userID,
		Status:     auth.EnabledStatus,
		Permission: "admin",
	}

	policy := auth.Policy{
		SubjectType:     policies.UserType,
		SubjectID:       userID,
		SubjectRelation: "admin",
		Relation:        "admin",
		ObjectType:      policies.DomainType,
		ObjectID:        domainID,
	}

	_, err := repo.Save(context.Background(), domain)
	require.Nil(t, err, fmt.Sprintf("failed to save domain %s", domain.ID))

	err = repo.SavePolicies(context.Background(), policy)
	require.Nil(t, err, fmt.Sprintf("failed to save policy %s", policy.SubjectID))

	cases := []struct {
		desc          string
		domainID      string
		policySubject string
		response      []string
		err           error
	}{
		{
			desc:          "retrieve existing permissions with valid domaiinID and policySubject",
			domainID:      domain.ID,
			policySubject: userID,
			response:      []string{"admin"},
			err:           nil,
		},
		{
			desc:          "retreieve permissions with invalid domainID",
			domainID:      inValid,
			policySubject: userID,
			response:      []string{},
			err:           nil,
		},
		{
			desc:          "retreieve permissions with invalid policySubject",
			domainID:      domain.ID,
			policySubject: inValid,
			response:      []string{},
			err:           nil,
		},
	}

	for _, tc := range cases {
		d, err := repo.RetrievePermissions(context.Background(), tc.policySubject, tc.domainID)
		assert.Equal(t, tc.response, d, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, d))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveAllByIDs(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM domains")
		require.Nil(t, err, fmt.Sprintf("clean domains unexpected error: %s", err))
	})

	repo := postgres.NewDomainRepository(database)

	items := []auth.Domain{}
	for i := 0; i < 10; i++ {
		domain := auth.Domain{
			ID:    testsutil.GenerateUUID(t),
			Name:  fmt.Sprintf(`"test%d"`, i),
			Alias: fmt.Sprintf(`"test%d"`, i),
			Tags:  []string{"test"},
			Metadata: map[string]interface{}{
				"test": "test",
			},
			CreatedBy: userID,
			UpdatedBy: userID,
			Status:    auth.EnabledStatus,
		}
		if i%5 == 0 {
			domain.Status = auth.DisabledStatus
			domain.Tags = []string{"test", "admin"}
			domain.Metadata = map[string]interface{}{
				"test1": "test1",
			}
		}
		_, err := repo.Save(context.Background(), domain)
		require.Nil(t, err, fmt.Sprintf("save domain unexpected error: %s", err))
		items = append(items, domain)
	}

	cases := []struct {
		desc     string
		pm       auth.Page
		response auth.DomainsPage
		err      error
	}{
		{
			desc: "retrieve by ids successfully",
			pm: auth.Page{
				Offset: 0,
				Limit:  10,
				IDs:    []string{items[1].ID, items[2].ID},
			},
			response: auth.DomainsPage{
				Total:   2,
				Offset:  0,
				Limit:   10,
				Domains: []auth.Domain{items[1], items[2]},
			},
			err: nil,
		},
		{
			desc: "retrieve by ids with empty ids",
			pm: auth.Page{
				Offset: 0,
				Limit:  10,
				IDs:    []string{},
			},
			response: auth.DomainsPage{
				Total:  0,
				Offset: 0,
				Limit:  0,
			},
			err: nil,
		},
		{
			desc: "retrieve by ids with invalid ids",
			pm: auth.Page{
				Offset: 0,
				Limit:  10,
				IDs:    []string{inValid},
			},
			response: auth.DomainsPage{
				Total:  0,
				Offset: 0,
				Limit:  10,
			},
			err: nil,
		},
		{
			desc: "retrieve by ids and status",
			pm: auth.Page{
				Offset: 0,
				Limit:  10,
				IDs:    []string{items[0].ID, items[1].ID},
				Status: auth.DisabledStatus,
			},
			response: auth.DomainsPage{
				Total:   1,
				Offset:  0,
				Limit:   10,
				Domains: []auth.Domain{items[0]},
			},
		},
		{
			desc: "retrieve by ids and status with invalid status",
			pm: auth.Page{
				Offset: 0,
				Limit:  10,
				IDs:    []string{items[0].ID, items[1].ID},
				Status: 5,
			},
			response: auth.DomainsPage{
				Total:   2,
				Offset:  0,
				Limit:   10,
				Domains: []auth.Domain{items[0], items[1]},
			},
		},
		{
			desc: "retrieve by ids and tags",
			pm: auth.Page{
				Offset: 0,
				Limit:  10,
				IDs:    []string{items[0].ID, items[1].ID},
				Tag:    "test",
			},
			response: auth.DomainsPage{
				Total:   1,
				Offset:  0,
				Limit:   10,
				Domains: []auth.Domain{items[1]},
			},
		},
		{
			desc: " retrieve by ids and metadata",
			pm: auth.Page{
				Offset: 0,
				Limit:  10,
				IDs:    []string{items[1].ID, items[2].ID},
				Metadata: map[string]interface{}{
					"test": "test",
				},
				Status: auth.EnabledStatus,
			},
			response: auth.DomainsPage{
				Total:   2,
				Offset:  0,
				Limit:   10,
				Domains: items[1:3],
			},
		},
		{
			desc: "retrieve by ids and metadata with invalid metadata",
			pm: auth.Page{
				Offset: 0,
				Limit:  10,
				IDs:    []string{items[1].ID, items[2].ID},
				Metadata: map[string]interface{}{
					"test1": "test1",
				},
				Status: auth.EnabledStatus,
			},
			response: auth.DomainsPage{
				Total:  0,
				Offset: 0,
				Limit:  10,
			},
		},
		{
			desc: "retrieve by ids and malfomed metadata",
			pm: auth.Page{
				Offset: 0,
				Limit:  10,
				IDs:    []string{items[1].ID, items[2].ID},
				Metadata: map[string]interface{}{
					"key": make(chan int),
				},
				Status: auth.EnabledStatus,
			},
			response: auth.DomainsPage{},
			err:      repoerr.ErrViewEntity,
		},
		{
			desc: "retrieve all by ids and id",
			pm: auth.Page{
				Offset: 0,
				Limit:  10,
				ID:     items[1].ID,
				IDs:    []string{items[1].ID, items[2].ID},
			},
			response: auth.DomainsPage{
				Total:   1,
				Offset:  0,
				Limit:   10,
				Domains: []auth.Domain{items[1]},
			},
		},
		{
			desc: "retrieve all by ids and id with invalid id",
			pm: auth.Page{
				Offset: 0,
				Limit:  10,
				ID:     inValid,
				IDs:    []string{items[1].ID, items[2].ID},
			},
			response: auth.DomainsPage{
				Total:  0,
				Offset: 0,
				Limit:  10,
			},
		},
		{
			desc: "retrieve all by ids and name",
			pm: auth.Page{
				Offset: 0,
				Limit:  10,
				Name:   items[1].Name,
				IDs:    []string{items[1].ID, items[2].ID},
			},
			response: auth.DomainsPage{
				Total:   1,
				Offset:  0,
				Limit:   10,
				Domains: []auth.Domain{items[1]},
			},
		},
		{
			desc:     "retrieve all by ids with empty page",
			pm:       auth.Page{},
			response: auth.DomainsPage{},
		},
	}

	for _, tc := range cases {
		d, err := repo.RetrieveAllByIDs(context.Background(), tc.pm)
		assert.Equal(t, tc.response, d, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, d))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.err, err))
	}
}

func TestListDomains(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM domains")
		require.Nil(t, err, fmt.Sprintf("clean domains unexpected error: %s", err))
	})

	repo := postgres.NewDomainRepository(database)

	items := []auth.Domain{}
	rDomains := []auth.Domain{}
	policyList := []auth.Policy{}
	for i := 0; i < 10; i++ {
		domain := auth.Domain{
			ID:    testsutil.GenerateUUID(t),
			Name:  fmt.Sprintf(`"test%d"`, i),
			Alias: fmt.Sprintf(`"test%d"`, i),
			Tags:  []string{"test"},
			Metadata: map[string]interface{}{
				"test": "test",
			},
			CreatedBy: userID,
			UpdatedBy: userID,
			Status:    auth.EnabledStatus,
		}
		if i%5 == 0 {
			domain.Status = auth.DisabledStatus
			domain.Tags = []string{"test", "admin"}
			domain.Metadata = map[string]interface{}{
				"test1": "test1",
			}
		}
		policy := auth.Policy{
			SubjectType:     policies.UserType,
			SubjectID:       userID,
			SubjectRelation: policies.AdministratorRelation,
			Relation:        policies.DomainRelation,
			ObjectType:      policies.DomainType,
			ObjectID:        domain.ID,
		}
		_, err := repo.Save(context.Background(), domain)
		require.Nil(t, err, fmt.Sprintf("save domain unexpected error: %s", err))
		items = append(items, domain)
		policyList = append(policyList, policy)
		rDomain := domain
		rDomain.Permission = "domain"
		rDomains = append(rDomains, rDomain)
	}

	err := repo.SavePolicies(context.Background(), policyList...)
	require.Nil(t, err, fmt.Sprintf("failed to save policies %s", policyList))

	cases := []struct {
		desc     string
		pm       auth.Page
		response auth.DomainsPage
		err      error
	}{
		{
			desc: "list all domains successfully",
			pm: auth.Page{
				Offset: 0,
				Limit:  10,
				Status: auth.AllStatus,
			},
			response: auth.DomainsPage{
				Total:   10,
				Offset:  0,
				Limit:   10,
				Domains: items,
			},
			err: nil,
		},
		{
			desc: "list domains with empty page",
			pm: auth.Page{
				Offset: 0,
				Limit:  0,
			},
			response: auth.DomainsPage{
				Total:  8,
				Offset: 0,
				Limit:  0,
			},
			err: nil,
		},
		{
			desc: "list domains with enabled status",
			pm: auth.Page{
				Offset: 0,
				Limit:  10,
				Status: auth.EnabledStatus,
			},
			response: auth.DomainsPage{
				Total:   8,
				Offset:  0,
				Limit:   10,
				Domains: []auth.Domain{items[1], items[2], items[3], items[4], items[6], items[7], items[8], items[9]},
			},
			err: nil,
		},
		{
			desc: "list domains with disabled status",
			pm: auth.Page{
				Offset: 0,
				Limit:  10,
				Status: auth.DisabledStatus,
			},
			response: auth.DomainsPage{
				Total:   2,
				Offset:  0,
				Limit:   10,
				Domains: []auth.Domain{items[0], items[5]},
			},
			err: nil,
		},
		{
			desc: "list domains with subject ID",
			pm: auth.Page{
				Offset:    0,
				Limit:     10,
				SubjectID: userID,
				Status:    auth.AllStatus,
			},
			response: auth.DomainsPage{
				Total:   10,
				Offset:  0,
				Limit:   10,
				Domains: rDomains,
			},
			err: nil,
		},
		{
			desc: "list domains with subject ID and status",
			pm: auth.Page{
				Offset:    0,
				Limit:     10,
				SubjectID: userID,
				Status:    auth.EnabledStatus,
			},
			response: auth.DomainsPage{
				Total:   8,
				Offset:  0,
				Limit:   10,
				Domains: []auth.Domain{rDomains[1], rDomains[2], rDomains[3], rDomains[4], rDomains[6], rDomains[7], rDomains[8], rDomains[9]},
			},
			err: nil,
		},
		{
			desc: "list domains with subject Id and permission",
			pm: auth.Page{
				Offset:     0,
				Limit:      10,
				SubjectID:  userID,
				Permission: "domain",
				Status:     auth.AllStatus,
			},
			response: auth.DomainsPage{
				Total:   10,
				Offset:  0,
				Limit:   10,
				Domains: rDomains,
			},
			err: nil,
		},
		{
			desc: "list domains with subject id and tags",
			pm: auth.Page{
				Offset:    0,
				Limit:     10,
				SubjectID: userID,
				Tag:       "test",
				Status:    auth.AllStatus,
			},
			response: auth.DomainsPage{
				Total:   10,
				Offset:  0,
				Limit:   10,
				Domains: rDomains,
			},
			err: nil,
		},
		{
			desc: "list domains with subject id and metadata",
			pm: auth.Page{
				Offset:    0,
				Limit:     10,
				SubjectID: userID,
				Metadata: map[string]interface{}{
					"test": "test",
				},
				Status: auth.AllStatus,
			},
			response: auth.DomainsPage{
				Total:   8,
				Offset:  0,
				Limit:   10,
				Domains: []auth.Domain{rDomains[1], rDomains[2], rDomains[3], rDomains[4], rDomains[6], rDomains[7], rDomains[8], rDomains[9]},
			},
		},
		{
			desc: "list domains with subject id and metadata with malforned metadata",
			pm: auth.Page{
				Offset:    0,
				Limit:     10,
				SubjectID: userID,
				Metadata: map[string]interface{}{
					"key": make(chan int),
				},
				Status: auth.AllStatus,
			},
			response: auth.DomainsPage{},
			err:      repoerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		d, err := repo.ListDomains(context.Background(), tc.pm)
		assert.Equal(t, tc.response, d, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, d))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.err, err))
	}
}

func TestUpdate(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM domains")
		require.Nil(t, err, fmt.Sprintf("clean domains unexpected error: %s", err))
	})

	updatedName := "test1"
	updatedMetadata := clients.Metadata{
		"test1": "test1",
	}
	updatedTags := []string{"test1"}
	updatedStatus := auth.DisabledStatus
	updatedAlias := "test1"

	repo := postgres.NewDomainRepository(database)

	domain := auth.Domain{
		ID:    domainID,
		Name:  "test",
		Alias: "test",
		Tags:  []string{"test"},
		Metadata: map[string]interface{}{
			"test": "test",
		},
		CreatedBy: userID,
		UpdatedBy: userID,
		Status:    auth.EnabledStatus,
	}

	_, err := repo.Save(context.Background(), domain)
	require.Nil(t, err, fmt.Sprintf("failed to save client %s", domain.ID))

	cases := []struct {
		desc     string
		domainID string
		d        auth.DomainReq
		response auth.Domain
		err      error
	}{
		{
			desc:     "update existing domain name and metadata",
			domainID: domain.ID,
			d: auth.DomainReq{
				Name:     &updatedName,
				Metadata: &updatedMetadata,
			},
			response: auth.Domain{
				ID:    domainID,
				Name:  "test1",
				Alias: "test",
				Tags:  []string{"test"},
				Metadata: map[string]interface{}{
					"test1": "test1",
				},
				CreatedBy: userID,
				UpdatedBy: userID,
				Status:    auth.EnabledStatus,
				UpdatedAt: time.Now(),
			},
			err: nil,
		},
		{
			desc:     "update existing domain name, metadata, tags, status and alias",
			domainID: domain.ID,
			d: auth.DomainReq{
				Name:     &updatedName,
				Metadata: &updatedMetadata,
				Tags:     &updatedTags,
				Status:   &updatedStatus,
				Alias:    &updatedAlias,
			},
			response: auth.Domain{
				ID:    domainID,
				Name:  "test1",
				Alias: "test1",
				Tags:  []string{"test1"},
				Metadata: map[string]interface{}{
					"test1": "test1",
				},
				CreatedBy: userID,
				UpdatedBy: userID,
				Status:    auth.DisabledStatus,
				UpdatedAt: time.Now(),
			},
			err: nil,
		},
		{
			desc:     "update non-existing domain",
			domainID: inValid,
			d: auth.DomainReq{
				Name:     &updatedName,
				Metadata: &updatedMetadata,
			},
			response: auth.Domain{},
			err:      repoerr.ErrFailedOpDB,
		},
		{
			desc:     "update domain with empty ID",
			domainID: "",
			d: auth.DomainReq{
				Name:     &updatedName,
				Metadata: &updatedMetadata,
			},
			response: auth.Domain{},
			err:      repoerr.ErrFailedOpDB,
		},
		{
			desc:     "update domain with malformed metadata",
			domainID: domainID,
			d: auth.DomainReq{
				Name:     &updatedName,
				Metadata: &clients.Metadata{"key": make(chan int)},
			},
			response: auth.Domain{},
			err:      repoerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		d, err := repo.Update(context.Background(), tc.domainID, userID, tc.d)
		d.UpdatedAt = tc.response.UpdatedAt
		assert.Equal(t, tc.response, d, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, d))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestDelete(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM domains")
		require.Nil(t, err, fmt.Sprintf("clean domains unexpected error: %s", err))
	})

	repo := postgres.NewDomainRepository(database)

	domain := auth.Domain{
		ID:    domainID,
		Name:  "test",
		Alias: "test",
		Tags:  []string{"test"},
		Metadata: map[string]interface{}{
			"test": "test",
		},
		CreatedBy: userID,
		UpdatedBy: userID,
		Status:    auth.EnabledStatus,
	}

	_, err := repo.Save(context.Background(), domain)
	require.Nil(t, err, fmt.Sprintf("failed to save client %s", domain.ID))

	cases := []struct {
		desc     string
		domainID string
		err      error
	}{
		{
			desc:     "delete existing domain",
			domainID: domain.ID,
			err:      nil,
		},
		{
			desc:     "delete non-existing domain",
			domainID: inValid,
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "delete domain with empty ID",
			domainID: "",
			err:      repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := repo.Delete(context.Background(), tc.domainID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestCheckPolicy(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM policies")
		require.Nil(t, err, fmt.Sprintf("clean policies unexpected error: %s", err))
	})

	repo := postgres.NewDomainRepository(database)

	policy := auth.Policy{
		SubjectType:     policies.UserType,
		SubjectID:       userID,
		SubjectRelation: policies.AdministratorRelation,
		Relation:        policies.DomainRelation,
		ObjectType:      policies.DomainType,
		ObjectID:        domainID,
	}

	err := repo.SavePolicies(context.Background(), policy)
	require.Nil(t, err, fmt.Sprintf("failed to save policy %s", policy.SubjectID))

	cases := []struct {
		desc   string
		policy auth.Policy
		err    error
	}{
		{
			desc:   "check valid policy",
			policy: policy,
			err:    nil,
		},
		{
			desc: "check policy with invalid subject type",
			policy: auth.Policy{
				SubjectType:     inValid,
				SubjectID:       userID,
				SubjectRelation: policies.AdministratorRelation,
				Relation:        policies.DomainRelation,
				ObjectType:      policies.DomainType,
				ObjectID:        domainID,
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "check policy with invalid subject id",
			policy: auth.Policy{
				SubjectType:     policies.UserType,
				SubjectID:       inValid,
				SubjectRelation: policies.AdministratorRelation,
				Relation:        policies.DomainRelation,
				ObjectType:      policies.DomainType,
				ObjectID:        domainID,
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "check policy with invalid subject relation",
			policy: auth.Policy{
				SubjectType:     policies.UserType,
				SubjectID:       userID,
				SubjectRelation: inValid,
				Relation:        policies.DomainRelation,
				ObjectType:      policies.DomainType,
				ObjectID:        domainID,
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "check policy with invalid relation",
			policy: auth.Policy{
				SubjectType:     policies.UserType,
				SubjectID:       userID,
				SubjectRelation: policies.AdministratorRelation,
				Relation:        inValid,
				ObjectType:      policies.DomainType,
				ObjectID:        domainID,
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "check policy with invalid object type",
			policy: auth.Policy{
				SubjectType:     policies.UserType,
				SubjectID:       userID,
				SubjectRelation: policies.AdministratorRelation,
				Relation:        policies.DomainRelation,
				ObjectType:      inValid,
				ObjectID:        domainID,
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "check policy with invalid object id",
			policy: auth.Policy{
				SubjectType:     policies.UserType,
				SubjectID:       userID,
				SubjectRelation: policies.AdministratorRelation,
				Relation:        policies.DomainRelation,
				ObjectType:      policies.DomainType,
				ObjectID:        inValid,
			},
			err: repoerr.ErrNotFound,
		},
	}
	for _, tc := range cases {
		err := repo.CheckPolicy(context.Background(), tc.policy)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.err, err))
	}
}

func TestDeleteUserPolicies(t *testing.T) {
	repo := postgres.NewDomainRepository(database)

	domain := auth.Domain{
		ID:    domainID,
		Name:  "test",
		Alias: "test",
		Tags:  []string{"test"},
		Metadata: map[string]interface{}{
			"test": "test",
		},
		CreatedBy:  userID,
		UpdatedBy:  userID,
		Status:     auth.EnabledStatus,
		Permission: "admin",
	}

	policy := auth.Policy{
		SubjectType:     policies.UserType,
		SubjectID:       userID,
		SubjectRelation: "admin",
		Relation:        "admin",
		ObjectType:      policies.DomainType,
		ObjectID:        domainID,
	}

	_, err := repo.Save(context.Background(), domain)
	require.Nil(t, err, fmt.Sprintf("failed to save domain %s", domain.ID))

	err = repo.SavePolicies(context.Background(), policy)
	require.Nil(t, err, fmt.Sprintf("failed to save policy %s", policy.SubjectID))

	cases := []struct {
		desc string
		id   string
		err  error
	}{
		{
			desc: "delete valid user policy",
			id:   userID,
			err:  nil,
		},
		{
			desc: "delete invalid user policy",
			id:   inValid,
			err:  nil,
		},
	}

	for _, tc := range cases {
		err := repo.DeleteUserPolicies(context.Background(), tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.err, err))
	}
}
