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
	"github.com/absmach/magistrala/re"
	"github.com/absmach/magistrala/re/mocks"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/messaging"
	pubsubmocks "github.com/absmach/supermq/pkg/messaging/mocks"
	mgjson "github.com/absmach/supermq/pkg/transformers/json"
	"github.com/absmach/supermq/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	namegen      = namegenerator.NewGenerator()
	userID       = testsutil.GenerateUUID(&testing.T{})
	domainID     = testsutil.GenerateUUID(&testing.T{})
	ruleName     = namegen.Generate()
	ruleID       = testsutil.GenerateUUID(&testing.T{})
	inputChannel = "test.channel"
	schedule     = re.Schedule{
		StartDateTime:   time.Now().Add(-time.Hour),
		Recurring:       re.Daily,
		RecurringPeriod: 1,
		Time:            time.Now().Add(-time.Hour),
	}
	rule = re.Rule{
		ID:           testsutil.GenerateUUID(&testing.T{}),
		Name:         namegen.Generate(),
		InputChannel: inputChannel,
		Status:       re.EnabledStatus,
		Schedule:     schedule,
	}
	futureRule = re.Rule{
		ID:           testsutil.GenerateUUID(&testing.T{}),
		Name:         namegen.Generate(),
		InputChannel: inputChannel,
		Status:       re.EnabledStatus,
		Schedule: re.Schedule{
			StartDateTime: time.Now().Add(24 * time.Hour),
			Recurring:     re.None,
		},
	}
)

func newService(t *testing.T) (re.Service, *mocks.Repository, *pubsubmocks.PubSub, *mocks.Ticker) {
	repo := new(mocks.Repository)
	mockTicker := new(mocks.Ticker)
	idProvider := uuid.NewMock()
	pubsub := pubsubmocks.NewPubSub(t)
	return re.NewService(repo, idProvider, pubsub, mockTicker), repo, pubsub, mockTicker
}

func TestAddRule(t *testing.T) {
	svc, repo, _, _ := newService(t)
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
				Schedule: re.Schedule{
					Recurring:       re.Daily,
					RecurringPeriod: 1,
					Time:            now,
				},
			},
			res: re.Rule{
				Name:         ruleName,
				ID:           ruleID,
				InputChannel: inputChannel,
				Schedule: re.Schedule{
					Recurring:       re.Daily,
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
				Schedule: re.Schedule{
					Recurring:       re.Daily,
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
	svc, repo, _, _ := newService(t)

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
				Schedule: re.Schedule{
					Recurring:       re.Daily,
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
	svc, repo, _, _ := newService(t)

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
				Schedule: re.Schedule{
					Recurring:       re.Daily,
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
				Schedule: re.Schedule{
					Recurring:       re.Daily,
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
				Schedule: re.Schedule{
					Recurring:       re.Daily,
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

func TestListRules(t *testing.T) {
	svc, repo, _, _ := newService(t)
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
			Schedule: re.Schedule{
				Recurring:       re.Daily,
				Time:            now.Add(1 * time.Hour),
				RecurringPeriod: 1,
				StartDateTime:   now.Add(-1 * time.Hour),
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
				PageMeta: re.PageMeta{
					Total:  uint64(numRules),
					Offset: 0,
					Limit:  10,
				},
				Rules: rules[0:10],
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
				PageMeta: re.PageMeta{
					Total:  uint64(numRules),
					Offset: 0,
					Limit:  100,
				},
				Rules: rules[0:numRules],
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
				PageMeta: re.PageMeta{
					Total:  uint64(numRules),
					Offset: 20,
					Limit:  10,
				},
				Rules: rules[20:30],
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
	svc, repo, _, _ := newService(t)

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
	svc, repo, _, _ := newService(t)

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
			repoCall := repo.On("UpdateRuleStatus", context.Background(), tc.id, tc.status).Return(tc.res, tc.err)
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
	svc, repo, _, _ := newService(t)

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
			repoCall := repo.On("UpdateRuleStatus", mock.Anything, tc.id, tc.status).Return(tc.res, tc.err)
			res, err := svc.DisableRule(context.Background(), tc.session, tc.id)

			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				assert.Equal(t, tc.res, res)
			}
			defer repoCall.Unset()
		})
	}
}

func TestConsumeAsync(t *testing.T) {
	svc, repo, pubmocks, _ := newService(t)
	now := time.Now()

	cases := []struct {
		desc       string
		message    any
		pageMeta   re.PageMeta
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
			pageMeta: re.PageMeta{
				InputChannel: inputChannel,
				Status:       re.EnabledStatus,
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
			pageMeta: re.PageMeta{
				InputChannel: inputChannel,
				Status:       re.EnabledStatus,
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
						OutputChannel: "output.channel",
						Schedule:      schedule,
					},
				},
			},
			listErr: nil,
		},
		{
			desc:    "consume message with unsupported message type",
			message: "unsupported message type",
			pageMeta: re.PageMeta{
				InputChannel: inputChannel,
				Status:       re.EnabledStatus,
			},
			page: re.Page{},
		},
		{
			desc:    "consume json message",
			message: mgjson.Message{},
			pageMeta: re.PageMeta{
				InputChannel: inputChannel,
				Status:       re.EnabledStatus,
			},
			page:    re.Page{},
			listErr: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			defer cancel()

			var err error

			repoCall := repo.On("ListRules", mock.Anything, tc.pageMeta).Return(tc.page, tc.listErr).Run(func(args mock.Arguments) {
				if tc.listErr != nil {
					err = tc.listErr
				}
			})
			repoCall1 := pubmocks.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return(tc.publishErr)

			svc.ConsumeAsync(ctx, tc.message)

			assert.True(t, errors.Contains(err, tc.listErr), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.listErr, err))

			repoCall.Unset()
			repoCall1.Unset()
		})
	}
}

func TestStartScheduler(t *testing.T) {
	now := time.Now().Truncate(time.Minute)
	svc, repo, _, ticker := newService(t)

	noRecurringPeriod := re.Rule{
		ID:           testsutil.GenerateUUID(t),
		Name:         namegen.Generate(),
		InputChannel: inputChannel,
		Status:       re.EnabledStatus,
		Schedule: re.Schedule{
			StartDateTime:   time.Now().Add(-time.Hour),
			Recurring:       re.None,
			RecurringPeriod: 0,
			Time:            time.Now().Add(-time.Hour),
		},
	}

	weeklyRule := re.Rule{
		ID:           testsutil.GenerateUUID(t),
		Name:         namegen.Generate(),
		InputChannel: inputChannel,
		Status:       re.EnabledStatus,
		Schedule: re.Schedule{
			StartDateTime:   time.Now().Add(-time.Hour),
			Recurring:       re.Weekly,
			RecurringPeriod: 1,
			Time:            time.Now().Add(-time.Hour),
		},
	}

	monthlyRule := re.Rule{
		ID:           testsutil.GenerateUUID(t),
		Name:         namegen.Generate(),
		InputChannel: inputChannel,
		Status:       re.EnabledStatus,
		Schedule: re.Schedule{
			StartDateTime:   time.Now().Add(-time.Hour),
			Recurring:       re.Monthly,
			RecurringPeriod: 1,
			Time:            time.Now().Add(-time.Hour),
		},
	}

	pastRule := re.Rule{
		ID:           testsutil.GenerateUUID(t),
		Name:         namegen.Generate(),
		InputChannel: inputChannel,
		Status:       re.EnabledStatus,
		Schedule: re.Schedule{
			StartDateTime:   time.Now().Add(-time.Hour),
			Recurring:       re.None,
			RecurringPeriod: 1,
			Time:            time.Now().Add(-time.Hour),
		},
	}

	cases := []struct {
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
				return context.WithCancel(context.Background())
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
		{
			desc: "start scheduler successfully processes rules",
			err:  context.Canceled,
			pageMeta: re.PageMeta{
				Status:          re.EnabledStatus,
				ScheduledBefore: &now,
			},
			page: re.Page{
				Rules: []re.Rule{rule},
			},
			setupCtx: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
		},
		{
			desc: "start scheduler with list error",
			err:  repoerr.ErrViewEntity,
			pageMeta: re.PageMeta{
				Status:          re.EnabledStatus,
				ScheduledBefore: &now,
			},
			page:    re.Page{},
			listErr: repoerr.ErrViewEntity,
			setupCtx: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
		},
		{
			desc: "start scheduler with rule to be run in the future",
			err:  context.Canceled,
			pageMeta: re.PageMeta{
				Status:          re.EnabledStatus,
				ScheduledBefore: &now,
			},
			page: re.Page{
				Rules: []re.Rule{futureRule},
			},
			setupCtx: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
		},
		{
			desc: "start scheduler successfully with no recurring period",
			err:  context.Canceled,
			pageMeta: re.PageMeta{
				Status:          re.EnabledStatus,
				ScheduledBefore: &now,
			},
			page: re.Page{
				Rules: []re.Rule{noRecurringPeriod},
			},
			setupCtx: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
		},
		{
			desc: "start scheduler successfully with weekly schedule",
			err:  context.Canceled,
			pageMeta: re.PageMeta{
				Status:          re.EnabledStatus,
				ScheduledBefore: &now,
			},
			page: re.Page{
				Rules: []re.Rule{weeklyRule},
			},
			setupCtx: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
		},
		{
			desc: "start scheduler successfully with monthly schedule",
			err:  context.Canceled,
			pageMeta: re.PageMeta{
				Status:          re.EnabledStatus,
				ScheduledBefore: &now,
			},
			page: re.Page{
				Rules: []re.Rule{monthlyRule},
			},
			setupCtx: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
		},
		{
			desc: "start scheduler successfully processes rules with past schedule",
			err:  context.Canceled,
			pageMeta: re.PageMeta{
				Status:          re.EnabledStatus,
				ScheduledBefore: &now,
			},
			page: re.Page{
				Rules: []re.Rule{pastRule},
			},
			setupCtx: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
		},
	}

	for _, tc := range cases {
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

			switch tc.desc {
			case "start scheduler with canceled context":
				cancel()
			case "start scheduler with list error":
				tickChan <- time.Now()
				time.Sleep(100 * time.Millisecond)
				if err := svc.Errors(); err != nil {
					cancel()
				}
			default:
				tickChan <- time.Now()
				time.Sleep(100 * time.Millisecond)
				cancel()
			}

			err := <-errc
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v but got %v", tc.err, err))
			repoCall.Unset()
			tickCall.Unset()
			tickCall1.Unset()
		})
	}
}
