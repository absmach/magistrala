// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package reports

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	requiredFields = []string{
		"{{.Title}}",
		"{{.GeneratedDate}}",
		"{{.GeneratedTime}}",
		"{{$.Title}}",
		"{{$.GeneratedDate}}",
		"{{$.GeneratedTime}}",
		"{{.Metric.Name}}",
		"{{.Metric.ClientID}}",
		"{{.Metric.ChannelID}}",
		"{{len .Messages}}",
		"{{range .Messages}}",
		"{{formatTime .Time}}",
		"{{formatValue .}}",
		"{{.Unit}}",
		"{{.Protocol}}",
		"{{.Subtopic}}",
		"{{end}}",
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

	requiredCSS = []string{
		".page",
		".header",
		".content-area",
		".metrics-section",
		".data-table",
		".footer",
	}
)

type ReportTemplate string

func (temp ReportTemplate) String() string {
    return string(temp)
}

func (temp ReportTemplate) Validate() error {
	template := string(temp)

	for _, required := range requiredStructure {
		if !strings.Contains(template, required) {
			return fmt.Errorf("missing required HTML element: %s", required)
		}
	}

	if !strings.HasPrefix(strings.TrimSpace(template), "<!DOCTYPE html>") {
		return fmt.Errorf("template must start with <!DOCTYPE html>")
	}

	for _, field := range requiredFields {
		if !strings.Contains(template, field) {
			return fmt.Errorf("missing required template field: %s", field)
		}
	}

	rangePattern := regexp.MustCompile(`\{\{range\s+\.\w+\}\}`)
	endPattern := regexp.MustCompile(`\{\{end\}\}`)

	rangeMatches := rangePattern.FindAllString(template, -1)
	endMatches := endPattern.FindAllString(template, -1)

	if len(rangeMatches) != len(endMatches) {
		return fmt.Errorf("unmatched {{range}} and {{end}} blocks")
	}

	for _, class := range requiredCSS {
		pattern := fmt.Sprintf(`\.%s\s*\{`, strings.TrimPrefix(class, "."))
		matched, _ := regexp.MatchString(pattern, template)
		if !matched {
			return fmt.Errorf("missing required CSS class: %s", class)
		}
	}

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

	expectedHeaders := []string{"Time", "Value", "Unit", "Protocol", "Subtopic"}
	for _, header := range expectedHeaders {
		if !strings.Contains(template, header) {
			return fmt.Errorf("missing expected table header: %s", header)
		}
	}

	requiredFunctions := []string{
		"formatTime",
		"formatValue",
	}

	for _, function := range requiredFunctions {
		pattern := fmt.Sprintf(`function\s+%s\s*\(`, function)
		matched, _ := regexp.MatchString(pattern, template)
		if !matched {
			return fmt.Errorf("missing required JavaScript function: %s", function)
		}
	}

	return nil
}
