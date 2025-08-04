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
		"add":         func(a, b int) int { return a + b },
		"formatTime":  func(t interface{}) string { return "" },
		"formatValue": func(v interface{}) string { return "" },
	})

	parsedTmpl, err := tmpl.Parse(templateStr)
	if err != nil {
		return fmt.Errorf("template syntax error: %w", err)
	}

	// Validate essential fields are present using template parsing
	if err := validateEssentialFields(parsedTmpl.Tree.Root); err != nil {
		return err
	}

	return nil
}

func validateEssentialFields(node parse.Node) error {
	var hasTitle, hasRange, hasFormatTime, hasFormatValue, hasEnd bool

	var traverse func(parse.Node)
	traverse = func(n parse.Node) {
		if n == nil {
			return
		}

		switch node := n.(type) {
		case *parse.ListNode:
			for _, sub := range node.Nodes {
				traverse(sub)
			}

		case *parse.ActionNode:
			if node.Pipe != nil {
				for _, cmd := range node.Pipe.Cmds {
					cmdStr := cmd.String()
					if cmdStr == "$.Title" {
						hasTitle = true
					}
					if len(cmd.Args) > 0 {
						firstArg := cmd.Args[0].String()
						if firstArg == "formatTime" {
							hasFormatTime = true
						}
						if firstArg == "formatValue" {
							hasFormatValue = true
						}
					}
				}
			}

		case *parse.RangeNode:
			if node.Pipe != nil && len(node.Pipe.Cmds) > 0 {
				cmdStr := node.Pipe.Cmds[0].String()
				if cmdStr == ".Messages" {
					hasRange = true
				}
			}
			traverse(node.List)
			if node.ElseList != nil {
				traverse(node.ElseList)
			}
			hasEnd = true

		case *parse.IfNode:
			traverse(node.List)
			if node.ElseList != nil {
				traverse(node.ElseList)
			}

		case *parse.WithNode:
			traverse(node.List)
			if node.ElseList != nil {
				traverse(node.ElseList)
			}
		}
	}

	traverse(node)

	if !hasTitle {
		return fmt.Errorf("missing essential template field: {{$.Title}}")
	}
	if !hasRange {
		return fmt.Errorf("missing essential template field: {{range .Messages}}")
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
