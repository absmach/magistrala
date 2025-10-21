// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	pkglog "github.com/absmach/magistrala/pkg/logger"
	"github.com/absmach/magistrala/re/outputs"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/absmach/supermq/pkg/messaging"
)

var (
	scheduledTrue  = true
	scheduledFalse = false
)

const (
	maxPayload     = 100 * 1024
	pldExceededFmt = "max payload size of 100kB exceeded: "
	protocol       = "nats"
)

func (re *re) Handle(msg *messaging.Message) error {
	// Limit payload for RE so we don't get to process large JSON.
	if n := len(msg.Payload); n > maxPayload {
		return errors.New(pldExceededFmt + strconv.Itoa(n))
	}
	// Skip filtering by message topic and fetch all non-scheduled rules instead.
	// It's cleaner and more efficient to match wildcards in Go, but we can
	// revisit this if it ever becomes a performance bottleneck.
	pm := PageMeta{
		Domain:       msg.Domain,
		InputChannel: msg.Channel,
		Status:       EnabledStatus,
		Scheduled:    &scheduledFalse,
	}
	ctx := context.Background()
	page, err := re.repo.ListRules(ctx, pm)
	if err != nil {
		return err
	}

	for _, r := range page.Rules {
		if matchSubject(msg.Subtopic, r.InputTopic) {
			go func(ctx context.Context, rule Rule) {
				info := re.process(ctx, rule, msg)
				re.runInfo <- info
				re.saveLog(ctx, rule, info)
			}(ctx, r)
		}
	}

	return nil
}

// Match NATS subject to support wildcards.
func matchSubject(published, subscribed string) bool {
	p := strings.Split(published, ".")
	s := strings.Split(subscribed, ".")
	n := len(p)

	for i := range s {
		if s[i] == ">" {
			return true
		}
		if i >= n {
			return false
		}
		if s[i] != "*" && p[i] != s[i] {
			return false
		}
	}
	return len(s) == n
}

func (re *re) process(ctx context.Context, r Rule, msg *messaging.Message) pkglog.RunInfo {
	execTime := time.Now().UTC()
	details := []slog.Attr{
		slog.String("domain_id", r.DomainID),
		slog.String("rule_id", r.ID),
		slog.String("rule_name", r.Name),
	}
	var info pkglog.RunInfo
	switch r.Logic.Type {
	case GoType:
		info = re.processGo(ctx, details, r, msg)
	default:
		info = re.processLua(ctx, details, r, msg)
	}
	info.ExecTime = execTime
	return info
}

func (re *re) handleOutput(ctx context.Context, o Runnable, r Rule, msg *messaging.Message, val any) error {
	switch o := o.(type) {
	case *outputs.Alarm:
		o.AlarmsPub = re.alarmsPub
		o.RuleID = r.ID
		return o.Run(ctx, msg, val)
	case *outputs.Email:
		o.Emailer = re.email
		return o.Run(ctx, msg, val)
	case *outputs.ChannelPublisher:
		o.RePubSub = re.rePubSub
		return o.Run(ctx, msg, val)
	case *outputs.SenML:
		o.WritersPub = re.writersPub
		return o.Run(ctx, msg, val)
	case *outputs.Postgres, *outputs.Slack:
		return o.Run(ctx, msg, val)
	default:
		return fmt.Errorf("unknown output type: %T", o)
	}
}

func (re *re) saveLog(ctx context.Context, r Rule, info pkglog.RunInfo) {
	id, err := re.idp.ID()
	if err != nil {
		return
	}

	errMsg := ""
	if info.Error != nil {
		errMsg = info.Error.Error()
	}

	log := RuleLog{
		ID:        id,
		RuleID:    r.ID,
		Level:     info.Level.String(),
		Message:   info.Message,
		Error:     errMsg,
		ExecTime:  info.ExecTime,
		CreatedAt: time.Now().UTC(),
	}

	if err := re.repo.AddLog(ctx, log); err != nil {
		re.runInfo <- pkglog.RunInfo{
			Level:   slog.LevelError,
			Message: fmt.Sprintf("failed to persist log: %s", err),
			Details: []slog.Attr{slog.String("rule_id", r.ID)},
		}
	}
}

func (re *re) StartScheduler(ctx context.Context) error {
	defer re.ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-re.ticker.Tick():
			due := time.Now().UTC()
			pm := PageMeta{
				Status:          EnabledStatus,
				Scheduled:       &scheduledTrue,
				ScheduledBefore: &due,
			}

			page, err := re.repo.ListRules(ctx, pm)
			if err != nil {
				re.runInfo <- pkglog.RunInfo{
					Level:   slog.LevelError,
					Message: fmt.Sprintf("failed to list rules: %s", err),
					Details: []slog.Attr{slog.Time("due", due)},
				}

				continue
			}

			for _, r := range page.Rules {
				go func(rule Rule) {
					if _, err := re.repo.UpdateRuleDue(ctx, rule.ID, rule.Schedule.NextDue()); err != nil {
						re.runInfo <- pkglog.RunInfo{Level: slog.LevelError, Message: fmt.Sprintf("failed to update rule: %s", err), Details: []slog.Attr{slog.Time("time", time.Now().UTC())}}
						return
					}

					msg := &messaging.Message{
						Domain:   rule.DomainID,
						Channel:  rule.InputChannel,
						Subtopic: rule.InputTopic,
						Protocol: protocol,
						Created:  due.Unix(),
					}
					info := re.process(ctx, rule, msg)
					re.runInfo <- info
					re.saveLog(ctx, rule, info)
				}(r)
			}
			// Reset due, it will reset in the page meta as well.
			due = time.Now().UTC()
		}
	}
}
