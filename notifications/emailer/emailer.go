// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package emailer

import (
	"context"
	"fmt"
	"unicode"
	"unicode/utf8"

	grpcUsersV1 "github.com/absmach/magistrala/api/grpc/users/v1"
	"github.com/absmach/magistrala/internal/email"
	"github.com/absmach/magistrala/notifications"
	"github.com/absmach/magistrala/pkg/errors"
)

var (
	errFetchingUser = errors.New("failed to fetch user information")
	errSendingEmail = errors.New("failed to send email")
)

const (
	inviterRecipient = "inviter"
	inviteeRecipient = "invitee"
)

var _ notifications.Notifier = (*notifier)(nil)

type notifier struct {
	usersClient   grpcUsersV1.UsersServiceClient
	agents        map[notifications.NotificationType]*email.Agent
	fromName      string
	domainAltName string
}

// Config represents the emailer configuration.
type Config struct {
	FromAddress        string
	FromName           string
	DomainAltName      string
	InvitationTemplate string
	AcceptanceTemplate string
	RejectionTemplate  string
	EmailHost          string
	EmailPort          string
	EmailUsername      string
	EmailPassword      string
}

// New creates a new email notifier.
func New(usersClient grpcUsersV1.UsersServiceClient, cfg Config) (notifications.Notifier, error) {
	templates := map[notifications.NotificationType]string{
		notifications.Invitation: cfg.InvitationTemplate,
		notifications.Acceptance: cfg.AcceptanceTemplate,
		notifications.Rejection:  cfg.RejectionTemplate,
	}

	agents := make(map[notifications.NotificationType]*email.Agent)
	for notifType, template := range templates {
		emailCfg := &email.Config{
			Host:        cfg.EmailHost,
			Port:        cfg.EmailPort,
			Username:    cfg.EmailUsername,
			Password:    cfg.EmailPassword,
			FromAddress: cfg.FromAddress,
			FromName:    cfg.FromName,
			Template:    template,
		}
		agent, err := email.New(emailCfg)
		if err != nil {
			return nil, err
		}
		agents[notifType] = agent
	}

	return &notifier{
		usersClient:   usersClient,
		agents:        agents,
		fromName:      cfg.FromName,
		domainAltName: cfg.DomainAltName,
	}, nil
}

func (n *notifier) Notify(ctx context.Context, notif notifications.Notification) error {
	users, err := n.fetchUsers(ctx, []string{notif.InviterID, notif.InviteeID})
	if err != nil {
		return errors.Wrap(errFetchingUser, err)
	}

	inviter, ok := users[notif.InviterID]
	if !ok {
		return errors.Wrap(errFetchingUser, fmt.Errorf("inviter not found: %s", notif.InviterID))
	}

	invitee, ok := users[notif.InviteeID]
	if !ok {
		return errors.Wrap(errFetchingUser, fmt.Errorf("invitee not found: %s", notif.InviteeID))
	}

	inviterName := n.userDisplayName(inviter)
	inviteeName := n.userDisplayName(invitee)

	domainName := notif.DomainName
	if domainName == "" {
		domainName = notif.DomainID
	}

	roleName := notif.RoleName
	if roleName == "" {
		roleName = notif.RoleID
	}

	subject, content, recipient, err := n.buildEmailContent(notif.Type, inviterName, inviteeName, domainName, roleName)
	if err != nil {
		return err
	}
	recipientEmail := inviter.Email
	recipientName := inviterName
	if recipient == inviteeRecipient {
		recipientEmail = invitee.Email
		recipientName = inviteeName
	}

	agent, ok := n.agents[notif.Type]
	if !ok || agent == nil {
		return errors.Wrap(errSendingEmail, fmt.Errorf("no email agent configured for notification type: %d", notif.Type))
	}

	if err := agent.Send([]string{recipientEmail}, "", subject, "", recipientName, content, n.fromName, nil); err != nil {
		return errors.Wrap(errSendingEmail, err)
	}

	return nil
}

func (n *notifier) buildEmailContent(notifType notifications.NotificationType, inviterName, inviteeName, domainName, roleName string) (subject, content, recipient string, err error) {
	switch notifType {
	case notifications.Invitation:
		return fmt.Sprintf("%s Invitation", titleFirst(n.domainAltName)),
			fmt.Sprintf("%s has invited you to join the %s %s as %s.", n.domainAltName, inviterName, domainName, roleName),
			inviteeRecipient,
			nil
	case notifications.Acceptance:
		return "Invitation Accepted",
			fmt.Sprintf("%s has accepted your invitation to join the %s %s as %s.", n.domainAltName, inviteeName, domainName, roleName),
			inviterRecipient,
			nil
	case notifications.Rejection:
		return "Invitation Declined",
			fmt.Sprintf("%s has declined your invitation to join the %s %s as %s.", n.domainAltName, inviteeName, domainName, roleName),
			inviterRecipient,
			nil
	default:
		return "", "", "", errors.Wrap(errSendingEmail, fmt.Errorf("unsupported notification type: %d", notifType))
	}
}

func (n *notifier) fetchUsers(ctx context.Context, userIDs []string) (map[string]*grpcUsersV1.User, error) {
	req := &grpcUsersV1.RetrieveUsersReq{
		Ids:    userIDs,
		Limit:  uint64(len(userIDs)),
		Offset: 0,
	}

	res, err := n.usersClient.RetrieveUsers(ctx, req)
	if err != nil {
		return nil, err
	}

	users := make(map[string]*grpcUsersV1.User)
	for _, user := range res.Users {
		users[user.Id] = user
	}

	return users, nil
}

func (n *notifier) userDisplayName(user *grpcUsersV1.User) string {
	if user.FirstName != "" && user.LastName != "" {
		return fmt.Sprintf("%s %s", user.FirstName, user.LastName)
	}
	if user.FirstName != "" {
		return user.FirstName
	}
	if user.Username != "" {
		return user.Username
	}
	if user.Email != "" {
		return user.Email
	}
	return user.Id
}

func titleFirst(s string) string {
	if s == "" {
		return s
	}
	r, size := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[size:]
}
