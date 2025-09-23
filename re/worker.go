// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"context"
	"sync"
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
		select {
		case <-w.updateChan:
		default:
		}
		w.updateChan <- rule
	}
}

// GetRule returns the current rule configuration.
func (w *RuleWorker) GetRule() Rule {
	return w.rule
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

type WorkerCommandType uint8

const (
	CmdAdd WorkerCommandType = iota
	CmdRemove
	CmdUpdate
	CmdSend
	CmdStopAll
	CmdCount
	CmdList
)

func (c WorkerCommandType) String() string {
	switch c {
	case CmdAdd:
		return "add"
	case CmdRemove:
		return "remove"
	case CmdUpdate:
		return "update"
	case CmdSend:
		return "send"
	case CmdStopAll:
		return "stop_all"
	case CmdCount:
		return "count"
	case CmdList:
		return "list"
	default:
		return "unknown"
	}
}

// WorkerManagerCommand represents commands for worker management.
type WorkerManagerCommand struct {
	Type     WorkerCommandType
	Rule     Rule
	RuleID   string
	Message  *messaging.Message
	Response chan interface{}
}

// WorkerManager manages all rule workers.
type WorkerManager struct {
	workers   map[string]*RuleWorker
	engine    *re
	g         *errgroup.Group
	ctx       context.Context
	commandCh chan WorkerManagerCommand
	errorCh   chan error
	mu        sync.RWMutex
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
		errorCh:   make(chan error, 100),
		running:   0,
	}

	// Start the worker manager goroutine
	wm.g.Go(func() error {
		return wm.manageWorkers(ctx)
	})

	atomic.StoreInt32(&wm.running, 1)
	return wm
}

func (wm *WorkerManager) manageWorkers(ctx context.Context) error {
	defer atomic.StoreInt32(&wm.running, 0)

	for {
		select {
		case <-ctx.Done():
			for _, worker := range wm.workers {
				if err := worker.Stop(); err != nil {
					select {
					case wm.errorCh <- err:
					default:
					}
				}
			}
			wm.workers = make(map[string]*RuleWorker)
			return ctx.Err()

		case cmd := <-wm.commandCh:
			wm.handleCommand(cmd)
		}
	}
}

func (wm *WorkerManager) handleCommand(cmd WorkerManagerCommand) {
	switch cmd.Type {
	case CmdAdd:
		if err := wm.addWorker(cmd.Rule); err != nil {
			select {
			case wm.errorCh <- err:
			default:
			}
		}
	case CmdRemove:
		if err := wm.removeWorker(cmd.RuleID); err != nil {
			select {
			case wm.errorCh <- err:
			default:
			}
		}
	case CmdUpdate:
		if err := wm.updateWorker(cmd.Rule); err != nil {
			select {
			case wm.errorCh <- err:
			default:
			}
		}
	case CmdSend:
		result := wm.sendMessage(cmd.Message, cmd.Rule)
		if cmd.Response != nil {
			cmd.Response <- result
		}
	case CmdStopAll:
		if err := wm.stopAll(); err != nil {
			select {
			case wm.errorCh <- err:
			default:
			}
		}
		if cmd.Response != nil {
			cmd.Response <- true
		}
	case CmdCount:
		wm.mu.RLock()
		count := len(wm.workers)
		wm.mu.RUnlock()
		if cmd.Response != nil {
			cmd.Response <- count
		}
	case CmdList:
		wm.mu.RLock()
		ruleIDs := make([]string, 0, len(wm.workers))
		for ruleID := range wm.workers {
			ruleIDs = append(ruleIDs, ruleID)
		}
		wm.mu.RUnlock()
		if cmd.Response != nil {
			cmd.Response <- ruleIDs
		}
	}
}

func (wm *WorkerManager) addWorker(rule Rule) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if existing, ok := wm.workers[rule.ID]; ok {
		if err := existing.Stop(); err != nil {
			return err
		}
	}

	if rule.Status != EnabledStatus {
		delete(wm.workers, rule.ID)
		return nil
	}

	worker := NewRuleWorker(rule, wm.engine)
	worker.Start(wm.ctx)
	wm.workers[rule.ID] = worker
	return nil
}

func (wm *WorkerManager) removeWorker(ruleID string) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if worker, ok := wm.workers[ruleID]; ok {
		if err := worker.Stop(); err != nil {
			return err
		}
		delete(wm.workers, ruleID)
	}
	return nil
}

func (wm *WorkerManager) updateWorker(rule Rule) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if worker, ok := wm.workers[rule.ID]; ok {
		if err := worker.Stop(); err != nil {
			return err
		}
		delete(wm.workers, rule.ID)
	}

	if rule.Status != EnabledStatus {
		delete(wm.workers, rule.ID)
		return nil
	}

	worker := NewRuleWorker(rule, wm.engine)
	worker.Start(wm.ctx)
	wm.workers[rule.ID] = worker
	return nil
}

func (wm *WorkerManager) sendMessage(msg *messaging.Message, rule Rule) bool {
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

func (wm *WorkerManager) stopAll() error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	for _, worker := range wm.workers {
		if err := worker.Stop(); err != nil {
			return err
		}
	}
	wm.workers = make(map[string]*RuleWorker)
	return nil
}

func (wm *WorkerManager) AddWorker(ctx context.Context, rule Rule) {
	if atomic.LoadInt32(&wm.running) == 0 {
		return
	}

	cmd := WorkerManagerCommand{
		Type: CmdAdd,
		Rule: rule,
	}

	select {
	case wm.commandCh <- cmd:
	case <-ctx.Done():
	}
}

func (wm *WorkerManager) RemoveWorker(ruleID string) {
	if atomic.LoadInt32(&wm.running) == 0 {
		return
	}

	cmd := WorkerManagerCommand{
		Type:   CmdRemove,
		RuleID: ruleID,
	}

	wm.commandCh <- cmd
}

func (wm *WorkerManager) UpdateWorker(ctx context.Context, rule Rule) {
	if atomic.LoadInt32(&wm.running) == 0 {
		return
	}

	cmd := WorkerManagerCommand{
		Type: CmdUpdate,
		Rule: rule,
	}

	select {
	case wm.commandCh <- cmd:
	case <-ctx.Done():
	}
}

func (wm *WorkerManager) SendMessage(msg *messaging.Message, rule Rule) bool {
	if atomic.LoadInt32(&wm.running) == 0 {
		return false
	}

	responseCh := make(chan interface{}, 1)
	cmd := WorkerManagerCommand{
		Type:     CmdSend,
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

func (wm *WorkerManager) StopAll() error {
	if !atomic.CompareAndSwapInt32(&wm.running, 1, 0) {
		return nil
	}

	responseCh := make(chan interface{}, 1)
	cmd := WorkerManagerCommand{
		Type:     CmdStopAll,
		Response: responseCh,
	}

	wm.commandCh <- cmd
	<-responseCh

	return wm.g.Wait()
}

func (wm *WorkerManager) GetWorkerCount() int {
	if atomic.LoadInt32(&wm.running) == 0 {
		return 0
	}

	responseCh := make(chan interface{}, 1)
	cmd := WorkerManagerCommand{
		Type:     CmdCount,
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

func (wm *WorkerManager) ListWorkers() []string {
	if atomic.LoadInt32(&wm.running) == 0 {
		return nil
	}

	responseCh := make(chan interface{}, 1)
	cmd := WorkerManagerCommand{
		Type:     CmdList,
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

func (wm *WorkerManager) ErrorChan() <-chan error {
	return wm.errorCh
}

func (wm *WorkerManager) RefreshWorkers(ctx context.Context, rules []Rule) {
	if atomic.LoadInt32(&wm.running) == 0 {
		return
	}

	for _, rule := range rules {
		if rule.Status == EnabledStatus {
			wm.UpdateWorker(ctx, rule)
		} else {
			wm.RemoveWorker(rule.ID)
		}
	}
}
