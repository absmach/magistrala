package sdk_test

// import (
// 	"fmt"
// 	"net/http"
// 	"net/http/httptest"
// 	"testing"
// 	"time"

// 	"github.com/go-zoo/bone"
// 	"github.com/mainflux/mainflux/internal/apiutil"
// 	gmocks "github.com/mainflux/mainflux/internal/groups/mocks"
// 	"github.com/mainflux/mainflux/internal/testsutil"
// 	"github.com/mainflux/mainflux/logger"
// 	mfclients "github.com/mainflux/mainflux/pkg/clients"
// 	"github.com/mainflux/mainflux/pkg/errors"
// 	"github.com/mainflux/mainflux/pkg/groups"
// 	mfgroups "github.com/mainflux/mainflux/pkg/groups"
// 	sdk "github.com/mainflux/mainflux/pkg/sdk/go"
// 	"github.com/mainflux/mainflux/users/clients"
// 	cmocks "github.com/mainflux/mainflux/users/clients/mocks"
// 	"github.com/mainflux/mainflux/users/groups/api"
// 	"github.com/mainflux/mainflux/users/jwt"
// 	pmocks "github.com/mainflux/mainflux/users/policies/mocks"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/mock"
// )

// func newGroupsServer(svc groups.Service) *httptest.Server {
// 	logger := logger.NewMock()
// 	mux := bone.New()
// 	api.MakeHandler(svc, mux, logger)

// 	return httptest.NewServer(mux)
// }

// func TestCreateGroup(t *testing.T) {
// 	cRepo := new(cmocks.Repository)
// 	gRepo := new(gmocks.Repository)
// 	pRepo := new(pmocks.Repository)
// 	tokenizer := jwt.NewRepository([]byte(secret), accessDuration, refreshDuration)

// 	csvc := clients.NewService(cRepo, pRepo, tokenizer, emailer, phasher, idProvider, passRegex)
// 	svc := groups.NewService(gRepo, pRepo, tokenizer, idProvider)
// 	ts := newGroupsServer(svc)
// 	defer ts.Close()
// 	group := sdk.Group{
// 		Name:     "groupName",
// 		Metadata: validMetadata,
// 		Status:   mfclients.EnabledStatus.String(),
// 	}

// 	conf := sdk.Config{
// 		UsersURL: ts.URL,
// 	}
// 	mfsdk := sdk.NewSDK(conf)
// 	cases := []struct {
// 		desc  string
// 		group sdk.Group
// 		token string
// 		err   errors.SDKError
// 	}{
// 		{
// 			desc:  "create group successfully",
// 			group: group,
// 			token: token,
// 			err:   nil,
// 		},
// 		{
// 			desc:  "create group with existing name",
// 			group: group,
// 			err:   nil,
// 		},
// 		{
// 			desc: "create group with parent",
// 			group: sdk.Group{
// 				Name:     gName,
// 				ParentID: testsutil.GenerateUUID(t, idProvider),
// 				Status:   mfclients.EnabledStatus.String(),
// 			},
// 			err: nil,
// 		},
// 		{
// 			desc: "create group with invalid parent",
// 			group: sdk.Group{
// 				Name:     gName,
// 				ParentID: gmocks.WrongID,
// 				Status:   mfclients.EnabledStatus.String(),
// 			},
// 			err: errors.NewSDKErrorWithStatus(errors.ErrCreateEntity, http.StatusInternalServerError),
// 		},
// 		{
// 			desc: "create group with invalid owner",
// 			group: sdk.Group{
// 				Name:    gName,
// 				OwnerID: gmocks.WrongID,
// 				Status:  mfclients.EnabledStatus.String(),
// 			},
// 			err: errors.NewSDKErrorWithStatus(sdk.ErrFailedCreation, http.StatusInternalServerError),
// 		},
// 		{
// 			desc: "create group with missing name",
// 			group: sdk.Group{
// 				Status: mfclients.EnabledStatus.String(),
// 			},
// 			err: errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrNameSize), http.StatusBadRequest),
// 		},
// 		{
// 			desc: "create a group with every field defined",
// 			group: sdk.Group{
// 				ID:          generateUUID(t),
// 				OwnerID:     "owner",
// 				ParentID:    "parent",
// 				Name:        "name",
// 				Description: description,
// 				Metadata:    validMetadata,
// 				Level:       1,
// 				Children:    []*sdk.Group{&group},
// 				CreatedAt:   time.Now(),
// 				UpdatedAt:   time.Now(),
// 				Status:      mfclients.EnabledStatus.String(),
// 			},
// 			token: token,
// 			err:   nil,
// 		},
// 		{
// 			desc: "create a group that can't be marshalled",
// 			group: sdk.Group{
// 				Name: "test",
// 				Metadata: map[string]interface{}{
// 					"test": make(chan int),
// 				},
// 			},
// 			token: token,
// 			err:   errors.NewSDKError(fmt.Errorf("json: unsupported type: chan int")),
// 		},
// 	}
// 	for _, tc := range cases {
// 		repoCall := gRepo.On("Save", mock.Anything, mock.Anything).Return(convertGroup(sdk.Group{}), tc.err)
// 		rGroup, err := mfsdk.CreateGroup(tc.group, generateValidToken(t, csvc, cRepo))
// 		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
// 		if err == nil {
// 			assert.NotEmpty(t, rGroup, fmt.Sprintf("%s: expected not nil on client ID", tc.desc))
// 			ok := repoCall.Parent.AssertCalled(t, "Save", mock.Anything, mock.Anything)
// 			assert.True(t, ok, fmt.Sprintf("Save was not called on %s", tc.desc))
// 		}
// 		repoCall.Unset()
// 	}
// }

// func TestListGroups(t *testing.T) {
// 	cRepo := new(cmocks.Repository)
// 	gRepo := new(gmocks.Repository)
// 	pRepo := new(pmocks.Repository)
// 	tokenizer := jwt.NewRepository([]byte(secret), accessDuration, refreshDuration)

// 	csvc := clients.NewService(cRepo, pRepo, tokenizer, emailer, phasher, idProvider, passRegex)
// 	svc := groups.NewService(gRepo, pRepo, tokenizer, idProvider)
// 	ts := newGroupsServer(svc)
// 	defer ts.Close()

// 	var grps []sdk.Group
// 	conf := sdk.Config{
// 		UsersURL: ts.URL,
// 	}
// 	mfsdk := sdk.NewSDK(conf)

// 	for i := 10; i < 100; i++ {
// 		gr := sdk.Group{
// 			ID:       generateUUID(t),
// 			Name:     fmt.Sprintf("group_%d", i),
// 			Metadata: sdk.Metadata{"name": fmt.Sprintf("user_%d", i)},
// 			Status:   mfclients.EnabledStatus.String(),
// 		}
// 		grps = append(grps, gr)
// 	}

// 	cases := []struct {
// 		desc     string
// 		token    string
// 		status   mfclients.Status
// 		total    uint64
// 		offset   uint64
// 		limit    uint64
// 		level    int
// 		name     string
// 		ownerID  string
// 		metadata sdk.Metadata
// 		err      errors.SDKError
// 		response []sdk.Group
// 	}{
// 		{
// 			desc:     "get a list of groups",
// 			token:    token,
// 			limit:    limit,
// 			offset:   offset,
// 			total:    total,
// 			err:      nil,
// 			response: grps[offset:limit],
// 		},
// 		{
// 			desc:     "get a list of groups with invalid token",
// 			token:    invalidToken,
// 			offset:   offset,
// 			limit:    limit,
// 			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, sdk.ErrFailedList), http.StatusInternalServerError),
// 			response: nil,
// 		},
// 		{
// 			desc:     "get a list of groups with empty token",
// 			token:    "",
// 			offset:   offset,
// 			limit:    limit,
// 			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, sdk.ErrFailedList), http.StatusInternalServerError),
// 			response: nil,
// 		},
// 		{
// 			desc:     "get a list of groups with zero limit",
// 			token:    token,
// 			offset:   offset,
// 			limit:    0,
// 			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, sdk.ErrFailedList), http.StatusInternalServerError),
// 			response: nil,
// 		},
// 		{
// 			desc:     "get a list of groups with limit greater than max",
// 			token:    token,
// 			offset:   offset,
// 			limit:    110,
// 			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, sdk.ErrFailedList), http.StatusInternalServerError),
// 			response: []sdk.Group(nil),
// 		},
// 		{
// 			desc:     "get a list of groups with given name",
// 			token:    token,
// 			offset:   0,
// 			limit:    1,
// 			err:      nil,
// 			metadata: sdk.Metadata{},
// 			response: []sdk.Group{grps[89]},
// 		},
// 		{
// 			desc:     "get a list of groups with level",
// 			token:    token,
// 			offset:   0,
// 			limit:    1,
// 			level:    1,
// 			err:      nil,
// 			response: []sdk.Group{grps[0]},
// 		},
// 		{
// 			desc:     "get a list of groups with metadata",
// 			token:    token,
// 			offset:   0,
// 			limit:    1,
// 			err:      nil,
// 			metadata: sdk.Metadata{},
// 			response: []sdk.Group{grps[89]},
// 		},
// 	}

// 	for _, tc := range cases {
// 		repoCall := pRepo.On("CheckAdmin", mock.Anything, mock.Anything).Return(nil)
// 		repoCall1 := gRepo.On("RetrieveAll", mock.Anything, mock.Anything).Return(mfgroups.Page{Groups: convertGroups(tc.response)}, tc.err)
// 		pm := sdk.PageMetadata{}
// 		page, err := mfsdk.Groups(pm, generateValidToken(t, csvc, cRepo))
// 		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
// 		assert.Equal(t, len(tc.response), len(page.Groups), fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))
// 		if tc.err == nil {
// 			ok := repoCall1.Parent.AssertCalled(t, "RetrieveAll", mock.Anything, mock.Anything)
// 			assert.True(t, ok, fmt.Sprintf("RetrieveAll was not called on %s", tc.desc))
// 		}
// 		repoCall.Unset()
// 		repoCall1.Unset()
// 	}
// }

// func TestListParentGroups(t *testing.T) {
// 	cRepo := new(cmocks.Repository)
// 	gRepo := new(gmocks.Repository)
// 	pRepo := new(pmocks.Repository)
// 	tokenizer := jwt.NewRepository([]byte(secret), accessDuration, refreshDuration)

// 	csvc := clients.NewService(cRepo, pRepo, tokenizer, emailer, phasher, idProvider, passRegex)
// 	svc := groups.NewService(gRepo, pRepo, tokenizer, idProvider)
// 	ts := newGroupsServer(svc)
// 	defer ts.Close()

// 	var grps []sdk.Group
// 	conf := sdk.Config{
// 		UsersURL: ts.URL,
// 	}
// 	mfsdk := sdk.NewSDK(conf)

// 	parentID := ""
// 	for i := 10; i < 100; i++ {
// 		gr := sdk.Group{
// 			ID:       generateUUID(t),
// 			Name:     fmt.Sprintf("group_%d", i),
// 			Metadata: sdk.Metadata{"name": fmt.Sprintf("user_%d", i)},
// 			Status:   mfclients.EnabledStatus.String(),
// 			ParentID: parentID,
// 		}
// 		parentID = gr.ID
// 		grps = append(grps, gr)
// 	}

// 	cases := []struct {
// 		desc     string
// 		token    string
// 		status   mfclients.Status
// 		total    uint64
// 		offset   uint64
// 		limit    uint64
// 		level    int
// 		name     string
// 		ownerID  string
// 		metadata sdk.Metadata
// 		err      errors.SDKError
// 		response []sdk.Group
// 	}{
// 		{
// 			desc:     "get a list of groups",
// 			token:    token,
// 			limit:    limit,
// 			offset:   offset,
// 			total:    total,
// 			err:      nil,
// 			response: grps[offset:limit],
// 		},
// 		{
// 			desc:     "get a list of groups with invalid token",
// 			token:    invalidToken,
// 			offset:   offset,
// 			limit:    limit,
// 			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, sdk.ErrFailedList), http.StatusInternalServerError),
// 			response: nil,
// 		},
// 		{
// 			desc:     "get a list of groups with empty token",
// 			token:    "",
// 			offset:   offset,
// 			limit:    limit,
// 			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, sdk.ErrFailedList), http.StatusInternalServerError),
// 			response: nil,
// 		},
// 		{
// 			desc:     "get a list of groups with zero limit",
// 			token:    token,
// 			offset:   offset,
// 			limit:    0,
// 			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, sdk.ErrFailedList), http.StatusInternalServerError),
// 			response: nil,
// 		},
// 		{
// 			desc:     "get a list of groups with limit greater than max",
// 			token:    token,
// 			offset:   offset,
// 			limit:    110,
// 			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, sdk.ErrFailedList), http.StatusInternalServerError),
// 			response: []sdk.Group(nil),
// 		},
// 		{
// 			desc:     "get a list of groups with given name",
// 			token:    token,
// 			offset:   0,
// 			limit:    1,
// 			err:      nil,
// 			metadata: sdk.Metadata{},
// 			response: []sdk.Group{grps[89]},
// 		},
// 		{
// 			desc:     "get a list of groups with level",
// 			token:    token,
// 			offset:   0,
// 			limit:    1,
// 			level:    1,
// 			err:      nil,
// 			response: []sdk.Group{grps[0]},
// 		},
// 		{
// 			desc:     "get a list of groups with metadata",
// 			token:    token,
// 			offset:   0,
// 			limit:    1,
// 			err:      nil,
// 			metadata: sdk.Metadata{},
// 			response: []sdk.Group{grps[89]},
// 		},
// 	}

// 	for _, tc := range cases {
// 		repoCall := pRepo.On("CheckAdmin", mock.Anything, mock.Anything).Return(nil)
// 		repoCall1 := gRepo.On("RetrieveAll", mock.Anything, mock.Anything).Return(mfgroups.Page{Groups: convertGroups(tc.response)}, tc.err)
// 		pm := sdk.PageMetadata{}
// 		page, err := mfsdk.Parents(parentID, pm, generateValidToken(t, csvc, cRepo))
// 		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
// 		assert.Equal(t, len(tc.response), len(page.Groups), fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))
// 		if tc.err == nil {
// 			ok := repoCall1.Parent.AssertCalled(t, "RetrieveAll", mock.Anything, mock.Anything)
// 			assert.True(t, ok, fmt.Sprintf("RetrieveAll was not called on %s", tc.desc))
// 		}
// 		repoCall.Unset()
// 		repoCall1.Unset()
// 	}
// }

// func TestListChildrenGroups(t *testing.T) {
// 	cRepo := new(cmocks.Repository)
// 	gRepo := new(gmocks.Repository)
// 	pRepo := new(pmocks.Repository)
// 	tokenizer := jwt.NewRepository([]byte(secret), accessDuration, refreshDuration)

// 	csvc := clients.NewService(cRepo, pRepo, tokenizer, emailer, phasher, idProvider, passRegex)
// 	svc := groups.NewService(gRepo, pRepo, tokenizer, idProvider)
// 	ts := newGroupsServer(svc)
// 	defer ts.Close()

// 	var grps []sdk.Group
// 	conf := sdk.Config{
// 		UsersURL: ts.URL,
// 	}
// 	mfsdk := sdk.NewSDK(conf)

// 	parentID := ""
// 	for i := 10; i < 100; i++ {
// 		gr := sdk.Group{
// 			ID:       generateUUID(t),
// 			Name:     fmt.Sprintf("group_%d", i),
// 			Metadata: sdk.Metadata{"name": fmt.Sprintf("user_%d", i)},
// 			Status:   mfclients.EnabledStatus.String(),
// 			ParentID: parentID,
// 		}
// 		parentID = gr.ID
// 		grps = append(grps, gr)
// 	}
// 	childID := grps[0].ID

// 	cases := []struct {
// 		desc     string
// 		token    string
// 		status   mfclients.Status
// 		total    uint64
// 		offset   uint64
// 		limit    uint64
// 		level    int
// 		name     string
// 		ownerID  string
// 		metadata sdk.Metadata
// 		err      errors.SDKError
// 		response []sdk.Group
// 	}{
// 		{
// 			desc:     "get a list of groups",
// 			token:    token,
// 			limit:    limit,
// 			offset:   offset,
// 			total:    total,
// 			err:      nil,
// 			response: grps[offset:limit],
// 		},
// 		{
// 			desc:     "get a list of groups with invalid token",
// 			token:    invalidToken,
// 			offset:   offset,
// 			limit:    limit,
// 			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, sdk.ErrFailedList), http.StatusInternalServerError),
// 			response: nil,
// 		},
// 		{
// 			desc:     "get a list of groups with empty token",
// 			token:    "",
// 			offset:   offset,
// 			limit:    limit,
// 			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, sdk.ErrFailedList), http.StatusInternalServerError),
// 			response: nil,
// 		},
// 		{
// 			desc:     "get a list of groups with zero limit",
// 			token:    token,
// 			offset:   offset,
// 			limit:    0,
// 			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, sdk.ErrFailedList), http.StatusInternalServerError),
// 			response: nil,
// 		},
// 		{
// 			desc:     "get a list of groups with limit greater than max",
// 			token:    token,
// 			offset:   offset,
// 			limit:    110,
// 			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, sdk.ErrFailedList), http.StatusInternalServerError),
// 			response: []sdk.Group(nil),
// 		},
// 		{
// 			desc:     "get a list of groups with given name",
// 			token:    token,
// 			offset:   0,
// 			limit:    1,
// 			err:      nil,
// 			metadata: sdk.Metadata{},
// 			response: []sdk.Group{grps[89]},
// 		},
// 		{
// 			desc:     "get a list of groups with level",
// 			token:    token,
// 			offset:   0,
// 			limit:    1,
// 			level:    1,
// 			err:      nil,
// 			response: []sdk.Group{grps[0]},
// 		},
// 		{
// 			desc:     "get a list of groups with metadata",
// 			token:    token,
// 			offset:   0,
// 			limit:    1,
// 			err:      nil,
// 			metadata: sdk.Metadata{},
// 			response: []sdk.Group{grps[89]},
// 		},
// 	}

// 	for _, tc := range cases {
// 		repoCall := pRepo.On("CheckAdmin", mock.Anything, mock.Anything).Return(nil)
// 		repoCall1 := gRepo.On("RetrieveAll", mock.Anything, mock.Anything).Return(mfgroups.Page{Groups: convertGroups(tc.response)}, tc.err)
// 		pm := sdk.PageMetadata{}
// 		page, err := mfsdk.Children(childID, pm, generateValidToken(t, csvc, cRepo))
// 		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
// 		assert.Equal(t, len(tc.response), len(page.Groups), fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))
// 		if tc.err == nil {
// 			ok := repoCall1.Parent.AssertCalled(t, "RetrieveAll", mock.Anything, mock.Anything)
// 			assert.True(t, ok, fmt.Sprintf("RetrieveAll was not called on %s", tc.desc))
// 		}
// 		repoCall.Unset()
// 		repoCall1.Unset()
// 	}
// }

// func TestViewGroup(t *testing.T) {
// 	cRepo := new(cmocks.Repository)
// 	gRepo := new(gmocks.Repository)
// 	pRepo := new(pmocks.Repository)
// 	tokenizer := jwt.NewRepository([]byte(secret), accessDuration, refreshDuration)

// 	csvc := clients.NewService(cRepo, pRepo, tokenizer, emailer, phasher, idProvider, passRegex)
// 	svc := groups.NewService(gRepo, pRepo, tokenizer, idProvider)
// 	ts := newGroupsServer(svc)
// 	defer ts.Close()

// 	group := sdk.Group{
// 		Name:        "groupName",
// 		Description: description,
// 		Metadata:    validMetadata,
// 		Children:    []*sdk.Group{},
// 		Status:      mfclients.EnabledStatus.String(),
// 	}

// 	conf := sdk.Config{
// 		UsersURL: ts.URL,
// 	}
// 	mfsdk := sdk.NewSDK(conf)
// 	group.ID = generateUUID(t)

// 	cases := []struct {
// 		desc     string
// 		token    string
// 		groupID  string
// 		response sdk.Group
// 		err      errors.SDKError
// 	}{
// 		{
// 			desc:     "view group",
// 			token:    generateValidToken(t, csvc, cRepo),
// 			groupID:  group.ID,
// 			response: group,
// 			err:      nil,
// 		},
// 		{
// 			desc:     "view group with invalid token",
// 			token:    "wrongtoken",
// 			groupID:  group.ID,
// 			response: sdk.Group{Children: []*sdk.Group{}},
// 			err:      errors.NewSDKErrorWithStatus(errors.Wrap(errors.ErrAuthentication, sdk.ErrInvalidJWT), http.StatusUnauthorized),
// 		},
// 		{
// 			desc:     "view group for wrong id",
// 			token:    generateValidToken(t, csvc, cRepo),
// 			groupID:  gmocks.WrongID,
// 			response: sdk.Group{Children: []*sdk.Group{}},
// 			err:      errors.NewSDKErrorWithStatus(errors.ErrNotFound, http.StatusNotFound),
// 		},
// 	}

// 	for _, tc := range cases {
// 		repoCall := pRepo.On("CheckAdmin", mock.Anything, mock.Anything).Return(nil)
// 		repoCall1 := gRepo.On("RetrieveByID", mock.Anything, tc.groupID).Return(convertGroup(tc.response), tc.err)
// 		grp, err := mfsdk.Group(tc.groupID, tc.token)
// 		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
// 		if len(tc.response.Children) == 0 {
// 			tc.response.Children = nil
// 		}
// 		if len(grp.Children) == 0 {
// 			grp.Children = nil
// 		}
// 		assert.Equal(t, tc.response, grp, fmt.Sprintf("%s: expected metadata %v got %v\n", tc.desc, tc.response, grp))
// 		if tc.err == nil {
// 			ok := repoCall.Parent.AssertCalled(t, "CheckAdmin", mock.Anything, mock.Anything)
// 			assert.True(t, ok, fmt.Sprintf("CheckAdmin was not called on %s", tc.desc))
// 			ok = repoCall1.Parent.AssertCalled(t, "RetrieveByID", mock.Anything, tc.groupID)
// 			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
// 		}
// 		repoCall.Unset()
// 		repoCall1.Unset()
// 	}
// }

// func TestUpdateGroup(t *testing.T) {
// 	cRepo := new(cmocks.Repository)
// 	gRepo := new(gmocks.Repository)
// 	pRepo := new(pmocks.Repository)
// 	tokenizer := jwt.NewRepository([]byte(secret), accessDuration, refreshDuration)

// 	csvc := clients.NewService(cRepo, pRepo, tokenizer, emailer, phasher, idProvider, passRegex)
// 	svc := groups.NewService(gRepo, pRepo, tokenizer, idProvider)
// 	ts := newGroupsServer(svc)
// 	defer ts.Close()

// 	group := sdk.Group{
// 		ID:          generateUUID(t),
// 		Name:        "groupName",
// 		Description: description,
// 		Metadata:    validMetadata,
// 	}

// 	conf := sdk.Config{
// 		UsersURL: ts.URL,
// 	}
// 	mfsdk := sdk.NewSDK(conf)

// 	group.ID = generateUUID(t)

// 	cases := []struct {
// 		desc     string
// 		token    string
// 		group    sdk.Group
// 		response sdk.Group
// 		err      errors.SDKError
// 	}{
// 		{
// 			desc: "update group name",
// 			group: sdk.Group{
// 				ID:   group.ID,
// 				Name: "NewName",
// 			},
// 			response: sdk.Group{
// 				ID:   group.ID,
// 				Name: "NewName",
// 			},
// 			token: generateValidToken(t, csvc, cRepo),
// 			err:   nil,
// 		},
// 		{
// 			desc: "update group description",
// 			group: sdk.Group{
// 				ID:          group.ID,
// 				Description: "NewDescription",
// 			},
// 			response: sdk.Group{
// 				ID:          group.ID,
// 				Description: "NewDescription",
// 			},
// 			token: generateValidToken(t, csvc, cRepo),
// 			err:   nil,
// 		},
// 		{
// 			desc: "update group metadata",
// 			group: sdk.Group{
// 				ID: group.ID,
// 				Metadata: sdk.Metadata{
// 					"field": "value2",
// 				},
// 			},
// 			response: sdk.Group{
// 				ID: group.ID,
// 				Metadata: sdk.Metadata{
// 					"field": "value2",
// 				},
// 			},
// 			token: generateValidToken(t, csvc, cRepo),
// 			err:   nil,
// 		},
// 		{
// 			desc: "update group name with invalid group id",
// 			group: sdk.Group{
// 				ID:   gmocks.WrongID,
// 				Name: "NewName",
// 			},
// 			response: sdk.Group{},
// 			token:    generateValidToken(t, csvc, cRepo),
// 			err:      errors.NewSDKErrorWithStatus(errors.ErrNotFound, http.StatusNotFound),
// 		},
// 		{
// 			desc: "update group description with invalid group id",
// 			group: sdk.Group{
// 				ID:          gmocks.WrongID,
// 				Description: "NewDescription",
// 			},
// 			response: sdk.Group{},
// 			token:    generateValidToken(t, csvc, cRepo),
// 			err:      errors.NewSDKErrorWithStatus(errors.ErrNotFound, http.StatusNotFound),
// 		},
// 		{
// 			desc: "update group metadata with invalid group id",
// 			group: sdk.Group{
// 				ID: gmocks.WrongID,
// 				Metadata: sdk.Metadata{
// 					"field": "value2",
// 				},
// 			},
// 			response: sdk.Group{},
// 			token:    generateValidToken(t, csvc, cRepo),
// 			err:      errors.NewSDKErrorWithStatus(errors.ErrNotFound, http.StatusNotFound),
// 		},
// 		{
// 			desc: "update group name with invalid token",
// 			group: sdk.Group{
// 				ID:   group.ID,
// 				Name: "NewName",
// 			},
// 			response: sdk.Group{},
// 			token:    invalidToken,
// 			err:      errors.NewSDKErrorWithStatus(errors.Wrap(errors.ErrAuthentication, sdk.ErrInvalidJWT), http.StatusUnauthorized),
// 		},
// 		{
// 			desc: "update group description with invalid token",
// 			group: sdk.Group{
// 				ID:          group.ID,
// 				Description: "NewDescription",
// 			},
// 			response: sdk.Group{},
// 			token:    invalidToken,
// 			err:      errors.NewSDKErrorWithStatus(errors.Wrap(errors.ErrAuthentication, sdk.ErrInvalidJWT), http.StatusUnauthorized),
// 		},
// 		{
// 			desc: "update group metadata with invalid token",
// 			group: sdk.Group{
// 				ID: group.ID,
// 				Metadata: sdk.Metadata{
// 					"field": "value2",
// 				},
// 			},
// 			response: sdk.Group{},
// 			token:    invalidToken,
// 			err:      errors.NewSDKErrorWithStatus(errors.Wrap(errors.ErrAuthentication, sdk.ErrInvalidJWT), http.StatusUnauthorized),
// 		},
// 		{
// 			desc: "update a group that can't be marshalled",
// 			group: sdk.Group{
// 				Name: "test",
// 				Metadata: map[string]interface{}{
// 					"test": make(chan int),
// 				},
// 			},
// 			response: sdk.Group{},
// 			token:    token,
// 			err:      errors.NewSDKError(fmt.Errorf("json: unsupported type: chan int")),
// 		},
// 	}

// 	for _, tc := range cases {
// 		repoCall := pRepo.On("CheckAdmin", mock.Anything, mock.Anything).Return(nil)
// 		repoCall1 := gRepo.On("Update", mock.Anything, mock.Anything).Return(convertGroup(tc.response), tc.err)
// 		_, err := mfsdk.UpdateGroup(tc.group, tc.token)
// 		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
// 		if tc.err == nil {
// 			ok := repoCall.Parent.AssertCalled(t, "CheckAdmin", mock.Anything, mock.Anything)
// 			assert.True(t, ok, fmt.Sprintf("CheckAdmin was not called on %s", tc.desc))
// 			ok = repoCall1.Parent.AssertCalled(t, "Update", mock.Anything, mock.Anything)
// 			assert.True(t, ok, fmt.Sprintf("Update was not called on %s", tc.desc))
// 		}
// 		repoCall.Unset()
// 		repoCall1.Unset()
// 	}
// }

// func TestListMemberships(t *testing.T) {
// 	cRepo := new(cmocks.Repository)
// 	gRepo := new(gmocks.Repository)
// 	pRepo := new(pmocks.Repository)
// 	tokenizer := jwt.NewRepository([]byte(secret), accessDuration, refreshDuration)

// 	csvc := clients.NewService(cRepo, pRepo, tokenizer, emailer, phasher, idProvider, passRegex)
// 	svc := groups.NewService(gRepo, pRepo, tokenizer, idProvider)
// 	ts := newGroupsServer(svc)
// 	defer ts.Close()

// 	conf := sdk.Config{
// 		UsersURL: ts.URL,
// 	}
// 	mfsdk := sdk.NewSDK(conf)

// 	nGroups := uint64(100)
// 	aGroups := []sdk.Group{}

// 	for i := uint64(1); i < nGroups; i++ {
// 		group := sdk.Group{
// 			Name:     fmt.Sprintf("membership_%d@example.com", i),
// 			Metadata: sdk.Metadata{"role": "group"},
// 			Status:   mfclients.EnabledStatus.String(),
// 		}
// 		aGroups = append(aGroups, group)
// 	}

// 	cases := []struct {
// 		desc     string
// 		token    string
// 		clientID string
// 		page     sdk.PageMetadata
// 		response []sdk.Group
// 		err      errors.SDKError
// 	}{
// 		{
// 			desc:     "list clients with authorized token",
// 			token:    generateValidToken(t, csvc, cRepo),
// 			clientID: testsutil.GenerateUUID(t, idProvider),
// 			page:     sdk.PageMetadata{},
// 			response: aGroups,
// 			err:      nil,
// 		},
// 		{
// 			desc:     "list clients with offset and limit",
// 			token:    generateValidToken(t, csvc, cRepo),
// 			clientID: testsutil.GenerateUUID(t, idProvider),
// 			page: sdk.PageMetadata{
// 				Offset: 6,
// 				Total:  nGroups,
// 				Limit:  nGroups,
// 				Status: mfclients.AllStatus.String(),
// 			},
// 			response: aGroups[6 : nGroups-1],
// 			err:      nil,
// 		},
// 		{
// 			desc:     "list clients with given name",
// 			token:    generateValidToken(t, csvc, cRepo),
// 			clientID: testsutil.GenerateUUID(t, idProvider),
// 			page: sdk.PageMetadata{
// 				Name:   gName,
// 				Offset: 6,
// 				Total:  nGroups,
// 				Limit:  nGroups,
// 				Status: mfclients.AllStatus.String(),
// 			},
// 			response: aGroups[6 : nGroups-1],
// 			err:      nil,
// 		},
// 		{
// 			desc:     "list clients with given level",
// 			token:    generateValidToken(t, csvc, cRepo),
// 			clientID: testsutil.GenerateUUID(t, idProvider),
// 			page: sdk.PageMetadata{
// 				Level:  1,
// 				Offset: 6,
// 				Total:  nGroups,
// 				Limit:  nGroups,
// 				Status: mfclients.AllStatus.String(),
// 			},
// 			response: aGroups[6 : nGroups-1],
// 			err:      nil,
// 		},
// 		{
// 			desc:     "list clients with metadata",
// 			token:    generateValidToken(t, csvc, cRepo),
// 			clientID: testsutil.GenerateUUID(t, idProvider),
// 			page: sdk.PageMetadata{
// 				Metadata: validMetadata,
// 				Offset:   6,
// 				Total:    nGroups,
// 				Limit:    nGroups,
// 				Status:   mfclients.AllStatus.String(),
// 			},
// 			response: aGroups[6 : nGroups-1],
// 			err:      nil,
// 		},
// 		{
// 			desc:     "list clients with an invalid token",
// 			token:    invalidToken,
// 			clientID: testsutil.GenerateUUID(t, idProvider),
// 			page:     sdk.PageMetadata{},
// 			response: []sdk.Group(nil),
// 			err:      errors.NewSDKErrorWithStatus(errors.Wrap(errors.ErrAuthentication, sdk.ErrInvalidJWT), http.StatusUnauthorized),
// 		},
// 		{
// 			desc:     "list clients with an invalid id",
// 			token:    generateValidToken(t, csvc, cRepo),
// 			clientID: gmocks.WrongID,
// 			page:     sdk.PageMetadata{},
// 			response: []sdk.Group(nil),
// 			err:      errors.NewSDKErrorWithStatus(errors.ErrNotFound, http.StatusNotFound),
// 		},
// 	}

// 	for _, tc := range cases {
// 		repoCall := pRepo.On("CheckAdmin", mock.Anything, mock.Anything).Return(nil)
// 		repoCall1 := gRepo.On("Memberships", mock.Anything, tc.clientID, mock.Anything).Return(convertMembershipsPage(sdk.MembershipsPage{Memberships: tc.response}), tc.err)
// 		page, err := mfsdk.Memberships(tc.clientID, tc.page, tc.token)
// 		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
// 		assert.Equal(t, tc.response, page.Memberships, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page.Memberships))
// 		if tc.err == nil {
// 			ok := repoCall.Parent.AssertCalled(t, "CheckAdmin", mock.Anything, mock.Anything)
// 			assert.True(t, ok, fmt.Sprintf("CheckAdmin was not called on %s", tc.desc))
// 			ok = repoCall1.Parent.AssertCalled(t, "Memberships", mock.Anything, tc.clientID, mock.Anything)
// 			assert.True(t, ok, fmt.Sprintf("Memberships was not called on %s", tc.desc))
// 		}
// 		repoCall.Unset()
// 		repoCall1.Unset()
// 	}
// }

// func TestEnableGroup(t *testing.T) {
// 	cRepo := new(cmocks.Repository)
// 	gRepo := new(gmocks.Repository)
// 	pRepo := new(pmocks.Repository)
// 	tokenizer := jwt.NewRepository([]byte(secret), accessDuration, refreshDuration)

// 	csvc := clients.NewService(cRepo, pRepo, tokenizer, emailer, phasher, idProvider, passRegex)
// 	svc := groups.NewService(gRepo, pRepo, tokenizer, idProvider)
// 	ts := newGroupsServer(svc)
// 	defer ts.Close()

// 	conf := sdk.Config{
// 		UsersURL: ts.URL,
// 	}
// 	mfsdk := sdk.NewSDK(conf)

// 	creationTime := time.Now().UTC()
// 	group := sdk.Group{
// 		ID:        generateUUID(t),
// 		Name:      gName,
// 		OwnerID:   generateUUID(t),
// 		CreatedAt: creationTime,
// 		UpdatedAt: creationTime,
// 		Status:    mfclients.Disabled,
// 	}

// 	repoCall := pRepo.On("CheckAdmin", mock.Anything, mock.Anything).Return(nil)
// 	repoCall1 := gRepo.On("RetrieveByID", mock.Anything, mock.Anything).Return(nil)
// 	repoCall2 := gRepo.On("ChangeStatus", mock.Anything, mock.Anything).Return(sdk.ErrFailedRemoval)
// 	_, err := mfsdk.EnableGroup("wrongID", generateValidToken(t, csvc, cRepo))
// 	assert.Equal(t, err, errors.NewSDKErrorWithStatus(errors.ErrNotFound, http.StatusNotFound), fmt.Sprintf("Enable group with wrong id: expected %v got %v", errors.ErrNotFound, err))
// 	ok := repoCall.Parent.AssertCalled(t, "CheckAdmin", mock.Anything, mock.Anything)
// 	assert.True(t, ok, "CheckAdmin was not called on enabling group")
// 	ok = repoCall1.Parent.AssertCalled(t, "RetrieveByID", mock.Anything, "wrongID")
// 	assert.True(t, ok, "RetrieveByID was not called on enabling group")
// 	repoCall.Unset()
// 	repoCall1.Unset()
// 	repoCall2.Unset()

// 	g := mfgroups.Group{
// 		ID:        group.ID,
// 		Name:      group.Name,
// 		Owner:     group.OwnerID,
// 		CreatedAt: creationTime,
// 		UpdatedAt: creationTime,
// 		Status:    mfclients.DisabledStatus,
// 	}

// 	repoCall = pRepo.On("CheckAdmin", mock.Anything, mock.Anything).Return(nil)
// 	repoCall1 = gRepo.On("RetrieveByID", mock.Anything, mock.Anything).Return(g, nil)
// 	repoCall2 = gRepo.On("ChangeStatus", mock.Anything, mock.Anything).Return(g, nil)
// 	res, err := mfsdk.EnableGroup(group.ID, generateValidToken(t, csvc, cRepo))
// 	assert.Nil(t, err, fmt.Sprintf("Enable group with correct id: expected %v got %v", nil, err))
// 	assert.Equal(t, group, res, fmt.Sprintf("Enable group with correct id: expected %v got %v", group, res))
// 	ok = repoCall.Parent.AssertCalled(t, "CheckAdmin", mock.Anything, mock.Anything)
// 	assert.True(t, ok, "CheckAdmin was not called on enabling group")
// 	ok = repoCall1.Parent.AssertCalled(t, "RetrieveByID", mock.Anything, group.ID)
// 	assert.True(t, ok, "RetrieveByID was not called on enabling group")
// 	ok = repoCall2.Parent.AssertCalled(t, "ChangeStatus", mock.Anything, mock.Anything)
// 	assert.True(t, ok, "ChangeStatus was not called on enabling group")
// 	repoCall.Unset()
// 	repoCall1.Unset()
// 	repoCall2.Unset()
// }

// func TestDisableGroup(t *testing.T) {
// 	cRepo := new(cmocks.Repository)
// 	gRepo := new(gmocks.Repository)
// 	pRepo := new(pmocks.Repository)
// 	tokenizer := jwt.NewRepository([]byte(secret), accessDuration, refreshDuration)

// 	csvc := clients.NewService(cRepo, pRepo, tokenizer, emailer, phasher, idProvider, passRegex)
// 	svc := groups.NewService(gRepo, pRepo, tokenizer, idProvider)
// 	ts := newGroupsServer(svc)
// 	defer ts.Close()

// 	conf := sdk.Config{
// 		UsersURL: ts.URL,
// 	}
// 	mfsdk := sdk.NewSDK(conf)

// 	creationTime := time.Now().UTC()
// 	group := sdk.Group{
// 		ID:        generateUUID(t),
// 		Name:      gName,
// 		OwnerID:   generateUUID(t),
// 		CreatedAt: creationTime,
// 		UpdatedAt: creationTime,
// 		Status:    mfclients.Enabled,
// 	}

// 	repoCall := pRepo.On("CheckAdmin", mock.Anything, mock.Anything).Return(nil)
// 	repoCall1 := gRepo.On("ChangeStatus", mock.Anything, mock.Anything).Return(sdk.ErrFailedRemoval)
// 	repoCall2 := gRepo.On("RetrieveByID", mock.Anything, mock.Anything).Return(nil)
// 	_, err := mfsdk.DisableGroup("wrongID", generateValidToken(t, csvc, cRepo))
// 	assert.Equal(t, err, errors.NewSDKErrorWithStatus(errors.ErrNotFound, http.StatusNotFound), fmt.Sprintf("Disable group with wrong id: expected %v got %v", errors.ErrNotFound, err))
// 	ok := repoCall.Parent.AssertCalled(t, "CheckAdmin", mock.Anything, mock.Anything)
// 	assert.True(t, ok, "CheckAdmin was not called on disabling group with wrong id")
// 	ok = repoCall1.Parent.AssertCalled(t, "RetrieveByID", mock.Anything, "wrongID")
// 	assert.True(t, ok, "Memberships was not called on disabling group with wrong id")
// 	repoCall.Unset()
// 	repoCall1.Unset()
// 	repoCall2.Unset()

// 	g := mfgroups.Group{
// 		ID:        group.ID,
// 		Name:      group.Name,
// 		Owner:     group.OwnerID,
// 		CreatedAt: creationTime,
// 		UpdatedAt: creationTime,
// 		Status:    mfclients.EnabledStatus,
// 	}

// 	repoCall = pRepo.On("CheckAdmin", mock.Anything, mock.Anything).Return(nil)
// 	repoCall1 = gRepo.On("ChangeStatus", mock.Anything, mock.Anything).Return(g, nil)
// 	repoCall2 = gRepo.On("RetrieveByID", mock.Anything, mock.Anything).Return(g, nil)
// 	res, err := mfsdk.DisableGroup(group.ID, generateValidToken(t, csvc, cRepo))
// 	assert.Nil(t, err, fmt.Sprintf("Disable group with correct id: expected %v got %v", nil, err))
// 	assert.Equal(t, group, res, fmt.Sprintf("Disable group with correct id: expected %v got %v", group, res))
// 	ok = repoCall.Parent.AssertCalled(t, "CheckAdmin", mock.Anything, mock.Anything)
// 	assert.True(t, ok, "CheckAdmin was not called on disabling group with correct id")
// 	ok = repoCall1.Parent.AssertCalled(t, "RetrieveByID", mock.Anything, group.ID)
// 	assert.True(t, ok, "RetrieveByID was not called on disabling group with correct id")
// 	ok = repoCall2.Parent.AssertCalled(t, "ChangeStatus", mock.Anything, mock.Anything)
// 	assert.True(t, ok, "ChangeStatus was not called on disabling group with correct id")
// 	repoCall.Unset()
// 	repoCall1.Unset()
// 	repoCall2.Unset()
// }
