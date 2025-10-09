// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package reports

import (
	"encoding/json"
	"fmt"
	"text/template"
	"text/template/parse"
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
	templateStr := string(temp)

	// Validate template syntax using Go's template parser
	tmpl := template.New("validate").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"div": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a / b
		},
		"mod": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a % b
		},
		"eq":          func(a, b int) bool { return a == b },
		"ge":          func(a, b int) bool { return a >= b },
		"lt":          func(a, b int) bool { return a < b },
		"iterate":     func(count int) []int { return make([]int, count) },
		"getStartRow": func(pageNum, firstPageRows, continuationPageRows int) int { return 0 },
		"getEndRow":   func(pageNum, firstPageRows, continuationPageRows, totalMessages int) int { return 0 },
		"formatTime":  func(t any) string { return "" },
		"formatValue": func(v any) string { return "" },
	})

	parsed, err := tmpl.Parse(templateStr)
	if err != nil {
		return fmt.Errorf("template syntax error: %w", err)
	}

	var hasTitle, hasRange, hasFormatTime, hasFormatValue, hasEnd bool
	// Validate essential fields are present using template parsing
	if err := validateEssentialFields(parsed.Tree.Root, &hasTitle, &hasRange, &hasFormatTime, &hasFormatValue, &hasEnd); err != nil {
		return err
	}

	if !hasTitle {
		return fmt.Errorf("missing essential template field: {{$.Title}}")
	}
	if !hasRange {
		return fmt.Errorf("missing essential template field: {{range .Messages}} or {{range .Reports}}")
	}
	if !hasFormatTime {
		return fmt.Errorf("missing essential template field: {{formatTime .Time}}")
	}
	if !hasFormatValue {
		return fmt.Errorf("missing essential template field: {{formatValue .}}")
	}
	if !hasEnd {
		return fmt.Errorf("missing essential template field: {{end}}")
	}

	return nil
}

func validateEssentialFields(node parse.Node, hasTitle, hasRange, hasFormatTime, hasFormatValue, hasEnd *bool) error {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *parse.ListNode:
		for _, sub := range n.Nodes {
			if err := validateEssentialFields(sub, hasTitle, hasRange, hasFormatTime, hasFormatValue, hasEnd); err != nil {
				return err
			}
		}

	case *parse.ActionNode:
		if n.Pipe != nil {
			for _, cmd := range n.Pipe.Cmds {
				cmdStr := cmd.String()
				if cmdStr == "$.Title" {
					*hasTitle = true
				}
				if len(cmd.Args) > 0 {
					firstArg := cmd.Args[0].String()
					if firstArg == "formatTime" {
						*hasFormatTime = true
					}
					if firstArg == "formatValue" {
						*hasFormatValue = true
					}
				}
			}
		}

	case *parse.RangeNode:
		if n.Pipe != nil && len(n.Pipe.Cmds) > 0 {
			cmdStr := n.Pipe.Cmds[0].String()
			// Accept .Messages, .Reports, or $report.Messages
			if cmdStr == ".Messages" || cmdStr == ".Reports" || cmdStr == "$report.Messages" {
				*hasRange = true
			}
		}
		if err := validateEssentialFields(n.List, hasTitle, hasRange, hasFormatTime, hasFormatValue, hasEnd); err != nil {
			return err
		}
		if n.ElseList != nil {
			if err := validateEssentialFields(n.ElseList, hasTitle, hasRange, hasFormatTime, hasFormatValue, hasEnd); err != nil {
				return err
			}
		}
		*hasEnd = true

	case *parse.IfNode:
		if err := validateEssentialFields(n.List, hasTitle, hasRange, hasFormatTime, hasFormatValue, hasEnd); err != nil {
			return err
		}
		if n.ElseList != nil {
			if err := validateEssentialFields(n.ElseList, hasTitle, hasRange, hasFormatTime, hasFormatValue, hasEnd); err != nil {
				return err
			}
		}

	case *parse.WithNode:
		if err := validateEssentialFields(n.List, hasTitle, hasRange, hasFormatTime, hasFormatValue, hasEnd); err != nil {
			return err
		}
		if n.ElseList != nil {
			if err := validateEssentialFields(n.ElseList, hasTitle, hasRange, hasFormatTime, hasFormatValue, hasEnd); err != nil {
				return err
			}
		}
	}

	return nil
}
