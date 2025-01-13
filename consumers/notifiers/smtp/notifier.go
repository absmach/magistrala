// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package smtp

import (
	"fmt"

	"github.com/absmach/magistrala/internal/email"
	"github.com/absmach/supermq/consumers"
	"github.com/absmach/supermq/pkg/messaging"
)

const (
	footer          = "Sent by SuperMQ SMTP Notification"
	contentTemplate = "A publisher with an id %s sent the message over %s with the following values \n %s"
)

var _ consumers.Notifier = (*notifier)(nil)

type notifier struct {
	agent *email.Agent
}

// New instantiates SMTP message notifier.
func New(agent *email.Agent) consumers.Notifier {
	return &notifier{agent: agent}
}

func (n *notifier) Notify(from string, to []string, msg *messaging.Message) error {
	subject := fmt.Sprintf(`Notification for Channel %s`, msg.GetChannel())
	if msg.GetSubtopic() != "" {
		subject = fmt.Sprintf("%s and subtopic %s", subject, msg.GetSubtopic())
	}

	values := string(msg.GetPayload())
	content := fmt.Sprintf(contentTemplate, msg.GetPublisher(), msg.GetProtocol(), values)

	return n.agent.Send(to, from, subject, "", "", content, footer)
}
