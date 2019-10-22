// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package gopcua

import (
	"context"
	"fmt"
	"log"

	opcuaGopcua "github.com/gopcua/opcua"
	uaGopcua "github.com/gopcua/opcua/ua"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/opcua"
)

var _ opcua.Reader = (*reader)(nil)

type reader struct {
	ctx    context.Context
	svc    opcua.Service
	logger logger.Logger
}

// NewReader returns new OPC-UA reader instance.
func NewReader(ctx context.Context, svc opcua.Service, log logger.Logger) opcua.Reader {
	return reader{
		ctx:    ctx,
		svc:    svc,
		logger: log,
	}
}

// Read reads a given OPC-UA Server endpoint.
func (r reader) Read(cfg opcua.Config) error {
	c := opcuaGopcua.NewClient(cfg.ServerURI, opcuaGopcua.SecurityMode(uaGopcua.MessageSecurityModeNone))
	if err := c.Connect(r.ctx); err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	nid := fmt.Sprintf("ns=%s;i=%s", cfg.NodeNamespace, cfg.NodeIdintifier)
	id, err := uaGopcua.ParseNodeID(nid)
	if err != nil {
		r.logger.Error(fmt.Sprintf("invalid node id: %v", err))
	}

	req := &uaGopcua.ReadRequest{
		MaxAge: 2000,
		NodesToRead: []*uaGopcua.ReadValueID{
			&uaGopcua.ReadValueID{NodeID: id},
		},
		TimestampsToReturn: uaGopcua.TimestampsToReturnBoth,
	}

	resp, err := c.Read(req)
	if err != nil {
		r.logger.Error(fmt.Sprintf("Read failed: %s", err))
	}
	if resp.Results[0].Status != uaGopcua.StatusOK {
		r.logger.Error(fmt.Sprintf("Status not OK: %v", resp.Results[0].Status))
	}

	// Publish on Mainflux NATS broker
	msg := opcua.Message{
		Namespace: cfg.NodeNamespace,
		ID:        cfg.NodeIdintifier,
		Data:      resp.Results[0].Value.Float(),
	}
	r.svc.Publish(r.ctx, "", msg)

	return nil
}
