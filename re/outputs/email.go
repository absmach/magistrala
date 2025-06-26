// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package outputs

import (
	"bytes"
	"context"
	"encoding/json"
	"text/template"

	"github.com/absmach/magistrala/pkg/emailer"
	"github.com/absmach/supermq/pkg/messaging"
)

type Email struct {
	To      []string        `json:"to"`
	Subject string          `json:"subject"`
	Content string          `json:"content"`
	Emailer emailer.Emailer `json:"-"`
}

func (e *Email) Run(ctx context.Context, msg *messaging.Message, val interface{}) error {
	templData := templateVal{
		Message: msg,
		Result:  val,
	}

	tmpl, err := template.New("email").Parse(e.Content)
	if err != nil {
		return err
	}

	var output bytes.Buffer
	if err := tmpl.Execute(&output, templData); err != nil {
		return err
	}

	content := output.String()

	if err := e.Emailer.SendEmailNotification(e.To, "", e.Subject, "", "", content, "", make(map[string][]byte)); err != nil {
		return err
	}
	return nil
}

func (e *Email) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"type":    EmailType.String(),
		"to":      e.To,
		"subject": e.Subject,
		"content": e.Content,
	})
}
