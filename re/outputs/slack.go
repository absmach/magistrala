// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package outputs

import (
	"context"
	"encoding/json"

	"github.com/absmach/supermq/pkg/messaging"
	"github.com/slack-go/slack"
)

type Slack struct {
	Token     string `json:"token"`
	ChannelID string `json:"channel_id"`
}

func (s *Slack) Run(ctx context.Context, msg *messaging.Message, val any) error {
	slackClient := slack.New(s.Token)
	_, _, err := slackClient.PostMessage(s.ChannelID, slack.MsgOptionText(msg.String(), false))
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
	})
}
