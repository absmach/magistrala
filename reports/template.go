// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package reports

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var (
	essentialFields = []string{
		"{{$.Title}}",
		"{{range .Messages}}",
		"{{formatTime .Time}}",
		"{{formatValue .}}",
		"{{end}}",
	}

	recommendedFields = []string{
		"{{$.GeneratedDate}}",
		"{{$.GeneratedTime}}",
		"{{.Metric.Name}}",
		"{{.Metric.ChannelID}}",
		"{{len .Messages}}",
	}

	conditionalFields = []string{
		"{{.Metric.ClientID}}",
		"{{.Unit}}",
		"{{.Protocol}}",
		"{{.Subtopic}}",
	}

	requiredStructure = []string{
		"<!DOCTYPE html>",
		"<html",
		"<head>",
		"<body>",
		"<style>",
		"</style>",
		"</head>",
		"</body>",
		"</html>",
	}

	essentialCSS = []string{
		".page",
		".data-table",
	}

	recommendedCSS = []string{
		".header",
		".content-area",
		".metrics-section",
		".footer",
	}
)

type ReportTemplate string

func (temp ReportTemplate) String() string {
	return string(temp)
}

func (temp ReportTemplate) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(temp))
}

func (temp *ReportTemplate) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	*temp = ReportTemplate(s)
	return nil
}

func (temp ReportTemplate) Validate() error {
	template := string(temp)

	for _, required := range requiredStructure {
		if !strings.Contains(template, required) {
			return fmt.Errorf("missing required HTML element: %s", required)
		}
	}

	cleaned := strings.TrimSpace(template)
	commentPattern := regexp.MustCompile(`^(?s:<!--.*?-->\s*)*`)
	cleaned = commentPattern.ReplaceAllString(cleaned, "")

	if !strings.HasPrefix(cleaned, "<!DOCTYPE html>") {
		return fmt.Errorf("template must start with <!DOCTYPE html>")
	}

	for _, field := range essentialFields {
		if !strings.Contains(template, field) {
			return fmt.Errorf("missing essential template field: %s", field)
		}
	}

	blockStartPattern := regexp.MustCompile(`\{\{\s*(range|if|with)\b[^{}]*\}\}`)
	blockEndPattern := regexp.MustCompile(`\{\{\s*end\s*\}\}`)

	blockStarts := blockStartPattern.FindAllString(template, -1)
	blockEnds := blockEndPattern.FindAllString(template, -1)

	if len(blockStarts) != len(blockEnds) {
		return fmt.Errorf("unmatched template blocks: found %d block start(s) (range/if/with) and %d end(s)",
			len(blockStarts), len(blockEnds))
	}

	for _, class := range essentialCSS {
		pattern := fmt.Sprintf(`\.%s\s*\{`, strings.TrimPrefix(class, "."))
		matched, _ := regexp.MatchString(pattern, template)
		if !matched {
			return fmt.Errorf("missing essential CSS class: %s", class)
		}
	}

	if strings.Contains(template, ".data-table") {
		requiredTableElements := []string{
			"<table",
			"<thead>",
			"<tbody>",
			"<th",
			"<td",
		}

		for _, element := range requiredTableElements {
			if !strings.Contains(template, element) {
				return fmt.Errorf("missing required table element: %s", element)
			}
		}
	}

	return nil
}

func (temp ReportTemplate) ValidateWithRecommendations() ([]string, error) {
	if err := temp.Validate(); err != nil {
		return nil, err
	}

	var warnings []string
	template := string(temp)

	for _, field := range recommendedFields {
		if !strings.Contains(template, field) {
			warnings = append(warnings, fmt.Sprintf("recommended field missing: %s", field))
		}
	}

	for _, class := range recommendedCSS {
		pattern := fmt.Sprintf(`\.%s\s*\{`, strings.TrimPrefix(class, "."))
		matched, _ := regexp.MatchString(pattern, template)
		if !matched {
			warnings = append(warnings, fmt.Sprintf("recommended CSS class missing: %s", class))
		}
	}

	return warnings, nil
}
