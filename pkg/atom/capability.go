// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package atom

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// capabilityDeclarationKey is where BuildCapabilitySchema stores the capability
// document verbatim. It lives in ui_schema rather than json_schema because Atom
// compiles json_schema with a JSON Schema validator on every entity write,
// while ui_schema is never compiled — so the declaration cannot affect what
// Atom accepts, whatever it contains.
const capabilityDeclarationKey = "x-magistrala-device-type"

// capabilityOrderKey records declaration order, which json_schema's properties
// object cannot preserve.
const capabilityOrderKey = "ui:order"

// Value types a measurement or command parameter may take. These are the JSON
// Schema names; normalizeValueType also accepts the shorthand a declaration is
// likely to be written with.
const (
	ValueTypeNumber  = "number"
	ValueTypeInteger = "integer"
	ValueTypeString  = "string"
	ValueTypeBoolean = "boolean"
)

// Measurement access. Access is advisory: it is rendered as JSON Schema's
// readOnly annotation, which validators record but do not enforce.
const (
	MeasurementAccessRead      = "r"
	MeasurementAccessReadWrite = "rw"
)

// BuildCapabilitySchema renders a capability document to the json_schema and
// ui_schema a device type version stores.
//
// json_schema constrains a device's *attributes*, which is what Atom validates
// on entity write. It does not constrain telemetry — messages never reach Atom.
//
// additionalProperties is deliberately left open. A Magistrala device carries
// housekeeping attributes no device type declares (source, tags, metadata,
// gateways, timestamps — see EntityFromFields), so closing the schema would
// reject every device write.
func BuildCapabilitySchema(doc CapabilityDocument) (jsonSchema, uiSchema map[string]any, err error) {
	normalized, err := normalizeCapabilityDocument(doc)
	if err != nil {
		return nil, nil, err
	}

	properties := make(map[string]any, len(normalized.Measurements))
	required := make([]string, 0, len(normalized.Measurements))
	order := make([]string, 0, len(normalized.Measurements))

	for _, measurement := range normalized.Measurements {
		property := map[string]any{"type": measurement.Type}
		if measurement.Unit != "" {
			property["unit"] = measurement.Unit
		}
		if measurement.Access == MeasurementAccessRead {
			property["readOnly"] = true
		}
		if measurement.Min != nil {
			property["minimum"] = *measurement.Min
		}
		if measurement.Max != nil {
			property["maximum"] = *measurement.Max
		}
		if len(measurement.Enum) > 0 {
			options := make([]any, len(measurement.Enum))
			for i, option := range measurement.Enum {
				options[i] = option
			}
			property["enum"] = options
		}
		properties[measurement.Name] = property
		order = append(order, measurement.Name)
		if measurement.Required {
			required = append(required, measurement.Name)
		}
	}

	jsonSchema = map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		jsonSchema["required"] = required
	}

	declaration, err := jsonValue(normalized)
	if err != nil {
		return nil, nil, err
	}
	uiSchema = map[string]any{capabilityDeclarationKey: declaration}
	if len(order) > 0 {
		uiSchema[capabilityOrderKey] = order
	}

	return jsonSchema, uiSchema, nil
}

// ParseCapabilityDocument reads a capability document back out of a stored
// device type version, so callers never hand-parse JSON Schema.
//
// It prefers the declaration BuildCapabilitySchema stashed in ui_schema, which
// round-trips exactly. Failing that — a version whose schema was written by
// hand — it reconstructs what it can from json_schema's properties. That
// fallback recovers measurements only: commands have no json_schema
// representation, since a command is not device state.
func ParseCapabilityDocument(jsonSchema, uiSchema map[string]any) (CapabilityDocument, error) {
	if declaration, ok := uiSchema[capabilityDeclarationKey]; ok {
		raw, err := json.Marshal(declaration)
		if err != nil {
			return CapabilityDocument{}, fmt.Errorf("atom: encode device type declaration: %w", err)
		}
		var doc CapabilityDocument
		if err := json.Unmarshal(raw, &doc); err != nil {
			return CapabilityDocument{}, fmt.Errorf("atom: decode device type declaration: %w", err)
		}
		return normalizeCapabilityDocument(doc)
	}
	return capabilityDocumentFromSchema(jsonSchema)
}

// capabilityDocumentFromSchema reconstructs measurements from a hand-written
// json_schema. Properties whose type the capability model cannot express
// (objects, arrays, unions) are skipped rather than failing the whole read —
// an advanced schema stays usable, it just does not round-trip in full.
func capabilityDocumentFromSchema(jsonSchema map[string]any) (CapabilityDocument, error) {
	properties, _ := jsonSchema["properties"].(map[string]any)
	if len(properties) == 0 {
		return CapabilityDocument{}, nil
	}

	required := map[string]bool{}
	for _, name := range jsonSlice(jsonSchema["required"]) {
		if name, ok := name.(string); ok {
			required[name] = true
		}
	}

	names := make([]string, 0, len(properties))
	for name := range properties {
		names = append(names, name)
	}
	sort.Strings(names)

	var doc CapabilityDocument
	for _, name := range names {
		property, ok := properties[name].(map[string]any)
		if !ok {
			continue
		}
		declared, _ := property["type"].(string)
		valueType, err := normalizeValueType(declared)
		if err != nil {
			continue
		}

		measurement := Measurement{
			Name:     name,
			Type:     valueType,
			Access:   MeasurementAccessReadWrite,
			Required: required[name],
		}
		if unit, ok := property["unit"].(string); ok {
			measurement.Unit = unit
		}
		if readOnly, ok := property["readOnly"].(bool); ok && readOnly {
			measurement.Access = MeasurementAccessRead
		}
		if min, ok := floatValue(property["minimum"]); ok {
			measurement.Min = &min
		}
		if max, ok := floatValue(property["maximum"]); ok {
			measurement.Max = &max
		}
		for _, option := range jsonSlice(property["enum"]) {
			if option, ok := option.(string); ok {
				measurement.Enum = append(measurement.Enum, option)
			}
		}
		doc.Measurements = append(doc.Measurements, measurement)
	}

	return normalizeCapabilityDocument(doc)
}

// normalizeCapabilityDocument fills in defaults and rejects declarations that
// cannot render to a schema. It is idempotent, so a document survives any
// number of build/parse cycles unchanged.
func normalizeCapabilityDocument(doc CapabilityDocument) (CapabilityDocument, error) {
	var out CapabilityDocument

	seen := make(map[string]struct{}, len(doc.Measurements))
	for _, measurement := range doc.Measurements {
		name := strings.TrimSpace(measurement.Name)
		if name == "" {
			return CapabilityDocument{}, errors.New("atom: device type measurement has no name")
		}
		if _, duplicate := seen[name]; duplicate {
			return CapabilityDocument{}, fmt.Errorf("atom: duplicate device type measurement %q", name)
		}
		seen[name] = struct{}{}

		declared := strings.TrimSpace(measurement.Type)
		if declared == "" {
			declared = ValueTypeNumber
		}
		valueType, err := normalizeValueType(declared)
		if err != nil {
			return CapabilityDocument{}, fmt.Errorf("atom: measurement %q: %w", name, err)
		}
		access, err := normalizeAccess(measurement.Access)
		if err != nil {
			return CapabilityDocument{}, fmt.Errorf("atom: measurement %q: %w", name, err)
		}
		if measurement.Min != nil && measurement.Max != nil && *measurement.Min > *measurement.Max {
			return CapabilityDocument{}, fmt.Errorf("atom: measurement %q has min above max", name)
		}
		if len(measurement.Enum) > 0 && valueType != ValueTypeString {
			return CapabilityDocument{}, fmt.Errorf("atom: measurement %q has enum but type %q, want %q", name, valueType, ValueTypeString)
		}

		out.Measurements = append(out.Measurements, Measurement{
			Name:     name,
			Unit:     strings.TrimSpace(measurement.Unit),
			Access:   access,
			Type:     valueType,
			Required: measurement.Required,
			Min:      copyFloat(measurement.Min),
			Max:      copyFloat(measurement.Max),
			Enum:     cloneStrings(measurement.Enum),
		})
	}

	seen = make(map[string]struct{}, len(doc.Commands))
	for _, command := range doc.Commands {
		name := strings.TrimSpace(command.Name)
		if name == "" {
			return CapabilityDocument{}, errors.New("atom: device type command has no name")
		}
		if _, duplicate := seen[name]; duplicate {
			return CapabilityDocument{}, fmt.Errorf("atom: duplicate device type command %q", name)
		}
		seen[name] = struct{}{}

		var params map[string]string
		if len(command.Params) > 0 {
			params = make(map[string]string, len(command.Params))
			for param, declared := range command.Params {
				param = strings.TrimSpace(param)
				if param == "" {
					return CapabilityDocument{}, fmt.Errorf("atom: command %q has an unnamed parameter", name)
				}
				valueType, err := normalizeValueType(declared)
				if err != nil {
					return CapabilityDocument{}, fmt.Errorf("atom: command %q parameter %q: %w", name, param, err)
				}
				params[param] = valueType
			}
		}

		out.Commands = append(out.Commands, Command{
			Name:        name,
			Description: strings.TrimSpace(command.Description),
			Params:      params,
		})
	}

	return out, nil
}

func normalizeValueType(valueType string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(valueType)) {
	case ValueTypeNumber, "float", "double":
		return ValueTypeNumber, nil
	case ValueTypeInteger, "int":
		return ValueTypeInteger, nil
	case ValueTypeString, "text":
		return ValueTypeString, nil
	case ValueTypeBoolean, "bool":
		return ValueTypeBoolean, nil
	default:
		return "", fmt.Errorf("unsupported value type %q", valueType)
	}
}

func normalizeAccess(access string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(access)) {
	case "", MeasurementAccessRead:
		return MeasurementAccessRead, nil
	case MeasurementAccessReadWrite:
		return MeasurementAccessReadWrite, nil
	default:
		return "", fmt.Errorf("unsupported access %q, want %q or %q", access, MeasurementAccessRead, MeasurementAccessReadWrite)
	}
}

// jsonValue converts a value to the plain map/slice/scalar form it will have
// after a round trip through Atom, so what is stored matches what is read back.
func jsonValue(v any) (map[string]any, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("atom: encode device type declaration: %w", err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("atom: decode device type declaration: %w", err)
	}
	return out, nil
}

func jsonSlice(v any) []any {
	switch v := v.(type) {
	case []any:
		return v
	case []string:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = item
		}
		return out
	default:
		return nil
	}
}

func floatValue(v any) (float64, bool) {
	switch v := v.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case json.Number:
		f, err := v.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

func copyFloat(v *float64) *float64 {
	if v == nil {
		return nil
	}
	out := *v
	return &out
}
