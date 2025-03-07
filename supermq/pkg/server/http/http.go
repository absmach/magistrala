// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/absmach/supermq/pkg/server"
)

const (
	httpProtocol  = "http"
	httpsProtocol = "https"
)

type httpServer struct {
	server.BaseServer
	server *http.Server
}

var _ server.Server = (*httpServer)(nil)

func NewServer(ctx context.Context, cancel context.CancelFunc, name string, config server.Config, handler http.Handler, logger *slog.Logger) server.Server {
	baseServer := server.NewBaseServer(ctx, cancel, name, config, logger)
	hserver := &http.Server{Addr: baseServer.Address, Handler: handler}

	return &httpServer{
		BaseServer: baseServer,
		server:     hserver,
	}
}

func (s *httpServer) Start() error {
	errCh := make(chan error)
	s.Protocol = httpProtocol
	switch {
	case s.Config.CertFile != "" || s.Config.KeyFile != "":
		s.Protocol = httpsProtocol
		s.Logger.Info(fmt.Sprintf("%s service %s server listening at %s with TLS cert %s and key %s", s.Name, s.Protocol, s.Address, s.Config.CertFile, s.Config.KeyFile))
		go func() {
			errCh <- s.server.ListenAndServeTLS(s.Config.CertFile, s.Config.KeyFile)
		}()
	default:
		s.Logger.Info(fmt.Sprintf("%s service %s server listening at %s without TLS", s.Name, s.Protocol, s.Address))
		go func() {
			errCh <- s.server.ListenAndServe()
		}()
	}
	select {
	case <-s.Ctx.Done():
		return s.Stop()
	case err := <-errCh:
		return err
	}
}

func (s *httpServer) Stop() error {
	defer s.Cancel()
	ctx, cancel := context.WithTimeout(context.Background(), server.StopWaitTime)
	defer cancel()
	if err := s.server.Shutdown(ctx); err != nil {
		s.Logger.Error(fmt.Sprintf("%s service %s server error occurred during shutdown at %s: %s", s.Name, s.Protocol, s.Address, err))
		return fmt.Errorf("%s service %s server error occurred during shutdown at %s: %w", s.Name, s.Protocol, s.Address, err)
	}
	s.Logger.Info(fmt.Sprintf("%s %s service shutdown of http at %s", s.Name, s.Protocol, s.Address))
	return nil
}
