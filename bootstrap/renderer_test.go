// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package bootstrap_test

import (
	"fmt"
	"testing"

	"github.com/absmach/magistrala/bootstrap"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestRendererStructuredOutputValidation(t *testing.T) {
	renderer := bootstrap.NewRenderer()

	cases := []struct {
		desc     string
		format   bootstrap.ContentFormat
		template string
		err      error
	}{
		{
			desc:     "valid JSON output",
			format:   bootstrap.ContentFormatJSON,
			template: `{"device_id":"{{ .Device.ID }}"}`,
		},
		{
			desc:     "invalid output for JSON format",
			format:   bootstrap.ContentFormatJSON,
			template: `[unclosed bracket`,
			err:      bootstrap.ErrRenderFailed,
		},
		{
			desc:     "valid YAML output",
			format:   bootstrap.ContentFormatYAML,
			template: "device_id: {{ .Device.ID }}",
		},
		{
			desc:     "invalid output for YAML format",
			format:   bootstrap.ContentFormatYAML,
			template: "[unclosed bracket",
			err:      bootstrap.ErrRenderFailed,
		},
		{
			desc:   "valid TOML output",
			format: bootstrap.ContentFormatTOML,
			template: `[device]
			device_id = "{{ .Device.ID }}"`,
		},
		{
			desc:     "invalid output for TOML format",
			format:   bootstrap.ContentFormatTOML,
			template: `[unclosed bracket`,
			err:      bootstrap.ErrRenderFailed,
		},
		{
			desc:     "JSON template auto-converted to TOML",
			format:   bootstrap.ContentFormatTOML,
			template: `{"device_id":"{{ .Device.ID }}"}`,
		},
		{
			desc:     "TOML template auto-converted to JSON",
			format:   bootstrap.ContentFormatJSON,
			template: `device_id = "{{ .Device.ID }}"`,
		},
		{
			desc:     "YAML template auto-converted to TOML",
			format:   bootstrap.ContentFormatTOML,
			template: "device_id: {{ .Device.ID }}",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			_, err := renderer.Render(
				bootstrap.Profile{
					ContentFormat:   tc.format,
					ContentTemplate: tc.template,
				},
				bootstrap.Config{ID: "config-id"},
				nil,
			)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.err, err))
		})
	}
}
