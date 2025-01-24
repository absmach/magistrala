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
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/re"
	"github.com/absmach/magistrala/re/mocks"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	pubsubmocks "github.com/absmach/supermq/pkg/messaging/mocks"
	"github.com/absmach/supermq/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	namegen = namegenerator.NewGenerator()
	rule    = re.Rule{
		ID:           testsutil.GenerateUUID(&testing.T{}),
		Name:         namegen.Generate(),
		InputChannel: "test.channel",
		Status:       re.EnabledStatus,
		Schedule: re.Schedule{
			StartDateTime:   time.Now().Add(-time.Hour), // Started an hour ago
			Recurring:       re.Daily,
			RecurringPeriod: 1,
			Time:            time.Now().Add(-time.Hour),
		},
	}
	futureRule = re.Rule{
		ID:           testsutil.GenerateUUID(&testing.T{}),
		Name:         namegen.Generate(),
		InputChannel: "test.channel",
		Status:       re.EnabledStatus,
		Schedule: re.Schedule{
			StartDateTime: time.Now().Add(24 * time.Hour),
			Recurring:     re.None,
		},
	}
)

func newService(t *testing.T) (re.Service, *mocks.Repository, *mocks.Ticker) {
	repo := new(mocks.Repository)
	mockTicker := new(mocks.Ticker)
	idProvider := uuid.NewMock()
	pubsub := pubsubmocks.NewPubSub(t)
	return re.NewService(repo, idProvider, pubsub, mockTicker), repo, mockTicker
}

func TestStartScheduler(t *testing.T) {
	now := time.Now().Truncate(time.Minute)
	svc, repo, ticker := newService(t)

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
			case "start scheduler successfully processes rules":
				tickChan <- time.Now()
				time.Sleep(100 * time.Millisecond)
				cancel()
			case "start scheduler with rule to be run in the future":
				tickChan <- time.Now()
				time.Sleep(100 * time.Millisecond)
				cancel()
			case "start scheduler with list error":
				tickChan <- time.Now()
				time.Sleep(100 * time.Millisecond)
				if err := svc.Errors(); err != nil {
					cancel()
				}
			}

			err := <-errc
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v but got %v", tc.err, err))
			repoCall.Unset()
			tickCall.Unset()
			tickCall1.Unset()
		})
	}
}
