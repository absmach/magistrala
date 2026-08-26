// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli_test

import (
	"encoding/json"
	"testing"

	"github.com/absmach/magistrala/cli"
	smqsdk "github.com/absmach/magistrala/pkg/sdk"
	sdkmocks "github.com/absmach/magistrala/pkg/sdk/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthCmdReportsUnsupportedService(t *testing.T) {
	mockSDK := sdkmocks.NewSDK(t)
	cli.SetSDK(mockSDK)

	out := executeCommand(t, cli.NewHealthCmd(), "certs")

	assert.Contains(t, out, `unsupported health service "certs"`)
}

func TestHealthCmdChecksFluxMQ(t *testing.T) {
	mockSDK := sdkmocks.NewSDK(t)
	mockSDK.On("Health", "fluxmq").Return(smqsdk.HealthInfo{Status: "healthy"}, nil).Once()
	cli.SetSDK(mockSDK)

	out := executeCommand(t, cli.NewHealthCmd(), "fluxmq")

	var got smqsdk.HealthInfo
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	assert.Equal(t, "healthy", got.Status)
}
