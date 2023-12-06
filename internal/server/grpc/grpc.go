// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/absmach/magistrala/internal/server"
	mglog "github.com/absmach/magistrala/logger"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	stopWaitTime = 5 * time.Second
)

type Server struct {
	server.BaseServer
	server          *grpc.Server
	registerService serviceRegister
}

type serviceRegister func(srv *grpc.Server)

var _ server.Server = (*Server)(nil)

func New(ctx context.Context, cancel context.CancelFunc, name string, config server.Config, registerService serviceRegister, logger mglog.Logger) server.Server {
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
		registerService: registerService,
	}
}

func (s *Server) Start() error {
	errCh := make(chan error)
	grpcServerOptions := []grpc.ServerOption{
		grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor()),
	}

	listener, err := net.Listen("tcp", s.Address)
	if err != nil {
		return fmt.Errorf("failed to listen on port %s: %w", s.Address, err)
	}
	creds := grpc.Creds(insecure.NewCredentials())

	switch {
	case s.Config.CertFile != "" || s.Config.KeyFile != "":
		certificate, err := tls.LoadX509KeyPair(s.Config.CertFile, s.Config.KeyFile)
		if err != nil {
			return fmt.Errorf("failed to load auth certificates: %w", err)
		}
		tlsConfig := &tls.Config{
			ClientAuth:   tls.RequireAndVerifyClientCert,
			Certificates: []tls.Certificate{certificate},
		}

		var mtlsCA string
		// Loading Server CA file
		rootCA, err := loadCertFile(s.Config.ServerCAFile)
		if err != nil {
			return fmt.Errorf("failed to load root ca file: %w", err)
		}
		if len(rootCA) > 0 {
			if tlsConfig.RootCAs == nil {
				tlsConfig.RootCAs = x509.NewCertPool()
			}
			if !tlsConfig.RootCAs.AppendCertsFromPEM(rootCA) {
				return fmt.Errorf("failed to append root ca to tls.Config")
			}
			mtlsCA = fmt.Sprintf("root ca %s", s.Config.ServerCAFile)
		}

		// Loading Client CA File
		clientCA, err := loadCertFile(s.Config.ClientCAFile)
		if err != nil {
			return fmt.Errorf("failed to load client ca file: %w", err)
		}
		if len(clientCA) > 0 {
			if tlsConfig.ClientCAs == nil {
				tlsConfig.ClientCAs = x509.NewCertPool()
			}
			if !tlsConfig.ClientCAs.AppendCertsFromPEM(clientCA) {
				return fmt.Errorf("failed to append client ca to tls.Config")
			}
			mtlsCA = fmt.Sprintf("%s client ca %s", mtlsCA, s.Config.ClientCAFile)
		}
		creds = grpc.Creds(credentials.NewTLS(tlsConfig))
		switch {
		case len(mtlsCA) > 0:
			s.Logger.Info(fmt.Sprintf("%s service gRPC server listening at %s with TLS/mTLS cert %s , key %s and %s", s.Name, s.Address, s.Config.CertFile, s.Config.KeyFile, mtlsCA))
		default:
			s.Logger.Info(fmt.Sprintf("%s service gRPC server listening at %s with TLS cert %s and key %s", s.Name, s.Address, s.Config.CertFile, s.Config.KeyFile))
		}
	default:
		s.Logger.Info(fmt.Sprintf("%s service gRPC server listening at %s without TLS", s.Name, s.Address))
	}

	grpcServerOptions = append(grpcServerOptions, creds)

	s.server = grpc.NewServer(grpcServerOptions...)
	s.registerService(s.server)

	go func() {
		errCh <- s.server.Serve(listener)
	}()

	select {
	case <-s.Ctx.Done():
		return s.Stop()
	case err := <-errCh:
		s.Cancel()
		return err
	}
}

func (s *Server) Stop() error {
	defer s.Cancel()
	c := make(chan bool)
	go func() {
		defer close(c)
		s.server.GracefulStop()
	}()
	select {
	case <-c:
	case <-time.After(stopWaitTime):
	}
	s.Logger.Info(fmt.Sprintf("%s gRPC service shutdown at %s", s.Name, s.Address))

	return nil
}

func loadCertFile(certFile string) ([]byte, error) {
	if certFile != "" {
		return os.ReadFile(certFile)
	}
	return []byte{}, nil
}
