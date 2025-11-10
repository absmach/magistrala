// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re_test

import (
	"context"
	"fmt"
	"log/slog"
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

// unknownOutput is a mock output type that doesn't match any known output type.
type unknownOutput struct{}

func (u *unknownOutput) Run(ctx context.Context, msg *messaging.Message, val any) error {
	return nil
}

func (u *unknownOutput) MarshalJSON() ([]byte, error) {
	return []byte(`{"type": "unknown"}`), nil
}

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

func newService(t *testing.T, runInfo chan pkglog.RunInfo) (re.Service, *mocks.Repository, *pubsubmocks.PubSub, *tmocks.Ticker, *emocks.Emailer) {
	repo := new(mocks.Repository)
	mockTicker := new(tmocks.Ticker)
	idProvider := uuid.NewMock()
	pubsub := pubsubmocks.NewPubSub(t)
	readersSvc := new(readmocks.ReadersServiceClient)
	e := new(emocks.Emailer)
	return re.NewService(repo, runInfo, idProvider, pubsub, pubsub, pubsub, mockTicker, e, readersSvc), repo, pubsub, mockTicker, e
}

func TestAddRule(t *testing.T) {
	// nolint:dogsled
	svc, repo, _, _, _ := newService(t, make(chan pkglog.RunInfo))
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
		{
			desc: "Add rule with non-zero StartDateTime",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			rule: re.Rule{
				Name:         ruleName,
				InputChannel: inputChannel,
				Schedule: pkgSch.Schedule{
					StartDateTime:   now,
					Recurring:       pkgSch.Weekly,
					RecurringPeriod: 2,
					Time:            now.Add(2 * time.Hour),
				},
			},
			res: re.Rule{
				Name:         ruleName,
				ID:           ruleID,
				InputChannel: inputChannel,
				Schedule: pkgSch.Schedule{
					StartDateTime:   now,
					Recurring:       pkgSch.Weekly,
					RecurringPeriod: 2,
					Time:            now.Add(2 * time.Hour),
				},
				Status:    re.EnabledStatus,
				CreatedBy: userID,
				DomainID:  domainID,
			},
			err: nil,
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
	// nolint:dogsled
	svc, repo, _, _, _ := newService(t, make(chan pkglog.RunInfo))

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
	// nolint:dogsled
	svc, repo, _, _, _ := newService(t, make(chan pkglog.RunInfo))

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
	// nolint:dogsled
	svc, repo, _, _, _ := newService(t, make(chan pkglog.RunInfo))

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

func TestUpdateRuleSchedule(t *testing.T) {
	// nolint:dogsled
	svc, repo, _, _, _ := newService(t, make(chan pkglog.RunInfo))

	now := time.Now().UTC()
	future := now.Add(2 * time.Hour)
	newSchedule := pkgSch.Schedule{
		StartDateTime:   future,
		Time:            future.Add(time.Hour),
		Recurring:       pkgSch.Weekly,
		RecurringPeriod: 2,
	}

	cases := []struct {
		desc      string
		session   authn.Session
		updateReq re.Rule
		repoResp  re.Rule
		repoErr   error
		err       error
	}{
		{
			desc: "update rule schedule successfully",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			updateReq: re.Rule{
				ID:       testsutil.GenerateUUID(t),
				Schedule: newSchedule,
			},
			repoResp: re.Rule{
				ID:        testsutil.GenerateUUID(t),
				Schedule:  newSchedule,
				UpdatedAt: now,
				UpdatedBy: userID,
			},
		},
		{
			desc: "update rule schedule with repo error",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			updateReq: re.Rule{
				ID:       testsutil.GenerateUUID(t),
				Schedule: newSchedule,
			},
			repoErr: repoerr.ErrNotFound,
			err:     svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("UpdateRuleSchedule", context.Background(), mock.Anything).Return(tc.repoResp, tc.repoErr)
			got, err := svc.UpdateRuleSchedule(context.Background(), tc.session, tc.updateReq)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v to contain %v", err, tc.err))
			if err == nil {
				assert.Equal(t, tc.repoResp, got)
				ok := repo.AssertCalled(t, "UpdateRuleSchedule", context.Background(), mock.Anything)
				assert.True(t, ok, fmt.Sprintf("UpdateRuleSchedule was not called on %s", tc.desc))
			}
			repoCall.Unset()
		})
	}
}

func TestListRules(t *testing.T) {
	// nolint:dogsled
	svc, repo, _, _, _ := newService(t, make(chan pkglog.RunInfo))
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

	goRule := re.Rule{
		ID:        testsutil.GenerateUUID(t),
		Name:      namegen.Generate(),
		DomainID:  domainID,
		Status:    re.EnabledStatus,
		CreatedAt: now,
		CreatedBy: userID,
		Logic: re.Script{
			Type:  re.GoType,
			Value: "func() bool { return true }",
		},
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
			desc: "list rules with go type",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			pageMeta: re.PageMeta{},
			res: re.Page{
				Total:  1,
				Offset: 0,
				Limit:  10,
				Rules:  []re.Rule{goRule},
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
	// nolint:dogsled
	svc, repo, _, _, _ := newService(t, make(chan pkglog.RunInfo))

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
	// nolint:dogsled
	svc, repo, _, _, _ := newService(t, make(chan pkglog.RunInfo))

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
	// nolint:dogsled
	svc, repo, _, _, _ := newService(t, make(chan pkglog.RunInfo))

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
	svc, repo, pubmocks, _, emailer := newService(t, make(chan pkglog.RunInfo))
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
			desc: "consume message with Lua script returning true",
			message: &messaging.Message{
				Channel: inputChannel,
				Created: now.Unix(),
				Payload: []byte(`{"temperature": 25.5}`),
			},
			page: re.Page{
				Rules: []re.Rule{
					{
						ID:           testsutil.GenerateUUID(t),
						Name:         namegen.Generate(),
						InputChannel: inputChannel,
						Status:       re.EnabledStatus,
						Logic: re.Script{
							Type:  re.LuaType,
							Value: "return message.payload",
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
			desc: "consume message with Lua script returning false",
			message: &messaging.Message{
				Channel: inputChannel,
				Created: now.Unix(),
				Payload: []byte(`{"temperature": 25.5}`),
			},
			page: re.Page{
				Rules: []re.Rule{
					{
						ID:           testsutil.GenerateUUID(t),
						Name:         namegen.Generate(),
						InputChannel: inputChannel,
						Status:       re.EnabledStatus,
						Logic: re.Script{
							Type:  re.LuaType,
							Value: "return false",
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
			desc: "consume message with Lua script with no outputs",
			message: &messaging.Message{
				Channel: inputChannel,
				Created: now.Unix(),
				Payload: []byte(`{"temperature": 25.5}`),
			},
			page: re.Page{
				Rules: []re.Rule{
					{
						ID:           testsutil.GenerateUUID(t),
						Name:         namegen.Generate(),
						InputChannel: inputChannel,
						Status:       re.EnabledStatus,
						Logic: re.Script{
							Type:  re.LuaType,
							Value: "return message.payload",
						},
						Outputs:  re.Outputs{},
						Schedule: schedule,
					},
				},
			},
			listErr: nil,
		},
		{
			desc: "consume message with Lua script returning nil",
			message: &messaging.Message{
				Channel: inputChannel,
				Created: now.Unix(),
				Payload: []byte(`{"temperature": 25.5}`),
			},
			page: re.Page{
				Rules: []re.Rule{
					{
						ID:           testsutil.GenerateUUID(t),
						Name:         namegen.Generate(),
						InputChannel: inputChannel,
						Status:       re.EnabledStatus,
						Logic: re.Script{
							Type:  re.LuaType,
							Value: "return nil",
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
			desc: "consume message with Lua script with invalid syntax",
			message: &messaging.Message{
				Channel: inputChannel,
				Created: now.Unix(),
				Payload: []byte(`{"temperature": 25.5}`),
			},
			page: re.Page{
				Rules: []re.Rule{
					{
						ID:           testsutil.GenerateUUID(t),
						Name:         namegen.Generate(),
						InputChannel: inputChannel,
						Status:       re.EnabledStatus,
						Logic: re.Script{
							Type:  re.LuaType,
							Value: "invalid lua syntax {{{",
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
			desc: "consume message with Lua script and Alarm output",
			message: &messaging.Message{
				Channel: inputChannel,
				Created: now.Unix(),
				Payload: []byte(`{"temperature": 30.5}`),
			},
			page: re.Page{
				Rules: []re.Rule{
					{
						ID:           testsutil.GenerateUUID(t),
						Name:         namegen.Generate(),
						InputChannel: inputChannel,
						Status:       re.EnabledStatus,
						Logic: re.Script{
							Type:  re.LuaType,
							Value: `return {severity = 2, description = "High temperature"}`,
						},
						Outputs: re.Outputs{
							&outputs.Alarm{
								RuleID: testsutil.GenerateUUID(t),
							},
						},
						Schedule: schedule,
					},
				},
			},
			listErr: nil,
		},
		{
			desc: "consume message with Lua script and SenML output",
			message: &messaging.Message{
				Channel: inputChannel,
				Created: now.Unix(),
				Payload: []byte(`{"temperature": 25.5}`),
			},
			page: re.Page{
				Rules: []re.Rule{
					{
						ID:           testsutil.GenerateUUID(t),
						Name:         namegen.Generate(),
						InputChannel: inputChannel,
						Status:       re.EnabledStatus,
						Logic: re.Script{
							Type:  re.LuaType,
							Value: `return {bn = "sensor1", n = "temperature", v = 25.5}`,
						},
						Outputs: re.Outputs{
							&outputs.SenML{},
						},
						Schedule: schedule,
					},
				},
			},
			listErr: nil,
		},
		{
			desc: "consume message with Lua script and Email output",
			message: &messaging.Message{
				Channel: inputChannel,
				Created: now.Unix(),
				Payload: []byte(`{"temperature": 25.5}`),
			},
			page: re.Page{
				Rules: []re.Rule{
					{
						ID:           testsutil.GenerateUUID(t),
						Name:         namegen.Generate(),
						InputChannel: inputChannel,
						Status:       re.EnabledStatus,
						Logic: re.Script{
							Type:  re.LuaType,
							Value: `return message.payload`,
						},
						Outputs: re.Outputs{
							&outputs.Email{
								To:      []string{"test@example.com"},
								Subject: "Temperature Alert",
								Content: "Temperature: {{.Result}}",
							},
						},
						Schedule: schedule,
					},
				},
			},
			listErr: nil,
		},
		{
			desc: "consume message with rules using GoType",
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
							Type:  re.GoType,
							Value: "func() bool { return true }",
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
			desc: "consume message with GoType logic returning false",
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
							Type:  re.GoType,
							Value: "func() bool { return false }",
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
			desc: "consume message with GoType invalid logic value",
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
							Type:  re.GoType,
							Value: "invalid go code {{{",
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
			desc: "consume message with GoType missing logicFunction",
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
							Type:  re.GoType,
							Value: "func someOtherFunc() bool { return true }",
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
			desc: "consume message with GoType invalid function signature",
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
							Type:  re.GoType,
							Value: "var logicFunction = 42",
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
			desc: "consume message with GoType function logicFunction properly named",
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
							Type:  re.GoType,
							Value: "func logicFunction() any { return true }",
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
			desc: "consume message with GoType returning non-bool",
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
							Type:  re.GoType,
							Value: "func() any { return \"not a bool\" }",
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
			desc: "consume message with GoType and JSON payload",
			message: &messaging.Message{
				Channel: inputChannel,
				Created: now.Unix(),
				Payload: []byte(`{"temperature": 25, "humidity": 60}`),
			},
			page: re.Page{
				Rules: []re.Rule{
					{
						ID:           testsutil.GenerateUUID(t),
						Name:         namegen.Generate(),
						InputChannel: inputChannel,
						Status:       re.EnabledStatus,
						Logic: re.Script{
							Type:  re.GoType,
							Value: "func() bool { return true }",
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
			desc: "consume message with GoType and invalid JSON payload",
			message: &messaging.Message{
				Channel: inputChannel,
				Created: now.Unix(),
				Payload: []byte(`invalid json {{{`),
			},
			page: re.Page{
				Rules: []re.Rule{
					{
						ID:           testsutil.GenerateUUID(t),
						Name:         namegen.Generate(),
						InputChannel: inputChannel,
						Status:       re.EnabledStatus,
						Logic: re.Script{
							Type:  re.GoType,
							Value: "func() bool { return true }",
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
			desc: "consume message with Lua script and Postgres output",
			message: &messaging.Message{
				Channel: inputChannel,
				Created: now.Unix(),
				Payload: []byte(`{"temperature": 25.5, "humidity": 60}`),
			},
			page: re.Page{
				Rules: []re.Rule{
					{
						ID:           testsutil.GenerateUUID(t),
						Name:         namegen.Generate(),
						InputChannel: inputChannel,
						Status:       re.EnabledStatus,
						Logic: re.Script{
							Type:  re.LuaType,
							Value: `return message.payload`,
						},
						Outputs: re.Outputs{
							&outputs.Postgres{
								Host:     "localhost",
								Port:     5432,
								User:     "test",
								Password: "test",
								Database: "testdb",
								Table:    "sensor_data",
								Mapping:  `{"temperature": {{.Result.temperature}}, "humidity": {{.Result.humidity}}}`,
							},
						},
						Schedule: schedule,
					},
				},
			},
			listErr: nil,
		},
		{
			desc: "consume message with Lua script and Slack output",
			message: &messaging.Message{
				Channel: inputChannel,
				Created: now.Unix(),
				Payload: []byte(`{"temperature": 25.5}`),
			},
			page: re.Page{
				Rules: []re.Rule{
					{
						ID:           testsutil.GenerateUUID(t),
						Name:         namegen.Generate(),
						InputChannel: inputChannel,
						Status:       re.EnabledStatus,
						Logic: re.Script{
							Type:  re.LuaType,
							Value: `return message.payload`,
						},
						Outputs: re.Outputs{
							&outputs.Slack{
								Token:     "xoxb-test-token",
								ChannelID: "C12345678",
								Message:   `{"text": "Temperature: {{.Result.temperature}}"}`,
							},
						},
						Schedule: schedule,
					},
				},
			},
			listErr: nil,
		},
		{
			desc: "consume message with Lua script and unknown output type",
			message: &messaging.Message{
				Channel: inputChannel,
				Created: now.Unix(),
				Payload: []byte(`{"temperature": 25.5}`),
			},
			page: re.Page{
				Rules: []re.Rule{
					{
						ID:           testsutil.GenerateUUID(t),
						Name:         namegen.Generate(),
						InputChannel: inputChannel,
						Status:       re.EnabledStatus,
						Logic: re.Script{
							Type:  re.LuaType,
							Value: `return message.payload`,
						},
						Outputs: re.Outputs{
							&unknownOutput{},
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
			repoCall1 := pubmocks.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return(tc.publishErr).Maybe()
			repoCall2 := emailer.On("SendEmailNotification", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

			err = svc.Handle(tc.message)
			assert.Nil(t, err)

			time.Sleep(100 * time.Millisecond)

			assert.True(t, errors.Contains(err, tc.listErr), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.listErr, err))

			repoCall.Unset()
			repoCall1.Unset()
			repoCall2.Unset()
		})
	}
}

func TestStartScheduler(t *testing.T) {
	now := time.Now().Truncate(time.Minute)
	ri := make(chan pkglog.RunInfo)
	svc, repo, _, ticker, _ := newService(t, ri)

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

	schedulerCases := []struct {
		desc            string
		rules           []re.Rule
		listErr         error
		updateDueErr    error
		expectedRunInfo int
	}{
		{
			desc: "start scheduler with successful rule processing",
			rules: []re.Rule{
				{
					ID:           testsutil.GenerateUUID(t),
					Name:         namegen.Generate(),
					DomainID:     domainID,
					InputChannel: inputChannel,
					Status:       re.EnabledStatus,
					Schedule: pkgSch.Schedule{
						StartDateTime:   now.Add(-time.Hour),
						Time:            now.Add(time.Hour),
						Recurring:       pkgSch.Daily,
						RecurringPeriod: 1,
					},
					Logic: re.Script{
						Type:  re.LuaType,
						Value: "return true",
					},
				},
			},
			listErr:         nil,
			updateDueErr:    nil,
			expectedRunInfo: 1,
		},
		{
			desc: "start scheduler with multiple rules",
			rules: []re.Rule{
				{
					ID:           testsutil.GenerateUUID(t),
					Name:         namegen.Generate(),
					DomainID:     domainID,
					InputChannel: inputChannel,
					Status:       re.EnabledStatus,
					Schedule: pkgSch.Schedule{
						StartDateTime:   now.Add(-time.Hour),
						Time:            now.Add(time.Hour),
						Recurring:       pkgSch.Daily,
						RecurringPeriod: 1,
					},
					Logic: re.Script{
						Type:  re.LuaType,
						Value: "return true",
					},
				},
				{
					ID:           testsutil.GenerateUUID(t),
					Name:         namegen.Generate(),
					DomainID:     domainID,
					InputChannel: inputChannel,
					Status:       re.EnabledStatus,
					Schedule: pkgSch.Schedule{
						StartDateTime:   now.Add(-time.Hour),
						Time:            now.Add(time.Hour),
						Recurring:       pkgSch.Weekly,
						RecurringPeriod: 1,
					},
					Logic: re.Script{
						Type:  re.GoType,
						Value: "func() bool { return true }",
					},
				},
			},
			listErr:         nil,
			updateDueErr:    nil,
			expectedRunInfo: 2,
		},
		{
			desc:            "start scheduler with list rules error",
			rules:           []re.Rule{},
			listErr:         repoerr.ErrViewEntity,
			updateDueErr:    nil,
			expectedRunInfo: 1,
		},
		{
			desc: "start scheduler with update due error",
			rules: []re.Rule{
				{
					ID:           testsutil.GenerateUUID(t),
					Name:         namegen.Generate(),
					DomainID:     domainID,
					InputChannel: inputChannel,
					Status:       re.EnabledStatus,
					Schedule: pkgSch.Schedule{
						StartDateTime:   now.Add(-time.Hour),
						Time:            now.Add(time.Hour),
						Recurring:       pkgSch.Daily,
						RecurringPeriod: 1,
					},
					Logic: re.Script{
						Type:  re.LuaType,
						Value: "return true",
					},
				},
			},
			listErr:         nil,
			updateDueErr:    repoerr.ErrUpdateEntity,
			expectedRunInfo: 1,
		},
	}

	for _, tc := range schedulerCases {
		t.Run(tc.desc, func(t *testing.T) {
			page := re.Page{
				Rules: tc.rules,
				Total: uint64(len(tc.rules)),
			}

			repoCall := repo.On("ListRules", mock.Anything, mock.Anything).Return(page, tc.listErr)
			repoCall2 := repo.On("UpdateRuleDue", mock.Anything, mock.Anything, mock.Anything).Return(re.Rule{}, tc.updateDueErr)
			tickChan := make(chan time.Time, 1)
			tickCall := ticker.On("Tick").Return((<-chan time.Time)(tickChan))
			tickCall1 := ticker.On("Stop").Return()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			go func() {
				_ = svc.StartScheduler(ctx)
			}()

			tickChan <- now

			collected := 0
			timeout := time.After(500 * time.Millisecond)
			for collected < tc.expectedRunInfo {
				select {
				case info := <-ri:
					collected++
					if tc.listErr != nil {
						assert.Equal(t, slog.LevelError, info.Level)
						assert.Contains(t, info.Message, "failed to list rules")
					} else if tc.updateDueErr != nil {
						assert.Equal(t, slog.LevelError, info.Level)
						assert.Contains(t, info.Message, "failed to update rule")
					} else {
						assert.True(t, info.Level == slog.LevelInfo || info.Level == slog.LevelWarn || info.Level == slog.LevelError)
					}
				case <-timeout:
					t.Fatalf("timeout waiting for runInfo messages, expected %d got %d", tc.expectedRunInfo, collected)
				}
			}

			cancel()
			time.Sleep(50 * time.Millisecond)

			repoCall.Unset()
			repoCall2.Unset()
			tickCall.Unset()
			tickCall1.Unset()
		})
	}
}
