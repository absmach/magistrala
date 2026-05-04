// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/absmach/magistrala/pkg/errors"
	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

// Renderer renders a Profile's content template into a concrete device
// configuration. All input data must already be stored in Bootstrap — no
// external service calls are allowed inside Render.
type Renderer interface {
	Render(profile Profile, enrollment Config, bindings []BindingSnapshot) ([]byte, error)
}

// ErrRenderFailed is returned when template execution or output validation fails.
var ErrRenderFailed = errors.New("failed to render profile template")

type renderer struct{}

// NewRenderer returns the default Renderer implementation using Go text/template.
func NewRenderer() Renderer {
	return renderer{}
}

func (r renderer) Render(profile Profile, enrollment Config, bindings []BindingSnapshot) ([]byte, error) {
	rctx := buildRenderContext(profile, enrollment, bindings)

	switch profile.TemplateFormat {
	case TemplateFormatRaw:
		return []byte(profile.ContentTemplate), nil
	case TemplateFormatGoTemplate, TemplateFormatJSON, TemplateFormatYAML, TemplateFormatTOML, "":
		return r.renderTemplate(profile, rctx)
	default:
		return nil, fmt.Errorf("%w: unsupported template format %q", ErrRenderFailed, profile.TemplateFormat)
	}
}

func (r renderer) renderTemplate(profile Profile, rctx RenderContext) ([]byte, error) {
	t, err := template.New("bootstrap").
		Option("missingkey=error").
		Funcs(allowlistedFuncs()).
		Parse(profile.ContentTemplate)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrRenderFailed, err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, rctx); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrRenderFailed, err)
	}

	out, err := validateRenderedOutput(buf.Bytes(), profile.TemplateFormat)
	if err != nil {
		return nil, err
	}

	return out, nil
}

// validateRenderedOutput checks that the rendered bytes are valid for the
// declared output format. It returns the original bytes on success and wraps
// ErrRenderFailed on failure.
func validateRenderedOutput(out []byte, format TemplateFormat) ([]byte, error) {
	// Unrecognised formats are passed through. Recognised structured formats
	// must parse successfully so broken templates fail before reaching devices.
	switch format {
	case TemplateFormatJSON:
		var v any
		if err := json.Unmarshal(out, &v); err != nil {
			return nil, fmt.Errorf("%w: invalid json output: %w", ErrRenderFailed, err)
		}
	case TemplateFormatYAML:
		var v any
		if err := yaml.Unmarshal(out, &v); err != nil {
			return nil, fmt.Errorf("%w: invalid yaml output: %w", ErrRenderFailed, err)
		}
	case TemplateFormatTOML:
		var v any
		if err := toml.Unmarshal(out, &v); err != nil {
			return nil, fmt.Errorf("%w: invalid toml output: %w", ErrRenderFailed, err)
		}
	}
	return out, nil
}

// buildRenderContext constructs the typed RenderContext from stored data.
// No external calls are made here.
func buildRenderContext(profile Profile, enrollment Config, bindings []BindingSnapshot) RenderContext {
	vars := make(map[string]any)
	for k, v := range profile.Defaults {
		vars[k] = v
	}
	for k, v := range enrollment.RenderContext {
		vars[k] = v
	}

	bctx := make(map[string]BindingContext, len(bindings))
	for _, b := range bindings {
		bctx[b.Slot] = BindingContext{
			Type:     b.Type,
			ID:       b.ResourceID,
			Snapshot: b.Snapshot,
			Secret:   b.SecretSnapshot,
		}
	}

	return RenderContext{
		Device: DeviceContext{
			ID:         enrollment.ID,
			ExternalID: enrollment.ExternalID,
			DomainID:   enrollment.DomainID,
		},
		Vars:     vars,
		Bindings: bctx,
	}
}

// allowlistedFuncs returns the safe set of template helper functions.
// No function in this map may call an external service or perform I/O.
func allowlistedFuncs() template.FuncMap {
	return template.FuncMap{
		"toJSON": func(v any) (string, error) {
			b, err := json.Marshal(v)
			if err != nil {
				return "", err
			}
			return string(b), nil
		},
		"default": func(def, val any) any {
			if val == nil || val == "" {
				return def
			}
			return val
		},
		"required": func(key string, val any) (any, error) {
			if val == nil || val == "" {
				return nil, fmt.Errorf("required value %q is missing", key)
			}
			return val, nil
		},
	}
}
