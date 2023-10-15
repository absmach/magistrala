package sdk_test

// import (
// 	"fmt"
// 	"net/http"
// 	"net/http/httptest"
// 	"testing"
// 	"time"

// 	"github.com/go-zoo/bone"
// 	"github.com/mainflux/mainflux/internal/apiutil"
// 	"github.com/mainflux/mainflux/internal/testsutil"
// 	mflog "github.com/mainflux/mainflux/logger"
// 	mfclients "github.com/mainflux/mainflux/pkg/clients"
// 	"github.com/mainflux/mainflux/pkg/errors"
// 	sdk "github.com/mainflux/mainflux/pkg/sdk/go"
// 	"github.com/mainflux/mainflux/things/clients"
// 	"github.com/mainflux/mainflux/things/clients/api"
// 	"github.com/mainflux/mainflux/things/clients/mocks"
// 	gmocks "github.com/mainflux/mainflux/things/groups/mocks"
// 	"github.com/mainflux/mainflux/things/policies"
// 	papi "github.com/mainflux/mainflux/things/policies/api/http"
// 	pmocks "github.com/mainflux/mainflux/things/policies/mocks"
// 	cmocks "github.com/mainflux/mainflux/users/clients/mocks"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/mock"
// )

// var (
// 	adminToken        = "token"
// 	userToken         = "userToken"
// 	adminID           = generateUUID(&testing.T{})
// 	userID            = generateUUID(&testing.T{})
// 	users             = map[string]string{adminToken: adminID, userToken: userID}
// 	adminRelationKeys = []string{"c_update", "c_list", "c_delete", "c_share"}
// 	uadminPolicy      = cmocks.SubjectSet{Subject: adminID, Relation: adminRelationKeys}
// )

// func newThingsServer(svc clients.Service, psvc policies.Service) *httptest.Server {
// 	logger := mflog.NewMock()
// 	mux := bone.New()
// 	api.MakeHandler(svc, mux, logger, instanceID)
// 	papi.MakeHandler(svc, psvc, mux, logger)
// 	return httptest.NewServer(mux)
// }

// func TestCreateThing(t *testing.T) {
// 	cRepo := new(mocks.Repository)
// 	gRepo := new(gmocks.Repository)
// 	uauth := cmocks.NewAuthService(users, map[string][]cmocks.SubjectSet{adminID: {uadminPolicy}})
// 	thingCache := mocks.NewCache()
// 	policiesCache := pmocks.NewCache()

// 	pRepo := new(pmocks.Repository)
// 	psvc := policies.NewService(uauth, pRepo, policiesCache, idProvider)

// 	svc := clients.NewService(uauth, psvc, cRepo, gRepo, thingCache, idProvider)
// 	ts := newThingsServer(svc, psvc)
// 	defer ts.Close()

// 	thing := sdk.Thing{
// 		Name:   "test",
// 		Status: mfclients.EnabledStatus.String(),
// 	}
// 	conf := sdk.Config{
// 		ThingsURL: ts.URL,
// 	}
// 	mfsdk := sdk.NewSDK(conf)

// 	cases := []struct {
// 		desc     string
// 		client   sdk.Thing
// 		response sdk.Thing
// 		token    string
// 		repoErr  error
// 		err      errors.SDKError
// 	}{
// 		{
// 			desc:     "register new thing",
// 			client:   thing,
// 			response: thing,
// 			token:    token,
// 			repoErr:  nil,
// 			err:      nil,
// 		},
// 		{
// 			desc:     "register existing thing",
// 			client:   thing,
// 			response: sdk.Thing{},
// 			token:    token,
// 			repoErr:  sdk.ErrFailedCreation,
// 			err:      errors.NewSDKErrorWithStatus(sdk.ErrFailedCreation, http.StatusInternalServerError),
// 		},
// 		{
// 			desc:     "register empty thing",
// 			client:   sdk.Thing{},
// 			response: sdk.Thing{},
// 			token:    token,
// 			repoErr:  errors.ErrMalformedEntity,
// 			err:      errors.NewSDKErrorWithStatus(errors.ErrMalformedEntity, http.StatusBadRequest),
// 		},
// 		{
// 			desc: "register a thing that can't be marshalled",
// 			client: sdk.Thing{
// 				Name: "test",
// 				Metadata: map[string]interface{}{
// 					"test": make(chan int),
// 				},
// 			},
// 			response: sdk.Thing{},
// 			token:    token,
// 			err:      errors.NewSDKError(fmt.Errorf("json: unsupported type: chan int")),
// 			repoErr:  errors.NewSDKError(fmt.Errorf("json: unsupported type: chan int")),
// 		},
// 		{
// 			desc: "register thing with empty secret",
// 			client: sdk.Thing{
// 				Name: "emptysecret",
// 				Credentials: sdk.Credentials{
// 					Secret: "",
// 				},
// 			},
// 			response: sdk.Thing{
// 				Name: "emptysecret",
// 				Credentials: sdk.Credentials{
// 					Secret: "",
// 				},
// 			},
// 			token:   token,
// 			err:     nil,
// 			repoErr: nil,
// 		},
// 		{
// 			desc: "register thing with empty identity",
// 			client: sdk.Thing{
// 				Credentials: sdk.Credentials{
// 					Identity: "",
// 					Secret:   secret,
// 				},
// 			},
// 			response: sdk.Thing{
// 				Credentials: sdk.Credentials{
// 					Identity: "",
// 					Secret:   secret,
// 				},
// 			},
// 			token:   token,
// 			repoErr: nil,
// 			err:     nil,
// 		},
// 		{
// 			desc: "register thing with every field defined",
// 			client: sdk.Thing{
// 				ID:          id,
// 				Name:        "name",
// 				Tags:        []string{"tag1", "tag2"},
// 				Owner:       id,
// 				Credentials: user.Credentials,
// 				Metadata:    validMetadata,
// 				CreatedAt:   time.Now(),
// 				UpdatedAt:   time.Now(),
// 				Status:      mfclients.EnabledStatus.String(),
// 			},
// 			response: sdk.Thing{
// 				ID:          id,
// 				Name:        "name",
// 				Tags:        []string{"tag1", "tag2"},
// 				Owner:       id,
// 				Credentials: user.Credentials,
// 				Metadata:    validMetadata,
// 				CreatedAt:   time.Now(),
// 				UpdatedAt:   time.Now(),
// 				Status:      mfclients.EnabledStatus.String(),
// 			},
// 			token:   token,
// 			repoErr: nil,
// 			err:     nil,
// 		},
// 	}
// 	for _, tc := range cases {
// 		repoCall := cRepo.On("Save", mock.Anything, mock.Anything).Return(tc.response, tc.repoErr)
// 		rThing, err := mfsdk.CreateThing(tc.client, tc.token)

// 		tc.response.ID = rThing.ID
// 		tc.response.Owner = rThing.Owner
// 		tc.response.CreatedAt = rThing.CreatedAt
// 		tc.response.UpdatedAt = rThing.UpdatedAt
// 		rThing.Credentials.Secret = tc.response.Credentials.Secret
// 		rThing.Status = tc.response.Status
// 		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
// 		assert.Equal(t, tc.response, rThing, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, rThing))
// 		if tc.err == nil {
// 			ok := repoCall.Parent.AssertCalled(t, "Save", mock.Anything, mock.Anything)
// 			assert.True(t, ok, fmt.Sprintf("Save was not called on %s", tc.desc))
// 		}
// 		repoCall.Unset()
// 	}
// }

// func TestCreateThings(t *testing.T) {
// 	cRepo := new(mocks.Repository)
// 	gRepo := new(gmocks.Repository)
// 	uauth := cmocks.NewAuthService(users, map[string][]cmocks.SubjectSet{adminID: {uadminPolicy}})
// 	thingCache := mocks.NewCache()
// 	policiesCache := pmocks.NewCache()

// 	pRepo := new(pmocks.Repository)
// 	psvc := policies.NewService(uauth, pRepo, policiesCache, idProvider)

// 	svc := clients.NewService(uauth, psvc, cRepo, gRepo, thingCache, idProvider)
// 	ts := newThingsServer(svc, psvc)
// 	defer ts.Close()

// 	things := []sdk.Thing{
// 		{
// 			Name:   "test",
// 			Status: mfclients.EnabledStatus.String(),
// 		},
// 		{
// 			Name:   "test2",
// 			Status: mfclients.EnabledStatus.String(),
// 		},
// 	}
// 	conf := sdk.Config{
// 		ThingsURL: ts.URL,
// 	}
// 	mfsdk := sdk.NewSDK(conf)

// 	cases := []struct {
// 		desc     string
// 		things   []sdk.Thing
// 		response []sdk.Thing
// 		token    string
// 		err      errors.SDKError
// 	}{
// 		{
// 			desc:     "register new things",
// 			things:   things,
// 			response: things,
// 			token:    token,
// 			err:      nil,
// 		},
// 		{
// 			desc:     "register existing things",
// 			things:   things,
// 			response: []sdk.Thing{},
// 			token:    token,
// 			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, sdk.ErrFailedCreation), http.StatusInternalServerError),
// 		},
// 		{
// 			desc:     "register empty things",
// 			things:   []sdk.Thing{},
// 			response: []sdk.Thing{},
// 			token:    token,
// 			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrEmptyList), http.StatusBadRequest),
// 		},
// 		{
// 			desc: "register things that can't be marshalled",
// 			things: []sdk.Thing{
// 				{
// 					Name: "test",
// 					Metadata: map[string]interface{}{
// 						"test": make(chan int),
// 					},
// 				},
// 			},
// 			response: []sdk.Thing{},
// 			token:    token,
// 			err:      errors.NewSDKError(fmt.Errorf("json: unsupported type: chan int")),
// 		},
// 	}
// 	for _, tc := range cases {
// 		repoCall := cRepo.On("Save", mock.Anything, mock.Anything).Return(tc.response, tc.err)
// 		rThing, err := mfsdk.CreateThings(tc.things, tc.token)
// 		for i, t := range rThing {
// 			tc.response[i].ID = t.ID
// 			tc.response[i].Owner = t.Owner
// 			tc.response[i].CreatedAt = t.CreatedAt
// 			tc.response[i].UpdatedAt = t.UpdatedAt
// 			tc.response[i].Credentials.Secret = t.Credentials.Secret
// 			t.Status = tc.response[i].Status
// 		}
// 		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
// 		assert.Equal(t, tc.response, rThing, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, rThing))
// 		if tc.err == nil {
// 			ok := repoCall.Parent.AssertCalled(t, "Save", mock.Anything, mock.Anything)
// 			assert.True(t, ok, fmt.Sprintf("Save was not called on %s", tc.desc))
// 		}
// 		repoCall.Unset()
// 	}
// }

// func TestListThings(t *testing.T) {
// 	cRepo := new(mocks.Repository)
// 	gRepo := new(gmocks.Repository)
// 	uauth := cmocks.NewAuthService(users, map[string][]cmocks.SubjectSet{adminID: {uadminPolicy}})
// 	thingCache := mocks.NewCache()
// 	policiesCache := pmocks.NewCache()

// 	pRepo := new(pmocks.Repository)
// 	psvc := policies.NewService(uauth, pRepo, policiesCache, idProvider)

// 	svc := clients.NewService(uauth, psvc, cRepo, gRepo, thingCache, idProvider)
// 	ts := newThingsServer(svc, psvc)
// 	defer ts.Close()

// 	var ths []sdk.Thing
// 	conf := sdk.Config{
// 		ThingsURL: ts.URL,
// 	}
// 	mfsdk := sdk.NewSDK(conf)

// 	owner := generateUUID(t)
// 	for i := 10; i < 100; i++ {
// 		th := sdk.Thing{
// 			ID:   generateUUID(t),
// 			Name: fmt.Sprintf("thing_%d", i),
// 			Credentials: sdk.Credentials{
// 				Identity: fmt.Sprintf("identity_%d", i),
// 				Secret:   generateUUID(t),
// 			},
// 			Metadata: sdk.Metadata{"name": fmt.Sprintf("thing_%d", i)},
// 			Status:   mfclients.EnabledStatus.String(),
// 		}
// 		if i == 50 {
// 			th.Owner = owner
// 			th.Status = mfclients.DisabledStatus.String()
// 			th.Tags = []string{"tag1", "tag2"}
// 		}
// 		ths = append(ths, th)
// 	}

// 	cases := []struct {
// 		desc       string
// 		token      string
// 		status     string
// 		total      uint64
// 		offset     uint64
// 		limit      uint64
// 		name       string
// 		identifier string
// 		ownerID    string
// 		tag        string
// 		metadata   sdk.Metadata
// 		err        errors.SDKError
// 		response   []sdk.Thing
// 	}{
// 		{
// 			desc:     "get a list of things",
// 			token:    token,
// 			limit:    limit,
// 			offset:   offset,
// 			total:    total,
// 			err:      nil,
// 			response: ths[offset:limit],
// 		},
// 		{
// 			desc:     "get a list of things with invalid token",
// 			token:    invalidToken,
// 			offset:   offset,
// 			limit:    limit,
// 			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, sdk.ErrFailedList), http.StatusInternalServerError),
// 			response: nil,
// 		},
// 		{
// 			desc:     "get a list of things with empty token",
// 			token:    "",
// 			offset:   offset,
// 			limit:    limit,
// 			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, sdk.ErrFailedList), http.StatusInternalServerError),
// 			response: nil,
// 		},
// 		{
// 			desc:     "get a list of things with zero limit",
// 			token:    token,
// 			offset:   offset,
// 			limit:    0,
// 			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrLimitSize), http.StatusInternalServerError),
// 			response: nil,
// 		},
// 		{
// 			desc:     "get a list of things with limit greater than max",
// 			token:    token,
// 			offset:   offset,
// 			limit:    110,
// 			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrLimitSize), http.StatusInternalServerError),
// 			response: []sdk.Thing(nil),
// 		},
// 		{
// 			desc:       "get a list of things with same identity",
// 			token:      token,
// 			offset:     0,
// 			limit:      1,
// 			err:        nil,
// 			identifier: Identity,
// 			metadata:   sdk.Metadata{},
// 			response:   []sdk.Thing{ths[89]},
// 		},
// 		{
// 			desc:       "get a list of things with same identity and metadata",
// 			token:      token,
// 			offset:     0,
// 			limit:      1,
// 			err:        nil,
// 			identifier: Identity,
// 			metadata: sdk.Metadata{
// 				"name": "client99",
// 			},
// 			response: []sdk.Thing{ths[89]},
// 		},
// 		{
// 			desc:   "list things with given metadata",
// 			token:  adminToken,
// 			offset: 0,
// 			limit:  1,
// 			metadata: sdk.Metadata{
// 				"name": "client99",
// 			},
// 			response: []sdk.Thing{ths[89]},
// 			err:      nil,
// 		},
// 		{
// 			desc:     "list things with given name",
// 			token:    adminToken,
// 			offset:   0,
// 			limit:    1,
// 			name:     "client10",
// 			response: []sdk.Thing{ths[0]},
// 			err:      nil,
// 		},
// 		{
// 			desc:     "list things with given owner",
// 			token:    adminToken,
// 			offset:   0,
// 			limit:    1,
// 			ownerID:  owner,
// 			response: []sdk.Thing{ths[50]},
// 			err:      nil,
// 		},
// 		{
// 			desc:     "list things with given status",
// 			token:    adminToken,
// 			offset:   0,
// 			limit:    1,
// 			status:   mfclients.DisabledStatus.String(),
// 			response: []sdk.Thing{ths[50]},
// 			err:      nil,
// 		},
// 		{
// 			desc:     "list things with given tag",
// 			token:    adminToken,
// 			offset:   0,
// 			limit:    1,
// 			tag:      "tag1",
// 			response: []sdk.Thing{ths[50]},
// 			err:      nil,
// 		},
// 	}

// 	for _, tc := range cases {
// 		pm := sdk.PageMetadata{
// 			Status:   tc.status,
// 			Total:    total,
// 			Offset:   tc.offset,
// 			Limit:    tc.limit,
// 			Name:     tc.name,
// 			OwnerID:  tc.ownerID,
// 			Metadata: tc.metadata,
// 			Tag:      tc.tag,
// 		}

// 		repoCall := cRepo.On("RetrieveAll", mock.Anything, mock.Anything).Return(mfclients.ClientsPage{Page: convertClientPage(pm), Clients: convertThings(tc.response)}, tc.err)
// 		page, err := mfsdk.Things(pm, adminToken)
// 		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
// 		assert.Equal(t, tc.response, page.Things, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))
// 		repoCall.Unset()
// 	}
// }

// func TestListThingsByChannel(t *testing.T) {
// 	cRepo := new(mocks.Repository)
// 	gRepo := new(gmocks.Repository)
// 	uauth := cmocks.NewAuthService(users, map[string][]cmocks.SubjectSet{adminID: {uadminPolicy}})
// 	thingCache := mocks.NewCache()
// 	policiesCache := pmocks.NewCache()

// 	pRepo := new(pmocks.Repository)
// 	psvc := policies.NewService(uauth, pRepo, policiesCache, idProvider)

// 	svc := clients.NewService(uauth, psvc, cRepo, gRepo, thingCache, idProvider)
// 	ts := newThingsServer(svc, psvc)
// 	defer ts.Close()

// 	conf := sdk.Config{
// 		ThingsURL: ts.URL,
// 	}
// 	mfsdk := sdk.NewSDK(conf)

// 	nThing := uint64(10)
// 	aThings := []sdk.Thing{}

// 	for i := uint64(1); i < nThing; i++ {
// 		thing := sdk.Thing{
// 			Name: fmt.Sprintf("member_%d@example.com", i),
// 			Credentials: sdk.Credentials{
// 				Secret: generateUUID(t),
// 			},
// 			Tags:     []string{"tag1", "tag2"},
// 			Metadata: sdk.Metadata{"role": "client"},
// 			Status:   mfclients.EnabledStatus.String(),
// 		}
// 		aThings = append(aThings, thing)
// 	}

// 	cases := []struct {
// 		desc      string
// 		token     string
// 		channelID string
// 		page      sdk.PageMetadata
// 		response  []sdk.Thing
// 		err       errors.SDKError
// 	}{
// 		{
// 			desc:      "list things with authorized token",
// 			token:     adminToken,
// 			channelID: testsutil.GenerateUUID(t, idProvider),
// 			page:      sdk.PageMetadata{},
// 			response:  aThings,
// 			err:       nil,
// 		},
// 		{
// 			desc:      "list things with offset and limit",
// 			token:     adminToken,
// 			channelID: testsutil.GenerateUUID(t, idProvider),
// 			page: sdk.PageMetadata{
// 				Offset: 4,
// 				Limit:  nThing,
// 			},
// 			response: aThings[4:],
// 			err:      nil,
// 		},
// 		{
// 			desc:      "list things with given name",
// 			token:     adminToken,
// 			channelID: testsutil.GenerateUUID(t, idProvider),
// 			page: sdk.PageMetadata{
// 				Name:   Identity,
// 				Offset: 6,
// 				Limit:  nThing,
// 			},
// 			response: aThings[6:],
// 			err:      nil,
// 		},

// 		{
// 			desc:      "list things with given ownerID",
// 			token:     adminToken,
// 			channelID: testsutil.GenerateUUID(t, idProvider),
// 			page: sdk.PageMetadata{
// 				OwnerID: user.Owner,
// 				Offset:  6,
// 				Limit:   nThing,
// 			},
// 			response: aThings[6:],
// 			err:      nil,
// 		},
// 		{
// 			desc:      "list things with given subject",
// 			token:     adminToken,
// 			channelID: testsutil.GenerateUUID(t, idProvider),
// 			page: sdk.PageMetadata{
// 				Subject: subject,
// 				Offset:  6,
// 				Limit:   nThing,
// 			},
// 			response: aThings[6:],
// 			err:      nil,
// 		},
// 		{
// 			desc:      "list things with given object",
// 			token:     adminToken,
// 			channelID: testsutil.GenerateUUID(t, idProvider),
// 			page: sdk.PageMetadata{
// 				Object: object,
// 				Offset: 6,
// 				Limit:  nThing,
// 			},
// 			response: aThings[6:],
// 			err:      nil,
// 		},
// 		{
// 			desc:      "list things with an invalid token",
// 			token:     invalidToken,
// 			channelID: testsutil.GenerateUUID(t, idProvider),
// 			page:      sdk.PageMetadata{},
// 			response:  []sdk.Thing(nil),
// 			err:       errors.NewSDKErrorWithStatus(errors.ErrAuthentication, http.StatusUnauthorized),
// 		},
// 		{
// 			desc:      "list things with an invalid id",
// 			token:     adminToken,
// 			channelID: mocks.WrongID,
// 			page:      sdk.PageMetadata{},
// 			response:  []sdk.Thing(nil),
// 			err:       errors.NewSDKErrorWithStatus(errors.ErrNotFound, http.StatusNotFound),
// 		},
// 	}

// 	for _, tc := range cases {
// 		repoCall := cRepo.On("Members", mock.Anything, tc.channelID, mock.Anything).Return(mfclients.MembersPage{Members: convertThings(tc.response)}, tc.err)
// 		membersPage, err := mfsdk.ThingsByChannel(tc.channelID, tc.page, tc.token)
// 		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
// 		assert.Equal(t, tc.response, membersPage.Things, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, membersPage.Things))
// 		if tc.err == nil {
// 			ok := repoCall.Parent.AssertCalled(t, "Members", mock.Anything, tc.channelID, mock.Anything)
// 			assert.True(t, ok, fmt.Sprintf("Members was not called on %s", tc.desc))
// 		}
// 		repoCall.Unset()
// 	}
// }

// func TestThing(t *testing.T) {
// 	cRepo := new(mocks.Repository)
// 	gRepo := new(gmocks.Repository)
// 	uauth := cmocks.NewAuthService(users, map[string][]cmocks.SubjectSet{adminID: {uadminPolicy}})
// 	thingCache := mocks.NewCache()
// 	policiesCache := pmocks.NewCache()

// 	pRepo := new(pmocks.Repository)
// 	psvc := policies.NewService(uauth, pRepo, policiesCache, idProvider)

// 	svc := clients.NewService(uauth, psvc, cRepo, gRepo, thingCache, idProvider)
// 	ts := newThingsServer(svc, psvc)
// 	defer ts.Close()

// 	thing := sdk.Thing{
// 		Name:        "thingname",
// 		Tags:        []string{"tag1", "tag2"},
// 		Credentials: sdk.Credentials{Identity: "clientidentity", Secret: generateUUID(t)},
// 		Metadata:    validMetadata,
// 		Status:      mfclients.EnabledStatus.String(),
// 	}
// 	conf := sdk.Config{
// 		ThingsURL: ts.URL,
// 	}
// 	mfsdk := sdk.NewSDK(conf)

// 	cases := []struct {
// 		desc     string
// 		token    string
// 		thingID  string
// 		response sdk.Thing
// 		err      errors.SDKError
// 	}{
// 		{
// 			desc:     "view thing successfully",
// 			response: thing,
// 			token:    adminToken,
// 			thingID:  generateUUID(t),
// 			err:      nil,
// 		},
// 		{
// 			desc:     "view thing with an invalid token",
// 			response: sdk.Thing{},
// 			token:    invalidToken,
// 			thingID:  generateUUID(t),
// 			err:      errors.NewSDKErrorWithStatus(errors.ErrAuthentication, http.StatusUnauthorized),
// 		},
// 		{
// 			desc:     "view thing with valid token and invalid thing id",
// 			response: sdk.Thing{},
// 			token:    adminToken,
// 			thingID:  mocks.WrongID,
// 			err:      errors.NewSDKErrorWithStatus(errors.ErrNotFound, http.StatusNotFound),
// 		},
// 		{
// 			desc:     "view thing with an invalid token and invalid thing id",
// 			response: sdk.Thing{},
// 			token:    invalidToken,
// 			thingID:  mocks.WrongID,
// 			err:      errors.NewSDKErrorWithStatus(errors.ErrAuthentication, http.StatusUnauthorized),
// 		},
// 	}

// 	for _, tc := range cases {
// 		repoCall := pRepo.On("EvaluateThingAccess", mock.Anything, mock.Anything).Return(policies.Policy{}, nil)
// 		repoCall1 := cRepo.On("RetrieveByID", mock.Anything, tc.thingID).Return(convertThing(tc.response), tc.err)
// 		rClient, err := mfsdk.Thing(tc.thingID, tc.token)
// 		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
// 		assert.Equal(t, tc.response, rClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, rClient))
// 		if tc.err == nil {
// 			ok := repoCall1.Parent.AssertCalled(t, "RetrieveByID", mock.Anything, tc.thingID)
// 			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
// 		}
// 		repoCall1.Unset()
// 		repoCall.Unset()
// 	}
// }

// func TestUpdateThing(t *testing.T) {
// 	cRepo := new(mocks.Repository)
// 	gRepo := new(gmocks.Repository)
// 	uauth := cmocks.NewAuthService(users, map[string][]cmocks.SubjectSet{adminID: {uadminPolicy}})
// 	thingCache := mocks.NewCache()
// 	policiesCache := pmocks.NewCache()

// 	pRepo := new(pmocks.Repository)
// 	psvc := policies.NewService(uauth, pRepo, policiesCache, idProvider)

// 	svc := clients.NewService(uauth, psvc, cRepo, gRepo, thingCache, idProvider)
// 	ts := newThingsServer(svc, psvc)
// 	defer ts.Close()

// 	conf := sdk.Config{
// 		ThingsURL: ts.URL,
// 	}
// 	mfsdk := sdk.NewSDK(conf)

// 	thing := sdk.Thing{
// 		ID:          generateUUID(t),
// 		Name:        "clientname",
// 		Credentials: sdk.Credentials{Secret: generateUUID(t)},
// 		Metadata:    validMetadata,
// 		Status:      mfclients.EnabledStatus.String(),
// 	}

// 	thing1 := thing
// 	thing1.Name = "Updated client"

// 	thing2 := thing
// 	thing2.Metadata = sdk.Metadata{"role": "test"}
// 	thing2.ID = invalidIdentity

// 	cases := []struct {
// 		desc     string
// 		thing    sdk.Thing
// 		response sdk.Thing
// 		token    string
// 		err      errors.SDKError
// 	}{
// 		{
// 			desc:     "update thing name with valid token",
// 			thing:    thing1,
// 			response: thing1,
// 			token:    adminToken,
// 			err:      nil,
// 		},
// 		{
// 			desc:     "update thing name with invalid token",
// 			thing:    thing1,
// 			response: sdk.Thing{},
// 			token:    invalidToken,
// 			err:      errors.NewSDKErrorWithStatus(errors.ErrAuthentication, http.StatusUnauthorized),
// 		},
// 		{
// 			desc:     "update thing name with invalid id",
// 			thing:    thing2,
// 			response: sdk.Thing{},
// 			token:    adminToken,
// 			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, sdk.ErrFailedUpdate), http.StatusInternalServerError),
// 		},
// 		{
// 			desc: "update thing that can't be marshalled",
// 			thing: sdk.Thing{
// 				Name: "test",
// 				Metadata: map[string]interface{}{
// 					"test": make(chan int),
// 				},
// 			},
// 			response: sdk.Thing{},
// 			token:    token,
// 			err:      errors.NewSDKError(fmt.Errorf("json: unsupported type: chan int")),
// 		},
// 	}

// 	for _, tc := range cases {
// 		repoCall := pRepo.On("EvaluateThingAccess", mock.Anything, mock.Anything).Return(policies.Policy{}, nil)
// 		repoCall1 := cRepo.On("Update", mock.Anything, mock.Anything).Return(convertThing(tc.response), tc.err)
// 		uClient, err := mfsdk.UpdateThing(tc.thing, tc.token)
// 		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
// 		assert.Equal(t, tc.response, uClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, uClient))
// 		if tc.err == nil {
// 			ok := repoCall.Parent.AssertCalled(t, "EvaluateThingAccess", mock.Anything, mock.Anything)
// 			assert.True(t, ok, fmt.Sprintf("EvaluateThingAccess was not called on %s", tc.desc))
// 			ok = repoCall1.Parent.AssertCalled(t, "Update", mock.Anything, mock.Anything)
// 			assert.True(t, ok, fmt.Sprintf("Update was not called on %s", tc.desc))
// 		}
// 		repoCall.Unset()
// 		repoCall1.Unset()
// 	}
// }

// func TestUpdateThingTags(t *testing.T) {
// 	cRepo := new(mocks.Repository)
// 	gRepo := new(gmocks.Repository)
// 	uauth := cmocks.NewAuthService(users, map[string][]cmocks.SubjectSet{adminID: {uadminPolicy}})
// 	thingCache := mocks.NewCache()
// 	policiesCache := pmocks.NewCache()

// 	pRepo := new(pmocks.Repository)
// 	psvc := policies.NewService(uauth, pRepo, policiesCache, idProvider)

// 	svc := clients.NewService(uauth, psvc, cRepo, gRepo, thingCache, idProvider)
// 	ts := newThingsServer(svc, psvc)
// 	defer ts.Close()

// 	conf := sdk.Config{
// 		ThingsURL: ts.URL,
// 	}
// 	mfsdk := sdk.NewSDK(conf)

// 	thing := sdk.Thing{
// 		ID:          generateUUID(t),
// 		Name:        "clientname",
// 		Tags:        []string{"tag1", "tag2"},
// 		Credentials: sdk.Credentials{Secret: generateUUID(t)},
// 		Status:      mfclients.EnabledStatus.String(),
// 	}

// 	thing1 := thing
// 	thing1.Tags = []string{"updatedTag1", "updatedTag2"}

// 	thing2 := thing
// 	thing2.ID = invalidIdentity

// 	cases := []struct {
// 		desc     string
// 		thing    sdk.Thing
// 		response sdk.Thing
// 		token    string
// 		err      error
// 	}{
// 		{
// 			desc:     "update thing name with valid token",
// 			thing:    thing,
// 			response: thing1,
// 			token:    adminToken,
// 			err:      nil,
// 		},
// 		{
// 			desc:     "update thing name with invalid token",
// 			thing:    thing1,
// 			response: sdk.Thing{},
// 			token:    invalidToken,
// 			err:      errors.NewSDKErrorWithStatus(errors.ErrAuthentication, http.StatusUnauthorized),
// 		},
// 		{
// 			desc:     "update thing name with invalid id",
// 			thing:    thing2,
// 			response: sdk.Thing{},
// 			token:    adminToken,
// 			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, sdk.ErrFailedUpdate), http.StatusInternalServerError),
// 		},
// 		{
// 			desc: "update thing that can't be marshalled",
// 			thing: sdk.Thing{
// 				Name: "test",
// 				Metadata: map[string]interface{}{
// 					"test": make(chan int),
// 				},
// 			},
// 			response: sdk.Thing{},
// 			token:    token,
// 			err:      errors.NewSDKError(fmt.Errorf("json: unsupported type: chan int")),
// 		},
// 	}

// 	for _, tc := range cases {
// 		repoCall := pRepo.On("EvaluateThingAccess", mock.Anything, mock.Anything).Return(policies.Policy{}, nil)
// 		repoCall1 := cRepo.On("UpdateTags", mock.Anything, mock.Anything).Return(convertThing(tc.response), tc.err)
// 		uClient, err := mfsdk.UpdateThingTags(tc.thing, tc.token)
// 		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
// 		assert.Equal(t, tc.response, uClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, uClient))
// 		if tc.err == nil {
// 			ok := repoCall.Parent.AssertCalled(t, "EvaluateThingAccess", mock.Anything, mock.Anything)
// 			assert.True(t, ok, fmt.Sprintf("EvaluateThingAccess was not called on %s", tc.desc))
// 			ok = repoCall1.Parent.AssertCalled(t, "UpdateTags", mock.Anything, mock.Anything)
// 			assert.True(t, ok, fmt.Sprintf("UpdateTags was not called on %s", tc.desc))
// 		}
// 		repoCall.Unset()
// 		repoCall1.Unset()
// 	}
// }

// func TestUpdateThingSecret(t *testing.T) {
// 	cRepo := new(mocks.Repository)
// 	gRepo := new(gmocks.Repository)
// 	uauth := cmocks.NewAuthService(users, map[string][]cmocks.SubjectSet{adminID: {uadminPolicy}})
// 	thingCache := mocks.NewCache()
// 	policiesCache := pmocks.NewCache()

// 	pRepo := new(pmocks.Repository)
// 	psvc := policies.NewService(uauth, pRepo, policiesCache, idProvider)

// 	svc := clients.NewService(uauth, psvc, cRepo, gRepo, thingCache, idProvider)
// 	ts := newThingsServer(svc, psvc)
// 	defer ts.Close()

// 	conf := sdk.Config{
// 		ThingsURL: ts.URL,
// 	}
// 	mfsdk := sdk.NewSDK(conf)

// 	user.ID = generateUUID(t)
// 	rthing := thing
// 	rthing.Credentials.Secret, _ = phasher.Hash(user.Credentials.Secret)

// 	cases := []struct {
// 		desc      string
// 		oldSecret string
// 		newSecret string
// 		token     string
// 		response  sdk.Thing
// 		repoErr   error
// 		err       error
// 	}{
// 		{
// 			desc:      "update thing secret with valid token",
// 			oldSecret: thing.Credentials.Secret,
// 			newSecret: "newSecret",
// 			token:     adminToken,
// 			response:  rthing,
// 			repoErr:   nil,
// 			err:       nil,
// 		},
// 		{
// 			desc:      "update thing secret with invalid token",
// 			oldSecret: thing.Credentials.Secret,
// 			newSecret: "newPassword",
// 			token:     "non-existent",
// 			response:  sdk.Thing{},
// 			repoErr:   errors.ErrAuthorization,
// 			err:       errors.NewSDKErrorWithStatus(errors.ErrAuthentication, http.StatusUnauthorized),
// 		},
// 		{
// 			desc:      "update thing secret with wrong old secret",
// 			oldSecret: "oldSecret",
// 			newSecret: "newSecret",
// 			token:     adminToken,
// 			response:  sdk.Thing{},
// 			repoErr:   apiutil.ErrInvalidSecret,
// 			err:       errors.NewSDKErrorWithStatus(apiutil.ErrInvalidSecret, http.StatusBadRequest),
// 		},
// 	}
// 	for _, tc := range cases {
// 		repoCall := pRepo.On("EvaluateThingAccess", mock.Anything, mock.Anything).Return(policies.Policy{}, nil)
// 		repoCall1 := cRepo.On("UpdateSecret", mock.Anything, mock.Anything).Return(convertThing(tc.response), tc.repoErr)
// 		uClient, err := mfsdk.UpdateThingSecret(tc.oldSecret, tc.newSecret, tc.token)
// 		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
// 		assert.Equal(t, tc.response, uClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, uClient))
// 		if tc.err == nil {
// 			ok := repoCall1.Parent.AssertCalled(t, "UpdateSecret", mock.Anything, mock.Anything)
// 			assert.True(t, ok, fmt.Sprintf("UpdateSecret was not called on %s", tc.desc))
// 		}
// 		repoCall.Unset()
// 		repoCall1.Unset()
// 	}
// }

// func TestUpdateThingOwner(t *testing.T) {
// 	cRepo := new(mocks.Repository)
// 	gRepo := new(gmocks.Repository)
// 	uauth := cmocks.NewAuthService(users, map[string][]cmocks.SubjectSet{adminID: {uadminPolicy}})
// 	thingCache := mocks.NewCache()
// 	policiesCache := pmocks.NewCache()

// 	pRepo := new(pmocks.Repository)
// 	psvc := policies.NewService(uauth, pRepo, policiesCache, idProvider)

// 	svc := clients.NewService(uauth, psvc, cRepo, gRepo, thingCache, idProvider)
// 	ts := newThingsServer(svc, psvc)
// 	defer ts.Close()

// 	conf := sdk.Config{
// 		ThingsURL: ts.URL,
// 	}
// 	mfsdk := sdk.NewSDK(conf)

// 	thing = sdk.Thing{
// 		ID:          generateUUID(t),
// 		Name:        "clientname",
// 		Tags:        []string{"tag1", "tag2"},
// 		Credentials: sdk.Credentials{Identity: "clientidentity", Secret: generateUUID(t)},
// 		Metadata:    validMetadata,
// 		Status:      mfclients.EnabledStatus.String(),
// 		Owner:       "owner",
// 	}

// 	thing2 := thing
// 	thing2.ID = invalidIdentity

// 	cases := []struct {
// 		desc     string
// 		thing    sdk.Thing
// 		response sdk.Thing
// 		token    string
// 		err      errors.SDKError
// 	}{
// 		{
// 			desc:     "update thing name with valid token",
// 			thing:    thing,
// 			response: thing,
// 			token:    adminToken,
// 			err:      nil,
// 		},
// 		{
// 			desc:     "update thing name with invalid token",
// 			thing:    thing2,
// 			response: sdk.Thing{},
// 			token:    invalidToken,
// 			err:      errors.NewSDKErrorWithStatus(errors.ErrAuthentication, http.StatusUnauthorized),
// 		},
// 		{
// 			desc:     "update thing name with invalid id",
// 			thing:    thing2,
// 			response: sdk.Thing{},
// 			token:    adminToken,
// 			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, sdk.ErrFailedUpdate), http.StatusInternalServerError),
// 		},
// 		{
// 			desc: "update thing that can't be marshalled",
// 			thing: sdk.Thing{
// 				Name: "test",
// 				Metadata: map[string]interface{}{
// 					"test": make(chan int),
// 				},
// 			},
// 			response: sdk.Thing{},
// 			token:    token,
// 			err:      errors.NewSDKError(fmt.Errorf("json: unsupported type: chan int")),
// 		},
// 	}

// 	for _, tc := range cases {
// 		repoCall := pRepo.On("EvaluateThingAccess", mock.Anything, mock.Anything).Return(policies.Policy{}, nil)
// 		repoCall1 := cRepo.On("UpdateOwner", mock.Anything, mock.Anything).Return(convertThing(tc.response), tc.err)
// 		uClient, err := mfsdk.UpdateThingOwner(tc.thing, tc.token)
// 		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
// 		assert.Equal(t, tc.response, uClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, uClient))
// 		if tc.err == nil {
// 			ok := repoCall.Parent.AssertCalled(t, "EvaluateThingAccess", mock.Anything, mock.Anything)
// 			assert.True(t, ok, fmt.Sprintf("EvaluateThingAccess was not called on %s", tc.desc))
// 			ok = repoCall1.Parent.AssertCalled(t, "UpdateOwner", mock.Anything, mock.Anything)
// 			assert.True(t, ok, fmt.Sprintf("UpdateOwner was not called on %s", tc.desc))
// 		}
// 		repoCall.Unset()
// 		repoCall1.Unset()
// 	}
// }

// func TestEnableThing(t *testing.T) {
// 	cRepo := new(mocks.Repository)
// 	gRepo := new(gmocks.Repository)
// 	uauth := cmocks.NewAuthService(users, map[string][]cmocks.SubjectSet{adminID: {uadminPolicy}})
// 	thingCache := mocks.NewCache()
// 	policiesCache := pmocks.NewCache()

// 	pRepo := new(pmocks.Repository)
// 	psvc := policies.NewService(uauth, pRepo, policiesCache, idProvider)

// 	svc := clients.NewService(uauth, psvc, cRepo, gRepo, thingCache, idProvider)
// 	ts := newThingsServer(svc, psvc)
// 	defer ts.Close()

// 	conf := sdk.Config{
// 		ThingsURL: ts.URL,
// 	}
// 	mfsdk := sdk.NewSDK(conf)

// 	enabledThing1 := sdk.Thing{ID: testsutil.GenerateUUID(t, idProvider), Credentials: sdk.Credentials{Identity: "client1@example.com", Secret: generateUUID(t)}, Status: mfclients.EnabledStatus.String()}
// 	disabledThing1 := sdk.Thing{ID: testsutil.GenerateUUID(t, idProvider), Credentials: sdk.Credentials{Identity: "client3@example.com", Secret: generateUUID(t)}, Status: mfclients.DisabledStatus.String()}
// 	endisabledThing1 := disabledThing1
// 	endisabledThing1.Status = mfclients.EnabledStatus.String()
// 	endisabledThing1.ID = testsutil.GenerateUUID(t, idProvider)

// 	cases := []struct {
// 		desc     string
// 		id       string
// 		token    string
// 		thing    sdk.Thing
// 		response sdk.Thing
// 		repoErr  error
// 		err      errors.SDKError
// 	}{
// 		{
// 			desc:     "enable disabled thing",
// 			id:       disabledThing1.ID,
// 			token:    adminToken,
// 			thing:    disabledThing1,
// 			response: endisabledThing1,
// 			repoErr:  nil,
// 			err:      nil,
// 		},
// 		{
// 			desc:     "enable enabled thing",
// 			id:       enabledThing1.ID,
// 			token:    adminToken,
// 			thing:    enabledThing1,
// 			response: sdk.Thing{},
// 			repoErr:  sdk.ErrFailedEnable,
// 			err:      errors.NewSDKErrorWithStatus(errors.Wrap(sdk.ErrFailedEnable, sdk.ErrFailedEnable), http.StatusInternalServerError),
// 		},
// 		{
// 			desc:     "enable non-existing thing",
// 			id:       mocks.WrongID,
// 			token:    adminToken,
// 			thing:    sdk.Thing{},
// 			response: sdk.Thing{},
// 			repoErr:  sdk.ErrFailedEnable,
// 			err:      errors.NewSDKErrorWithStatus(errors.Wrap(sdk.ErrFailedEnable, errors.ErrNotFound), http.StatusNotFound),
// 		},
// 	}

// 	for _, tc := range cases {
// 		repoCall := pRepo.On("EvaluateThingAccess", mock.Anything, mock.Anything).Return(policies.Policy{}, nil)
// 		repoCall1 := cRepo.On("RetrieveByID", mock.Anything, tc.id).Return(convertThing(tc.thing), tc.repoErr)
// 		repoCall2 := cRepo.On("ChangeStatus", mock.Anything, mock.Anything).Return(convertThing(tc.response), tc.repoErr)
// 		eClient, err := mfsdk.EnableThing(tc.id, tc.token)
// 		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
// 		assert.Equal(t, tc.response, eClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, eClient))
// 		if tc.err == nil {
// 			ok := repoCall.Parent.AssertCalled(t, "EvaluateThingAccess", mock.Anything, mock.Anything)
// 			assert.True(t, ok, fmt.Sprintf("EvaluateThingAccess was not called on %s", tc.desc))
// 			ok = repoCall1.Parent.AssertCalled(t, "RetrieveByID", mock.Anything, tc.id)
// 			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
// 			ok = repoCall2.Parent.AssertCalled(t, "ChangeStatus", mock.Anything, mock.Anything)
// 			assert.True(t, ok, fmt.Sprintf("ChangeStatus was not called on %s", tc.desc))
// 		}
// 		repoCall.Unset()
// 		repoCall1.Unset()
// 		repoCall2.Unset()
// 	}

// 	cases2 := []struct {
// 		desc     string
// 		token    string
// 		status   string
// 		metadata sdk.Metadata
// 		response sdk.ThingsPage
// 		size     uint64
// 	}{
// 		{
// 			desc:   "list enabled clients",
// 			status: mfclients.EnabledStatus.String(),
// 			size:   2,
// 			response: sdk.ThingsPage{
// 				Things: []sdk.Thing{enabledThing1, endisabledThing1},
// 			},
// 		},
// 		{
// 			desc:   "list disabled clients",
// 			status: mfclients.DisabledStatus.String(),
// 			size:   1,
// 			response: sdk.ThingsPage{
// 				Things: []sdk.Thing{disabledThing1},
// 			},
// 		},
// 		{
// 			desc:   "list enabled and disabled clients",
// 			status: mfclients.AllStatus.String(),
// 			size:   3,
// 			response: sdk.ThingsPage{
// 				Things: []sdk.Thing{enabledThing1, disabledThing1, endisabledThing1},
// 			},
// 		},
// 	}

// 	for _, tc := range cases2 {
// 		pm := sdk.PageMetadata{
// 			Total:  100,
// 			Offset: 0,
// 			Limit:  100,
// 			Status: tc.status,
// 		}
// 		repoCall := pRepo.On("CheckAdmin", mock.Anything, mock.Anything).Return(nil)
// 		repoCall1 := cRepo.On("RetrieveAll", mock.Anything, mock.Anything).Return(convertThingsPage(tc.response), nil)
// 		clientsPage, err := mfsdk.Things(pm, adminToken)
// 		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
// 		size := uint64(len(clientsPage.Things))
// 		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", tc.desc, tc.size, size))
// 		repoCall.Unset()
// 		repoCall1.Unset()
// 	}
// }

// func TestDisableThing(t *testing.T) {
// 	cRepo := new(mocks.Repository)
// 	gRepo := new(gmocks.Repository)
// 	uauth := cmocks.NewAuthService(users, map[string][]cmocks.SubjectSet{adminID: {uadminPolicy}})
// 	thingCache := mocks.NewCache()
// 	policiesCache := pmocks.NewCache()

// 	pRepo := new(pmocks.Repository)
// 	psvc := policies.NewService(uauth, pRepo, policiesCache, idProvider)

// 	svc := clients.NewService(uauth, psvc, cRepo, gRepo, thingCache, idProvider)
// 	ts := newThingsServer(svc, psvc)
// 	defer ts.Close()

// 	conf := sdk.Config{
// 		ThingsURL: ts.URL,
// 	}
// 	mfsdk := sdk.NewSDK(conf)

// 	enabledThing1 := sdk.Thing{ID: testsutil.GenerateUUID(t, idProvider), Credentials: sdk.Credentials{Identity: "client1@example.com", Secret: generateUUID(t)}, Status: mfclients.EnabledStatus.String()}
// 	disabledThing1 := sdk.Thing{ID: testsutil.GenerateUUID(t, idProvider), Credentials: sdk.Credentials{Identity: "client3@example.com", Secret: generateUUID(t)}, Status: mfclients.DisabledStatus.String()}
// 	disenabledThing1 := enabledThing1
// 	disenabledThing1.Status = mfclients.DisabledStatus.String()
// 	disenabledThing1.ID = testsutil.GenerateUUID(t, idProvider)

// 	cases := []struct {
// 		desc     string
// 		id       string
// 		token    string
// 		thing    sdk.Thing
// 		response sdk.Thing
// 		repoErr  error
// 		err      errors.SDKError
// 	}{
// 		{
// 			desc:     "disable enabled thing",
// 			id:       enabledThing1.ID,
// 			token:    adminToken,
// 			thing:    enabledThing1,
// 			response: disenabledThing1,
// 			repoErr:  nil,
// 			err:      nil,
// 		},
// 		{
// 			desc:     "disable disabled thing",
// 			id:       disabledThing1.ID,
// 			token:    adminToken,
// 			thing:    disabledThing1,
// 			response: sdk.Thing{},
// 			repoErr:  sdk.ErrFailedDisable,
// 			err:      errors.NewSDKErrorWithStatus(errors.Wrap(sdk.ErrFailedDisable, sdk.ErrFailedDisable), http.StatusInternalServerError),
// 		},
// 		{
// 			desc:     "disable non-existing thing",
// 			id:       mocks.WrongID,
// 			thing:    sdk.Thing{},
// 			token:    adminToken,
// 			response: sdk.Thing{},
// 			repoErr:  sdk.ErrFailedDisable,
// 			err:      errors.NewSDKErrorWithStatus(errors.Wrap(sdk.ErrFailedDisable, errors.ErrNotFound), http.StatusNotFound),
// 		},
// 	}

// 	for _, tc := range cases {
// 		repoCall := pRepo.On("EvaluateThingAccess", mock.Anything, mock.Anything).Return(policies.Policy{}, nil)
// 		repoCall1 := cRepo.On("RetrieveByID", mock.Anything, tc.id).Return(convertThing(tc.thing), tc.repoErr)
// 		repoCall2 := cRepo.On("ChangeStatus", mock.Anything, mock.Anything).Return(convertThing(tc.response), tc.repoErr)
// 		dThing, err := mfsdk.DisableThing(tc.id, tc.token)
// 		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
// 		assert.Equal(t, tc.response, dThing, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, dThing))
// 		if tc.err == nil {
// 			ok := repoCall.Parent.AssertCalled(t, "EvaluateThingAccess", mock.Anything, mock.Anything)
// 			assert.True(t, ok, fmt.Sprintf("EvaluateThingAccess was not called on %s", tc.desc))
// 			ok = repoCall1.Parent.AssertCalled(t, "RetrieveByID", mock.Anything, tc.id)
// 			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
// 			ok = repoCall2.Parent.AssertCalled(t, "ChangeStatus", mock.Anything, mock.Anything)
// 			assert.True(t, ok, fmt.Sprintf("ChangeStatus was not called on %s", tc.desc))
// 		}
// 		repoCall.Unset()
// 		repoCall1.Unset()
// 		repoCall2.Unset()
// 	}

// 	cases2 := []struct {
// 		desc     string
// 		token    string
// 		status   string
// 		metadata sdk.Metadata
// 		response sdk.ThingsPage
// 		size     uint64
// 	}{
// 		{
// 			desc:   "list enabled things",
// 			status: mfclients.EnabledStatus.String(),
// 			size:   2,
// 			response: sdk.ThingsPage{
// 				Things: []sdk.Thing{enabledThing1, disenabledThing1},
// 			},
// 		},
// 		{
// 			desc:   "list disabled things",
// 			status: mfclients.DisabledStatus.String(),
// 			size:   1,
// 			response: sdk.ThingsPage{
// 				Things: []sdk.Thing{disabledThing1},
// 			},
// 		},
// 		{
// 			desc:   "list enabled and disabled things",
// 			status: mfclients.AllStatus.String(),
// 			size:   3,
// 			response: sdk.ThingsPage{
// 				Things: []sdk.Thing{enabledThing1, disabledThing1, disenabledThing1},
// 			},
// 		},
// 	}

// 	for _, tc := range cases2 {
// 		pm := sdk.PageMetadata{
// 			Total:  100,
// 			Offset: 0,
// 			Limit:  100,
// 			Status: tc.status,
// 		}
// 		repoCall := pRepo.On("CheckAdmin", mock.Anything, mock.Anything).Return(nil)
// 		repoCall1 := cRepo.On("RetrieveAll", mock.Anything, mock.Anything).Return(convertThingsPage(tc.response), nil)
// 		page, err := mfsdk.Things(pm, adminToken)
// 		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
// 		size := uint64(len(page.Things))
// 		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", tc.desc, tc.size, size))
// 		repoCall.Unset()
// 		repoCall1.Unset()
// 	}
// }

// func TestIdentify(t *testing.T) {
// 	cRepo := new(mocks.Repository)
// 	gRepo := new(gmocks.Repository)
// 	uauth := cmocks.NewAuthService(users, map[string][]cmocks.SubjectSet{adminID: {uadminPolicy}})
// 	thingCache := mocks.NewCache()
// 	policiesCache := pmocks.NewCache()

// 	pRepo := new(pmocks.Repository)
// 	psvc := policies.NewService(uauth, pRepo, policiesCache, idProvider)

// 	svc := clients.NewService(uauth, psvc, cRepo, gRepo, thingCache, idProvider)
// 	ts := newThingsServer(svc, psvc)
// 	defer ts.Close()

// 	conf := sdk.Config{
// 		ThingsURL: ts.URL,
// 	}
// 	mfsdk := sdk.NewSDK(conf)

// 	thing = sdk.Thing{
// 		ID:          generateUUID(t),
// 		Name:        "clientname",
// 		Credentials: sdk.Credentials{Identity: "clientidentity", Secret: generateUUID(t)},
// 		Status:      mfclients.EnabledStatus.String(),
// 	}

// 	cases := []struct {
// 		desc     string
// 		secret   string
// 		response string
// 		repoErr  error
// 		err      errors.SDKError
// 	}{
// 		{
// 			desc:     "identify thing successfully",
// 			response: thing.ID,
// 			secret:   thing.Credentials.Secret,
// 			repoErr:  nil,
// 			err:      nil,
// 		},
// 		{
// 			desc:     "identify thing with an invalid token",
// 			response: "",
// 			secret:   invalidToken,
// 			repoErr:  errors.ErrAuthentication,
// 			err:      errors.NewSDKErrorWithStatus(errors.ErrAuthentication, http.StatusUnauthorized),
// 		},
// 	}

// 	for _, tc := range cases {
// 		repoCall := cRepo.On("RetrieveBySecret", mock.Anything, mock.Anything).Return(convertThing(thing), tc.repoErr)
// 		id, err := mfsdk.IdentifyThing(tc.secret)
// 		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
// 		assert.Equal(t, tc.response, id, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, id))
// 		if tc.err == nil {
// 			ok := repoCall.Parent.AssertCalled(t, "RetrieveBySecret", mock.Anything, mock.Anything)
// 			assert.True(t, ok, fmt.Sprintf("RetrieveBySecret was not called on %s", tc.desc))
// 		}
// 		repoCall.Unset()
// 	}
// }

// func TestShareThing(t *testing.T) {
// 	thingID := generateUUID(t)
// 	cRepo := new(mocks.Repository)
// 	gRepo := new(gmocks.Repository)
// 	aPolicy := cmocks.SubjectSet{Subject: "things", Relation: []string{"g_add", "c_share"}}
// 	uPolicy := cmocks.SubjectSet{Subject: thingID, Relation: []string{"g_add"}}

// 	uauth := cmocks.NewAuthService(users, map[string][]cmocks.SubjectSet{adminID: {aPolicy}, userID: {uPolicy}})
// 	thingCache := mocks.NewCache()
// 	policiesCache := pmocks.NewCache()

// 	pRepo := new(pmocks.Repository)
// 	psvc := policies.NewService(uauth, pRepo, policiesCache, idProvider)

// 	svc := clients.NewService(uauth, psvc, cRepo, gRepo, thingCache, idProvider)
// 	ts := newThingsServer(svc, psvc)
// 	defer ts.Close()

// 	conf := sdk.Config{
// 		ThingsURL: ts.URL,
// 	}
// 	mfsdk := sdk.NewSDK(conf)

// 	cases := []struct {
// 		desc      string
// 		channelID string
// 		thingID   string
// 		token     string
// 		err       errors.SDKError
// 		repoErr   error
// 	}{
// 		{
// 			desc:      "share thing with valid token",
// 			channelID: generateUUID(t),
// 			thingID:   thingID,
// 			token:     adminToken,
// 			err:       nil,
// 		},
// 		{
// 			desc:      "share thing with invalid token",
// 			channelID: generateUUID(t),
// 			thingID:   thingID,
// 			token:     invalidToken,
// 			err:       errors.NewSDKErrorWithStatus(errors.Wrap(errors.ErrAuthorization, errors.ErrAuthentication), http.StatusUnauthorized),
// 		},
// 		{
// 			desc:      "share thing with valid token for unauthorized user",
// 			channelID: generateUUID(t),
// 			thingID:   thingID,
// 			token:     userToken,
// 			err:       errors.NewSDKErrorWithStatus(errors.ErrAuthorization, http.StatusForbidden),
// 			repoErr:   errors.ErrAuthorization,
// 		},
// 	}

// 	for _, tc := range cases {
// 		repoCall := pRepo.On("EvaluateGroupAccess", mock.Anything, mock.Anything).Return(policies.Policy{}, tc.repoErr)
// 		repoCall1 := pRepo.On("EvaluateThingAccess", mock.Anything, mock.Anything).Return(policies.Policy{}, tc.repoErr)
// 		repoCall2 := pRepo.On("Retrieve", mock.Anything, mock.Anything).Return(policies.PolicyPage{}, nil)
// 		repoCall3 := pRepo.On("Save", mock.Anything, mock.Anything).Return(policies.Policy{}, nil)
// 		err := mfsdk.ShareThing(tc.channelID, tc.thingID, []string{"c_list", "c_delete"}, tc.token)
// 		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
// 		if tc.err == nil {
// 			ok := repoCall3.Parent.AssertCalled(t, "Save", mock.Anything, mock.Anything)
// 			assert.True(t, ok, fmt.Sprintf("Save was not called on %s", tc.desc))
// 		}
// 		repoCall.Unset()
// 		repoCall1.Unset()
// 		repoCall2.Unset()
// 		repoCall3.Unset()
// 	}
// }
