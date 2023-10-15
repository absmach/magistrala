// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"github.com/mainflux/mainflux/pkg/errors"
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
	errGrpcConnect = errors.New("failed to connect to grpc server")
	errGrpcClose   = errors.New("failed to close grpc connection")
)

type Config struct {
	ClientCert   string        `env:"CLIENT_CERT"      envDefault:""`
	ClientKey    string        `env:"CLIENT_KEY"       envDefault:""`
	ServerCAFile string        `env:"SERVER_CA_CERTS"  envDefault:""`
	URL          string        `env:"URL"              envDefault:""`
	Timeout      time.Duration `env:"TIMEOUT"          envDefault:"1s"`
}

type ClientHandler interface {
	Close() error
	IsSecure() bool
	Secure() string
}

type Client struct {
	*grpc.ClientConn
	secure security
}

var _ ClientHandler = (*Client)(nil)

// NewClientHandler create new client handler for gRPC client.
func NewClientHandler(c *Client) ClientHandler {
	return c
}

// Connect creates new gRPC client and connect to gRPC server.
func Connect(cfg Config) (*grpc.ClientConn, security, error) {
	opts := []grpc.DialOption{
		grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor()),
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

	conn, err := grpc.Dial(cfg.URL, opts...)
	if err != nil {
		return nil, secure, err
	}
	return conn, secure, nil
}

// Setup load gRPC configuration from environment variable, creates new gRPC client and connect to gRPC server.
func Setup(config Config, svcName string) (*Client, ClientHandler, error) {
	secure := withoutTLS

	// connect to auth grpc server
	grpcClient, secure, err := Connect(config)
	if err != nil {
		return nil, nil, errors.Wrap(errGrpcConnect, err)
	}

	c := &Client{grpcClient, secure}

	return c, NewClientHandler(c), nil
}

// Close shuts down trace provider.
func (c *Client) Close() error {
	var retErr error
	err := c.ClientConn.Close()
	if err != nil {
		retErr = errors.Wrap(errGrpcClose, err)
	}
	return retErr
}

// IsSecure is utility method for checking if
// the client is running with TLS enabled.
func (c *Client) IsSecure() bool {
	switch c.secure {
	case withTLS, withmTLS:
		return true
	case withoutTLS:
		fallthrough
	default:
		return true
	}
}

// Secure is used for pretty printing TLS info.
func (c *Client) Secure() string {
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
