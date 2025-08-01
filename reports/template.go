// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package reports

import (
	"encoding/json"
	"fmt"
	"text/template"
	"text/template/parse"

	"github.com/absmach/supermq/pkg/errors"
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
	tmpl := template.New("validate").Funcs(template.FuncMap{
		"add":         func(a, b int) int { return a + b },
		"formatTime":  func(t interface{}) string { return "" },
		"formatValue": func(v interface{}) string { return "" },
	})

	parsedTmpl, err := tmpl.Parse(string(temp))
	if err != nil {
		return fmt.Errorf("template parsing error: %w", err)
	}

	if err := walkNode(parsedTmpl.Tree.Root); err != nil {
		return err
	}

	return nil
}

func walkNode(node parse.Node) error {
	switch n := node.(type) {

	case *parse.ListNode:
		for _, sub := range n.Nodes {
			if err := walkNode(sub); err != nil {
				return err
			}
		}

	case *parse.ActionNode:
		fmt.Printf("Action: %s\n", n.String())
		if len(n.Pipe.Cmds) == 0 {
			return fmt.Errorf("empty action command: %s", n.String())
		}

	case *parse.IfNode:
		fmt.Printf("If block: %s\n", n.String())
		if err := walkNode(n.List); err != nil {
			return err
		}
		if n.ElseList != nil {
			if err := walkNode(n.ElseList); err != nil {
				return err
			}
		}

	case *parse.RangeNode:
		if n.Pipe != nil && len(n.Pipe.Cmds) == 1 {
			cmdStr := n.Pipe.Cmds[0].String()
			if cmdStr != ".Reports" && cmdStr != ".Messages" {
				return errors.New("range allowed only on Reports or Messages")
			}
		}
		if err := walkNode(n.List); err != nil {
			return err
		}
		if n.ElseList != nil {
			if err := walkNode(n.ElseList); err != nil {
				return err
			}
		}

	case *parse.TemplateNode:
		fmt.Printf("Template call: %s\n", n.String())

	case *parse.TextNode:
		fmt.Printf("Text: %q\n", string(n.Text))

	case *parse.WithNode:
		fmt.Printf("With block: %s\n", n.String())
		if err := walkNode(n.List); err != nil {
			return err
		}
		if n.ElseList != nil {
			if err := walkNode(n.ElseList); err != nil {
				return err
			}
		}

	case *parse.VariableNode:
		fmt.Printf("Variable: %s\n", n.Ident)

	case *parse.CommandNode:
		fmt.Printf("Command: %s\n", n.String())

	default:
		return fmt.Errorf("unknown node type: %T", n)
	}

	return nil
}
