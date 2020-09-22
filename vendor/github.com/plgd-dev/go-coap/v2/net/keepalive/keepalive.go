package keepalive

import (
	"context"
	"time"
)

type ClientConn interface {
	Context() context.Context
	Ping(context.Context) error
	Close() error
}

type RetryFunc = func() (when time.Time, err error)
type RetryFuncFactory = func() RetryFunc

// Config KeepAlive config
type Config struct {
	// Interval between two success pings
	Interval time.Duration
	// WaitForPong how long it will waits for pong response.
	WaitForPong time.Duration
	// NewRetryPolicy creates retry policy for the connection when ping fails.
	NewRetryPolicy RetryFuncFactory
}

var defaultOptions = options{
	config: MakeConfig(time.Second * 6),
}

type CfgOpt struct {
	config Config
}

func (o *CfgOpt) apply(opts *options) {
	opts.config = o.config
}

func WithConfig(config Config) *CfgOpt {
	return &CfgOpt{
		config: config,
	}
}

type options struct {
	config Config
}

// MakeConfig creates a policy that detects dropped connections within the connTimeout limit
// while attempting to make 3 pings during that period.
func MakeConfig(connTimeout time.Duration) Config {
	// 3 times ping-pong
	duration := connTimeout / 6
	return Config{
		Interval:    duration,
		WaitForPong: duration,
		NewRetryPolicy: func() RetryFunc {
			// The first failure is detected after 2*duration:
			// 1 since the previous ping, plus 1 for the next ping-pong to timeout
			start := time.Now()
			attempt := time.Duration(1)
			return func() (time.Time, error) {
				attempt++
				// Try to send ping and wait for pong 2 more times
				if time.Since(start) <= 2*2*duration {
					return start.Add(attempt * duration), nil
				}
				return time.Time{}, ErrKeepAliveDeadlineExceeded
			}
		},
	}
}

// A Option sets options such as config etc.
type Option interface {
	apply(*options)
}

type KeepAlive struct {
	cfg options
}

func New(opts ...Option) *KeepAlive {
	cfg := defaultOptions
	for _, o := range opts {
		o.apply(&cfg)
	}
	return &KeepAlive{
		cfg: cfg,
	}
}

func ping(c ClientConn, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(c.Context(), timeout)
	defer cancel()
	return c.Ping(ctx)
}

func (k *KeepAlive) doPing(c ClientConn) (bool, error) {
	err := ping(c, k.cfg.config.WaitForPong)
	if err == nil {
		return true, nil
	}
	if err != context.DeadlineExceeded {
		if err == context.Canceled {
			return false, nil
		}
		c.Close()
		return false, err
	}
	retryPolicy := k.cfg.config.NewRetryPolicy()
	for {
		when, err := retryPolicy()
		if err != nil {
			c.Close()
			return false, err
		}
		select {
		case <-c.Context().Done():
			return false, nil
		case <-time.After(time.Until(when)):
			err := ping(c, k.cfg.config.WaitForPong)
			if err == context.DeadlineExceeded {
				continue
			}
			if err == nil {
				return true, nil
			}
			c.Close()
			return false, err
		}
	}
}

func (k *KeepAlive) Run(c ClientConn) error {
	ticker := time.NewTicker(k.cfg.config.Interval)
	defer ticker.Stop()
	ok := true
	for ok {
		select {
		case <-ticker.C:
			var err error
			ok, err = k.doPing(c)
			if err != nil {
				return err
			}
		case <-c.Context().Done():
			return nil
		}
	}
	return nil
}
