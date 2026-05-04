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
		format   bootstrap.TemplateFormat
		template string
		err      error
	}{
		{
			desc:     "valid JSON output",
			format:   bootstrap.TemplateFormatJSON,
			template: `{"device_id":"{{ .Device.ID }}"}`,
		},
		{
			desc:     "invalid JSON output",
			format:   bootstrap.TemplateFormatJSON,
			template: `{"device_id":`,
			err:      bootstrap.ErrRenderFailed,
		},
		{
			desc:     "valid YAML output",
			format:   bootstrap.TemplateFormatYAML,
			template: "device_id: {{ .Device.ID }}",
		},
		{
			desc:     "invalid YAML output",
			format:   bootstrap.TemplateFormatYAML,
			template: "device_id: [",
			err:      bootstrap.ErrRenderFailed,
		},
		{
			desc:     "valid TOML output",
			format:   bootstrap.TemplateFormatTOML,
			template: `device_id = "{{ .Device.ID }}"`,
		},
		{
			desc:     "invalid TOML output",
			format:   bootstrap.TemplateFormatTOML,
			template: `device_id = `,
			err:      bootstrap.ErrRenderFailed,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			_, err := renderer.Render(
				bootstrap.Profile{
					TemplateFormat:  tc.format,
					ContentTemplate: tc.template,
				},
				bootstrap.Config{ID: "config-id"},
				nil,
			)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.err, err))
		})
	}
}
