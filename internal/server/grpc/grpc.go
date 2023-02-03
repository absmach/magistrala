package grpc

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/mainflux/mainflux/internal/server"
	"github.com/mainflux/mainflux/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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

func New(ctx context.Context, cancel context.CancelFunc, name string, config server.Config, registerService serviceRegister, logger logger.Logger) server.Server {
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

	listener, err := net.Listen("tcp", s.Address)
	if err != nil {
		return fmt.Errorf("failed to listen on port %s: %w", s.Address, err)
	}

	switch {
	case s.Config.CertFile != "" || s.Config.KeyFile != "":
		creds, err := credentials.NewServerTLSFromFile(s.Config.CertFile, s.Config.KeyFile)
		if err != nil {
			return fmt.Errorf("failed to load auth certificates: %w", err)
		}
		s.Logger.Info(fmt.Sprintf("%s service gRPC server listening at %s with TLS cert %s and key %s", s.Name, s.Address, s.Config.CertFile, s.Config.KeyFile))
		s.server = grpc.NewServer(grpc.Creds(creds))
	default:
		s.Logger.Info(fmt.Sprintf("%s service gRPC server listening at %s without TLS", s.Name, s.Address))
		s.server = grpc.NewServer()
	}

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
