// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"context"
	"strconv"
	"time"

	"github.com/absmach/supermq/pkg/errors"
	"github.com/absmach/supermq/pkg/messaging"
	lua "github.com/yuin/gopher-lua"
)

const (
	maxPayload     = 100 * 1024
	pldExceededFmt = "max payload size of 100kB exceeded: "
)

func (re *re) Handle(msg *messaging.Message) error {
	// Limit payload for RE so we don't get to process large JSON.
	if n := len(msg.Payload); n > maxPayload {
		return errors.New(pldExceededFmt + strconv.Itoa(n))
	}
	inputChannel := msg.Channel
	pm := PageMeta{
		InputChannel: inputChannel,
		Status:       EnabledStatus,
		InputTopic:   &msg.Subtopic,
	}
	ctx := context.Background()
	page, err := re.repo.ListRules(ctx, pm)
	if err != nil {
		return err
	}

	for _, r := range page.Rules {
		go func(ctx context.Context) {
			if err := re.process(ctx, r, msg); err != nil {
				re.errors <- err
			}
		}(ctx)
	}
	return nil
}

func (re *re) process(ctx context.Context, r Rule, msg *messaging.Message) error {
	l := lua.NewState()
	defer l.Close()
	preload(l)
	message := prepareMsg(l, msg)

	// Set the message object as a Lua global variable.
	l.SetGlobal("message", message)

	// set the email function as a Lua global function.
	l.SetGlobal("send_email", l.NewFunction(re.sendEmail))
	l.SetGlobal("send_alarm", l.NewFunction(re.sendAlarm(ctx, r.ID, msg)))

	if err := l.DoString(r.Logic.Value); err != nil {
		return err
	}
	// Get the last result.
	result := l.Get(-1)
	if result == lua.LNil {
		return nil
	}
	var err error
	for _, o := range r.Logic.Outputs {
		val := convertLua(result)
		// If value is false, don't run the follow-up.
		if v, ok := val.(bool); ok && !v {
			return nil
		}
		if e := re.handleOutput(ctx, o, r, msg, val); e != nil {
			err = errors.Wrap(e, err)
		}
	}
	return err
}

func (re *re) handleOutput(ctx context.Context, o ScriptOutput, r Rule, msg *messaging.Message, val interface{}) error {
	switch o {
	case Channels:
		if r.OutputChannel == "" {
			return nil
		}
		return re.publishChannel(ctx, val, r.OutputChannel, r.OutputTopic, msg)
	case SaveSenML:
		return re.saveSenml(ctx, val, msg)
	case Email:
		break
	}
	return nil
}

func (re *re) StartScheduler(ctx context.Context) error {
	defer re.ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-re.ticker.Tick():
			startTime := time.Now()

			pm := PageMeta{
				Status:          EnabledStatus,
				ScheduledBefore: &startTime,
			}

			page, err := re.repo.ListRules(ctx, pm)
			if err != nil {
				return err
			}

			for _, rule := range page.Rules {
				if rule.shouldRun(startTime) {
					go func(r Rule) {
						msg := &messaging.Message{
							Channel: r.InputChannel,
							Created: startTime.Unix(),
						}
						re.errors <- re.process(ctx, r, msg)
					}(rule)
				}
			}
		}
	}
}

func (r Rule) shouldRun(startTime time.Time) bool {
	// Don't run if the rule's start time is in the future
	// This allows scheduling rules to start at a specific future time
	if r.Schedule.StartDateTime.After(startTime) {
		return false
	}

	t := r.Schedule.Time.Truncate(time.Minute).UTC()
	startTimeOnly := time.Date(0, 1, 1, startTime.Hour(), startTime.Minute(), 0, 0, time.UTC)
	if t.Equal(startTimeOnly) {
		return true
	}

	if r.Schedule.RecurringPeriod == 0 {
		return false
	}

	period := int(r.Schedule.RecurringPeriod)

	switch r.Schedule.Recurring {
	case Daily:
		if r.Schedule.RecurringPeriod > 0 {
			daysSinceStart := startTime.Sub(r.Schedule.StartDateTime).Hours() / hoursInDay
			if int(daysSinceStart)%period == 0 {
				return true
			}
		}
	case Weekly:
		if r.Schedule.RecurringPeriod > 0 {
			weeksSinceStart := startTime.Sub(r.Schedule.StartDateTime).Hours() / (hoursInDay * daysInWeek)
			if int(weeksSinceStart)%period == 0 {
				return true
			}
		}
	case Monthly:
		if r.Schedule.RecurringPeriod > 0 {
			monthsSinceStart := (startTime.Year()-r.Schedule.StartDateTime.Year())*monthsInYear +
				int(startTime.Month()-r.Schedule.StartDateTime.Month())
			if monthsSinceStart%period == 0 {
				return true
			}
		}
	}

	return false
}
