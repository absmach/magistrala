// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package atom

import (
	"encoding/json"
	"reflect"
	"testing"
)

func floatPtr(v float64) *float64 { return &v }

// storedRoundTrip mimics what a schema goes through on the way to Atom and
// back: marshalled to JSON, stored as JSONB, decoded into map[string]any. A
// declaration that only survives in memory would still be broken in practice.
func storedRoundTrip(t *testing.T, schema map[string]any) map[string]any {
	t.Helper()
	if schema == nil {
		return nil
	}
	raw, err := json.Marshal(schema)
	if err != nil {
		t.Fatalf("marshal schema: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}
	return out
}

func buildAndParse(t *testing.T, doc CapabilityDocument) CapabilityDocument {
	t.Helper()
	jsonSchema, uiSchema, err := BuildCapabilitySchema(doc)
	if err != nil {
		t.Fatalf("build capability schema: %v", err)
	}
	parsed, err := ParseCapabilityDocument(storedRoundTrip(t, jsonSchema), storedRoundTrip(t, uiSchema))
	if err != nil {
		t.Fatalf("parse capability document: %v", err)
	}
	return parsed
}

func TestCapabilityDocumentRoundTrip(t *testing.T) {
	cases := []struct {
		name string
		doc  CapabilityDocument
		want CapabilityDocument
	}{
		{
			name: "watermeter",
			doc: CapabilityDocument{
				Measurements: []Measurement{
					{Name: "volume", Unit: "m3", Access: MeasurementAccessRead, Required: true},
					{Name: "battery", Unit: "%", Access: MeasurementAccessRead, Min: floatPtr(0), Max: floatPtr(100)},
				},
				Commands: []Command{
					{Name: "set_interval", Params: map[string]string{"seconds": "int"}},
				},
			},
			want: CapabilityDocument{
				Measurements: []Measurement{
					{Name: "volume", Unit: "m3", Access: MeasurementAccessRead, Type: ValueTypeNumber, Required: true},
					{Name: "battery", Unit: "%", Access: MeasurementAccessRead, Type: ValueTypeNumber, Min: floatPtr(0), Max: floatPtr(100)},
				},
				Commands: []Command{
					{Name: "set_interval", Params: map[string]string{"seconds": ValueTypeInteger}},
				},
			},
		},
		{
			name: "no commands",
			doc: CapabilityDocument{
				Measurements: []Measurement{{Name: "temperature", Unit: "Cel"}},
			},
			want: CapabilityDocument{
				Measurements: []Measurement{{Name: "temperature", Unit: "Cel", Access: MeasurementAccessRead, Type: ValueTypeNumber}},
			},
		},
		{
			name: "no units",
			doc: CapabilityDocument{
				Measurements: []Measurement{{Name: "pulse_count", Type: "int"}},
			},
			want: CapabilityDocument{
				Measurements: []Measurement{{Name: "pulse_count", Access: MeasurementAccessRead, Type: ValueTypeInteger}},
			},
		},
		{
			name: "rw access",
			doc: CapabilityDocument{
				Measurements: []Measurement{
					{Name: "setpoint", Unit: "Cel", Access: MeasurementAccessReadWrite, Min: floatPtr(5), Max: floatPtr(30)},
				},
			},
			want: CapabilityDocument{
				Measurements: []Measurement{
					{Name: "setpoint", Unit: "Cel", Access: MeasurementAccessReadWrite, Type: ValueTypeNumber, Min: floatPtr(5), Max: floatPtr(30)},
				},
			},
		},
		{
			name: "commands without params",
			doc: CapabilityDocument{
				Commands: []Command{{Name: "reboot", Description: "power cycle the modem"}},
			},
			want: CapabilityDocument{
				Commands: []Command{{Name: "reboot", Description: "power cycle the modem"}},
			},
		},
		{
			name: "empty declaration",
			doc:  CapabilityDocument{},
			want: CapabilityDocument{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildAndParse(t, tc.doc)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("round trip changed the declaration:\n got: %+v\nwant: %+v", got, tc.want)
			}
		})
	}
}

// The capability model is easy to freeze around the watermeter example alone.
// These are the shapes the PRD calls out as likely to break it.
func TestCapabilityDocumentRoundTripsRealisticDeviceTypes(t *testing.T) {
	cases := []struct {
		name string
		doc  CapabilityDocument
	}{
		{
			name: "multi-channel energy meter",
			doc: CapabilityDocument{
				Measurements: []Measurement{
					{Name: "l1_voltage", Unit: "V", Type: ValueTypeNumber},
					{Name: "l2_voltage", Unit: "V", Type: ValueTypeNumber},
					{Name: "l3_voltage", Unit: "V", Type: ValueTypeNumber},
					{Name: "total_energy", Unit: "kWh", Type: ValueTypeNumber, Required: true},
				},
				Commands: []Command{{Name: "reset_counters"}},
			},
		},
		{
			name: "smart valve with enumerated state",
			doc: CapabilityDocument{
				Measurements: []Measurement{
					{Name: "position", Type: ValueTypeString, Access: MeasurementAccessReadWrite, Enum: []string{"open", "closed", "partial"}, Required: true},
					{Name: "opening", Unit: "%", Min: floatPtr(0), Max: floatPtr(100), Access: MeasurementAccessReadWrite},
				},
				Commands: []Command{
					{Name: "set_position", Params: map[string]string{"position": "string", "ramp_seconds": "int"}},
				},
			},
		},
		{
			name: "lorawan concentrator that also meters",
			doc: CapabilityDocument{
				Measurements: []Measurement{
					{Name: "uplink_count", Type: ValueTypeInteger},
					{Name: "radio_enabled", Type: ValueTypeBoolean, Access: MeasurementAccessReadWrite},
					{Name: "firmware", Type: ValueTypeString},
					{Name: "volume", Unit: "m3"},
				},
				Commands: []Command{
					{Name: "restart_radio"},
					{Name: "set_region", Params: map[string]string{"region": "string"}},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			first := buildAndParse(t, tc.doc)
			// Normalization must be idempotent, or a device type would drift
			// every time it is read and written back.
			second := buildAndParse(t, first)
			if !reflect.DeepEqual(first, second) {
				t.Fatalf("re-rendering a parsed declaration changed it:\nfirst: %+v\nsecond: %+v", first, second)
			}
			if len(first.Measurements) != len(tc.doc.Measurements) {
				t.Fatalf("expected %d measurements, got %d", len(tc.doc.Measurements), len(first.Measurements))
			}
			if len(first.Commands) != len(tc.doc.Commands) {
				t.Fatalf("expected %d commands, got %d", len(tc.doc.Commands), len(first.Commands))
			}
		})
	}
}

func TestBuildCapabilitySchemaRendersConstraints(t *testing.T) {
	jsonSchema, _, err := BuildCapabilitySchema(CapabilityDocument{
		Measurements: []Measurement{
			{Name: "volume", Unit: "m3", Required: true},
			{Name: "interval", Type: "int", Access: MeasurementAccessReadWrite, Min: floatPtr(30), Max: floatPtr(86400)},
			{Name: "mode", Type: ValueTypeString, Enum: []string{"normal", "burst"}},
		},
	})
	if err != nil {
		t.Fatalf("build capability schema: %v", err)
	}

	properties, ok := jsonSchema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected properties object, got %T", jsonSchema["properties"])
	}

	volume, _ := properties["volume"].(map[string]any)
	if volume["type"] != ValueTypeNumber || volume["unit"] != "m3" || volume["readOnly"] != true {
		t.Fatalf("unexpected volume property: %+v", volume)
	}

	interval, _ := properties["interval"].(map[string]any)
	if interval["type"] != ValueTypeInteger || interval["minimum"] != float64(30) || interval["maximum"] != float64(86400) {
		t.Fatalf("unexpected interval property: %+v", interval)
	}
	if _, readOnly := interval["readOnly"]; readOnly {
		t.Fatalf("rw measurement must not be marked readOnly: %+v", interval)
	}

	mode, _ := properties["mode"].(map[string]any)
	if !reflect.DeepEqual(mode["enum"], []any{"normal", "burst"}) {
		t.Fatalf("unexpected mode enum: %+v", mode["enum"])
	}

	if !reflect.DeepEqual(jsonSchema["required"], []string{"volume"}) {
		t.Fatalf("unexpected required list: %+v", jsonSchema["required"])
	}
}

// A device carries housekeeping attributes no device type declares. Closing
// the schema would reject every Magistrala device write.
func TestBuildCapabilitySchemaLeavesAdditionalPropertiesOpen(t *testing.T) {
	jsonSchema, _, err := BuildCapabilitySchema(CapabilityDocument{
		Measurements: []Measurement{{Name: "volume", Unit: "m3"}},
	})
	if err != nil {
		t.Fatalf("build capability schema: %v", err)
	}
	if _, closed := jsonSchema["additionalProperties"]; closed {
		t.Fatalf("device type schemas must stay open to housekeeping attributes: %+v", jsonSchema)
	}
}

// The declaration belongs in ui_schema: Atom compiles json_schema on every
// entity write, so a custom keyword there is a validator's problem.
func TestBuildCapabilitySchemaKeepsDeclarationOutOfJSONSchema(t *testing.T) {
	jsonSchema, uiSchema, err := BuildCapabilitySchema(CapabilityDocument{
		Measurements: []Measurement{{Name: "volume", Unit: "m3"}},
		Commands:     []Command{{Name: "reboot"}},
	})
	if err != nil {
		t.Fatalf("build capability schema: %v", err)
	}
	if _, found := jsonSchema[capabilityDeclarationKey]; found {
		t.Fatalf("declaration must not appear in json_schema: %+v", jsonSchema)
	}
	if _, found := uiSchema[capabilityDeclarationKey]; !found {
		t.Fatalf("declaration must be stored in ui_schema: %+v", uiSchema)
	}
	if !reflect.DeepEqual(uiSchema[capabilityOrderKey], []string{"volume"}) {
		t.Fatalf("ui_schema must record declaration order: %+v", uiSchema[capabilityOrderKey])
	}
}

func TestBuildCapabilitySchemaRejectsUnusableDeclarations(t *testing.T) {
	cases := []struct {
		name string
		doc  CapabilityDocument
	}{
		{"measurement without a name", CapabilityDocument{Measurements: []Measurement{{Unit: "m3"}}}},
		{"duplicate measurements", CapabilityDocument{Measurements: []Measurement{{Name: "volume"}, {Name: "volume"}}}},
		{"unsupported measurement type", CapabilityDocument{Measurements: []Measurement{{Name: "volume", Type: "geometry"}}}},
		{"unsupported access", CapabilityDocument{Measurements: []Measurement{{Name: "volume", Access: "w"}}}},
		{"min above max", CapabilityDocument{Measurements: []Measurement{{Name: "volume", Min: floatPtr(10), Max: floatPtr(1)}}}},
		{"non-string enum", CapabilityDocument{Measurements: []Measurement{{Name: "volume", Type: ValueTypeInteger, Enum: []string{"1", "2"}}}}},
		{"command without a name", CapabilityDocument{Commands: []Command{{Description: "no name"}}}},
		{"duplicate commands", CapabilityDocument{Commands: []Command{{Name: "reboot"}, {Name: "reboot"}}}},
		{"parameter without a type", CapabilityDocument{Commands: []Command{{Name: "set", Params: map[string]string{"seconds": ""}}}}},
		{"unsupported parameter type", CapabilityDocument{Commands: []Command{{Name: "set", Params: map[string]string{"at": "timestamp"}}}}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := BuildCapabilitySchema(tc.doc); err == nil {
				t.Fatal("expected an error, got none")
			}
		})
	}
}

// A version whose schema was written by hand has no declaration to read, so
// the measurements are reconstructed from the schema itself.
func TestParseCapabilityDocumentFallsBackToJSONSchema(t *testing.T) {
	got, err := ParseCapabilityDocument(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"volume":   map[string]any{"type": "number", "unit": "m3", "readOnly": true},
			"interval": map[string]any{"type": "integer", "minimum": float64(30), "maximum": float64(600)},
			"mode":     map[string]any{"type": "string", "enum": []any{"normal", "burst"}},
		},
		"required": []any{"volume"},
	}, nil)
	if err != nil {
		t.Fatalf("parse capability document: %v", err)
	}

	want := CapabilityDocument{Measurements: []Measurement{
		{Name: "interval", Type: ValueTypeInteger, Access: MeasurementAccessReadWrite, Min: floatPtr(30), Max: floatPtr(600)},
		{Name: "mode", Type: ValueTypeString, Access: MeasurementAccessReadWrite, Enum: []string{"normal", "burst"}},
		{Name: "volume", Unit: "m3", Type: ValueTypeNumber, Access: MeasurementAccessRead, Required: true},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected reconstruction:\n got: %+v\nwant: %+v", got, want)
	}
}

// An advanced schema stays readable even where the capability model cannot
// describe it — the unmodelled properties are skipped, not fatal.
func TestParseCapabilityDocumentSkipsUnmodelledProperties(t *testing.T) {
	got, err := ParseCapabilityDocument(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"volume":   map[string]any{"type": "number"},
			"channels": map[string]any{"type": "array"},
			"location": map[string]any{"type": "object"},
			"untyped":  map[string]any{},
		},
	}, nil)
	if err != nil {
		t.Fatalf("parse capability document: %v", err)
	}
	if len(got.Measurements) != 1 || got.Measurements[0].Name != "volume" {
		t.Fatalf("expected only the modellable property, got: %+v", got.Measurements)
	}
}

func TestParseCapabilityDocumentPrefersStoredDeclaration(t *testing.T) {
	doc := CapabilityDocument{
		Measurements: []Measurement{{Name: "volume", Unit: "m3", Access: MeasurementAccessRead}},
		Commands:     []Command{{Name: "set_interval", Params: map[string]string{"seconds": "int"}}},
	}
	_, uiSchema, err := BuildCapabilitySchema(doc)
	if err != nil {
		t.Fatalf("build capability schema: %v", err)
	}

	// Commands have no json_schema representation, so recovering them proves
	// the declaration was read rather than the schema reverse-engineered.
	got, err := ParseCapabilityDocument(nil, storedRoundTrip(t, uiSchema))
	if err != nil {
		t.Fatalf("parse capability document: %v", err)
	}
	if len(got.Commands) != 1 || got.Commands[0].Name != "set_interval" {
		t.Fatalf("expected the stored declaration's commands, got: %+v", got.Commands)
	}
}
