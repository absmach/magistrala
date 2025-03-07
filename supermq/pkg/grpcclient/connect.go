// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpcclient

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"github.com/absmach/supermq/pkg/errors"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

type security int

const (
	withoutTLS security = iota
	withTLS
	withmTLS
)
const buffSize = 10 * 1024 * 1024

var (
	errGrpcConnect   = errors.New("failed to connect to grpc server")
	errGrpcClose     = errors.New("failed to close grpc connection")
	ErrSvcNotServing = errors.New("service is not serving")
)

type Config struct {
	URL               string        `env:"URL"              envDefault:""`
	Timeout           time.Duration `env:"TIMEOUT"          envDefault:"1s"`
	ClientCert        string        `env:"CLIENT_CERT"      envDefault:""`
	ClientKey         string        `env:"CLIENT_KEY"       envDefault:""`
	ServerCAFile      string        `env:"SERVER_CA_CERTS"  envDefault:""`
	BypassHealthCheck bool
}

// Handler is used to handle gRPC connection.
type Handler interface {
	// Close closes gRPC connection.
	Close() error

	// Secure is used for pretty printing TLS info.
	Secure() string

	// Connection returns the gRPC connection.
	Connection() *grpc.ClientConn
}

type client struct {
	*grpc.ClientConn
	cfg    Config
	secure security
}

var _ Handler = (*client)(nil)

func NewHandler(cfg Config) (Handler, error) {
	conn, secure, err := connect(cfg)
	if err != nil {
		return nil, err
	}

	return &client{
		ClientConn: conn,
		cfg:        cfg,
		secure:     secure,
	}, nil
}

func (c *client) Close() error {
	if err := c.ClientConn.Close(); err != nil {
		return errors.Wrap(errGrpcClose, err)
	}

	return nil
}

func (c *client) Connection() *grpc.ClientConn {
	return c.ClientConn
}

// Secure is used for pretty printing TLS info.
func (c *client) Secure() string {
	switch c.secure {
	case withTLS:
		return "with TLS"
	case withmTLS:
		return "with mTLS"
	case withoutTLS:
		fallthrough
	default:
		return "without TLS"
	}
}

// connect creates new gRPC client and connect to gRPC server.
func connect(cfg Config) (*grpc.ClientConn, security, error) {
	opts := []grpc.DialOption{
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	}
	secure := withoutTLS
	tc := insecure.NewCredentials()

	if cfg.ServerCAFile != "" {
		tlsConfig := &tls.Config{}

		// Loading root ca certificates file
		rootCA, err := os.ReadFile(cfg.ServerCAFile)
		if err != nil {
			return nil, secure, fmt.Errorf("failed to load root ca file: %w", err)
		}
		if len(rootCA) > 0 {
			capool := x509.NewCertPool()
			if !capool.AppendCertsFromPEM(rootCA) {
				return nil, secure, fmt.Errorf("failed to append root ca to tls.Config")
			}
			tlsConfig.RootCAs = capool
			secure = withTLS
		}

		// Loading mtls certificates file
		if cfg.ClientCert != "" || cfg.ClientKey != "" {
			certificate, err := tls.LoadX509KeyPair(cfg.ClientCert, cfg.ClientKey)
			if err != nil {
				return nil, secure, fmt.Errorf("failed to client certificate and key %w", err)
			}
			tlsConfig.Certificates = []tls.Certificate{certificate}
			secure = withmTLS
		}

		tc = credentials.NewTLS(tlsConfig)
	}

	opts = append(
		opts, grpc.WithTransportCredentials(tc),
		grpc.WithReadBufferSize(buffSize),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(buffSize/10), grpc.MaxCallSendMsgSize(buffSize/10)),
		grpc.WithWriteBufferSize(buffSize),
	)

	conn, err := grpc.NewClient(cfg.URL, opts...)
	if err != nil {
		return nil, secure, errors.Wrap(errGrpcConnect, err)
	}

	return conn, secure, nil
}
