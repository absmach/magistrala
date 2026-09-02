// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package fluxmq

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/absmach/magistrala/pkg/messaging"
)

// ErrInvalidType is returned when the provided value is not of the expected type.
var ErrInvalidType = errors.New("invalid type")

const msgPrefix = "m"

type options struct {
	prefix             string
	connectionName     string
	directTopicIngress bool
	directTopicOnly    bool
	preprovisioned     bool
	tlsConfig          *tls.Config
}

func defaultOptions() options {
	return options{
		prefix: msgPrefix,
	}
}

// Prefix sets the topic prefix for publisher and subscriber.
func Prefix(prefix string) messaging.Option {
	return func(val any) error {
		switch v := val.(type) {
		case *publisher:
			v.prefix = strings.TrimSpace(prefix)
			if v.prefix == "" {
				v.prefix = msgPrefix
			}
		case *pubsub:
			v.prefix = strings.TrimSpace(prefix)
			if v.prefix == "" {
				v.prefix = msgPrefix
			}
		default:
			return ErrInvalidType
		}

		return nil
	}
}

// InternalMetadata configures a trusted FluxMQ service connection that can
// exchange broker-internal message metadata. It presents a client certificate,
// verifies the broker against caFile, and uses the broker-provisioned stream
// instead of trying to declare it with the service principal's restricted ACL.
//
// All three paths are required: a half-configured client would silently connect
// without the identity the broker authorizes against.
func InternalMetadata(certFile, keyFile, caFile string) messaging.Option {
	return func(val any) error {
		cfg, err := mtlsConfig(certFile, keyFile, caFile)
		if err != nil {
			return err
		}
		switch v := val.(type) {
		case *publisher:
			v.tlsConfig = cfg
			v.preprovisioned = true
		case *pubsub:
			v.tlsConfig = cfg
			v.preprovisioned = true
		default:
			return ErrInvalidType
		}

		return nil
	}
}

func mtlsConfig(certFile, keyFile, caFile string) (*tls.Config, error) {
	if certFile == "" || keyFile == "" || caFile == "" {
		return nil, fmt.Errorf("%w: mTLS needs a certificate, a key, and a CA", ErrInvalidType)
	}
	certificate, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load FluxMQ client certificate: %w", err)
	}
	ca, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read FluxMQ CA certificate: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(ca) {
		return nil, fmt.Errorf("failed to parse FluxMQ CA certificate %q", caFile)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{certificate},
		RootCAs:      pool,
		MinVersion:   tls.VersionTLS12,
	}, nil
}

// ConnectionName sets a human-readable connection name sent to FluxMQ
// for identifying this client in the broker's admin UI.
func ConnectionName(name string) messaging.Option {
	return func(val any) error {
		switch v := val.(type) {
		case *publisher:
			v.connectionName = name
		case *pubsub:
			v.connectionName = name
		default:
			return ErrInvalidType
		}

		return nil
	}
}

// DirectTopicIngress enables direct MQTT topic delivery in addition to stream
// queue delivery. This is opt-in because direct topic messages are normalized
// from broker-native metadata instead of the protobuf queue envelope.
func DirectTopicIngress() messaging.Option {
	return func(val any) error {
		switch v := val.(type) {
		case *publisher:
			return nil
		case *pubsub:
			v.directTopicIngress = true
		default:
			return ErrInvalidType
		}

		return nil
	}
}

// DirectTopicOnly subscribes only to regular MQTT topics and skips stream queue
// consumption. This is intended for bridge services that observe broker-native
// topics without also consuming queued messages.
func DirectTopicOnly() messaging.Option {
	return func(val any) error {
		switch v := val.(type) {
		case *publisher:
			return nil
		case *pubsub:
			v.directTopicIngress = true
			v.directTopicOnly = true
		default:
			return ErrInvalidType
		}

		return nil
	}
}
