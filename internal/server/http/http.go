// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/absmach/magistrala/internal/server"
)

const (
	stopWaitTime  = 5 * time.Second
	httpProtocol  = "http"
	httpsProtocol = "https"
)

type Server struct {
	server.BaseServer
	server *http.Server
}

var _ server.Server = (*Server)(nil)

func New(ctx context.Context, cancel context.CancelFunc, name string, config server.Config, handler http.Handler, logger *slog.Logger) server.Server {
	listenFullAddress := fmt.Sprintf("%s:%s", config.Host, config.Port)
	httpServer := &http.Server{Addr: listenFullAddress, Handler: handler}
	return &Server{
		BaseServer: server.BaseServer{
			Ctx:     ctx,
			Cancel:  cancel,
			Name:    name,
			Address: listenFullAddress,
			Config:  config,
			Logger:  logger,
		},
		server: httpServer,
	}
}

func (s *Server) Start() error {
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

func (s *Server) Stop() error {
	defer s.Cancel()
	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), stopWaitTime)
	defer cancelShutdown()
	if err := s.server.Shutdown(ctxShutdown); err != nil {
		s.Logger.Error(fmt.Sprintf("%s service %s server error occurred during shutdown at %s: %s", s.Name, s.Protocol, s.Address, err))
		return fmt.Errorf("%s service %s server error occurred during shutdown at %s: %w", s.Name, s.Protocol, s.Address, err)
	}
	s.Logger.Info(fmt.Sprintf("%s %s service shutdown of http at %s", s.Name, s.Protocol, s.Address))
	return nil
}
