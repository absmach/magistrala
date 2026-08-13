// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package atom

import (
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

// atomSchemaError builds the error Atom returns for a schema rejection: the
// validator's own messages, joined, behind Atom's prefix.
func atomSchemaError(messages ...string) error {
	return Error{
		StatusCode: http.StatusBadRequest,
		Message:    atomSchemaValidationPrefix + strings.Join(messages, "; "),
	}
}

func TestTranslateSchemaErrorNamesTheOffendingField(t *testing.T) {
	cases := []struct {
		name       string
		message    string
		attributes Attributes
		want       SchemaViolation
	}{
		{
			name:    "missing required property",
			message: `"volume" is a required property`,
			want:    SchemaViolation{Field: "volume", Constraint: "required"},
		},
		{
			name:       "wrong type",
			message:    `"many" is not of type "number"`,
			attributes: Attributes{"volume": "many", "battery": 80},
			want:       SchemaViolation{Field: "volume", Constraint: "type", Expected: "number"},
		},
		{
			name:       "out of range above maximum",
			message:    `120 is greater than the maximum of 100`,
			attributes: Attributes{"battery": 120, "volume": 3.5},
			want:       SchemaViolation{Field: "battery", Constraint: "maximum", Expected: "100"},
		},
		{
			name:       "out of range below minimum",
			message:    `5 is less than the minimum of 30`,
			attributes: Attributes{"interval": 5},
			want:       SchemaViolation{Field: "interval", Constraint: "minimum", Expected: "30"},
		},
		{
			name:       "not one of the declared states",
			message:    `"sideways" is not one of ["open","closed"]`,
			attributes: Attributes{"position": "sideways"},
			want:       SchemaViolation{Field: "position", Constraint: "enum", Expected: `["open","closed"]`},
		},
		{
			name:       "exclusive maximum",
			message:    `100 is greater than or equal to the maximum of 100`,
			attributes: Attributes{"duty": 100},
			want:       SchemaViolation{Field: "duty", Constraint: "exclusiveMaximum", Expected: "100"},
		},
		{
			name:       "string too long",
			message:    `"abcdefghijk" is longer than 8 characters`,
			attributes: Attributes{"firmware": "abcdefghijk"},
			want:       SchemaViolation{Field: "firmware", Constraint: "maxLength", Expected: "8"},
		},
		{
			name:    "unexpected property",
			message: `Additional properties are not allowed ('flow' was unexpected)`,
			want:    SchemaViolation{Field: "flow", Constraint: "additionalProperties"},
		},
		{
			name:       "multiple type names",
			message:    `true is not of types "integer", "number"`,
			attributes: Attributes{"volume": true},
			want:       SchemaViolation{Field: "volume", Constraint: "type", Expected: `"integer", "number"`},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := translateSchemaError(atomSchemaError(tc.message), tc.attributes)

			schemaErr, ok := AsSchemaValidationError(err)
			if !ok {
				t.Fatalf("expected a schema validation error, got %T: %v", err, err)
			}
			if len(schemaErr.Violations) != 1 {
				t.Fatalf("expected one violation, got %+v", schemaErr.Violations)
			}

			got := schemaErr.Violations[0]
			if got.Message != tc.message {
				t.Fatalf("violation must keep Atom's own message, got %q", got.Message)
			}
			got.Message = ""
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("unexpected violation:\n got: %+v\nwant: %+v", got, tc.want)
			}
			if !strings.Contains(err.Error(), tc.want.Field) {
				t.Fatalf("error text must name the field, got %q", err.Error())
			}
		})
	}
}

func TestTranslateSchemaErrorSplitsEveryViolation(t *testing.T) {
	err := translateSchemaError(atomSchemaError(
		`"volume" is a required property`,
		`"warm" is not of type "number"`,
		`150 is greater than the maximum of 100`,
	), Attributes{"setpoint": "warm", "battery": 150})

	schemaErr, ok := AsSchemaValidationError(err)
	if !ok {
		t.Fatalf("expected a schema validation error, got %T: %v", err, err)
	}
	if got, want := len(schemaErr.Violations), 3; got != want {
		t.Fatalf("expected %d violations, got %d: %+v", want, got, schemaErr.Violations)
	}
	if got, want := schemaErr.Fields(), []string{"volume", "setpoint", "battery"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected fields: got %+v, want %+v", got, want)
	}
}

func TestTranslateSchemaErrorReportsEveryUnexpectedProperty(t *testing.T) {
	err := translateSchemaError(atomSchemaError(
		`Additional properties are not allowed ('flow', 'pressure' were unexpected)`,
	), nil)

	schemaErr, ok := AsSchemaValidationError(err)
	if !ok {
		t.Fatalf("expected a schema validation error, got %T: %v", err, err)
	}
	if got, want := schemaErr.Fields(), []string{"flow", "pressure"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected fields: got %+v, want %+v", got, want)
	}
}

// Naming the wrong attribute is worse than naming none, so an ambiguous value
// yields a violation with no field rather than a guess.
func TestTranslateSchemaErrorLeavesAmbiguousFieldUnnamed(t *testing.T) {
	err := translateSchemaError(
		atomSchemaError(`"warm" is not of type "number"`),
		Attributes{"setpoint": "warm", "fallback": "warm"},
	)

	schemaErr, ok := AsSchemaValidationError(err)
	if !ok {
		t.Fatalf("expected a schema validation error, got %T: %v", err, err)
	}
	if got := schemaErr.Violations[0].Field; got != "" {
		t.Fatalf("expected no field for an ambiguous value, got %q", got)
	}
	if got := schemaErr.Violations[0].Constraint; got != "type" {
		t.Fatalf("constraint must survive even without a field, got %q", got)
	}
}

// Rust renders a whole float as 5.0 where Go encodes it as 5.
func TestTranslateSchemaErrorMatchesNumbersByValue(t *testing.T) {
	err := translateSchemaError(
		atomSchemaError(`5.0 is less than the minimum of 30`),
		Attributes{"interval": float64(5)},
	)

	schemaErr, ok := AsSchemaValidationError(err)
	if !ok {
		t.Fatalf("expected a schema validation error, got %T: %v", err, err)
	}
	if got := schemaErr.Violations[0].Field; got != "interval" {
		t.Fatalf("expected the numeric attribute to be matched, got %q", got)
	}
}

// An unrecognised validator message still becomes a typed error carrying the
// original text, rather than being dropped back to a raw GraphQL error.
func TestTranslateSchemaErrorKeepsUnrecognisedMessages(t *testing.T) {
	err := translateSchemaError(atomSchemaError(`something the validator invented`), nil)

	schemaErr, ok := AsSchemaValidationError(err)
	if !ok {
		t.Fatalf("expected a schema validation error, got %T: %v", err, err)
	}
	if got := schemaErr.Violations[0].Message; got != "something the validator invented" {
		t.Fatalf("unexpected message: %q", got)
	}
}

func TestTranslateSchemaErrorLeavesOtherErrorsAlone(t *testing.T) {
	cases := []struct {
		name string
		err  error
	}{
		{"not found", Error{StatusCode: http.StatusNotFound, Message: "entity not found"}},
		{"conflict", Error{StatusCode: http.StatusConflict, Message: "already exists"}},
		{"non atom error", errors.New("dial tcp: connection refused")},
		{"nil", nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := translateSchemaError(tc.err, nil)
			if !errors.Is(got, tc.err) {
				t.Fatalf("expected the original error back, got %v", got)
			}
			if _, isSchema := AsSchemaValidationError(got); isSchema {
				t.Fatal("must not be reported as a schema violation")
			}
		})
	}
}

// The translated error wraps Atom's, so status-based checks keep working.
func TestSchemaValidationErrorWrapsAtomError(t *testing.T) {
	err := translateSchemaError(atomSchemaError(`"volume" is a required property`), nil)

	var atomErr Error
	if !errors.As(err, &atomErr) {
		t.Fatalf("expected the atom error to remain reachable, got %T", err)
	}
	if atomErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d", atomErr.StatusCode)
	}
	if IsNotFound(err) || IsConflict(err) {
		t.Fatal("a schema rejection is neither a conflict nor a not-found")
	}
}
