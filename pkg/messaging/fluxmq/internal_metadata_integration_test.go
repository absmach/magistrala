// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package fluxmq

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/absmach/magistrala/pkg/messaging"
	"github.com/stretchr/testify/require"
)

type integrationHandler struct{}

func (*integrationHandler) Handle(*messaging.Message) error { return nil }

func (*integrationHandler) Cancel() error { return nil }

func TestInternalMetadataPreprovisionedStream(t *testing.T) {
	brokerURL := os.Getenv("FLUXMQ_INTERNAL_METADATA_URL")
	if brokerURL == "" {
		t.Skip("set FLUXMQ_INTERNAL_METADATA_URL to run the trusted stream integration test")
	}

	ctx := context.Background()
	ps, err := NewPubSub(ctx, brokerURL, nil, InternalMetadata(
		os.Getenv("FLUXMQ_INTERNAL_METADATA_CERT"),
		os.Getenv("FLUXMQ_INTERNAL_METADATA_KEY"),
		os.Getenv("FLUXMQ_INTERNAL_METADATA_CA"),
	))
	require.NoError(t, err)
	t.Cleanup(func() { _ = ps.Close() })

	stamp := time.Now().UnixNano()
	topic := "m/integration-workspace/c/integration-channel/" + time.Unix(0, stamp).Format("150405.000000000")
	handler := &integrationHandler{}
	subscribeErr := ps.Subscribe(ctx, messaging.SubscriberConfig{
		ID:             "internal-metadata-integration",
		Topic:          "m/#",
		DeliveryPolicy: messaging.DeliverNewPolicy,
		Handler:        handler,
	})
	if os.Getenv("FLUXMQ_INTERNAL_METADATA_EXPECT_SUBSCRIBE_ERROR") == "true" {
		require.Error(t, subscribeErr)
		return
	}
	require.NoError(t, subscribeErr)

	msg := &messaging.Message{
		Workspace: "integration-workspace",
		Channel:   "integration-channel",
		Subtopic:  time.Unix(0, stamp).Format("150405.000000000"),
		Payload:   []byte("metadata-round-trip"),
		Publisher: "rules-engine",
		Protocol:  "internal",
		Metadata: map[string]string{
			"magistrala.re.trace": "signed-trace",
			"other":               "preserved",
		},
	}
	require.NoError(t, ps.Publish(ctx, topic, msg))
}
