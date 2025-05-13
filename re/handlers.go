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
	pm := PageMeta{
		Domain:       msg.Domain,
		InputChannel: msg.Channel,
		Status:       EnabledStatus,
		InputTopic:   &msg.Subtopic,
	}
	ctx := context.Background()
	page, err := re.repo.ListRules(ctx, pm)
	if err != nil {
		return err
	}

	reportConfigs, err := re.repo.ListReportsConfig(ctx, pm)
	if err != nil {
		return err
	}

	for _, r := range page.Rules {
		go func(ctx context.Context) {
			re.errors <- re.process(ctx, r, msg)
		}(ctx)
	}

	for _, cfg := range reportConfigs.ReportConfigs {
		go func(ctx context.Context) {
			if err := re.processReportConfig(ctx, cfg); err != nil {
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
	// Converting Lua is an expensive operation, so
	// don't do it if there are no outputs.
	if len(r.Logic.Outputs) == 0 {
		return nil
	}
	var err error
	res := convertLua(result)
	for _, o := range r.Logic.Outputs {
		// If value is false, don't run the follow-up.
		if v, ok := res.(bool); ok && !v {
			return nil
		}
		if e := re.handleOutput(ctx, o, r, msg, res); e != nil {
			err = errors.Wrap(e, err)
		}
	}
	return err
}

func (re *re) processReportConfig(ctx context.Context, cfg ReportConfig) error {
	if _, err := re.generateReport(ctx, cfg, EmailReport); err != nil {
		return err
	}
	return nil
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
			due := time.Now().UTC()

			pm := PageMeta{
				Status:          EnabledStatus,
				ScheduledBefore: &due,
			}

			page, err := re.repo.ListRules(ctx, pm)
			if err != nil {
				re.errors <- err
				continue
			}

			for _, r := range page.Rules {
				go func(rule Rule) {
					if _, err := re.repo.UpdateRuleDue(ctx, rule.ID, rule.Schedule.NextDue()); err != nil {
						re.errors <- err
						return
					}

					msg := &messaging.Message{
						Channel:  rule.InputChannel,
						Subtopic: rule.InputTopic,
						Protocol: protocol,
						Created:  due.Unix(),
					}
					re.errors <- re.process(ctx, rule, msg)
				}(r)
			}

			reportConfigs, err := re.repo.ListReportsConfig(ctx, pm)
			if err != nil {
				re.errors <- err
				continue
			}

			for _, c := range reportConfigs.ReportConfigs {
				go func(cfg ReportConfig) {
					if _, err := re.repo.UpdateReportDue(ctx, cfg.ID, cfg.Schedule.NextDue()); err != nil {
						re.errors <- err
						return
					}
					re.errors <- re.processReportConfig(ctx, cfg)
				}(c)
			}
		}
	}
}
