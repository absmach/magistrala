// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"time"

	"github.com/mainflux/mainflux/pkg/errors"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	gogrpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	errGrpcConnect = errors.New("failed to connect to grpc server")
	errGrpcClose   = errors.New("failed to close grpc connection")
)

type Config struct {
	ClientTLS bool          `env:"CLIENT_TLS"    envDefault:"false"`
	CACerts   string        `env:"CA_CERTS"      envDefault:""`
	URL       string        `env:"URL"           envDefault:""`
	Timeout   time.Duration `env:"TIMEOUT"       envDefault:"1s"`
}

type ClientHandler interface {
	Close() error
	IsSecure() bool
	Secure() string
}

type Client struct {
	*gogrpc.ClientConn
	secure bool
}

var _ ClientHandler = (*Client)(nil)

// NewClientHandler create new client handler for gRPC client.
func NewClientHandler(c *Client) ClientHandler {
	return c
}

// Connect creates new gRPC client and connect to gRPC server.
func Connect(cfg Config) (*gogrpc.ClientConn, bool, error) {
	var opts []gogrpc.DialOption
	secure := false
	tc := insecure.NewCredentials()

	if cfg.ClientTLS && cfg.CACerts != "" {
		var err error
		tc, err = credentials.NewClientTLSFromFile(cfg.CACerts, "")
		if err != nil {
			return nil, secure, err
		}
		secure = true
	}

	opts = append(opts, gogrpc.WithTransportCredentials(tc), gogrpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor()))

	conn, err := gogrpc.Dial(cfg.URL, opts...)
	if err != nil {
		return nil, secure, err
	}
	return conn, secure, nil
}

// Setup load gRPC configuration from environment variable, creates new gRPC client and connect to gRPC server.
func Setup(config Config, svcName string) (*Client, ClientHandler, error) {
	secure := false

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
	return c.secure
}

// Secure is used for pretty printing TLS info.
func (c *Client) Secure() string {
	if c.secure {
		return "with TLS"
	}
	return "without TLS"
}
