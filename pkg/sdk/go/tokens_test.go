package sdk_test

// import (
// 	"context"
// 	"fmt"
// 	"net/http"
// 	"testing"

// 	"github.com/mainflux/mainflux/internal/apiutil"
// 	"github.com/mainflux/mainflux/pkg/errors"
// 	sdk "github.com/mainflux/mainflux/pkg/sdk/go"
// 	"github.com/mainflux/mainflux/users/clients"
// 	"github.com/mainflux/mainflux/users/clients/mocks"
// 	"github.com/mainflux/mainflux/users/jwt"
// 	pmocks "github.com/mainflux/mainflux/users/policies/mocks"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/mock"
// )

// func TestIssueToken(t *testing.T) {
// 	cRepo := new(mocks.Repository)
// 	pRepo := new(pmocks.Repository)
// 	tokenizer := jwt.NewRepository([]byte(secret), accessDuration, refreshDuration)

// 	svc := clients.NewService(cRepo, pRepo, tokenizer, emailer, phasher, idProvider, passRegex)
// 	ts := newClientServer(svc)
// 	defer ts.Close()

// 	conf := sdk.Config{
// 		UsersURL: ts.URL,
// 	}
// 	mfsdk := sdk.NewSDK(conf)

// 	client := sdk.User{
// 		ID: generateUUID(t),
// 		Credentials: sdk.Credentials{
// 			Identity: "valid@example.com",
// 			Secret:   "secret",
// 		},
// 		Status: sdk.EnabledStatus,
// 	}
// 	rClient := client
// 	rClient.Credentials.Secret, _ = phasher.Hash(client.Credentials.Secret)

// 	wrongClient := client
// 	wrongClient.Credentials.Secret, _ = phasher.Hash("wrong")

// 	cases := []struct {
// 		desc     string
// 		client   sdk.User
// 		dbClient sdk.User
// 		err      errors.SDKError
// 	}{
// 		{
// 			desc:     "issue token for a new user",
// 			client:   client,
// 			dbClient: rClient,
// 			err:      nil,
// 		},
// 		{
// 			desc:   "issue token for an empty user",
// 			client: sdk.User{},
// 			err:    errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingIdentity), http.StatusInternalServerError),
// 		},
// 		{
// 			desc: "issue token for invalid identity",
// 			client: sdk.User{
// 				Credentials: sdk.Credentials{
// 					Identity: "invalid",
// 					Secret:   "secret",
// 				},
// 			},
// 			dbClient: wrongClient,
// 			err:      errors.NewSDKErrorWithStatus(errors.ErrAuthentication, http.StatusUnauthorized),
// 		},
// 	}
// 	for _, tc := range cases {
// 		repoCall := cRepo.On("RetrieveByIdentity", mock.Anything, mock.Anything).Return(convertClient(tc.dbClient), tc.err)
// 		token, err := mfsdk.CreateToken(tc.client)
// 		switch tc.err {
// 		case nil:
// 			assert.NotEmpty(t, token, fmt.Sprintf("%s: expected token, got empty", tc.desc))
// 			ok := repoCall.Parent.AssertCalled(t, "RetrieveByIdentity", mock.Anything, mock.Anything)
// 			assert.True(t, ok, fmt.Sprintf("RetrieveByIdentity was not called on %s", tc.desc))
// 		default:
// 			assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
// 		}
// 		repoCall.Unset()
// 	}
// }

// func TestRefreshToken(t *testing.T) {
// 	cRepo := new(mocks.Repository)
// 	pRepo := new(pmocks.Repository)
// 	tokenizer := jwt.NewRepository([]byte(secret), accessDuration, refreshDuration)

// 	svc := clients.NewService(cRepo, pRepo, tokenizer, emailer, phasher, idProvider, passRegex)
// 	ts := newClientServer(svc)
// 	defer ts.Close()

// 	conf := sdk.Config{
// 		UsersURL: ts.URL,
// 	}
// 	mfsdk := sdk.NewSDK(conf)

// 	user := sdk.User{
// 		ID:   generateUUID(t),
// 		Name: "validtoken",
// 		Credentials: sdk.Credentials{
// 			Identity: "validtoken",
// 			Secret:   "secret",
// 		},
// 		Status: sdk.EnabledStatus,
// 	}
// 	rUser := user
// 	rUser.Credentials.Secret, _ = phasher.Hash(user.Credentials.Secret)

// 	repoCall := cRepo.On("RetrieveByIdentity", context.Background(), user.Credentials.Identity).Return(convertClient(rUser), nil)
// 	token, err := svc.IssueToken(context.Background(), user.Credentials.Identity, user.Credentials.Secret)
// 	assert.True(t, errors.Contains(err, nil), fmt.Sprintf("Create token expected nil got %s\n", err))
// 	ok := repoCall.Parent.AssertCalled(t, "RetrieveByIdentity", context.Background(), user.Credentials.Identity)
// 	assert.True(t, ok, "RetrieveByIdentity was not called on creating token")
// 	repoCall.Unset()

// 	cases := []struct {
// 		desc  string
// 		token string
// 		err   errors.SDKError
// 	}{
// 		{
// 			desc:  "refresh token for a valid refresh token",
// 			token: token.RefreshToken,
// 			err:   nil,
// 		},
// 		{
// 			desc:  "refresh token for a valid access token",
// 			token: token.AccessToken,
// 			err:   errors.NewSDKErrorWithStatus(errors.ErrAuthentication, http.StatusUnauthorized),
// 		},
// 		{
// 			desc:  "refresh token for an empty token",
// 			token: "",
// 			err:   errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusInternalServerError),
// 		},
// 	}
// 	for _, tc := range cases {
// 		repoCall := cRepo.On("RetrieveByID", mock.Anything, mock.Anything).Return(convertClient(user), tc.err)
// 		_, err := mfsdk.RefreshToken(tc.token)
// 		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
// 		if tc.err == nil {
// 			assert.NotEmpty(t, token, fmt.Sprintf("%s: expected token, got empty", tc.desc))
// 			ok := repoCall.Parent.AssertCalled(t, "RetrieveByID", mock.Anything, mock.Anything)
// 			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
// 		}
// 		repoCall.Unset()
// 	}
// }
