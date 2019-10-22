// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package gopcua

import (
	"context"
	"fmt"
	"time"

	opcuaGopcua "github.com/gopcua/opcua"
	uaGopcua "github.com/gopcua/opcua/ua"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/opcua"
)

var _ opcua.Subscriber = (*client)(nil)

type client struct {
	ctx    context.Context
	svc    opcua.Service
	logger logger.Logger
}

// NewClient returns new OPC-UA client instance.
func NewClient(ctx context.Context, svc opcua.Service, log logger.Logger) opcua.Subscriber {
	return client{
		ctx:    ctx,
		svc:    svc,
		logger: log,
	}
}

// Subscribe subscribes to the OPC-UA Server.
func (b client) Subscribe(cfg opcua.Config) error {
	endpoints, err := opcuaGopcua.GetEndpoints(cfg.ServerURI)
	if err != nil {
		b.logger.Error(fmt.Sprintf("Failed to fetch OPC-UA server endpoints: %s", err.Error()))
	}

	ep := opcuaGopcua.SelectEndpoint(endpoints, cfg.Policy, uaGopcua.MessageSecurityModeFromString(cfg.Mode))
	if ep == nil {
		b.logger.Error("Failed to find suitable endpoint")
	}

	opts := []opcuaGopcua.Option{
		opcuaGopcua.SecurityPolicy(cfg.Policy),
		opcuaGopcua.SecurityModeString(cfg.Mode),
		opcuaGopcua.CertificateFile(cfg.CertFile),
		opcuaGopcua.PrivateKeyFile(cfg.KeyFile),
		opcuaGopcua.AuthAnonymous(),
		opcuaGopcua.SecurityFromEndpoint(ep, uaGopcua.UserTokenTypeAnonymous),
	}

	c := opcuaGopcua.NewClient(ep.EndpointURL, opts...)
	if errC := c.Connect(b.ctx); err != nil {
		b.logger.Error(errC.Error())
	}
	defer c.Close()

	sub, err := c.Subscribe(&opcuaGopcua.SubscriptionParameters{
		Interval: 2000 * time.Millisecond,
	})
	if err != nil {
		b.logger.Error(err.Error())
	}
	defer sub.Cancel()
	b.logger.Info(fmt.Sprintf("OPC-UA server URI: %s", ep.SecurityPolicyURI))
	b.logger.Info(fmt.Sprintf("Created subscription with id %v", sub.SubscriptionID))

	if err := b.runHandler(sub, cfg); err != nil {
		return err
	}

	return nil
}

func (b client) runHandler(sub *opcuaGopcua.Subscription, cfg opcua.Config) error {
	nid := fmt.Sprintf("ns=%s;i=%s", cfg.NodeNamespace, cfg.NodeIdintifier)
	nodeID, err := uaGopcua.ParseNodeID(nid)
	if err != nil {
		b.logger.Error(err.Error())
	}

	// arbitrary client handle for the monitoring item
	handle := uint32(42)
	miCreateRequest := opcuaGopcua.NewMonitoredItemCreateRequestWithDefaults(nodeID, uaGopcua.AttributeIDValue, handle)
	res, err := sub.Monitor(uaGopcua.TimestampsToReturnBoth, miCreateRequest)
	if err != nil || res.Results[0].StatusCode != uaGopcua.StatusOK {
		b.logger.Error(err.Error())
	}

	go sub.Run(b.ctx)

	for {
		select {
		case <-b.ctx.Done():
			return nil
		case res := <-sub.Notifs:
			if res.Error != nil {
				b.logger.Error(res.Error.Error())
				continue
			}

			switch x := res.Value.(type) {
			case *uaGopcua.DataChangeNotification:
				for _, item := range x.MonitoredItems {
					// Publish on Mainflux NATS broker
					msg := opcua.Message{
						Namespace: cfg.NodeNamespace,
						ID:        cfg.NodeIdintifier,
						Data:      item.Value.Value.Float(),
					}
					b.svc.Publish(b.ctx, "", msg)
				}

			default:
				b.logger.Info(fmt.Sprintf("what's this publish result? %T", res.Value))
			}
		}
	}
}
