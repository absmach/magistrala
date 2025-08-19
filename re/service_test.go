// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/0x6flab/namegenerator"
	"github.com/absmach/magistrala/internal/testsutil"
	emocks "github.com/absmach/magistrala/pkg/emailer/mocks"
	pkglog "github.com/absmach/magistrala/pkg/logger"
	pkgSch "github.com/absmach/magistrala/pkg/schedule"
	tmocks "github.com/absmach/magistrala/pkg/ticker/mocks"
	"github.com/absmach/magistrala/re"
	"github.com/absmach/magistrala/re/mocks"
	"github.com/absmach/magistrala/re/outputs"
	readmocks "github.com/absmach/magistrala/readers/mocks"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/messaging"
	pubsubmocks "github.com/absmach/supermq/pkg/messaging/mocks"
	"github.com/absmach/supermq/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	namegen       = namegenerator.NewGenerator()
	userID        = testsutil.GenerateUUID(&testing.T{})
	domainID      = testsutil.GenerateUUID(&testing.T{})
	ruleName      = namegen.Generate()
	ruleID        = testsutil.GenerateUUID(&testing.T{})
	Tags          = []string{"tag1", "tag2"}
	inputChannel  = "test.channel"
	testSubtopic  = "test"
	StartDateTime = time.Now().Add(-time.Hour)
	schedule      = pkgSch.Schedule{
		StartDateTime:   StartDateTime,
		Recurring:       pkgSch.Daily,
		RecurringPeriod: 1,
		Time:            time.Now().Add(-time.Hour),
	}
)

func newService(t *testing.T, runInfo chan pkglog.RunInfo) (re.Service, *mocks.Repository, *pubsubmocks.PubSub, *tmocks.Ticker) {
	repo := new(mocks.Repository)
	mockTicker := new(tmocks.Ticker)
	idProvider := uuid.NewMock()
	pubsub := pubsubmocks.NewPubSub(t)
	readersSvc := new(readmocks.ReadersServiceClient)
	e := new(emocks.Emailer)
	return re.NewService(repo, runInfo, idProvider, pubsub, pubsub, pubsub, mockTicker, e, readersSvc), repo, pubsub, mockTicker
}

func TestAddRule(t *testing.T) {
	svc, repo, _, _ := newService(t, make(chan pkglog.RunInfo))
	ruleName := namegen.Generate()
	now := time.Now().Add(time.Hour)
	cases := []struct {
		desc    string
		session authn.Session
		rule    re.Rule
		res     re.Rule
		err     error
	}{
		{
			desc: "Add rule successfully",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			rule: re.Rule{
				Name:         ruleName,
				InputChannel: inputChannel,
				Schedule: pkgSch.Schedule{
					Recurring:       pkgSch.Daily,
					RecurringPeriod: 1,
					Time:            now,
				},
			},
			res: re.Rule{
				Name:         ruleName,
				ID:           ruleID,
				InputChannel: inputChannel,
				Schedule: pkgSch.Schedule{
					Recurring:       pkgSch.Daily,
					RecurringPeriod: 1,
					Time:            now,
				},
				Status:    re.EnabledStatus,
				CreatedBy: userID,
				DomainID:  domainID,
			},
			err: nil,
		},
		{
			desc: "Add rule with failed repo",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			rule: re.Rule{
				Name:         ruleName,
				InputChannel: inputChannel,
				Schedule: pkgSch.Schedule{
					Recurring:       pkgSch.Daily,
					RecurringPeriod: 1,
					Time:            now,
				},
			},
			err: repoerr.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("AddRule", mock.Anything, mock.Anything).Return(tc.res, tc.err)
			res, err := svc.AddRule(context.Background(), tc.session, tc.rule)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				assert.NotEmpty(t, res.ID, "expected non-empty result in ID")
				assert.Equal(t, tc.rule.Name, res.Name)
				assert.Equal(t, tc.rule.Schedule, res.Schedule)
			}
			defer repoCall.Unset()
		})
	}
}

func TestViewRule(t *testing.T) {
	svc, repo, _, _ := newService(t, make(chan pkglog.RunInfo))

	now := time.Now().Add(time.Hour)
	cases := []struct {
		desc    string
		session authn.Session
		id      string
		res     re.Rule
		err     error
	}{
		{
			desc: "view rule successfully",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			id: ruleID,
			res: re.Rule{
				Name:         ruleName,
				ID:           ruleID,
				InputChannel: inputChannel,
				Schedule: pkgSch.Schedule{
					Recurring:       pkgSch.Daily,
					RecurringPeriod: 1,
					Time:            now,
				},
				Status:    re.EnabledStatus,
				CreatedBy: userID,
				DomainID:  domainID,
			},
			err: nil,
		},
		{
			desc: "view rule with failed repo",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			id:  ruleID,
			err: svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("ViewRule", mock.Anything, mock.Anything).Return(tc.res, tc.err)
			res, err := svc.ViewRule(context.Background(), tc.session, tc.id)

			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				assert.Equal(t, tc.res, res)
			}
			defer repoCall.Unset()
		})
	}
}

func TestUpdateRule(t *testing.T) {
	svc, repo, _, _ := newService(t, make(chan pkglog.RunInfo))

	newName := namegen.Generate()
	now := time.Now().Add(time.Hour)
	cases := []struct {
		desc    string
		session authn.Session
		rule    re.Rule
		res     re.Rule
		err     error
	}{
		{
			desc: "update rule successfully",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			rule: re.Rule{
				Name:         newName,
				ID:           ruleID,
				InputChannel: inputChannel,
				Schedule: pkgSch.Schedule{
					Recurring:       pkgSch.Daily,
					RecurringPeriod: 1,
					Time:            now,
				},
				Status:    re.EnabledStatus,
				CreatedBy: userID,
				DomainID:  domainID,
			},
			res: re.Rule{
				Name:         newName,
				ID:           ruleID,
				InputChannel: inputChannel,
				Schedule: pkgSch.Schedule{
					Recurring:       pkgSch.Daily,
					RecurringPeriod: 1,
					Time:            now,
				},
				Status:    re.EnabledStatus,
				CreatedBy: userID,
				DomainID:  domainID,
				UpdatedAt: now,
				UpdatedBy: userID,
			},
			err: nil,
		},
		{
			desc: "update rule with failed repo",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			rule: re.Rule{
				Name:         ruleName,
				ID:           ruleID,
				InputChannel: inputChannel,
				Schedule: pkgSch.Schedule{
					Recurring:       pkgSch.Daily,
					RecurringPeriod: 1,
					Time:            now,
				},
				Status:    re.EnabledStatus,
				CreatedBy: userID,
				DomainID:  domainID,
			},
			err: svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("UpdateRule", mock.Anything, mock.Anything).Return(tc.res, tc.err)
			res, err := svc.UpdateRule(context.Background(), tc.session, tc.rule)

			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				assert.Equal(t, tc.res, res)
			}
			defer repoCall.Unset()
		})
	}
}

func TestUpdateRuleTags(t *testing.T) {
	svc, repo, _, _ := newService(t, make(chan pkglog.RunInfo))

	cases := []struct {
		desc      string
		session   authn.Session
		updateReq re.Rule
		repoResp  re.Rule
		repoErr   error
		err       error
	}{
		{
			desc: "update rule tags successfully",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			updateReq: re.Rule{
				ID:   testsutil.GenerateUUID(t),
				Tags: []string{"tag1", "tag2"},
			},
			repoResp: re.Rule{
				ID:   testsutil.GenerateUUID(t),
				Tags: []string{"tag1", "tag2"},
			},
		},
		{
			desc: "update rule tags with repo error",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			updateReq: re.Rule{
				ID:   testsutil.GenerateUUID(t),
				Tags: []string{"tag1", "tag2"},
			},
			repoErr: repoerr.ErrNotFound,
			err:     svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("UpdateRuleTags", context.Background(), mock.Anything).Return(tc.repoResp, tc.repoErr)
			got, err := svc.UpdateRuleTags(context.Background(), tc.session, tc.updateReq)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			if err == nil {
				assert.Equal(t, tc.repoResp, got)
				ok := repo.AssertCalled(t, "UpdateRuleTags", context.Background(), mock.Anything)
				assert.True(t, ok, fmt.Sprintf("UpdateTags was not called on %s", tc.desc))
			}
			repoCall.Unset()
		})
	}
}

func TestListRules(t *testing.T) {
	svc, repo, _, _ := newService(t, make(chan pkglog.RunInfo))
	numRules := 50
	now := time.Now().Add(time.Hour)
	var rules []re.Rule
	for i := 0; i < numRules; i++ {
		r := re.Rule{
			ID:        testsutil.GenerateUUID(t),
			Name:      namegen.Generate(),
			DomainID:  domainID,
			Status:    re.EnabledStatus,
			CreatedAt: now,
			CreatedBy: userID,
			Schedule: pkgSch.Schedule{
				Recurring:       pkgSch.Daily,
				Time:            now.Add(1 * time.Hour),
				RecurringPeriod: 1,
				StartDateTime:   now,
			},
		}
		rules = append(rules, r)
	}

	cases := []struct {
		desc     string
		session  authn.Session
		pageMeta re.PageMeta
		res      re.Page
		err      error
	}{
		{
			desc: "list rules successfully",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			pageMeta: re.PageMeta{},
			res: re.Page{
				Total:  uint64(numRules),
				Offset: 0,
				Limit:  10,
				Rules:  rules[0:10],
			},
			err: nil,
		},
		{
			desc: "list rules successfully with limit",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			pageMeta: re.PageMeta{
				Limit: 100,
			},
			res: re.Page{
				Total:  uint64(numRules),
				Offset: 0,
				Limit:  100,
				Rules:  rules[0:numRules],
			},
			err: nil,
		},
		{
			desc: "list rules successfully with offset",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			pageMeta: re.PageMeta{
				Offset: 20,
				Limit:  10,
			},
			res: re.Page{
				Total:  uint64(numRules),
				Offset: 20,
				Limit:  10,
				Rules:  rules[20:30],
			},
			err: nil,
		},
		{
			desc: "list rules with failed repo",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			pageMeta: re.PageMeta{},
			err:      svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("ListRules", mock.Anything, mock.Anything).Return(tc.res, tc.err)
			res, err := svc.ListRules(context.Background(), tc.session, tc.pageMeta)

			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				assert.Equal(t, tc.res, res)
			}
			defer repoCall.Unset()
		})
	}
}

func TestRemoveRule(t *testing.T) {
	svc, repo, _, _ := newService(t, make(chan pkglog.RunInfo))

	cases := []struct {
		desc    string
		session authn.Session
		id      string
		err     error
	}{
		{
			desc: "remove rule successfully",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			id:  ruleID,
			err: nil,
		},
		{
			desc: "remove rule with failed repo",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			id:  ruleID,
			err: svcerr.ErrRemoveEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("RemoveRule", mock.Anything, mock.Anything).Return(tc.err)
			err := svc.RemoveRule(context.Background(), tc.session, tc.id)

			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			defer repoCall.Unset()
		})
	}
}

func TestEnableRule(t *testing.T) {
	svc, repo, _, _ := newService(t, make(chan pkglog.RunInfo))

	now := time.Now()

	cases := []struct {
		desc    string
		session authn.Session
		id      string
		status  re.Status
		res     re.Rule
		err     error
	}{
		{
			desc: "enable rule successfully",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			id:     ruleID,
			status: re.EnabledStatus,
			res: re.Rule{
				ID:           ruleID,
				Name:         ruleName,
				DomainID:     domainID,
				InputChannel: inputChannel,
				Status:       re.EnabledStatus,
				Schedule:     schedule,
				UpdatedBy:    userID,
				UpdatedAt:    now,
			},
			err: nil,
		},
		{
			desc: "enable rule with failed repo",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			id:     ruleID,
			status: re.EnabledStatus,
			err:    svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("UpdateRuleStatus", context.Background(), mock.Anything).Return(tc.res, tc.err)
			res, err := svc.EnableRule(context.Background(), tc.session, tc.id)

			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				assert.Equal(t, tc.res, res)
			}
			defer repoCall.Unset()
		})
	}
}

func TestDisableRule(t *testing.T) {
	svc, repo, _, _ := newService(t, make(chan pkglog.RunInfo))

	now := time.Now()

	cases := []struct {
		desc    string
		session authn.Session
		id      string
		status  re.Status
		res     re.Rule
		err     error
	}{
		{
			desc: "disable rule successfully",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			id:     ruleID,
			status: re.DisabledStatus,
			res: re.Rule{
				ID:           ruleID,
				Name:         ruleName,
				DomainID:     domainID,
				InputChannel: inputChannel,
				Status:       re.DisabledStatus,
				Schedule:     schedule,
				UpdatedBy:    userID,
				UpdatedAt:    now,
			},
			err: nil,
		},
		{
			desc: "disable rule with failed repo",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			id:     ruleID,
			status: re.DisabledStatus,
			err:    svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("UpdateRuleStatus", mock.Anything, mock.Anything).Return(tc.res, tc.err)
			res, err := svc.DisableRule(context.Background(), tc.session, tc.id)

			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				assert.Equal(t, tc.res, res)
			}
			defer repoCall.Unset()
		})
	}
}

func TestHandle(t *testing.T) {
	svc, repo, pubmocks, _ := newService(t, make(chan pkglog.RunInfo))
	now := time.Now()
	scheduled := false

	cases := []struct {
		desc       string
		message    *messaging.Message
		page       re.Page
		listErr    error
		publishErr error
		expectErr  bool
	}{
		{
			desc: "consume message with empty rules",
			message: &messaging.Message{
				Channel: inputChannel,
				Created: now.Unix(),
			},
			page: re.Page{
				Rules: []re.Rule{},
			},
			listErr: nil,
		},
		{
			desc: "consume message with rules",
			message: &messaging.Message{
				Channel: inputChannel,
				Created: now.Unix(),
			},
			page: re.Page{
				Rules: []re.Rule{
					{
						ID:           testsutil.GenerateUUID(t),
						Name:         namegen.Generate(),
						InputChannel: inputChannel,
						Status:       re.EnabledStatus,
						Logic: re.Script{
							Type: re.ScriptType(0),
						},
						Outputs: re.Outputs{
							&outputs.ChannelPublisher{
								Channel: "output.channel",
								Topic:   "output.topic",
							},
						},
						Schedule: schedule,
					},
				},
			},
			listErr: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			var err error

			repoCall := repo.On("ListRules", mock.Anything, re.PageMeta{Domain: tc.message.Domain, InputChannel: tc.message.Channel, Scheduled: &scheduled}).Return(tc.page, tc.listErr).Run(func(args mock.Arguments) {
				if tc.listErr != nil {
					err = tc.listErr
				}
			})
			repoCall1 := pubmocks.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return(tc.publishErr)

			err = svc.Handle(tc.message)
			assert.Nil(t, err)

			assert.True(t, errors.Contains(err, tc.listErr), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.listErr, err))

			repoCall.Unset()
			repoCall1.Unset()
		})
	}
}

func TestThrottledHandler(t *testing.T) {
	svc, repo, pubmocks, _ := newService(t, make(chan pkglog.RunInfo))
	now := time.Now()

	cases := []struct {
		desc        string
		config      re.ThrottlingConfig
		messages    []*messaging.Message
		page        re.Page
		listErr     error
		publishErr  error
		shouldBlock []bool
	}{
		{
			desc: "throttled handler allows messages under rate limit",
			config: re.ThrottlingConfig{
				RateLimit:     100,
				LoopThreshold: 5,
				LoopWindow:    5 * time.Second,
			},
			messages: []*messaging.Message{
				{
					Channel: inputChannel,
					Domain:  domainID,
					Created: now.Unix(),
				},
				{
					Channel: inputChannel,
					Domain:  domainID,
					Created: now.Unix(),
				},
			},
			page: re.Page{
				Rules: []re.Rule{
					{
						ID:           testsutil.GenerateUUID(t),
						Name:         namegen.Generate(),
						InputChannel: inputChannel,
						Status:       re.EnabledStatus,
						Logic: re.Script{
							Type: re.ScriptType(0),
						},
						Outputs: re.Outputs{
							&outputs.ChannelPublisher{
								Channel: "output.channel",
								Topic:   "output.topic",
							},
						},
						Schedule: schedule,
					},
				},
			},
			listErr:     nil,
			shouldBlock: []bool{false, false},
		},
		{
			desc: "throttled handler detects loops and blocks messages",
			config: re.ThrottlingConfig{
				RateLimit:     100,
				LoopThreshold: 3,
				LoopWindow:    5 * time.Second,
			},
			messages: []*messaging.Message{
				{
					Channel:  inputChannel,
					Domain:   domainID,
					Subtopic: testSubtopic,
					Created:  now.Unix(),
				},
				{
					Channel:  inputChannel,
					Domain:   domainID,
					Subtopic: testSubtopic,
					Created:  now.Unix(),
				},
				{
					Channel:  inputChannel,
					Domain:   domainID,
					Subtopic: testSubtopic,
					Created:  now.Unix(),
				},
				{
					Channel:  inputChannel,
					Domain:   domainID,
					Subtopic: testSubtopic,
					Created:  now.Unix(),
				},
				{
					Channel:  inputChannel,
					Domain:   domainID,
					Subtopic: testSubtopic,
					Created:  now.Unix(),
				},
			},
			page: re.Page{
				Rules: []re.Rule{
					{
						ID:           testsutil.GenerateUUID(t),
						Name:         namegen.Generate(),
						InputChannel: inputChannel,
						Status:       re.EnabledStatus,
						Logic: re.Script{
							Type: re.ScriptType(0),
						},
						Outputs: re.Outputs{
							&outputs.ChannelPublisher{
								Channel: "output.channel",
								Topic:   "output.topic",
							},
						},
						Schedule: schedule,
					},
				},
			},
			listErr:     nil,
			shouldBlock: []bool{false, false, false, true, true},
		},
		{
			desc: "throttled handler allows messages after window reset",
			config: re.ThrottlingConfig{
				RateLimit:     100,
				LoopThreshold: 2,
				LoopWindow:    100 * time.Millisecond,
			},
			messages: []*messaging.Message{
				{
					Channel:  inputChannel,
					Domain:   domainID,
					Subtopic: testSubtopic,
					Created:  now.Unix(),
				},
				{
					Channel:  inputChannel,
					Domain:   domainID,
					Subtopic: testSubtopic,
					Created:  now.Unix(),
				},
				{
					Channel:  inputChannel,
					Domain:   domainID,
					Subtopic: testSubtopic,
					Created:  now.Unix(),
				},
			},
			page: re.Page{
				Rules: []re.Rule{
					{
						ID:           testsutil.GenerateUUID(t),
						Name:         namegen.Generate(),
						InputChannel: inputChannel,
						Status:       re.EnabledStatus,
						Logic: re.Script{
							Type: re.ScriptType(0),
						},
						Outputs: re.Outputs{
							&outputs.ChannelPublisher{
								Channel: "output.channel",
								Topic:   "output.topic",
							},
						},
						Schedule: schedule,
					},
				},
			},
			listErr:     nil,
			shouldBlock: []bool{false, false, false},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			throttledSvc := re.NewThrottledHandler(svc, tc.config)
			callCount := 0

			repoCall := repo.On("ListRules", mock.Anything, mock.Anything).Return(tc.page, tc.listErr).Run(func(args mock.Arguments) {
				callCount++
			})
			pubCall := pubmocks.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return(tc.publishErr)

			for i, msg := range tc.messages {
				err := throttledSvc.Handle(msg)
				assert.Nil(t, err)

				if tc.desc == "throttled handler allows messages after window reset" && i == 1 {
					time.Sleep(150 * time.Millisecond)
				}
			}

			expectedCalls := 0
			for _, blocked := range tc.shouldBlock {
				if !blocked {
					expectedCalls++
				}
			}
			assert.Equal(t, expectedCalls, callCount)

			repoCall.Unset()
			pubCall.Unset()
		})
	}
}

func TestThrottledHandlerRateLimit(t *testing.T) {
	svc, repo, pubmocks, _ := newService(t, make(chan pkglog.RunInfo))

	config := re.ThrottlingConfig{
		RateLimit:     1,
		LoopThreshold: 10,
		LoopWindow:    5 * time.Second,
	}

	throttledSvc := re.NewThrottledHandler(svc, config)

	message := &messaging.Message{
		Channel: inputChannel,
		Domain:  domainID,
		Created: time.Now().Unix(),
	}

	page := re.Page{
		Rules: []re.Rule{
			{
				ID:           testsutil.GenerateUUID(t),
				Name:         namegen.Generate(),
				InputChannel: inputChannel,
				Status:       re.EnabledStatus,
			},
		},
	}

	callCount := 0
	repoCall := repo.On("ListRules", mock.Anything, mock.Anything).Return(page, nil).Run(func(args mock.Arguments) {
		callCount++
	})
	pubCall := pubmocks.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	for i := 0; i < 2; i++ {
		err := throttledSvc.Handle(message)
		assert.Nil(t, err)
	}

	assert.Equal(t, 1, callCount)

	repoCall.Unset()
	pubCall.Unset()
}

func TestThrottledHandlerDelegation(t *testing.T) {
	svc, repo, _, _ := newService(t, make(chan pkglog.RunInfo))

	config := re.ThrottlingConfig{
		RateLimit:     100,
		LoopThreshold: 5,
		LoopWindow:    5 * time.Second,
	}

	throttledSvc := re.NewThrottledHandler(svc, config)

	session := authn.Session{
		UserID:   userID,
		DomainID: domainID,
	}

	rule := re.Rule{
		Name:         ruleName,
		InputChannel: inputChannel,
		Schedule:     schedule,
	}

	expectedRule := re.Rule{
		Name:         ruleName,
		ID:           ruleID,
		InputChannel: inputChannel,
		Schedule:     schedule,
		Status:       re.EnabledStatus,
		CreatedBy:    userID,
		DomainID:     domainID,
	}

	t.Run("AddRule", func(t *testing.T) {
		repoCall := repo.On("AddRule", mock.Anything, mock.Anything).Return(expectedRule, nil)
		res, err := throttledSvc.AddRule(context.Background(), session, rule)
		assert.Nil(t, err)
		assert.Equal(t, expectedRule, res)
		repoCall.Unset()
	})

	t.Run("ViewRule", func(t *testing.T) {
		repoCall := repo.On("ViewRule", mock.Anything, mock.Anything).Return(expectedRule, nil)
		res, err := throttledSvc.ViewRule(context.Background(), session, ruleID)
		assert.Nil(t, err)
		assert.Equal(t, expectedRule, res)
		repoCall.Unset()
	})

	t.Run("UpdateRule", func(t *testing.T) {
		repoCall := repo.On("UpdateRule", mock.Anything, mock.Anything).Return(expectedRule, nil)
		res, err := throttledSvc.UpdateRule(context.Background(), session, rule)
		assert.Nil(t, err)
		assert.Equal(t, expectedRule, res)
		repoCall.Unset()
	})

	t.Run("RemoveRule", func(t *testing.T) {
		repoCall := repo.On("RemoveRule", mock.Anything, mock.Anything).Return(nil)
		err := throttledSvc.RemoveRule(context.Background(), session, ruleID)
		assert.Nil(t, err)
		repoCall.Unset()
	})

	t.Run("EnableRule", func(t *testing.T) {
		repoCall := repo.On("UpdateRuleStatus", mock.Anything, mock.Anything).Return(expectedRule, nil)
		res, err := throttledSvc.EnableRule(context.Background(), session, ruleID)
		assert.Nil(t, err)
		assert.Equal(t, expectedRule, res)
		repoCall.Unset()
	})

	t.Run("DisableRule", func(t *testing.T) {
		repoCall := repo.On("UpdateRuleStatus", mock.Anything, mock.Anything).Return(expectedRule, nil)
		res, err := throttledSvc.DisableRule(context.Background(), session, ruleID)
		assert.Nil(t, err)
		assert.Equal(t, expectedRule, res)
		repoCall.Unset()
	})

	t.Run("UpdateRuleTags", func(t *testing.T) {
		ruleWithTags := re.Rule{
			ID:   ruleID,
			Tags: []string{"tag1", "tag2"},
		}
		expectedRuleWithTags := re.Rule{
			ID:   ruleID,
			Tags: []string{"tag1", "tag2"},
		}
		repoCall := repo.On("UpdateRuleTags", context.Background(), mock.Anything).Return(expectedRuleWithTags, nil)
		res, err := throttledSvc.UpdateRuleTags(context.Background(), session, ruleWithTags)
		assert.Nil(t, err)
		assert.Equal(t, expectedRuleWithTags, res)
		repoCall.Unset()
	})

	t.Run("ListRules", func(t *testing.T) {
		pageMeta := re.PageMeta{
			Limit:  10,
			Offset: 0,
		}
		expectedPage := re.Page{
			Total:  1,
			Offset: 0,
			Limit:  10,
			Rules:  []re.Rule{expectedRule},
		}
		repoCall := repo.On("ListRules", mock.Anything, mock.Anything).Return(expectedPage, nil)
		res, err := throttledSvc.ListRules(context.Background(), session, pageMeta)
		assert.Nil(t, err)
		assert.Equal(t, expectedPage, res)
		repoCall.Unset()
	})

	t.Run("UpdateRuleSchedule", func(t *testing.T) {
		ruleWithSchedule := re.Rule{
			ID:       ruleID,
			Schedule: schedule,
		}
		expectedRuleWithSchedule := re.Rule{
			ID:       ruleID,
			Schedule: schedule,
		}
		repoCall := repo.On("UpdateRuleSchedule", mock.Anything, mock.Anything).Return(expectedRuleWithSchedule, nil)
		res, err := throttledSvc.UpdateRuleSchedule(context.Background(), session, ruleWithSchedule)
		assert.Nil(t, err)
		assert.Equal(t, expectedRuleWithSchedule, res)
		repoCall.Unset()
	})
}

func TestStartScheduler(t *testing.T) {
	now := time.Now().Truncate(time.Minute)
	ri := make(chan pkglog.RunInfo)
	svc, repo, _, ticker := newService(t, ri)

	ctxCases := []struct {
		desc     string
		err      error
		pageMeta re.PageMeta
		page     re.Page
		listErr  error
		setupCtx func() (context.Context, context.CancelFunc)
	}{
		{
			desc: "start scheduler with canceled context",
			err:  context.Canceled,
			pageMeta: re.PageMeta{
				Status:          re.EnabledStatus,
				ScheduledBefore: &now,
			},
			setupCtx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, cancel
			},
		},
		{
			desc: "start scheduler with timeout",
			err:  context.DeadlineExceeded,
			pageMeta: re.PageMeta{
				Status:          re.EnabledStatus,
				ScheduledBefore: &now,
			},
			setupCtx: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), time.Millisecond)
			},
		},
		{
			desc: "start scheduler with deadline exceeded",
			err:  context.DeadlineExceeded,
			pageMeta: re.PageMeta{
				Status:          re.EnabledStatus,
				ScheduledBefore: &now,
			},
			page: re.Page{},
			setupCtx: func() (context.Context, context.CancelFunc) {
				return context.WithDeadline(context.Background(), time.Now().Add(time.Millisecond))
			},
		},
	}

	for _, tc := range ctxCases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("ListRules", mock.Anything, mock.Anything).Return(tc.page, tc.listErr)
			tickChan := make(chan time.Time)
			tickCall := ticker.On("Tick").Return((<-chan time.Time)(tickChan))
			tickCall1 := ticker.On("Stop").Return()
			ctx, cancel := tc.setupCtx()
			defer cancel()
			errc := make(chan error)

			go func() {
				errc <- svc.StartScheduler(ctx)
			}()

			err := <-errc
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v but got %v", tc.err, err))
			repoCall.Unset()
			tickCall.Unset()
			tickCall1.Unset()
		})
	}
}
