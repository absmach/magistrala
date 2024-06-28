// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package coap

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/absmach/magistrala/pkg/server"
	gocoap "github.com/plgd-dev/go-coap/v3"
	"github.com/plgd-dev/go-coap/v3/mux"
)

type coapServer struct {
	server.BaseServer
	handler mux.HandlerFunc
}

var _ server.Server = (*coapServer)(nil)

func NewServer(ctx context.Context, cancel context.CancelFunc, name string, config server.Config, handler mux.HandlerFunc, logger *slog.Logger) server.Server {
	baseServer := server.NewBaseServer(ctx, cancel, name, config, logger)

	return &coapServer{
		BaseServer: baseServer,
		handler:    handler,
	}
}

func (s *coapServer) Start() error {
	errCh := make(chan error)
	s.Logger.Info(fmt.Sprintf("%s service started using http, exposed port %s", s.Name, s.Address))
	s.Logger.Info(fmt.Sprintf("%s service %s server listening at %s without TLS", s.Name, s.Protocol, s.Address))

	go func() {
		errCh <- gocoap.ListenAndServe("udp", s.Address, s.handler)
	}()

	select {
	case <-s.Ctx.Done():
		return s.Stop()
	case err := <-errCh:
		return err
	}
}

func (s *coapServer) Stop() error {
	defer s.Cancel()
	c := make(chan bool)
	defer close(c)
	select {
	case <-c:
	case <-time.After(server.StopWaitTime):
	}
	s.Logger.Info(fmt.Sprintf("%s service shutdown of http at %s", s.Name, s.Address))
	return nil
}
