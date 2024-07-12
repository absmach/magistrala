// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package twins_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/absmach/magistrala"
	authmocks "github.com/absmach/magistrala/auth/mocks"
	"github.com/absmach/magistrala/internal/testsutil"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/absmach/magistrala/twins"
	"github.com/absmach/magistrala/twins/mocks"
	"github.com/absmach/senml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	twinName     = "name"
	wrongID      = ""
	token        = "token"
	invalidToken = "invalidToken"
	email        = "user@example.com"
	numRecs      = 100
	retained     = "saved"
	validID      = "123e4567-e89b-12d3-a456-426614174000"
)

var (
	subtopics = []string{"engine", "chassis", "wheel_2"}
	channels  = []string{"01ec3c3e-0e66-4e69-9751-a0545b44e08f", "48061e4f-7c23-4f5c-9012-0f9b7cd9d18d", "5b2180e4-e96b-4469-9dc1-b6745078d0b6"}
)

func NewService() (twins.Service, *authmocks.AuthClient, *mocks.TwinRepository, *mocks.TwinCache, *mocks.StateRepository) {
	auth := new(authmocks.AuthClient)
	twinsRepo := new(mocks.TwinRepository)
	twinCache := new(mocks.TwinCache)
	statesRepo := new(mocks.StateRepository)
	idProvider := uuid.NewMock()
	subs := map[string]string{"chanID": "chanID"}
	broker := mocks.NewBroker(subs)

	return twins.New(broker, auth, twinsRepo, twinCache, statesRepo, idProvider, "chanID", mglog.NewMock()), auth, twinsRepo, twinCache, statesRepo
}

func TestAddTwin(t *testing.T) {
	svc, auth, twinRepo, twinCache, _ := NewService()
	twin := twins.Twin{}
	def := twins.Definition{}

	cases := []struct {
		desc        string
		twin        twins.Twin
		token       string
		err         error
		saveErr     error
		identifyErr error
		userID      string
	}{
		{
			desc:        "add new twin",
			twin:        twin,
			token:       token,
			err:         nil,
			saveErr:     nil,
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:        "add twin with wrong credentials",
			twin:        twin,
			token:       invalidToken,
			err:         svcerr.ErrAuthentication,
			saveErr:     svcerr.ErrCreateEntity,
			identifyErr: svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		authCall := auth.On("Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: tc.userID}, tc.identifyErr)
		repoCall := twinRepo.On("Save", context.Background(), mock.Anything).Return(retained, tc.saveErr)
		cacheCall := twinCache.On("Save", context.Background(), mock.Anything).Return(tc.err)
		_, err := svc.AddTwin(context.Background(), tc.token, tc.twin, def)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		authCall.Unset()
		repoCall.Unset()
		cacheCall.Unset()
	}
}

func TestUpdateTwin(t *testing.T) {
	svc, auth, twinRepo, twinCache, _ := NewService()

	other := twins.Twin{}
	def := twins.Definition{}
	twin := twins.Twin{
		Owner: email,
		ID:    testsutil.GenerateUUID(t),
		Name:  twinName,
	}

	other.ID = wrongID

	cases := []struct {
		desc        string
		twin        twins.Twin
		token       string
		err         error
		retrieveErr error
		updateErr   error
		identifyErr error
		userID      string
	}{
		{
			desc:        "update existing twin",
			twin:        twin,
			token:       token,
			err:         nil,
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:        "update twin with wrong credentials",
			twin:        twin,
			token:       invalidToken,
			err:         svcerr.ErrAuthentication,
			retrieveErr: svcerr.ErrNotFound,
			updateErr:   svcerr.ErrUpdateEntity,
			identifyErr: svcerr.ErrAuthentication,
		},
		{
			desc:        "update non-existing twin",
			twin:        other,
			token:       token,
			err:         svcerr.ErrNotFound,
			retrieveErr: svcerr.ErrNotFound,
			updateErr:   svcerr.ErrUpdateEntity,
			identifyErr: nil,
			userID:      validID,
		},
	}

	for _, tc := range cases {
		authCall := auth.On("Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: tc.userID}, tc.identifyErr)
		repoCall := twinRepo.On("RetrieveByID", context.Background(), tc.twin.ID).Return(tc.twin, tc.retrieveErr)
		repoCall1 := twinRepo.On("Update", context.Background(), mock.Anything).Return(tc.updateErr)
		cacheCall := twinCache.On("Update", context.Background(), mock.Anything).Return(tc.err)
		err := svc.UpdateTwin(context.Background(), tc.token, tc.twin, def)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		authCall.Unset()
		repoCall.Unset()
		repoCall1.Unset()
		cacheCall.Unset()
	}
}

func TestViewTwin(t *testing.T) {
	svc, auth, twinRepo, _, _ := NewService()

	twin := twins.Twin{
		Owner: email,
		ID:    testsutil.GenerateUUID(t),
		Name:  twinName,
	}

	cases := []struct {
		desc        string
		id          string
		token       string
		err         error
		identifyErr error
		userID      string
	}{
		{
			desc:        "view existing twin",
			id:          twin.ID,
			token:       token,
			err:         nil,
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:        "view twin with wrong credentials",
			id:          twin.ID,
			token:       invalidToken,
			err:         svcerr.ErrAuthentication,
			identifyErr: svcerr.ErrAuthentication,
		},
		{
			desc:        "view non-existing twin",
			id:          wrongID,
			token:       token,
			err:         svcerr.ErrNotFound,
			identifyErr: nil,
			userID:      validID,
		},
	}

	for _, tc := range cases {
		authCall := auth.On("Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: tc.userID}, tc.identifyErr)
		repoCall := twinRepo.On("RetrieveByID", context.Background(), tc.id).Return(twins.Twin{}, tc.err)
		_, err := svc.ViewTwin(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		authCall.Unset()
		repoCall.Unset()
	}
}

func TestListTwins(t *testing.T) {
	svc, auth, twinRepo, _, _ := NewService()
	twin := twins.Twin{Name: twinName, Owner: email}
	m := make(map[string]interface{})
	m["serial"] = "123456"
	twin.Metadata = m

	n := uint64(10)

	cases := []struct {
		desc        string
		token       string
		offset      uint64
		limit       uint64
		size        uint64
		metadata    map[string]interface{}
		err         error
		repoerr     error
		identifyErr error
		userID      string
	}{
		{
			desc:        "list all twins",
			token:       token,
			offset:      0,
			limit:       n,
			size:        n,
			err:         nil,
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:        "list with zero limit",
			token:       token,
			limit:       0,
			offset:      0,
			size:        0,
			err:         nil,
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:        "list with offset and limit",
			token:       token,
			offset:      8,
			limit:       5,
			size:        2,
			err:         nil,
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:        "list with wrong credentials",
			token:       invalidToken,
			limit:       0,
			offset:      n,
			err:         svcerr.ErrAuthentication,
			identifyErr: svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		authCall := auth.On("Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: tc.userID}, tc.identifyErr)
		repoCall := twinRepo.On("RetrieveAll", context.Background(), mock.Anything, tc.offset, tc.limit, twinName, mock.Anything).Return(twins.Page{}, tc.err)
		_, err := svc.ListTwins(context.Background(), tc.token, tc.offset, tc.limit, twinName, tc.metadata)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		authCall.Unset()
		repoCall.Unset()
	}
}

func TestRemoveTwin(t *testing.T) {
	svc, auth, twinRepo, twinCache, _ := NewService()
	twin := twins.Twin{
		Owner: email,
		ID:    testsutil.GenerateUUID(t),
		Name:  twinName,
	}

	cases := []struct {
		desc        string
		id          string
		token       string
		err         error
		removeErr   error
		identifyErr error
		userID      string
	}{
		{
			desc:        "remove twin with wrong credentials",
			id:          twin.ID,
			token:       invalidToken,
			err:         svcerr.ErrAuthentication,
			removeErr:   svcerr.ErrRemoveEntity,
			identifyErr: svcerr.ErrAuthentication,
		},
		{
			desc:        "remove existing twin",
			id:          twin.ID,
			token:       token,
			err:         nil,
			removeErr:   nil,
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:        "remove removed twin",
			id:          twin.ID,
			token:       token,
			err:         nil,
			removeErr:   nil,
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:        "remove non-existing twin",
			id:          wrongID,
			token:       token,
			err:         nil,
			removeErr:   nil,
			identifyErr: nil,
			userID:      validID,
		},
	}

	for _, tc := range cases {
		authCall := auth.On("Identify", context.Background(), &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: tc.userID}, tc.identifyErr)
		repoCall := twinRepo.On("Remove", context.Background(), tc.id).Return(tc.removeErr)
		cacheCall := twinCache.On("Remove", context.Background(), tc.id).Return(tc.err)
		err := svc.RemoveTwin(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		authCall.Unset()
		repoCall.Unset()
		cacheCall.Unset()
	}
}

func TestSaveStates(t *testing.T) {
	svc, auth, twinRepo, twinCache, stateRepo := NewService()

	def := mocks.CreateDefinition(channels[0:2], subtopics[0:2])
	twin := twins.Twin{
		Owner:       email,
		ID:          testsutil.GenerateUUID(t),
		Name:        twinName,
		Definitions: []twins.Definition{def},
	}

	attr := def.Attributes[0]
	attrSansTwin := mocks.CreateDefinition(channels[2:3], subtopics[2:3]).Attributes[0]

	defWildcard := mocks.CreateDefinition(channels[0:2], []string{twins.SubtopicWildcard, twins.SubtopicWildcard})
	twWildcard := twins.Twin{
		Definitions: []twins.Definition{defWildcard},
	}

	recs := make([]senml.Record, numRecs)

	var ttlAdded uint64

	cases := []struct {
		desc   string
		recs   []senml.Record
		attr   twins.Attribute
		size   uint64
		err    error
		String []string
		page   twins.StatesPage
	}{
		{
			desc: "add 100 states",
			recs: recs,
			attr: attr,
			size: numRecs,
			err:  nil,
			page: twins.StatesPage{
				PageMetadata: twins.PageMetadata{
					Total: numRecs,
				},
			},
		},
		{
			desc: "add 20 states",
			recs: recs[10:30],
			attr: attr,
			size: 20,
			err:  nil,
			page: twins.StatesPage{
				PageMetadata: twins.PageMetadata{
					Total: numRecs + 20,
				},
			},
		},
		{
			desc: "add 20 states for atttribute without twin",
			recs: recs[30:50],
			size: 0,
			attr: attrSansTwin,
			err:  svcerr.ErrNotFound,
			page: twins.StatesPage{
				PageMetadata: twins.PageMetadata{
					Total: numRecs + 20,
				},
			},
		},
		{
			desc: "use empty senml record",
			recs: []senml.Record{},
			attr: attr,
			size: 0,
			err:  nil,
			page: twins.StatesPage{
				PageMetadata: twins.PageMetadata{
					Total: numRecs + 20,
				},
			},
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", context.TODO(), &magistrala.IdentityReq{Token: token}).Return(&magistrala.IdentityRes{Id: testsutil.GenerateUUID(t)}, nil)
		message, err := mocks.CreateMessage(tc.attr, tc.recs)
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

		repoCall1 := twinRepo.On("RetrieveByAttribute", context.Background(), mock.Anything, mock.Anything).Return(tc.String, nil)
		repoCall2 := twinRepo.On("SaveIDs", context.Background(), mock.Anything, mock.Anything, mock.Anything).Return(tc.err)
		repoCall3 := twinCache.On("IDs", context.Background(), mock.Anything, mock.Anything).Return(tc.String, nil)
		err = svc.SaveStates(context.Background(), message)
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

		ttlAdded += tc.size
		repoCall4 := stateRepo.On("RetrieveAll", context.TODO(), mock.Anything, mock.Anything, twin.ID).Return(tc.page, nil)
		page, err := svc.ListStates(context.TODO(), token, 0, 10, twin.ID)
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		assert.Equal(t, ttlAdded, page.Total, fmt.Sprintf("%s: expected %d total got %d total\n", tc.desc, ttlAdded, page.Total))

		repoCall5 := stateRepo.On("RetrieveAll", context.TODO(), mock.Anything, mock.Anything, twWildcard.ID).Return(tc.page, nil)
		page, err = svc.ListStates(context.TODO(), token, 0, 10, twWildcard.ID)
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		assert.Equal(t, ttlAdded, page.Total, fmt.Sprintf("%s: expected %d total got %d total\n", tc.desc, ttlAdded, page.Total))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
		repoCall4.Unset()
		repoCall5.Unset()
	}
}

func TestListStates(t *testing.T) {
	svc, auth, _, _, stateRepo := NewService()

	def := mocks.CreateDefinition(channels[0:2], subtopics[0:2])
	twin := twins.Twin{
		Owner:       email,
		ID:          testsutil.GenerateUUID(t),
		Name:        twinName,
		Definitions: []twins.Definition{def},
	}

	tw2 := twins.Twin{
		Owner:       email,
		Definitions: []twins.Definition{mocks.CreateDefinition(channels[2:3], subtopics[2:3])},
	}

	cases := []struct {
		desc        string
		id          string
		token       string
		offset      uint64
		limit       uint64
		size        int
		err         error
		page        twins.StatesPage
		identifyErr error
		userID      string
	}{
		{
			desc:   "get a list of first 10 states",
			id:     twin.ID,
			token:  token,
			offset: 0,
			limit:  10,
			size:   10,
			err:    nil,
			page: twins.StatesPage{
				States: genStates(10),
			},
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:   "get a list of last 10 states",
			id:     twin.ID,
			token:  token,
			offset: numRecs - 10,
			limit:  numRecs,
			size:   10,
			err:    nil,
			page: twins.StatesPage{
				States: genStates(10),
			},
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:   "get a list of last 10 states with limit > numRecs",
			id:     twin.ID,
			token:  token,
			offset: numRecs - 10,
			limit:  numRecs + 10,
			size:   10,
			err:    nil,
			page: twins.StatesPage{
				States: genStates(10),
			},
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:        "get a list of first 10 states with offset == numRecs",
			id:          twin.ID,
			token:       token,
			offset:      numRecs,
			limit:       numRecs + 10,
			size:        0,
			err:         nil,
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:        "get a list with wrong user token",
			id:          twin.ID,
			token:       invalidToken,
			offset:      0,
			limit:       10,
			size:        0,
			err:         svcerr.ErrAuthentication,
			identifyErr: svcerr.ErrAuthentication,
		},
		{
			desc:        "get a list with id of non-existent twin",
			id:          "1234567890",
			token:       token,
			offset:      0,
			limit:       10,
			size:        0,
			err:         nil,
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:        "get a list with id of existing twin without states ",
			id:          tw2.ID,
			token:       token,
			offset:      0,
			limit:       10,
			size:        0,
			err:         nil,
			identifyErr: nil,
			userID:      validID,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", context.TODO(), &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: tc.userID}, tc.identifyErr)
		repoCall1 := stateRepo.On("RetrieveAll", context.TODO(), mock.Anything, mock.Anything, tc.id).Return(tc.page, nil)
		page, err := svc.ListStates(context.TODO(), tc.token, tc.offset, tc.limit, tc.id)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.size, len(page.States), fmt.Sprintf("%s: expected %d total got %d total\n", tc.desc, tc.size, len(page.States)))
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func genStates(length int) []twins.State {
	states := make([]twins.State, length)
	for i := range states {
		states[i] = twins.State{}
	}
	return states
}
