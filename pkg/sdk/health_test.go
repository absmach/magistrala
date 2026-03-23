// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"fmt"
	"testing"

	"github.com/absmach/supermq"
	"github.com/absmach/supermq/pkg/errors"
	sdk "github.com/absmach/supermq/pkg/sdk"
	"github.com/stretchr/testify/assert"
)

func TestHealth(t *testing.T) {
	clientsTs, _, _ := setupClients()
	defer clientsTs.Close()

	usersTs, _, _ := setupUsers()
	defer usersTs.Close()

	groupsTs, _, _ := setupGroups()
	defer groupsTs.Close()

	channelsTs, _, _ := setupChannels()
	defer channelsTs.Close()

	domainsTs, _, _ := setupDomains()
	defer domainsTs.Close()

	journalTs, _, _ := setupJournal()
	defer journalTs.Close()

	fluxmqTs := setupFluxMQ("any")
	defer fluxmqTs.Close()

	sdkConf := sdk.Config{
		ClientsURL:      clientsTs.URL,
		UsersURL:        usersTs.URL,
		HTTPAdapterURL:  fluxmqTs.URL,
		GroupsURL:       groupsTs.URL,
		ChannelsURL:     channelsTs.URL,
		DomainsURL:      domainsTs.URL,
		JournalURL:      journalTs.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mgsdk := sdk.NewSDK(sdkConf)
	cases := []struct {
		desc        string
		service     string
		empty       bool
		description string
		status      string
		err         errors.SDKError
	}{
		{
			desc:        "get clients service health check",
			service:     "clients",
			empty:       false,
			err:         nil,
			description: "clients service",
			status:      "pass",
		},
		{
			desc:        "get users service health check",
			service:     "users",
			empty:       false,
			err:         nil,
			description: "users service",
			status:      "pass",
		},
		{
			desc:        "get groups service health check",
			service:     "groups",
			empty:       false,
			err:         nil,
			description: "groups service",
			status:      "pass",
		},
		{
			desc:        "get channels service health check",
			service:     "channels",
			empty:       false,
			err:         nil,
			description: "channels service",
			status:      "pass",
		},
		{
			desc:        "get domains service health check",
			service:     "domains",
			empty:       false,
			err:         nil,
			description: "domains service",
			status:      "pass",
		},
		{
			desc:        "get journal service health check",
			service:     "journal",
			empty:       false,
			err:         nil,
			description: "journal-log service",
			status:      "pass",
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			h, err := mgsdk.Health(tc.service)
			assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
			assert.Equal(t, tc.status, h.Status, fmt.Sprintf("%s: expected %s status, got %s", tc.desc, tc.status, h.Status))
			assert.Equal(t, tc.empty, h.Version == "", fmt.Sprintf("%s: expected non-empty version", tc.desc))
			assert.Equal(t, supermq.Commit, h.Commit, fmt.Sprintf("%s: expected non-empty commit", tc.desc))
			assert.Equal(t, tc.description, h.Description, fmt.Sprintf("%s: expected proper description, got %s", tc.desc, h.Description))
			assert.Equal(t, supermq.BuildTime, h.BuildTime, fmt.Sprintf("%s: expected default epoch date, got %s", tc.desc, h.BuildTime))
		})
	}

	// FluxMQ returns a simpler health response without version/commit/description.
	t.Run("get fluxmq service health check", func(t *testing.T) {
		h, err := mgsdk.Health("fluxmq")
		assert.Nil(t, err)
		assert.Equal(t, "healthy", h.Status)
	})
}
