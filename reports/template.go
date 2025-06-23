// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package reports

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var (
	requiredFields = []string{
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

func (temp ReportTemplate) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(temp))
}

func (temp *ReportTemplate) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	if strings.HasSuffix(s, ".html") {
		content, err := os.ReadFile(s)
		if err != nil {
			return fmt.Errorf("failed to read template file: %w", err)
		}
		*temp = ReportTemplate(content)
	} else {
		*temp = ReportTemplate(s)
	}
	return nil
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

	blockStartPattern := regexp.MustCompile(`\{\{\s*(range|if|with)\b[^{}]*\}\}`)
	blockEndPattern := regexp.MustCompile(`\{\{\s*end\s*\}\}`)

	blockStarts := blockStartPattern.FindAllString(template, -1)
	blockEnds := blockEndPattern.FindAllString(template, -1)

	if len(blockStarts) != len(blockEnds) {
		return fmt.Errorf("unmatched template blocks: found %d block start(s) (range/if/with) and %d end(s)",
			len(blockStarts), len(blockEnds))
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

	return nil
}
