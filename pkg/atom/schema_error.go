// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package atom

import (
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
	"strings"
)

// atomSchemaValidationPrefix is what Atom puts in front of the validator's own
// output when a device's attributes fail its device type schema
// (src/schema.rs). Everything after it is the raw jsonschema crate messages,
// joined with "; ".
const atomSchemaValidationPrefix = "attributes failed profile schema validation: "

// SchemaViolation is one field-level reason a device's attributes were
// rejected.
//
// Field is best effort. The validator names the property directly for missing
// and unexpected properties, but for every other failure it reports only the
// offending *value* — so the field is recovered by matching that value against
// the attributes that were submitted, and is left empty when that is ambiguous.
type SchemaViolation struct {
	Field      string `json:"field,omitempty"`
	Constraint string `json:"constraint,omitempty"`
	Expected   string `json:"expected,omitempty"`
	Message    string `json:"message"`
}

func (v SchemaViolation) String() string {
	if v.Field == "" {
		return v.Message
	}
	return v.Field + ": " + v.Message
}

// SchemaValidationError is a device type schema rejection, translated from the
// validator output Atom returns verbatim. Untranslated, an operator creating a
// device sees a bare validator string with no indication of which attribute
// was wrong.
//
// It wraps the underlying atom Error, so IsConflict, IsNotFound and errors.As
// on Error keep working.
type SchemaValidationError struct {
	Violations []SchemaViolation
	err        error
}

func (e *SchemaValidationError) Error() string {
	parts := make([]string, 0, len(e.Violations))
	for _, violation := range e.Violations {
		parts = append(parts, violation.String())
	}
	if len(parts) == 0 {
		return "atom: attributes rejected by the device type schema"
	}
	return "atom: attributes rejected by the device type schema: " + strings.Join(parts, "; ")
}

func (e *SchemaValidationError) Unwrap() error { return e.err }

// Fields lists the attributes the violations name, in order and without
// duplicates.
func (e *SchemaValidationError) Fields() []string {
	var fields []string
	seen := map[string]struct{}{}
	for _, violation := range e.Violations {
		if violation.Field == "" {
			continue
		}
		if _, duplicate := seen[violation.Field]; duplicate {
			continue
		}
		seen[violation.Field] = struct{}{}
		fields = append(fields, violation.Field)
	}
	return fields
}

// AsSchemaValidationError reports whether err is a device type schema
// rejection, and if so returns it.
func AsSchemaValidationError(err error) (*SchemaValidationError, bool) {
	var schemaErr *SchemaValidationError
	if errors.As(err, &schemaErr) {
		return schemaErr, true
	}
	return nil, false
}

// violationPattern maps one jsonschema crate message shape to a constraint.
// The crate's Display output never includes the instance path, so patterns
// either capture the property name directly (field) or the offending value
// (instance), which is then matched back to an attribute.
type violationPattern struct {
	constraint string
	re         *regexp.Regexp
	field      int
	instance   int
	expected   int
}

// Ordered longest-match-first: "is not of types" must be tried before "is not
// of type", which would otherwise match it and capture nonsense.
var violationPatterns = []violationPattern{
	{constraint: "required", re: regexp.MustCompile(`^(.+) is a required property$`), field: 1},
	{constraint: "additionalProperties", re: regexp.MustCompile(`^Additional properties are not allowed \((.+) (?:was|were) unexpected\)$`), field: 1},
	{constraint: "unevaluatedProperties", re: regexp.MustCompile(`^Unevaluated properties are not allowed \((.+) (?:was|were) unexpected\)$`), field: 1},
	{constraint: jsonSchemaTypeKey, re: regexp.MustCompile(`^(.+) is not of types (.+)$`), instance: 1, expected: 2},
	{constraint: jsonSchemaTypeKey, re: regexp.MustCompile(`^(.+) is not of type (.+)$`), instance: 1, expected: 2},
	{constraint: "exclusiveMaximum", re: regexp.MustCompile(`^(.+) is greater than or equal to the maximum of (.+)$`), instance: 1, expected: 2},
	{constraint: "exclusiveMinimum", re: regexp.MustCompile(`^(.+) is less than or equal to the minimum of (.+)$`), instance: 1, expected: 2},
	{constraint: "maximum", re: regexp.MustCompile(`^(.+) is greater than the maximum of (.+)$`), instance: 1, expected: 2},
	{constraint: "minimum", re: regexp.MustCompile(`^(.+) is less than the minimum of (.+)$`), instance: 1, expected: 2},
	{constraint: "maxLength", re: regexp.MustCompile(`^(.+) is longer than (\d+) characters?$`), instance: 1, expected: 2},
	{constraint: "minLength", re: regexp.MustCompile(`^(.+) is shorter than (\d+) characters?$`), instance: 1, expected: 2},
	{constraint: "maxItems", re: regexp.MustCompile(`^(.+) has more than (\d+) items?$`), instance: 1, expected: 2},
	{constraint: "minItems", re: regexp.MustCompile(`^(.+) has less than (\d+) items?$`), instance: 1, expected: 2},
	{constraint: "enum", re: regexp.MustCompile(`^(.+) is not one of (.+)$`), instance: 1, expected: 2},
	{constraint: "pattern", re: regexp.MustCompile(`^(.+) does not match (".+")$`), instance: 1, expected: 2},
	{constraint: "multipleOf", re: regexp.MustCompile(`^(.+) is not a multiple of (.+)$`), instance: 1, expected: 2},
	{constraint: "uniqueItems", re: regexp.MustCompile(`^(.+) has non-unique elements$`), instance: 1},
}

// translateSchemaError turns Atom's schema rejection into a
// SchemaValidationError naming the offending fields. Anything else is returned
// unchanged, so no other error path behaves differently.
func translateSchemaError(err error, attributes Attributes) error {
	if err == nil {
		return nil
	}
	var atomErr Error
	if !errors.As(err, &atomErr) {
		return err
	}
	index := strings.Index(atomErr.Message, atomSchemaValidationPrefix)
	if index < 0 {
		return err
	}

	detail := atomErr.Message[index+len(atomSchemaValidationPrefix):]
	// The validator's own messages never contain "; " — lists inside them are
	// comma separated — so this splits exactly on Atom's join.
	messages := strings.Split(detail, "; ")

	violations := make([]SchemaViolation, 0, len(messages))
	for _, message := range messages {
		message = strings.TrimSpace(message)
		if message == "" {
			continue
		}
		violations = append(violations, parseViolations(message, attributes)...)
	}
	if len(violations) == 0 {
		return err
	}

	return &SchemaValidationError{Violations: violations, err: err}
}

func parseViolations(message string, attributes Attributes) []SchemaViolation {
	for _, pattern := range violationPatterns {
		match := pattern.re.FindStringSubmatch(message)
		if match == nil {
			continue
		}

		expected := ""
		if pattern.expected > 0 {
			expected = unquote(match[pattern.expected])
		}

		// One message can name several unexpected properties; each is its own
		// violation so a caller can report them per field.
		if pattern.field > 0 {
			names := strings.Split(match[pattern.field], ", ")
			violations := make([]SchemaViolation, 0, len(names))
			for _, name := range names {
				violations = append(violations, SchemaViolation{
					Field:      unquote(name),
					Constraint: pattern.constraint,
					Expected:   expected,
					Message:    message,
				})
			}
			return violations
		}

		violation := SchemaViolation{Constraint: pattern.constraint, Expected: expected, Message: message}
		if pattern.instance > 0 {
			violation.Field = fieldForInstance(attributes, match[pattern.instance])
		}
		return []SchemaViolation{violation}
	}

	return []SchemaViolation{{Message: message}}
}

// fieldForInstance finds which submitted attribute held the value the
// validator rejected. It answers only when exactly one attribute matches:
// naming the wrong field is worse than naming none.
func fieldForInstance(attributes Attributes, instance string) string {
	if len(attributes) == 0 {
		return ""
	}
	wantNumber, wantIsNumber := parseFloat(instance)

	var field string
	matches := 0
	for name, value := range attributes {
		encoded, err := json.Marshal(value)
		if err != nil {
			continue
		}
		if string(encoded) != instance {
			// Rust renders a whole-numbered float as "5.0" where Go encodes it
			// as "5", so numbers are also compared by value.
			gotNumber, gotIsNumber := parseFloat(string(encoded))
			if !wantIsNumber || !gotIsNumber || gotNumber != wantNumber {
				continue
			}
		}
		field = name
		matches++
	}
	if matches == 1 {
		return field
	}
	return ""
}

func parseFloat(s string) (float64, bool) {
	f, err := strconv.ParseFloat(s, 64)
	return f, err == nil
}

// unquote strips the quoting the validator puts around a single token. A
// capture holding several quoted tokens — a list of accepted types, say — is
// left exactly as it came, since it is not one name.
func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) < 2 {
		return s
	}
	switch {
	case s[0] == '\'' && s[len(s)-1] == '\'':
		return s[1 : len(s)-1]
	case s[0] == '"' && s[len(s)-1] == '"':
		if unquoted, err := strconv.Unquote(s); err == nil {
			return unquoted
		}
	}
	return s
}
