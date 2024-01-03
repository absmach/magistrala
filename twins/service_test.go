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
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/twins"
	"github.com/absmach/magistrala/twins/mocks"
	"github.com/absmach/senml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	twinName = "name"
	wrongID  = ""
	token    = "token"
	email    = "user@example.com"
	numRecs  = 100
)

var (
	subtopics = []string{"engine", "chassis", "wheel_2"}
	channels  = []string{"01ec3c3e-0e66-4e69-9751-a0545b44e08f", "48061e4f-7c23-4f5c-9012-0f9b7cd9d18d", "5b2180e4-e96b-4469-9dc1-b6745078d0b6"}
)

func TestAddTwin(t *testing.T) {
	svc, auth := mocks.NewService()
	twin := twins.Twin{}
	def := twins.Definition{}

	cases := []struct {
		desc  string
		twin  twins.Twin
		token string
		err   error
	}{
		{
			desc:  "add new twin",
			twin:  twin,
			token: token,
			err:   nil,
		},
		{
			desc:  "add twin with wrong credentials",
			twin:  twin,
			token: authmocks.InvalidValue,
			err:   svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: testsutil.GenerateUUID(t)}, nil)
		_, err := svc.AddTwin(context.Background(), tc.token, tc.twin, def)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
	}
}

func TestUpdateTwin(t *testing.T) {
	svc, auth := mocks.NewService()
	twin := twins.Twin{}
	other := twins.Twin{}
	def := twins.Definition{}

	other.ID = wrongID
	repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: token}).Return(&magistrala.IdentityRes{Id: testsutil.GenerateUUID(t)}, nil)
	saved, err := svc.AddTwin(context.Background(), token, twin, def)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	repoCall.Unset()

	saved.Name = twinName

	cases := []struct {
		desc  string
		twin  twins.Twin
		token string
		err   error
	}{
		{
			desc:  "update existing twin",
			twin:  saved,
			token: token,
			err:   nil,
		},
		{
			desc:  "update twin with wrong credentials",
			twin:  saved,
			token: authmocks.InvalidValue,
			err:   svcerr.ErrAuthentication,
		},
		{
			desc:  "update non-existing twin",
			twin:  other,
			token: token,
			err:   svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: testsutil.GenerateUUID(t)}, nil)
		err := svc.UpdateTwin(context.Background(), tc.token, tc.twin, def)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
	}
}

func TestViewTwin(t *testing.T) {
	svc, auth := mocks.NewService()
	twin := twins.Twin{}
	def := twins.Definition{}
	repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: token}).Return(&magistrala.IdentityRes{Id: testsutil.GenerateUUID(t)}, nil)
	saved, err := svc.AddTwin(context.Background(), token, twin, def)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	repoCall.Unset()

	cases := []struct {
		desc  string
		id    string
		token string
		err   error
	}{
		{
			desc:  "view existing twin",
			id:    saved.ID,
			token: token,
			err:   nil,
		},
		{
			desc:  "view twin with wrong credentials",
			id:    saved.ID,
			token: authmocks.InvalidValue,
			err:   svcerr.ErrAuthentication,
		},
		{
			desc:  "view non-existing twin",
			id:    wrongID,
			token: token,
			err:   svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: testsutil.GenerateUUID(t)}, nil)
		_, err := svc.ViewTwin(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
	}
}

func TestListTwins(t *testing.T) {
	svc, auth := mocks.NewService()
	twin := twins.Twin{Name: twinName, Owner: email}
	def := twins.Definition{}
	m := make(map[string]interface{})
	m["serial"] = "123456"
	twin.Metadata = m

	n := uint64(10)
	for i := uint64(0); i < n; i++ {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: token}).Return(&magistrala.IdentityRes{Id: testsutil.GenerateUUID(t)}, nil)
		_, err := svc.AddTwin(context.Background(), token, twin, def)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
		repoCall.Unset()
	}

	cases := []struct {
		desc     string
		token    string
		offset   uint64
		limit    uint64
		size     uint64
		metadata map[string]interface{}
		err      error
	}{
		{
			desc:   "list all twins",
			token:  token,
			offset: 0,
			limit:  n,
			size:   n,
			err:    nil,
		},
		{
			desc:   "list with zero limit",
			token:  token,
			limit:  0,
			offset: 0,
			size:   0,
			err:    nil,
		},
		{
			desc:   "list with offset and limit",
			token:  token,
			offset: 8,
			limit:  5,
			size:   2,
			err:    nil,
		},
		{
			desc:   "list with wrong credentials",
			token:  authmocks.InvalidValue,
			limit:  0,
			offset: n,
			err:    svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: testsutil.GenerateUUID(t)}, nil)
		_, err := svc.ListTwins(context.Background(), tc.token, tc.offset, tc.limit, twinName, tc.metadata)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
	}
}

func TestRemoveTwin(t *testing.T) {
	svc, auth := mocks.NewService()
	twin := twins.Twin{}
	def := twins.Definition{}
	repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: token}).Return(&magistrala.IdentityRes{Id: testsutil.GenerateUUID(t)}, nil)
	saved, err := svc.AddTwin(context.Background(), token, twin, def)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	repoCall.Unset()

	cases := []struct {
		desc  string
		id    string
		token string
		err   error
	}{
		{
			desc:  "remove twin with wrong credentials",
			id:    saved.ID,
			token: authmocks.InvalidValue,
			err:   svcerr.ErrAuthentication,
		},
		{
			desc:  "remove existing twin",
			id:    saved.ID,
			token: token,
			err:   nil,
		},
		{
			desc:  "remove removed twin",
			id:    saved.ID,
			token: token,
			err:   nil,
		},
		{
			desc:  "remove non-existing twin",
			id:    wrongID,
			token: token,
			err:   nil,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: testsutil.GenerateUUID(t)}, nil)
		err := svc.RemoveTwin(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
	}
}

func TestSaveStates(t *testing.T) {
	svc, auth := mocks.NewService()

	twin := twins.Twin{Owner: email}
	def := mocks.CreateDefinition(channels[0:2], subtopics[0:2])
	attr := def.Attributes[0]
	attrSansTwin := mocks.CreateDefinition(channels[2:3], subtopics[2:3]).Attributes[0]
	repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: token}).Return(&magistrala.IdentityRes{Id: testsutil.GenerateUUID(t)}, nil)
	tw, err := svc.AddTwin(context.Background(), token, twin, def)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	repoCall.Unset()

	defWildcard := mocks.CreateDefinition(channels[0:2], []string{twins.SubtopicWildcard, twins.SubtopicWildcard})
	repoCall = auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: token}).Return(&magistrala.IdentityRes{Id: testsutil.GenerateUUID(t)}, nil)
	twWildcard, err := svc.AddTwin(context.Background(), token, twin, defWildcard)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	repoCall.Unset()

	recs := make([]senml.Record, numRecs)
	mocks.CreateSenML(recs)

	var ttlAdded uint64

	cases := []struct {
		desc string
		recs []senml.Record
		attr twins.Attribute
		size uint64
		err  error
	}{
		{
			desc: "add 100 states",
			recs: recs,
			attr: attr,
			size: numRecs,
			err:  nil,
		},
		{
			desc: "add 20 states",
			recs: recs[10:30],
			attr: attr,
			size: 20,
			err:  nil,
		},
		{
			desc: "add 20 states for atttribute without twin",
			recs: recs[30:50],
			size: 0,
			attr: attrSansTwin,
			err:  svcerr.ErrNotFound,
		},
		{
			desc: "use empty senml record",
			recs: []senml.Record{},
			attr: attr,
			size: 0,
			err:  nil,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: token}).Return(&magistrala.IdentityRes{Id: testsutil.GenerateUUID(t)}, nil)
		message, err := mocks.CreateMessage(tc.attr, tc.recs)
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

		err = svc.SaveStates(context.Background(), message)
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

		ttlAdded += tc.size
		page, err := svc.ListStates(context.TODO(), token, 0, 10, tw.ID)
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		assert.Equal(t, ttlAdded, page.Total, fmt.Sprintf("%s: expected %d total got %d total\n", tc.desc, ttlAdded, page.Total))

		page, err = svc.ListStates(context.TODO(), token, 0, 10, twWildcard.ID)
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		assert.Equal(t, ttlAdded, page.Total, fmt.Sprintf("%s: expected %d total got %d total\n", tc.desc, ttlAdded, page.Total))
		repoCall.Unset()
	}
}

func TestListStates(t *testing.T) {
	svc, auth := mocks.NewService()

	twin := twins.Twin{Owner: email}
	def := mocks.CreateDefinition(channels[0:2], subtopics[0:2])
	attr := def.Attributes[0]
	repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: token}).Return(&magistrala.IdentityRes{Id: testsutil.GenerateUUID(t)}, nil)
	tw, err := svc.AddTwin(context.Background(), token, twin, def)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	repoCall.Unset()

	repoCall = auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: token}).Return(&magistrala.IdentityRes{Id: testsutil.GenerateUUID(t)}, nil)
	tw2, err := svc.AddTwin(context.Background(), token,
		twins.Twin{Owner: email},
		mocks.CreateDefinition(channels[2:3], subtopics[2:3]))
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	repoCall.Unset()

	recs := make([]senml.Record, numRecs)
	mocks.CreateSenML(recs)
	message, err := mocks.CreateMessage(attr, recs)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	repoCall = auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: token}).Return(&magistrala.IdentityRes{Id: testsutil.GenerateUUID(t)}, nil)
	err = svc.SaveStates(context.Background(), message)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	repoCall.Unset()

	cases := []struct {
		desc   string
		id     string
		token  string
		offset uint64
		limit  uint64
		size   int
		err    error
	}{
		{
			desc:   "get a list of first 10 states",
			id:     tw.ID,
			token:  token,
			offset: 0,
			limit:  10,
			size:   10,
			err:    nil,
		},
		{
			desc:   "get a list of last 10 states",
			id:     tw.ID,
			token:  token,
			offset: numRecs - 10,
			limit:  numRecs,
			size:   10,
			err:    nil,
		},
		{
			desc:   "get a list of last 10 states with limit > numRecs",
			id:     tw.ID,
			token:  token,
			offset: numRecs - 10,
			limit:  numRecs + 10,
			size:   10,
			err:    nil,
		},
		{
			desc:   "get a list of first 10 states with offset == numRecs",
			id:     tw.ID,
			token:  token,
			offset: numRecs,
			limit:  numRecs + 10,
			size:   0,
			err:    nil,
		},
		{
			desc:   "get a list with wrong user token",
			id:     tw.ID,
			token:  authmocks.InvalidValue,
			offset: 0,
			limit:  10,
			size:   0,
			err:    svcerr.ErrAuthentication,
		},
		{
			desc:   "get a list with id of non-existent twin",
			id:     "1234567890",
			token:  token,
			offset: 0,
			limit:  10,
			size:   0,
			err:    nil,
		},
		{
			desc:   "get a list with id of existing twin without states ",
			id:     tw2.ID,
			token:  token,
			offset: 0,
			limit:  10,
			size:   0,
			err:    nil,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: testsutil.GenerateUUID(t)}, nil)
		page, err := svc.ListStates(context.TODO(), tc.token, tc.offset, tc.limit, tc.id)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.size, len(page.States), fmt.Sprintf("%s: expected %d total got %d total\n", tc.desc, tc.size, len(page.States)))
		repoCall.Unset()
	}
}
