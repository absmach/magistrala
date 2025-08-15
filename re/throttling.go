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

// ThrottledHandler wraps a service with throttling capabilities to prevent loops
type ThrottledHandler struct {
	svc         Service
	rateLimiter *rate.Limiter
	backoff     *BackoffManager
	metrics     *ThrottlingMetrics
	logger      *slog.Logger

	// Circuit breaker state
	failures    int
	lastFailure time.Time
	maxFailures int
	resetTime   time.Duration
	mutex       sync.RWMutex
}

// BackoffManager handles exponential backoff for failed processing
type BackoffManager struct {
	initial    time.Duration
	max        time.Duration
	multiplier float64
	attempts   map[string]int
	lastTry    map[string]time.Time
	mutex      sync.RWMutex
}

// ThrottlingMetrics tracks performance and throttling statistics
type ThrottlingMetrics struct {
	ProcessedMessages   int64
	ThrottledMessages   int64
	FailedMessages      int64
	AverageProcessTime  time.Duration
	CircuitBreakerTrips int64
	mutex               sync.RWMutex
}

type ThrottlingConfig struct {
	RateLimit      int           // Messages per second
	MaxPending     int           // Maximum pending messages
	BackoffInitial time.Duration // Initial backoff delay
	BackoffMax     time.Duration // Maximum backoff delay
	MaxFailures    int           // Circuit breaker failure threshold
	ResetTime      time.Duration // Circuit breaker reset time
}

func NewThrottledHandler(svc Service, config ThrottlingConfig, logger *slog.Logger) *ThrottledHandler {
	return &ThrottledHandler{
		svc:         svc,
		rateLimiter: rate.NewLimiter(rate.Limit(config.RateLimit), config.RateLimit),
		backoff: &BackoffManager{
			initial:    config.BackoffInitial,
			max:        config.BackoffMax,
			multiplier: 2.0,
			attempts:   make(map[string]int),
			lastTry:    make(map[string]time.Time),
		},
		metrics:     &ThrottlingMetrics{},
		logger:      logger,
		maxFailures: config.MaxFailures,
		resetTime:   config.ResetTime,
	}
}

func (th *ThrottledHandler) Handle(msg *messaging.Message) error {
	if th.isCircuitOpen() {
		th.incrementThrottled()
		th.logger.Warn("Circuit breaker open, dropping message",
			slog.String("channel", msg.Channel),
			slog.String("subtopic", msg.Subtopic))
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := th.rateLimiter.Wait(ctx); err != nil {
		th.incrementThrottled()
		th.logger.Warn("Rate limit exceeded, dropping message",
			slog.String("channel", msg.Channel),
			slog.String("subtopic", msg.Subtopic))
		return nil
	}

	msgKey := th.generateMessageKey(msg)

	// Check if we should apply backoff for this message pattern
	if th.shouldBackoff(msgKey) {
		th.incrementThrottled()
		th.logger.Debug("Applying backoff, dropping message",
			slog.String("message_key", msgKey),
			slog.Duration("backoff_time", th.getBackoffDelay(msgKey)))
		return nil
	}

	start := time.Now().UTC()
	err := th.svc.Handle(msg)
	processingTime := time.Since(start)

	if err != nil {
		th.handleFailure(msgKey, err)
		th.incrementFailed()
	} else {
		th.handleSuccess(msgKey)
		th.incrementProcessed()
		th.updateProcessingTime(processingTime)
	}

	return err
}

func (th *ThrottledHandler) generateMessageKey(msg *messaging.Message) string {
	return msg.Domain + ":" + msg.Channel + ":" + msg.Subtopic
}

func (th *ThrottledHandler) shouldBackoff(msgKey string) bool {
	th.backoff.mutex.RLock()
	defer th.backoff.mutex.RUnlock()

	attempts, exists := th.backoff.attempts[msgKey]
	if !exists || attempts == 0 {
		return false
	}

	lastTry, exists := th.backoff.lastTry[msgKey]
	if !exists {
		return false
	}

	backoffDelay := th.calculateBackoffDelay(attempts)
	return time.Since(lastTry) < backoffDelay
}

func (th *ThrottledHandler) getBackoffDelay(msgKey string) time.Duration {
	th.backoff.mutex.RLock()
	defer th.backoff.mutex.RUnlock()

	attempts := th.backoff.attempts[msgKey]
	return th.calculateBackoffDelay(attempts)
}

func (th *ThrottledHandler) calculateBackoffDelay(attempts int) time.Duration {
	if attempts == 0 {
		return 0
	}

	delay := th.backoff.initial
	for i := 1; i < attempts; i++ {
		delay = time.Duration(float64(delay) * th.backoff.multiplier)
		if delay > th.backoff.max {
			delay = th.backoff.max
			break
		}
	}
	return delay
}

func (th *ThrottledHandler) handleFailure(msgKey string, err error) {
	th.backoff.mutex.Lock()
	defer th.backoff.mutex.Unlock()

	th.backoff.attempts[msgKey]++
	th.backoff.lastTry[msgKey] = time.Now().UTC()

	th.mutex.Lock()
	th.failures++
	th.lastFailure = time.Now().UTC()
	th.mutex.Unlock()

	th.logger.Error("Message processing failed",
		slog.String("message_key", msgKey),
		slog.Int("attempts", th.backoff.attempts[msgKey]),
		slog.Any("error", err))
}

func (th *ThrottledHandler) handleSuccess(msgKey string) {
	th.backoff.mutex.Lock()
	defer th.backoff.mutex.Unlock()

	delete(th.backoff.attempts, msgKey)
	delete(th.backoff.lastTry, msgKey)

	th.mutex.Lock()
	th.failures = 0
	th.mutex.Unlock()
}

func (th *ThrottledHandler) isCircuitOpen() bool {
	th.mutex.RLock()
	defer th.mutex.RUnlock()

	if th.failures < th.maxFailures {
		return false
	}

	return time.Since(th.lastFailure) < th.resetTime
}

func (th *ThrottledHandler) incrementProcessed() {
	th.metrics.mutex.Lock()
	defer th.metrics.mutex.Unlock()
	th.metrics.ProcessedMessages++
}

func (th *ThrottledHandler) incrementThrottled() {
	th.metrics.mutex.Lock()
	defer th.metrics.mutex.Unlock()
	th.metrics.ThrottledMessages++
}

func (th *ThrottledHandler) incrementFailed() {
	th.metrics.mutex.Lock()
	defer th.metrics.mutex.Unlock()
	th.metrics.FailedMessages++
}

func (th *ThrottledHandler) updateProcessingTime(duration time.Duration) {
	th.metrics.mutex.Lock()
	defer th.metrics.mutex.Unlock()

	if th.metrics.ProcessedMessages == 1 {
		th.metrics.AverageProcessTime = duration
	} else {
		th.metrics.AverageProcessTime = time.Duration(
			(int64(th.metrics.AverageProcessTime) + int64(duration)) / 2)
	}
}

func (th *ThrottledHandler) GetMetrics() ThrottlingMetrics {
	th.metrics.mutex.RLock()
	defer th.metrics.mutex.RUnlock()

	return ThrottlingMetrics{
		ProcessedMessages:   th.metrics.ProcessedMessages,
		ThrottledMessages:   th.metrics.ThrottledMessages,
		FailedMessages:      th.metrics.FailedMessages,
		AverageProcessTime:  th.metrics.AverageProcessTime,
		CircuitBreakerTrips: th.metrics.CircuitBreakerTrips,
	}
}

func (th *ThrottledHandler) Cleanup() {
	th.backoff.mutex.Lock()
	defer th.backoff.mutex.Unlock()

	cutoff := time.Now().UTC().Add(-th.backoff.max * 2)

	for key, lastTry := range th.backoff.lastTry {
		if lastTry.Before(cutoff) {
			delete(th.backoff.attempts, key)
			delete(th.backoff.lastTry, key)
		}
	}
}

func (th *ThrottledHandler) StartCleanupTask(ctx context.Context) {
	ticker := time.NewTicker(th.backoff.max)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			th.Cleanup()

			metrics := th.GetMetrics()
			th.logger.Info("Throttling metrics",
				slog.Int64("processed", metrics.ProcessedMessages),
				slog.Int64("throttled", metrics.ThrottledMessages),
				slog.Int64("failed", metrics.FailedMessages),
				slog.Duration("avg_process_time", metrics.AverageProcessTime))
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
