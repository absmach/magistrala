// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/absmach/supermq/certs/internal/certs"
	"github.com/absmach/supermq/pkg/server"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	grpchealth "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/stats"
)

type serviceRegister func(srv *grpc.Server)

type grpcServer struct {
	server.BaseServer
	server          *grpc.Server
	registerService serviceRegister
	health          *health.Server
	statsHandler    stats.Handler
	ocspVerifier    *certs.OCSP
}

var _ server.Server = (*grpcServer)(nil)

func NewServer(ctx context.Context, cancel context.CancelFunc, name string, config server.Config, registerService serviceRegister, logger *slog.Logger, statsHandler stats.Handler, ocspVerifier *certs.OCSP) server.Server {
	baseServer := server.NewBaseServer(ctx, cancel, name, config, logger)

	return &grpcServer{
		BaseServer:      baseServer,
		registerService: registerService,
		statsHandler:    statsHandler,
		ocspVerifier:    ocspVerifier,
	}
}

func (s *grpcServer) Start() error {
	errCh := make(chan error)
	grpcServerOptions := []grpc.ServerOption{
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	}

	listener, err := net.Listen("tcp", s.Address)
	if err != nil {
		return fmt.Errorf("failed to listen on port %s: %w", s.Address, err)
	}
	creds := grpc.Creds(insecure.NewCredentials())

	switch {
	case s.Config.CertFile != "" || s.Config.KeyFile != "":
		certificate, err := server.LoadX509KeyPair(s.Config.CertFile, s.Config.KeyFile)
		if err != nil {
			return fmt.Errorf("failed to load auth gRPC client certificates: %w", err)
		}
		tlsConfig := &tls.Config{
			ClientAuth:   tls.NoClientCert,
			Certificates: []tls.Certificate{certificate},
		}

		var mtlsCA string
		// Loading Server CA file.
		rootCA, err := server.LoadRootCACerts(s.Config.ServerCAFile)
		if err != nil {
			return fmt.Errorf("failed to load root ca file: %w", err)
		}
		if rootCA != nil {
			if tlsConfig.RootCAs == nil {
				tlsConfig.RootCAs = x509.NewCertPool()
			}
			tlsConfig.RootCAs = rootCA
			mtlsCA = fmt.Sprintf("root ca %s", s.Config.ServerCAFile)
		}

		// Loading Client CA File
		clientCA, err := server.LoadRootCACerts(s.Config.ClientCAFile)
		if err != nil {
			return fmt.Errorf("failed to load client ca file: %w", err)
		}
		if clientCA != nil {
			if tlsConfig.ClientCAs == nil {
				tlsConfig.ClientCAs = x509.NewCertPool()
			}
			tlsConfig.ClientCAs = clientCA
			mtlsCA = fmt.Sprintf("%s client ca %s", mtlsCA, s.Config.ClientCAFile)
		}
		creds = grpc.Creds(credentials.NewTLS(tlsConfig))
		switch {
		case mtlsCA != "":
			tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
			creds = grpc.Creds(credentials.NewTLS(tlsConfig))
			s.Logger.Info(fmt.Sprintf("%s service gRPC server listening at %s with TLS/mTLS", s.Name, s.Address))
		default:
			s.Logger.Info(fmt.Sprintf("%s service gRPC server listening at %s with TLS cert", s.Name, s.Address))
		}
	default:
		s.Logger.Info(fmt.Sprintf("%s service gRPC server listening at %s without TLS", s.Name, s.Address))
	}

	grpcServerOptions = append(grpcServerOptions, creds)

	s.server = grpc.NewServer(grpcServerOptions...)
	s.health = health.NewServer()
	grpchealth.RegisterHealthServer(s.server, s.health)
	s.registerService(s.server)
	s.health.SetServingStatus(s.Name, grpchealth.HealthCheckResponse_SERVING)

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

func (s *grpcServer) Stop() error {
	defer s.Cancel()
	c := make(chan bool)
	go func() {
		defer close(c)
		s.health.Shutdown()
		s.server.GracefulStop()
	}()
	select {
	case <-c:
	case <-time.After(server.StopWaitTime):
	}
	s.Logger.Info(fmt.Sprintf("%s gRPC service shutdown at %s", s.Name, s.Address))

	return nil
}
