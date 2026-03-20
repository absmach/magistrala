// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package ticker

import "time"

type Ticker interface {
	Tick() <-chan time.Time
	Stop()
}

type timeTicker struct {
	*time.Ticker
}

func NewTicker(d time.Duration) Ticker {
	return &timeTicker{time.NewTicker(d)}
}

func (t *timeTicker) Tick() <-chan time.Time {
	return t.C
}
