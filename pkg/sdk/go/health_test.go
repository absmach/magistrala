// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk_test

// func TestHealth(t *testing.T) {
// 	thingcRepo := new(thingsclientsmock.Repository)
// 	usercRepo := new(cmocks.Repository)
// 	gRepo := new(gmocks.Repository)
// 	uauth := cmocks.NewAuthService(users, map[string][]cmocks.SubjectSet{adminID: {uadminPolicy}})
// 	thingCache := thingsclientsmock.NewCache()
// 	policiesCache := thingspmocks.NewCache()
// 	tokenizer := jwt.NewRepository([]byte(secret), accessDuration, refreshDuration)

// 	thingspRepo := new(thingspmocks.Repository)
// 	psvc := policies.NewService(uauth, thingspRepo, policiesCache, idProvider)

// 	thsvc := thingsclients.NewService(uauth, psvc, thingcRepo, gRepo, thingCache, idProvider)
// 	ths := newThingsServer(thsvc, psvc)
// 	defer ths.Close()

// 	userspRepo := new(userspmocks.Repository)
// 	usSvc := usersclients.NewService(usercRepo, userspRepo, tokenizer, emailer, phasher, idProvider, passRegex)
// 	usclsv := newClientServer(usSvc)
// 	defer usclsv.Close()

// 	certSvc, err := newCertService()
// 	require.Nil(t, err, fmt.Sprintf("unexpected error during creating service: %s", err))
// 	CertTs := newCertServer(certSvc)
// 	defer CertTs.Close()

// 	sdkConf := sdk.Config{
// 		ThingsURL:       ths.URL,
// 		UsersURL:        usclsv.URL,
// 		CertsURL:        CertTs.URL,
// 		MsgContentType:  contentType,
// 		TLSVerification: false,
// 	}

// 	mfsdk := sdk.NewSDK(sdkConf)
// 	cases := map[string]struct {
// 		service     string
// 		empty       bool
// 		description string
// 		status      string
// 		err         errors.SDKError
// 	}{
// 		"get things service health check": {
// 			service:     "things",
// 			empty:       false,
// 			err:         nil,
// 			description: "things service",
// 			status:      "pass",
// 		},
// 		"get users service health check": {
// 			service:     "users",
// 			empty:       false,
// 			err:         nil,
// 			description: "users service",
// 			status:      "pass",
// 		},
// 		"get certs service health check": {
// 			service:     "certs",
// 			empty:       false,
// 			err:         nil,
// 			description: "certs service",
// 			status:      "pass",
// 		},
// 	}
// 	for desc, tc := range cases {
// 		h, err := mfsdk.Health(tc.service)
// 		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", desc, tc.err, err))
// 		assert.Equal(t, tc.status, h.Status, fmt.Sprintf("%s: expected %s status, got %s", desc, tc.status, h.Status))
// 		assert.Equal(t, tc.empty, h.Version == "", fmt.Sprintf("%s: expected non-empty version", desc))
// 		assert.Equal(t, mainflux.Commit, h.Commit, fmt.Sprintf("%s: expected non-empty commit", desc))
// 		assert.Equal(t, tc.description, h.Description, fmt.Sprintf("%s: expected proper description, got %s", desc, h.Description))
// 		assert.Equal(t, mainflux.BuildTime, h.BuildTime, fmt.Sprintf("%s: expected default epoch date, got %s", desc, h.BuildTime))
// 	}
// }
