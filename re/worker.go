// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"context"
	"sync"

	"github.com/absmach/supermq/pkg/messaging"
)

// WorkerMessage represents a message to be processed by a rule worker.
type WorkerMessage struct {
	Message *messaging.Message
	Rule    Rule
}

// RuleWorker manages execution of a single rule in its own goroutine.
type RuleWorker struct {
	rule     Rule
	engine   *re
	msgChan  chan WorkerMessage
	stopChan chan struct{}
	doneChan chan struct{}
	running  bool
	mu       sync.RWMutex
}

// NewRuleWorker creates a new rule worker for the given rule.
func NewRuleWorker(rule Rule, engine *re) *RuleWorker {
	return &RuleWorker{
		rule:     rule,
		engine:   engine,
		msgChan:  make(chan WorkerMessage, 100), // Buffer to prevent blocking
		stopChan: make(chan struct{}),
		doneChan: make(chan struct{}),
		running:  false,
	}
}

// Start begins the worker goroutine for processing messages.
func (w *RuleWorker) Start(ctx context.Context) {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return
	}
	w.running = true
	w.mu.Unlock()

	go w.run(ctx)
}

// Stop stops the worker goroutine and waits for it to finish.
func (w *RuleWorker) Stop() {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return
	}
	w.mu.Unlock()

	close(w.stopChan)
	<-w.doneChan
}

// Send sends a message to the worker for processing.
func (w *RuleWorker) Send(msg WorkerMessage) bool {
	w.mu.RLock()
	running := w.running
	w.mu.RUnlock()

	if !running {
		return false
	}

	select {
	case w.msgChan <- msg:
		return true
	default:
		return false
	}
}

// IsRunning returns true if the worker is currently running.
func (w *RuleWorker) IsRunning() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.running
}

// UpdateRule updates the rule configuration for this worker.
func (w *RuleWorker) UpdateRule(rule Rule) {
	w.mu.Lock()
	w.rule = rule
	w.mu.Unlock()
}

// GetRule returns the current rule configuration.
func (w *RuleWorker) GetRule() Rule {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.rule
}

// run is the main worker loop that processes messages.
func (w *RuleWorker) run(ctx context.Context) {
	defer func() {
		w.mu.Lock()
		w.running = false
		w.mu.Unlock()
		close(w.doneChan)
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopChan:
			return
		case workerMsg := <-w.msgChan:
			w.processMessage(ctx, workerMsg)
		}
	}
}

// processMessage processes a single message using the rule logic.
func (w *RuleWorker) processMessage(ctx context.Context, workerMsg WorkerMessage) {
	currentRule := w.GetRule()
	
	if currentRule.Status != EnabledStatus {
		return
	}

	runInfo := w.engine.process(ctx, currentRule, workerMsg.Message)
	
	// Send run info to the logging channel
	select {
	case w.engine.runInfo <- runInfo:
	default:
	}
}

// WorkerManager manages all rule workers.
type WorkerManager struct {
	workers map[string]*RuleWorker
	engine  *re
	mu      sync.RWMutex
}

// NewWorkerManager creates a new worker manager.
func NewWorkerManager(engine *re) *WorkerManager {
	return &WorkerManager{
		workers: make(map[string]*RuleWorker),
		engine:  engine,
	}
}

// AddWorker adds a new worker for the given rule.
func (wm *WorkerManager) AddWorker(ctx context.Context, rule Rule) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if existing, ok := wm.workers[rule.ID]; ok {
		existing.Stop()
	}

	if rule.Status != EnabledStatus {
		delete(wm.workers, rule.ID)
		return
	}

	worker := NewRuleWorker(rule, wm.engine)
	worker.Start(ctx)
	wm.workers[rule.ID] = worker
}

// RemoveWorker removes and stops the worker for the given rule ID.
func (wm *WorkerManager) RemoveWorker(ruleID string) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if worker, ok := wm.workers[ruleID]; ok {
		worker.Stop()
		delete(wm.workers, ruleID)
	}
}

// UpdateWorker updates the rule configuration for an existing worker.
func (wm *WorkerManager) UpdateWorker(ctx context.Context, rule Rule) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if rule.Status != EnabledStatus {
		if worker, ok := wm.workers[rule.ID]; ok {
			worker.Stop()
			delete(wm.workers, rule.ID)
		}
		return
	}

	if worker, ok := wm.workers[rule.ID]; ok {
		worker.UpdateRule(rule)
	} else {
		worker := NewRuleWorker(rule, wm.engine)
		worker.Start(ctx)
		wm.workers[rule.ID] = worker
	}
}

// SendMessage sends a message to the appropriate worker for processing.
func (wm *WorkerManager) SendMessage(msg *messaging.Message, rule Rule) bool {
	wm.mu.RLock()
	worker, ok := wm.workers[rule.ID]
	wm.mu.RUnlock()

	if !ok || !worker.IsRunning() {
		return false
	}

	return worker.Send(WorkerMessage{
		Message: msg,
		Rule:    rule,
	})
}

// StopAll stops all workers.
func (wm *WorkerManager) StopAll() {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	for _, worker := range wm.workers {
		worker.Stop()
	}
	wm.workers = make(map[string]*RuleWorker)
}

// GetWorkerCount returns the number of active workers.
func (wm *WorkerManager) GetWorkerCount() int {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	return len(wm.workers)
}

// ListWorkers returns a slice of rule IDs that have active workers.
func (wm *WorkerManager) ListWorkers() []string {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	ruleIDs := make([]string, 0, len(wm.workers))
	for ruleID := range wm.workers {
		ruleIDs = append(ruleIDs, ruleID)
	}
	return ruleIDs
}

// RefreshWorkers synchronizes workers with the current set of enabled rules.
func (wm *WorkerManager) RefreshWorkers(ctx context.Context, rules []Rule) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	currentRules := make(map[string]Rule)
	for _, rule := range rules {
		if rule.Status == EnabledStatus {
			currentRules[rule.ID] = rule
		}
	}

	for ruleID, worker := range wm.workers {
		if _, exists := currentRules[ruleID]; !exists {
			worker.Stop()
			delete(wm.workers, ruleID)
		}
	}

	for ruleID, rule := range currentRules {
		if worker, exists := wm.workers[ruleID]; exists {
			worker.UpdateRule(rule)
		} else {
			worker := NewRuleWorker(rule, wm.engine)
			worker.Start(ctx)
			wm.workers[ruleID] = worker
		}
	}
}
