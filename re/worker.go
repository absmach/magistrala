// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"context"
	"sync/atomic"

	"github.com/absmach/supermq/pkg/messaging"
	"golang.org/x/sync/errgroup"
)

// WorkerMessage represents a message to be processed by a rule worker.
type WorkerMessage struct {
	Message *messaging.Message
	Rule    Rule
}

// RuleWorker manages execution of a single rule in its own goroutine.
type RuleWorker struct {
	rule       Rule
	engine     *re
	msgChan    chan WorkerMessage
	updateChan chan Rule
	ctx        context.Context
	cancel     context.CancelFunc
	g          *errgroup.Group
	running    int32
}

// NewRuleWorker creates a new rule worker for the given rule.
func NewRuleWorker(rule Rule, engine *re) *RuleWorker {
	return &RuleWorker{
		rule:       rule,
		engine:     engine,
		msgChan:    make(chan WorkerMessage, 100),
		updateChan: make(chan Rule, 1),
		running:    0, // 0 = not running, 1 = running
	}
}

// Start begins the worker goroutine for processing messages.
func (w *RuleWorker) Start(ctx context.Context) {
	if !atomic.CompareAndSwapInt32(&w.running, 0, 1) {
		return
	}

	w.ctx, w.cancel = context.WithCancel(ctx)
	w.g, w.ctx = errgroup.WithContext(w.ctx)

	w.g.Go(func() error {
		return w.run(w.ctx)
	})
}

// Stop stops the worker goroutine and waits for it to finish.
func (w *RuleWorker) Stop() error {
	if !atomic.CompareAndSwapInt32(&w.running, 1, 0) {
		return nil
	}

	w.cancel()

	return w.g.Wait()
}

// Send sends a message to the worker for processing.
func (w *RuleWorker) Send(msg WorkerMessage) bool {
	if atomic.LoadInt32(&w.running) == 0 {
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
	return atomic.LoadInt32(&w.running) == 1
}

// UpdateRule updates the rule configuration for this worker.
func (w *RuleWorker) UpdateRule(rule Rule) {
	select {
	case w.updateChan <- rule:
	default:
		// If channel is full, just overwrite the current rule
		// This ensures we always have the latest rule
		select {
		case <-w.updateChan: // drain the channel
		default:
		}
		w.updateChan <- rule
	}
}

// GetRule returns the current rule configuration.
func (w *RuleWorker) GetRule() Rule {
	return w.rule // Since rule updates happen via channels in the worker loop, this is safe
}

// run is the main worker loop that processes messages.
func (w *RuleWorker) run(ctx context.Context) error {
	defer func() {
		atomic.StoreInt32(&w.running, 0)
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case rule := <-w.updateChan:
			// Update the rule configuration
			w.rule = rule
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

	select {
	case w.engine.runInfo <- runInfo:
	default:
	}
}

// WorkerManagerCommand represents commands for worker management
type WorkerManagerCommand struct {
	Type     string // "add", "remove", "update", "send", "stop_all"
	Rule     Rule
	RuleID   string
	Message  *messaging.Message
	Response chan interface{} // For responses (e.g., SendMessage result)
}

// WorkerManager manages all rule workers using channels instead of mutex
type WorkerManager struct {
	workers   map[string]*RuleWorker
	engine    *re
	g         *errgroup.Group
	ctx       context.Context
	commandCh chan WorkerManagerCommand
	running   int32
}

// NewWorkerManager creates a new worker manager.
func NewWorkerManager(engine *re, ctx context.Context) *WorkerManager {
	g, ctx := errgroup.WithContext(ctx)
	wm := &WorkerManager{
		workers:   make(map[string]*RuleWorker),
		engine:    engine,
		g:         g,
		ctx:       ctx,
		commandCh: make(chan WorkerManagerCommand, 100),
		running:   0,
	}
	
	// Start the worker manager goroutine
	wm.g.Go(func() error {
		return wm.manageWorkers(ctx)
	})
	
	atomic.StoreInt32(&wm.running, 1)
	return wm
}

// manageWorkers is the main loop that handles all worker management operations
func (wm *WorkerManager) manageWorkers(ctx context.Context) error {
	defer atomic.StoreInt32(&wm.running, 0)
	
	for {
		select {
		case <-ctx.Done():
			// Stop all workers before exiting
			for _, worker := range wm.workers {
				worker.Stop()
			}
			wm.workers = make(map[string]*RuleWorker)
			return ctx.Err()
			
		case cmd := <-wm.commandCh:
			wm.handleCommand(cmd)
		}
	}
}

// handleCommand processes worker management commands
func (wm *WorkerManager) handleCommand(cmd WorkerManagerCommand) {
	switch cmd.Type {
	case "add":
		wm.addWorkerUnsafe(cmd.Rule)
	case "remove":
		wm.removeWorkerUnsafe(cmd.RuleID)
	case "update":
		wm.updateWorkerUnsafe(cmd.Rule)
	case "send":
		result := wm.sendMessageUnsafe(cmd.Message, cmd.Rule)
		if cmd.Response != nil {
			cmd.Response <- result
		}
	case "stop_all":
		wm.stopAllUnsafe()
		if cmd.Response != nil {
			cmd.Response <- true
		}
	case "count":
		if cmd.Response != nil {
			cmd.Response <- len(wm.workers)
		}
	case "list":
		if cmd.Response != nil {
			ruleIDs := make([]string, 0, len(wm.workers))
			for ruleID := range wm.workers {
				ruleIDs = append(ruleIDs, ruleID)
			}
			cmd.Response <- ruleIDs
		}
	}
}

// addWorkerUnsafe adds a worker without locking (called from manager goroutine)
func (wm *WorkerManager) addWorkerUnsafe(rule Rule) {
	if existing, ok := wm.workers[rule.ID]; ok {
		existing.Stop()
	}

	if rule.Status != EnabledStatus {
		delete(wm.workers, rule.ID)
		return
	}

	worker := NewRuleWorker(rule, wm.engine)
	worker.Start(wm.ctx)
	wm.workers[rule.ID] = worker
}

// removeWorkerUnsafe removes a worker without locking (called from manager goroutine)
func (wm *WorkerManager) removeWorkerUnsafe(ruleID string) {
	if worker, ok := wm.workers[ruleID]; ok {
		worker.Stop()
		delete(wm.workers, ruleID)
	}
}

// updateWorkerUnsafe updates a worker without locking (called from manager goroutine)
func (wm *WorkerManager) updateWorkerUnsafe(rule Rule) {
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
		worker.Start(wm.ctx)
		wm.workers[rule.ID] = worker
	}
}

// sendMessageUnsafe sends a message to a worker without locking (called from manager goroutine)
func (wm *WorkerManager) sendMessageUnsafe(msg *messaging.Message, rule Rule) bool {
	worker, ok := wm.workers[rule.ID]
	if !ok || !worker.IsRunning() {
		return false
	}

	return worker.Send(WorkerMessage{
		Message: msg,
		Rule:    rule,
	})
}

// stopAllUnsafe stops all workers without locking (called from manager goroutine)
func (wm *WorkerManager) stopAllUnsafe() {
	for _, worker := range wm.workers {
		worker.Stop()
	}
	wm.workers = make(map[string]*RuleWorker)
}

// AddWorker adds a new worker for the given rule.
func (wm *WorkerManager) AddWorker(ctx context.Context, rule Rule) {
	if atomic.LoadInt32(&wm.running) == 0 {
		return
	}
	
	cmd := WorkerManagerCommand{
		Type: "add",
		Rule: rule,
	}
	
	select {
	case wm.commandCh <- cmd:
	case <-ctx.Done():
	}
}

// RemoveWorker removes and stops the worker for the given rule ID.
func (wm *WorkerManager) RemoveWorker(ruleID string) {
	if atomic.LoadInt32(&wm.running) == 0 {
		return
	}
	
	cmd := WorkerManagerCommand{
		Type:   "remove",
		RuleID: ruleID,
	}
	
	select {
	case wm.commandCh <- cmd:
	default:
		// Non-blocking, if channel is full, skip
	}
}

// UpdateWorker updates the rule configuration for an existing worker.
func (wm *WorkerManager) UpdateWorker(ctx context.Context, rule Rule) {
	if atomic.LoadInt32(&wm.running) == 0 {
		return
	}
	
	cmd := WorkerManagerCommand{
		Type: "update",
		Rule: rule,
	}
	
	select {
	case wm.commandCh <- cmd:
	case <-ctx.Done():
	}
}

// SendMessage sends a message to the appropriate worker for processing.
func (wm *WorkerManager) SendMessage(msg *messaging.Message, rule Rule) bool {
	if atomic.LoadInt32(&wm.running) == 0 {
		return false
	}
	
	responseCh := make(chan interface{}, 1)
	cmd := WorkerManagerCommand{
		Type:     "send",
		Rule:     rule,
		Message:  msg,
		Response: responseCh,
	}
	
	select {
	case wm.commandCh <- cmd:
		select {
		case result := <-responseCh:
			if b, ok := result.(bool); ok {
				return b
			}
			return false
		case <-wm.ctx.Done():
			return false
		}
	default:
		return false
	}
}

// StopAll stops all workers and waits for them to finish.
func (wm *WorkerManager) StopAll() error {
	if !atomic.CompareAndSwapInt32(&wm.running, 1, 0) {
		return nil
	}
	
	responseCh := make(chan interface{}, 1)
	cmd := WorkerManagerCommand{
		Type:     "stop_all",
		Response: responseCh,
	}
	
	select {
	case wm.commandCh <- cmd:
		<-responseCh // Wait for completion
	default:
		// Channel full, force stop
	}
	
	// Wait for all workers to finish
	return wm.g.Wait()
}// GetWorkerCount returns the number of active workers.
func (wm *WorkerManager) GetWorkerCount() int {
	if atomic.LoadInt32(&wm.running) == 0 {
		return 0
	}
	
	responseCh := make(chan interface{}, 1)
	cmd := WorkerManagerCommand{
		Type:     "count",
		Response: responseCh,
	}
	
	select {
	case wm.commandCh <- cmd:
		if result := <-responseCh; result != nil {
			if count, ok := result.(int); ok {
				return count
			}
		}
	default:
	}
	return 0
}

// ListWorkers returns a slice of rule IDs that have active workers.
func (wm *WorkerManager) ListWorkers() []string {
	if atomic.LoadInt32(&wm.running) == 0 {
		return nil
	}
	
	responseCh := make(chan interface{}, 1)
	cmd := WorkerManagerCommand{
		Type:     "list",
		Response: responseCh,
	}
	
	select {
	case wm.commandCh <- cmd:
		if result := <-responseCh; result != nil {
			if list, ok := result.([]string); ok {
				return list
			}
		}
	default:
	}
	return nil
}

// RefreshWorkers synchronizes workers with the current set of enabled rules.
func (wm *WorkerManager) RefreshWorkers(ctx context.Context, rules []Rule) {
	if atomic.LoadInt32(&wm.running) == 0 {
		return
	}
	
	// For simplicity, let's process refresh by individual add/update/remove commands
	// First get current workers, then sync
	for _, rule := range rules {
		if rule.Status == EnabledStatus {
			wm.UpdateWorker(ctx, rule)
		} else {
			wm.RemoveWorker(rule.ID)
		}
	}
}
