// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/messaging"
	"golang.org/x/time/rate"
)

type ThrottlingConfig struct {
	RateLimit     int
	LoopThreshold int
	LoopWindow    time.Duration
}

type ThrottledHandler struct {
	svc         Service
	rateLimiter *rate.Limiter
	logger      *slog.Logger

	messageCount map[string]int
	lastSeen     map[string]time.Time
	threshold    int
	window       time.Duration
	mutex        sync.RWMutex
}

func NewThrottledHandler(svc Service, config ThrottlingConfig, logger *slog.Logger) *ThrottledHandler {
	return &ThrottledHandler{
		svc:          svc,
		rateLimiter:  rate.NewLimiter(rate.Limit(config.RateLimit), config.RateLimit),
		logger:       logger,
		messageCount: make(map[string]int),
		lastSeen:     make(map[string]time.Time),
		threshold:    config.LoopThreshold,
		window:       config.LoopWindow,
	}
}

func (th *ThrottledHandler) Handle(msg *messaging.Message) error {
	if !th.rateLimiter.Allow() {
		th.logger.Warn("Rate limit exceeded, dropping message",
			slog.String("channel", msg.Channel),
			slog.String("subtopic", msg.Subtopic))
		return nil
	}

	msgKey := msg.Domain + ":" + msg.Channel + ":" + msg.Subtopic
	if th.isLoop(msgKey) {
		th.logger.Warn("Potential loop detected, dropping message",
			slog.String("message_key", msgKey))
		return nil
	}

	return th.svc.Handle(msg)
}

func (th *ThrottledHandler) isLoop(msgKey string) bool {
	th.mutex.Lock()
	defer th.mutex.Unlock()

	now := time.Now()
	lastTime, exists := th.lastSeen[msgKey]

	if !exists || now.Sub(lastTime) > th.window {
		th.messageCount[msgKey] = 1
		th.lastSeen[msgKey] = now
		return false
	}

	th.messageCount[msgKey]++
	th.lastSeen[msgKey] = now

	if th.messageCount[msgKey] > th.threshold {
		th.logger.Warn("Loop threshold exceeded",
			slog.String("message_key", msgKey),
			slog.Int("count", th.messageCount[msgKey]),
			slog.Int("threshold", th.threshold))
		return true
	}

	return false
}

func (th *ThrottledHandler) Cleanup() {
	th.mutex.Lock()
	defer th.mutex.Unlock()

	cutoff := time.Now().Add(-th.window * 2)
	for key, lastTime := range th.lastSeen {
		if lastTime.Before(cutoff) {
			delete(th.messageCount, key)
			delete(th.lastSeen, key)
		}
	}
}

func (th *ThrottledHandler) StartCleanupTask(ctx context.Context) {
	ticker := time.NewTicker(th.window)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			th.Cleanup()
		}
	}
}

func (th *ThrottledHandler) AddRule(ctx context.Context, session authn.Session, rule Rule) (Rule, error) {
	return th.svc.AddRule(ctx, session, rule)
}

func (th *ThrottledHandler) ViewRule(ctx context.Context, session authn.Session, id string) (Rule, error) {
	return th.svc.ViewRule(ctx, session, id)
}

func (th *ThrottledHandler) ListRules(ctx context.Context, session authn.Session, pm PageMeta) (Page, error) {
	return th.svc.ListRules(ctx, session, pm)
}

func (th *ThrottledHandler) UpdateRule(ctx context.Context, session authn.Session, rule Rule) (Rule, error) {
	return th.svc.UpdateRule(ctx, session, rule)
}

func (th *ThrottledHandler) UpdateRuleTags(ctx context.Context, session authn.Session, rule Rule) (Rule, error) {
	return th.svc.UpdateRuleTags(ctx, session, rule)
}

func (th *ThrottledHandler) UpdateRuleSchedule(ctx context.Context, session authn.Session, rule Rule) (Rule, error) {
	return th.svc.UpdateRuleSchedule(ctx, session, rule)
}

func (th *ThrottledHandler) RemoveRule(ctx context.Context, session authn.Session, id string) error {
	return th.svc.RemoveRule(ctx, session, id)
}

func (th *ThrottledHandler) EnableRule(ctx context.Context, session authn.Session, id string) (Rule, error) {
	return th.svc.EnableRule(ctx, session, id)
}

func (th *ThrottledHandler) DisableRule(ctx context.Context, session authn.Session, id string) (Rule, error) {
	return th.svc.DisableRule(ctx, session, id)
}

func (th *ThrottledHandler) StartScheduler(ctx context.Context) error {
	go th.StartCleanupTask(ctx)
	return th.svc.StartScheduler(ctx)
}

func (th *ThrottledHandler) Cancel() error {
	return th.svc.Cancel()
}
