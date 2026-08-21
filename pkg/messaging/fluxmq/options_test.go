// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package fluxmq

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestInternalMetadataRequiresMTLS(t *testing.T) {
	if err := InternalMetadata("", "", "")(&pubsub{}); err == nil {
		t.Fatal("expected empty certificate paths to be rejected")
	}

	dir := t.TempDir()
	certFile, keyFile, caFile := writeTestCertificate(t, dir)
	for name, paths := range map[string][3]string{
		"missing certificate": {"", keyFile, caFile},
		"missing key":         {certFile, "", caFile},
		"missing CA":          {certFile, keyFile, ""},
	} {
		t.Run(name, func(t *testing.T) {
			if err := InternalMetadata(paths[0], paths[1], paths[2])(&pubsub{}); err == nil {
				t.Fatal("expected partial mTLS configuration to be rejected")
			}
		})
	}

	tests := []struct {
		name string
		get  func() options
	}{
		{
			name: "publisher",
			get: func() options {
				var pub publisher
				if err := InternalMetadata(certFile, keyFile, caFile)(&pub); err != nil {
					t.Fatalf("configure publisher: %v", err)
				}
				return pub.options
			},
		},
		{
			name: "pubsub",
			get: func() options {
				var ps pubsub
				if err := InternalMetadata(certFile, keyFile, caFile)(&ps); err != nil {
					t.Fatalf("configure pubsub: %v", err)
				}
				return ps.options
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			opts := tc.get()
			if opts.tlsConfig == nil {
				t.Fatal("expected TLS configuration")
			}
			if !opts.preprovisioned {
				t.Fatal("expected broker-provisioned stream mode")
			}
		})
	}
}

func TestMQTTTopicToWireTopic(t *testing.T) {
	for input, want := range map[string]string{
		"m/workspace/c/channel/subtopic":  "m.workspace.c.channel.subtopic",
		"m/workspace/c/channel/sub.topic": "m.workspace.c.channel.sub%2Etopic",
	} {
		if got := mqttTopicToWireTopic(input); got != want {
			t.Fatalf("wire topic = %q, want %q", got, want)
		}
	}
}

func writeTestCertificate(t *testing.T, dir string) (certFile, keyFile, caFile string) {
	t.Helper()

	now := time.Now()
	caPublic, caPrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate CA key: %v", err)
	}
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             now.Add(-time.Minute),
		NotAfter:              now.Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, caPublic, caPrivate)
	if err != nil {
		t.Fatalf("create CA certificate: %v", err)
	}

	clientPublic, clientPrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate client key: %v", err)
	}
	clientTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "rules-engine"},
		NotBefore:    now.Add(-time.Minute),
		NotAfter:     now.Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	clientDER, err := x509.CreateCertificate(rand.Reader, clientTemplate, caTemplate, clientPublic, caPrivate)
	if err != nil {
		t.Fatalf("create client certificate: %v", err)
	}
	clientKey, err := x509.MarshalPKCS8PrivateKey(clientPrivate)
	if err != nil {
		t.Fatalf("marshal client key: %v", err)
	}

	certFile = filepath.Join(dir, "client.crt")
	keyFile = filepath.Join(dir, "client.key")
	caFile = filepath.Join(dir, "ca.crt")
	writePEM(t, certFile, "CERTIFICATE", clientDER)
	writePEM(t, keyFile, "PRIVATE KEY", clientKey)
	writePEM(t, caFile, "CERTIFICATE", caDER)

	return certFile, keyFile, caFile
}

func writePEM(t *testing.T, path, typ string, contents []byte) {
	t.Helper()

	data := pem.EncodeToMemory(&pem.Block{Type: typ, Bytes: contents})
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
