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

	svc := re.NewService(repo, runInfo, idProvider, pubsub, pubsub, pubsub, mockTicker, e, readersSvc)

	repocall1 := repo.On("ListRules", mock.Anything, mock.Anything).Return(re.Page{Rules: []re.Rule{}}, nil)
	defer repocall1.Unset()

	tickCh := make(chan time.Time)
	mockTicker.On("Tick").Return((<-chan time.Time)(tickCh))
	mockTicker.On("Stop").Return()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	schedulerDone := make(chan struct{})
	go func() {
		_ = svc.StartScheduler(ctx)
		close(schedulerDone)
	}()

	t.Cleanup(func() {
		cancel()

		// Wait for scheduler to stop (can take time for worker cleanup)
		<-schedulerDone

		time.Sleep(100 * time.Millisecond)
	})
	time.Sleep(50 * time.Millisecond)

	return svc, repo, pubsub, mockTicker
}

func newServiceForSchedulerTest(t *testing.T, runInfo chan pkglog.RunInfo) (re.Service, *mocks.Repository, *pubsubmocks.PubSub, *tmocks.Ticker) {
	repo := new(mocks.Repository)
	mockTicker := new(tmocks.Ticker)
	idProvider := uuid.NewMock()
	pubsub := pubsubmocks.NewPubSub(t)
	readersSvc := new(readmocks.ReadersServiceClient)
	e := new(emocks.Emailer)

	svc := re.NewService(repo, runInfo, idProvider, pubsub, pubsub, pubsub, mockTicker, e, readersSvc)

	tickCh := make(chan time.Time)
	mockTicker.On("Tick").Return((<-chan time.Time)(tickCh))
	mockTicker.On("Stop").Return()

	return svc, repo, pubsub, mockTicker
}

func TestAddRule(t *testing.T) {
	svc, repo, _, _ := newService(t, make(chan pkglog.RunInfo, 100))
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
	svc, repo, _, _ := newService(t, make(chan pkglog.RunInfo, 100))

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
	svc, repo, _, _ := newService(t, make(chan pkglog.RunInfo, 100))

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
	svc, repo, _, _ := newService(t, make(chan pkglog.RunInfo, 100))

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
	svc, repo, _, _ := newService(t, make(chan pkglog.RunInfo, 100))
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
			res: re.Page{
				Rules: []re.Rule{},
			},
			err: svcerr.ErrViewEntity,
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
	svc, repo, _, _ := newService(t, make(chan pkglog.RunInfo, 100))

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
	svc, repo, _, _ := newService(t, make(chan pkglog.RunInfo, 100))

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
	svc, repo, _, _ := newService(t, make(chan pkglog.RunInfo, 100))

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

func TestAbortRuleExecution(t *testing.T) {
	cases := []struct {
		desc    string
		session authn.Session
		id      string
		rule    re.Rule
		repoErr error
		err     error
	}{
		{
			desc: "abort rule execution with wrong status",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			id: ruleID,
			rule: re.Rule{
				ID:            ruleID,
				LastRunStatus: re.SuccessStatus,
			},
			repoErr: nil,
			err:     svcerr.ErrMalformedEntity,
		},
		{
			desc: "abort rule execution with never_run status",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			id: ruleID,
			rule: re.Rule{
				ID:            ruleID,
				LastRunStatus: re.NeverRunStatus,
			},
			repoErr: nil,
			err:     svcerr.ErrMalformedEntity,
		},
		{
			desc: "abort rule execution with non-existent rule",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			id:      "non-existent-rule",
			rule:    re.Rule{},
			repoErr: svcerr.ErrNotFound,
			err:     svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svc, repo, _, _ := newService(t, make(chan pkglog.RunInfo, 100))
			repo.On("ViewRule", mock.Anything, tc.id).Return(tc.rule, tc.repoErr)
			err := svc.AbortRuleExecution(context.Background(), tc.session, tc.id)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			repo.AssertExpectations(t)
		})
	}
}

func TestHandle(t *testing.T) {
	svc, repo, pubmocks, _ := newService(t, make(chan pkglog.RunInfo, 100))
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
			desc: "consume message with enabled rules",
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
		{
			desc: "consume message with disabled rules",
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
						Status:       re.DisabledStatus,
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

			enabledRulesCount := 0
			for _, rule := range tc.page.Rules {
				if rule.Status == re.EnabledStatus {
					enabledRulesCount++
				}
			}

			if enabledRulesCount > 0 {
				repoCall1 := pubmocks.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return(tc.publishErr).Times(enabledRulesCount)
				defer repoCall1.Unset()
			}

			err = svc.Handle(tc.message)

			assert.NoError(t, err, "Handle should not return errors with worker architecture")

			if tc.listErr != nil {
				assert.True(t, errors.Contains(err, tc.listErr), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.listErr, err))
			}

			repoCall.Unset()
		})
	}
}

func TestStartScheduler(t *testing.T) {
	now := time.Now().Truncate(time.Minute)
	ri := make(chan pkglog.RunInfo, 100)
	svc, repo, _, ticker := newServiceForSchedulerTest(t, ri)

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
