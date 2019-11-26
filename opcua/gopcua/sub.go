// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package gopcua

import (
	"context"
	"fmt"
	"time"

	opcuaGopcua "github.com/gopcua/opcua"
	uaGopcua "github.com/gopcua/opcua/ua"
	"github.com/mainflux/mainflux/errors"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/opcua"
)

var (
	errFailedConn          = errors.New("Failed to connect")
	errFailedRead          = errors.New("Failed to read")
	errFailedSub           = errors.New("Failed to subscribe")
	errFailedFindEndpoint  = errors.New("Failed to find suitable endpoint")
	errFailedFetchEndpoint = errors.New("Failed to fetch OPC-UA server endpoints")
	errFailedParseNodeID   = errors.New("Failed to parse NodeID")
	errFailedCreateReq     = errors.New("Failed to create request")
	errResponseStatus      = errors.New("Response status not OK")
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
		return errors.Wrap(errFailedFetchEndpoint, err)
	}

	ep := opcuaGopcua.SelectEndpoint(endpoints, cfg.Policy, uaGopcua.MessageSecurityModeFromString(cfg.Mode))
	if ep == nil {
		return errFailedFindEndpoint
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
	if err := c.Connect(b.ctx); err != nil {
		return errors.Wrap(errFailedConn, err)
	}
	defer c.Close()

	sub, err := c.Subscribe(&opcuaGopcua.SubscriptionParameters{
		Interval: 2000 * time.Millisecond,
	})
	if err != nil {
		return errors.Wrap(errFailedSub, err)
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
	nid := fmt.Sprintf("ns=%s;%s=%s", cfg.NodeNamespace, cfg.NodeIdentifierType, cfg.NodeIdentifier)
	nodeID, err := uaGopcua.ParseNodeID(nid)
	if err != nil {
		return errors.Wrap(errFailedParseNodeID, err)
	}

	// arbitrary client handle for the monitoring item
	handle := uint32(42)
	miCreateRequest := opcuaGopcua.NewMonitoredItemCreateRequestWithDefaults(nodeID, uaGopcua.AttributeIDValue, handle)
	res, err := sub.Monitor(uaGopcua.TimestampsToReturnBoth, miCreateRequest)
	if err != nil {
		return errors.Wrap(errFailedCreateReq, err)
	}
	if res.Results[0].StatusCode != uaGopcua.StatusOK {
		return errResponseStatus
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
						ID:        cfg.NodeIdentifier,
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
