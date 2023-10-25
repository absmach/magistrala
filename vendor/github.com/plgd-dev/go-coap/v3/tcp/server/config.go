package server

import (
	"fmt"
	"time"

	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/net/monitor/inactivity"
	"github.com/plgd-dev/go-coap/v3/net/responsewriter"
	"github.com/plgd-dev/go-coap/v3/options/config"
	"github.com/plgd-dev/go-coap/v3/tcp/client"
)

// The HandlerFunc type is an adapter to allow the use of
// ordinary functions as COAP handlers.
type HandlerFunc = func(*responsewriter.ResponseWriter[*client.Conn], *pool.Message)

type ErrorFunc = func(error)

type GoPoolFunc = func(func()) error

// OnNewConnFunc is the callback for new connections.
type OnNewConnFunc = func(cc *client.Conn)

var DefaultConfig = func() Config {
	opts := Config{
		Common: config.NewCommon[*client.Conn](),
		CreateInactivityMonitor: func() client.InactivityMonitor {
			maxRetries := uint32(2)
			timeout := time.Second * 16
			onInactive := func(cc *client.Conn) {
				_ = cc.Close()
			}
			keepalive := inactivity.NewKeepAlive(maxRetries, onInactive, func(cc *client.Conn, receivePong func()) (func(), error) {
				return cc.AsyncPing(receivePong)
			})
			return inactivity.New(timeout/time.Duration(maxRetries+1), keepalive.OnInactive)
		},
		OnNewConn: func(cc *client.Conn) {
			// do nothing by default
		},
		ConnectionCacheSize: 2 * 1024,
	}
	opts.Handler = func(w *responsewriter.ResponseWriter[*client.Conn], r *pool.Message) {
		if err := w.SetResponse(codes.NotFound, message.TextPlain, nil); err != nil {
			opts.Errors(fmt.Errorf("server handler: cannot set response: %w", err))
		}
	}
	return opts
}()

type Config struct {
	config.Common[*client.Conn]
	CreateInactivityMonitor         client.CreateInactivityMonitorFunc
	Handler                         HandlerFunc
	OnNewConn                       OnNewConnFunc
	ConnectionCacheSize             uint16
	DisablePeerTCPSignalMessageCSMs bool
	DisableTCPSignalMessageCSM      bool
}
