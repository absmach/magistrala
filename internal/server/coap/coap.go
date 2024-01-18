// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package coap

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"time"

	"github.com/absmach/magistrala/internal/server"
	gocoap "github.com/plgd-dev/go-coap/v2"
	"github.com/plgd-dev/go-coap/v2/mux"
)

const (
	stopWaitTime = 5 * time.Second
)

type Server struct {
	server.BaseServer
	handler mux.HandlerFunc
}

var _ server.Server = (*Server)(nil)

func New(ctx context.Context, cancel context.CancelFunc, name string, config server.Config, handler mux.HandlerFunc, logger *slog.Logger) server.Server {
	listenFullAddress := fmt.Sprintf("%s:%s", config.Host, config.Port)
	return &Server{
		BaseServer: server.BaseServer{
			Ctx:     ctx,
			Cancel:  cancel,
			Name:    name,
			Address: listenFullAddress,
			Config:  config,
			Logger:  logger,
		},
		handler: handler,
	}
}

func (s *Server) Start() error {
	errCh := make(chan error)
	s.Logger.Info(fmt.Sprintf("%s service started using http, exposed port %s", s.Name, s.Address))
	switch {
	case s.Config.CertFile != "" || s.Config.KeyFile != "":
		s.Logger.Info(fmt.Sprintf("%s service %s server listening at %s with TLS cert %s and key %s", s.Name, s.Protocol, s.Address, s.Config.CertFile, s.Config.KeyFile))
		certificate, err := tls.LoadX509KeyPair(s.Config.CertFile, s.Config.KeyFile)
		if err != nil {
			return fmt.Errorf("failed to load auth certificates: %w", err)
		}
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{certificate},
		}

		go func() {
			errCh <- gocoap.ListenAndServeTCPTLS("udp", s.Address, tlsConfig, s.handler)
		}()
	default:
		s.Logger.Info(fmt.Sprintf("%s service %s server listening at %s without TLS", s.Name, s.Protocol, s.Address))
		go func() {
			errCh <- gocoap.ListenAndServe("udp", s.Address, s.handler)
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
	c := make(chan bool)
	defer close(c)
	select {
	case <-c:
	case <-time.After(stopWaitTime):
	}
	s.Logger.Info(fmt.Sprintf("%s service shutdown of http at %s", s.Name, s.Address))
	return nil
}
