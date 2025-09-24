// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package outputs

import (
	"bytes"
	"context"
	"encoding/json"
	"text/template"

	"github.com/absmach/supermq/pkg/messaging"
	"github.com/slack-go/slack"
)

type Slack struct {
	Token     string `json:"token"`
	ChannelID string `json:"channel_id"`
	Message   string `json:"message"`
}

func (s *Slack) Run(ctx context.Context, msg *messaging.Message, val any) error {
	templData := templateVal{
		Message: msg,
		Result:  val,
	}

	tmpl, err := template.New("slack").Parse(s.Message)
	if err != nil {
		return err
	}

	var output bytes.Buffer
	if err := tmpl.Execute(&output, templData); err != nil {
		return err
	}

	mapping := output.String()

	var message slack.Msg
	if err := json.Unmarshal([]byte(mapping), &message); err != nil {
		return err
	}

	slackClient := slack.New(s.Token)

	var opts []slack.MsgOption

	if message.Text != "" {
		opts = append(opts, slack.MsgOptionText(message.Text, false))
	}
	if len(message.Attachments) > 0 {
		opts = append(opts, slack.MsgOptionAttachments(message.Attachments...))
	}
	if len(message.Blocks.BlockSet) > 0 {
		opts = append(opts, slack.MsgOptionBlocks(message.Blocks.BlockSet...))
	}
	_, _, err = slackClient.PostMessage(s.ChannelID, opts...)
	if err != nil {
		return err
	}
	return nil
}

func (s *Slack) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"type":       SlackType.String(),
		"token":      s.Token,
		"channel_id": s.ChannelID,
		"message":    s.Message,
	})
}
