// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package reports_test

import (
	"fmt"
	"testing"

	"github.com/absmach/magistrala/reports"
	"github.com/stretchr/testify/assert"
)

const (
	validTemplate = `<!DOCTYPE html>
<html>
<head>
    <title>{{$.Title}}</title>
    <style>
        body { font-family: Arial, sans-serif; }
        .header { background-color: #f0f0f0; padding: 10px; }
        .content { padding: 20px; }
    </style>
</head>
<body>
    <div class="header">
        <h1>{{$.Title}}</h1>
        <p>Generated on: {{$.GeneratedDate}}</p>
    </div>
    <div class="content">
        <h2>Messages</h2>
        {{range .Messages}}
        <div class="message">
            <p>Time: {{formatTime .Time}}</p>
            <p>Value: {{formatValue .}}</p>
        </div>
        {{end}}
    </div>
</body>
</html>`

	templateWithoutTitle = `<!DOCTYPE html>
<html>
<head>
    <title>Report</title>
    <style>
        body { font-family: Arial, sans-serif; }
    </style>
</head>
<body>
    <h1>Report</h1>
    {{range .Messages}}
    <p>Time: {{formatTime .Time}}</p>
    <p>Value: {{formatValue .}}</p>
    {{end}}
</body>
</html>`

	templateWithoutRange = `<!DOCTYPE html>
<html>
<head>
    <title>{{$.Title}}</title>
</head>
<body>
    <h1>{{$.Title}}</h1>
    <p>No messages to display</p>
</body>
</html>`

	templateWithoutFormatTime = `<!DOCTYPE html>
<html>
<head>
    <title>{{$.Title}}</title>
</head>
<body>
    <h1>{{$.Title}}</h1>
    {{range .Messages}}
    <p>Time: {{.Time}}</p>
    <p>Value: {{formatValue .}}</p>
    {{end}}
</body>
</html>`

	templateWithoutFormatValue = `<!DOCTYPE html>
<html>
<head>
    <title>{{$.Title}}</title>
</head>
<body>
    <h1>{{$.Title}}</h1>
    {{range .Messages}}
    <p>Time: {{formatTime .Time}}</p>
    <p>Value: {{.}}</p>
    {{end}}
</body>
</html>`

	templateWithoutEnd = `<!DOCTYPE html>
<html>
<head>
    <title>{{$.Title}}</title>
</head>
<body>
    <h1>{{$.Title}}</h1>
    <p>Time: {{formatTime "test"}}</p>
    <p>Value: {{formatValue "test"}}</p>
    <p>No range block with end</p>
</body>
</html>`

	templateWithSyntaxError = `<!DOCTYPE html>
<html>
<head>
    <title>{{$.Title}}</title>
</head>
<body>
    <h1>{{$.Title}}</h1>
    {{range .Messages}}
    <p>Time: {{formatTime .Time}}</p>
    <p>Value: {{formatValue .}}</p>
    {{end
</body>
</html>`

	templateWithUndefinedFunction = `<!DOCTYPE html>
<html>
<head>
    <title>{{$.Title}}</title>
</head>
<body>
    <h1>{{$.Title}}</h1>
    {{range .Messages}}
    <p>Time: {{formatTime .Time}}</p>
    <p>Value: {{formatValue .}}</p>
    <p>Custom: {{customFunction .}}</p>
    {{end}}
</body>
</html>`

	templateWithIfCondition = `<!DOCTYPE html>
<html>
<head>
    <title>{{$.Title}}</title>
</head>
<body>
    <h1>{{$.Title}}</h1>
    {{if .Messages}}
    {{range .Messages}}
    <p>Time: {{formatTime .Time}}</p>
    <p>Value: {{formatValue .}}</p>
    {{end}}
    {{else}}
    <p>No messages available</p>
    {{end}}
</body>
</html>`

	templateWithWithCondition = `<!DOCTYPE html>
<html>
<head>
    <title>{{$.Title}}</title>
</head>
<body>
    <h1>{{$.Title}}</h1>
    {{with .Data}}
    {{range .Messages}}
    <p>Time: {{formatTime .Time}}</p>
    <p>Value: {{formatValue .}}</p>
    {{end}}
    {{else}}
    <p>No data available</p>
    {{end}}
</body>
</html>`

	templateWithNestedConditions = `<!DOCTYPE html>
<html>
<head>
    <title>{{$.Title}}</title>
</head>
<body>
    <h1>{{$.Title}}</h1>
    {{if .HasMessages}}
        {{with .Data}}
            {{range .Messages}}
            <p>Time: {{formatTime .Time}}</p>
            <p>Value: {{formatValue .}}</p>
            {{end}}
        {{else}}
            <p>Data not available</p>
        {{end}}
    {{else}}
        <p>No messages flag set</p>
    {{end}}
</body>
</html>`

	templateWithIfMissingFields = `<!DOCTYPE html>
<html>
<head>
    <title>{{$.Title}}</title>
</head>
<body>
    <h1>{{$.Title}}</h1>
    {{if .Messages}}
    {{range .Messages}}
    <p>Time: {{.Time}}</p>
    <p>Value: {{.}}</p>
    {{end}}
    {{else}}
    <p>No messages available</p>
    {{end}}
</body>
</html>`

	templateWithWithMissingFields = `<!DOCTYPE html>
<html>
<head>
    <title>{{$.Title}}</title>
</head>
<body>
    <h1>{{$.Title}}</h1>
    {{with .Data}}
    {{range .Messages}}
    <p>Time: {{.Time}}</p>
    <p>Value: {{formatValue .}}</p>
    {{end}}
    {{else}}
    <p>No data available</p>
    {{end}}
</body>
</html>`
)

func TestReportTemplate_Validate(t *testing.T) {
	cases := []struct {
		desc     string
		template reports.ReportTemplate
		err      error
	}{
		{
			desc:     "validate template successfully",
			template: reports.ReportTemplate(validTemplate),
			err:      nil,
		},
		{
			desc:     "validate template without title field",
			template: reports.ReportTemplate(templateWithoutTitle),
			err:      fmt.Errorf("missing essential template field: {{$.Title}}"),
		},
		{
			desc:     "validate template without range field",
			template: reports.ReportTemplate(templateWithoutRange),
			err:      fmt.Errorf("missing essential template field: {{range .Messages}}"),
		},
		{
			desc:     "validate template without formatTime field",
			template: reports.ReportTemplate(templateWithoutFormatTime),
			err:      fmt.Errorf("missing essential template field: {{formatTime .Time}}"),
		},
		{
			desc:     "validate template without formatValue field",
			template: reports.ReportTemplate(templateWithoutFormatValue),
			err:      fmt.Errorf("missing essential template field: {{formatValue .}}"),
		},
		{
			desc:     "validate template without end field",
			template: reports.ReportTemplate(templateWithoutEnd),
			err:      fmt.Errorf("missing essential template field: {{range .Messages}}"),
		},
		{
			desc:     "validate template with syntax error",
			template: reports.ReportTemplate(templateWithSyntaxError),
			err:      fmt.Errorf("template syntax error"),
		},
		{
			desc:     "validate template with undefined function",
			template: reports.ReportTemplate(templateWithUndefinedFunction),
			err:      fmt.Errorf("template syntax error"),
		},
		{
			desc:     "validate empty template",
			template: reports.ReportTemplate(""),
			err:      fmt.Errorf("missing essential template field: {{$.Title}}"),
		},
		{
			desc:     "validate template with if condition successfully",
			template: reports.ReportTemplate(templateWithIfCondition),
			err:      nil,
		},
		{
			desc:     "validate template `with` with condition successfully",
			template: reports.ReportTemplate(templateWithWithCondition),
			err:      nil,
		},
		{
			desc:     "validate template with nested conditions successfully",
			template: reports.ReportTemplate(templateWithNestedConditions),
			err:      nil,
		},
		{
			desc:     "validate template with if condition missing formatTime",
			template: reports.ReportTemplate(templateWithIfMissingFields),
			err:      fmt.Errorf("missing essential template field: {{formatTime .Time}}"),
		},
		{
			desc:     "validate template `with` with condition missing formatTime",
			template: reports.ReportTemplate(templateWithWithMissingFields),
			err:      fmt.Errorf("missing essential template field: {{formatTime .Time}}"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.template.Validate()
			if tc.err != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestReportTemplate_String(t *testing.T) {
	template := reports.ReportTemplate(validTemplate)
	result := template.String()

	assert.Equal(t, validTemplate, result)
}

func TestReportTemplate_MarshalJSON(t *testing.T) {
	template := reports.ReportTemplate("simple template")
	data, err := template.MarshalJSON()

	assert.NoError(t, err)
	assert.NotNil(t, data)
	assert.Equal(t, `"simple template"`, string(data))
}

func TestReportTemplate_UnmarshalJSON(t *testing.T) {
	cases := []struct {
		desc     string
		data     []byte
		expected string
		err      error
	}{
		{
			desc:     "unmarshal valid JSON successfully",
			data:     []byte(`"simple template"`),
			expected: "simple template",
			err:      nil,
		},
		{
			desc: "unmarshal invalid JSON",
			data: []byte(`invalid json`),
			err:  fmt.Errorf("invalid character"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			var template reports.ReportTemplate
			err := template.UnmarshalJSON(tc.data)

			if tc.err != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, string(template))
			}
		})
	}
}
