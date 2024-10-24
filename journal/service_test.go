// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package journal_test

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/journal"
	"github.com/absmach/magistrala/journal/mocks"
	mgauthn "github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	validJournal = journal.Journal{
		Operation:  "user.create",
		OccurredAt: time.Now().Add(-time.Hour),
		Attributes: map[string]interface{}{
			"temperature": rand.Float64(),
			"humidity":    rand.Float64(),
		},
		Metadata: map[string]interface{}{
			"sensor_id": rand.Intn(1000),
		},
	}
	idProvider = uuid.New()
)

func TestSave(t *testing.T) {
	repo := new(mocks.Repository)
	svc := journal.NewService(idProvider, repo)

	cases := []struct {
		desc    string
		journal journal.Journal
		repoErr error
		err     error
	}{
		{
			desc:    "successful with ID and EntityType",
			journal: validJournal,
			repoErr: nil,
			err:     nil,
		},
		{
			desc:    "with repo error",
			repoErr: repoerr.ErrCreateEntity,
			err:     repoerr.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("Save", context.Background(), mock.Anything).Return(tc.repoErr)
			err := svc.Save(context.Background(), tc.journal)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			repoCall.Unset()
		})
	}
}

func TestReadAll(t *testing.T) {
	repo := new(mocks.Repository)
	svc := journal.NewService(idProvider, repo)

	validSession := mgauthn.Session{DomainUserID: testsutil.GenerateUUID(t), UserID: testsutil.GenerateUUID(t), DomainID: testsutil.GenerateUUID(t)}
	validPage := journal.Page{
		Offset:     0,
		Limit:      10,
		EntityID:   testsutil.GenerateUUID(t),
		EntityType: journal.ClientEntity,
	}

	cases := []struct {
		desc    string
		session mgauthn.Session
		page    journal.Page
		resp    journal.JournalsPage
		authErr error
		repoErr error
		err     error
	}{
		{
			desc:    "successful",
			session: validSession,
			page:    validPage,
			resp: journal.JournalsPage{
				Total:    1,
				Offset:   0,
				Limit:    10,
				Journals: []journal.Journal{validJournal},
			},
			authErr: nil,
			repoErr: nil,
			err:     nil,
		},
		{
			desc:    "successful for user",
			session: validSession,
			page: journal.Page{
				Offset:     0,
				Limit:      10,
				EntityID:   testsutil.GenerateUUID(t),
				EntityType: journal.UserEntity,
			},
			resp: journal.JournalsPage{
				Total:    1,
				Offset:   0,
				Limit:    10,
				Journals: []journal.Journal{validJournal},
			},
			authErr: nil,
			repoErr: nil,
			err:     nil,
		},
		{
			desc:    "with repo error",
			session: validSession,
			page:    validPage,
			resp:    journal.JournalsPage{},
			repoErr: repoerr.ErrViewEntity,
			err:     repoerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("RetrieveAll", context.Background(), tc.page).Return(tc.resp, tc.repoErr)
			resp, err := svc.RetrieveAll(context.Background(), tc.session, tc.page)
			if tc.err == nil {
				assert.Equal(t, tc.resp, resp, tc.desc)
			}
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			repoCall.Unset()
		})
	}
}
