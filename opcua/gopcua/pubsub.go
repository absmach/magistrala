// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package gopcua

import (
	"context"
	"fmt"
	"time"

	opcuaGopcua "github.com/gopcua/opcua"
	uaGopcua "github.com/gopcua/opcua/ua"
	"github.com/mainflux/mainflux"
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
	ctx        context.Context
	publisher  mainflux.MessagePublisher
	thingsRM   opcua.RouteMapRepository
	channelsRM opcua.RouteMapRepository
	connectRM  opcua.RouteMapRepository
	logger     logger.Logger
}

// NewPubSub returns new OPC-UA client instance.
func NewPubSub(ctx context.Context, pub mainflux.MessagePublisher, thingsRM, channelsRM, connectRM opcua.RouteMapRepository, log logger.Logger) opcua.Subscriber {
	return client{
		ctx:        ctx,
		publisher:  pub,
		thingsRM:   thingsRM,
		channelsRM: channelsRM,
		connectRM:  connectRM,
		logger:     log,
	}
}

// Subscribe subscribes to the OPC-UA Server.
func (c client) Subscribe(cfg opcua.Config) error {
	opts := []opcuaGopcua.Option{
		opcuaGopcua.SecurityMode(uaGopcua.MessageSecurityModeNone),
	}

	if cfg.Mode != "" {
		endpoints, err := opcuaGopcua.GetEndpoints(cfg.ServerURI)
		if err != nil {
			return errors.Wrap(errFailedFetchEndpoint, err)
		}

		ep := opcuaGopcua.SelectEndpoint(endpoints, cfg.Policy, uaGopcua.MessageSecurityModeFromString(cfg.Mode))
		if ep == nil {
			return errFailedFindEndpoint
		}

		opts = []opcuaGopcua.Option{
			opcuaGopcua.SecurityPolicy(cfg.Policy),
			opcuaGopcua.SecurityModeString(cfg.Mode),
			opcuaGopcua.CertificateFile(cfg.CertFile),
			opcuaGopcua.PrivateKeyFile(cfg.KeyFile),
			opcuaGopcua.AuthAnonymous(),
			opcuaGopcua.SecurityFromEndpoint(ep, uaGopcua.UserTokenTypeAnonymous),
		}
	}

	oc := opcuaGopcua.NewClient(cfg.ServerURI, opts...)
	if err := oc.Connect(c.ctx); err != nil {
		return errors.Wrap(errFailedConn, err)
	}
	defer oc.Close()

	sub, err := oc.Subscribe(&opcuaGopcua.SubscriptionParameters{
		Interval: 2000 * time.Millisecond,
	})
	if err != nil {
		return errors.Wrap(errFailedSub, err)
	}
	defer sub.Cancel()

	if err := c.runHandler(sub, cfg); err != nil {
		return err
	}

	return nil
}

func (c client) runHandler(sub *opcuaGopcua.Subscription, cfg opcua.Config) error {
	nodeID, err := uaGopcua.ParseNodeID(cfg.NodeID)
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

	go sub.Run(c.ctx)

	for {
		select {
		case <-c.ctx.Done():
			return nil
		case res := <-sub.Notifs:
			if res.Error != nil {
				c.logger.Error(res.Error.Error())
				continue
			}

			switch x := res.Value.(type) {
			case *uaGopcua.DataChangeNotification:
				for _, item := range x.MonitoredItems {
					msg := opcua.Message{
						ServerURI: cfg.ServerURI,
						NodeID:    cfg.NodeID,
						Type:      item.Value.Value.Type().String(),
					}

					switch item.Value.Value.Type() {
					case uaGopcua.TypeIDBoolean:
						msg.Data = item.Value.Value.Bool()
					case uaGopcua.TypeIDInt64:
						msg.Data = item.Value.Value.Int()
					case uaGopcua.TypeIDUint64:
						msg.Data = item.Value.Value.Uint()
					case uaGopcua.TypeIDFloat, uaGopcua.TypeIDDouble:
						msg.Data = item.Value.Value.Float()
					case uaGopcua.TypeIDString:
						msg.Data = item.Value.Value.String()
					default:
						msg.Data = 0
					}

					c.Publish(c.ctx, "", msg)
				}

			default:
				c.logger.Info(fmt.Sprintf("unknown publish result: %T", res.Value))
			}
		}
	}
}

// Publish forwards messages from OPC-UA MQTT broker to Mainflux NATS broker
func (c client) Publish(ctx context.Context, token string, m opcua.Message) error {
	// Get route-map of the OPC-UA ServerURI
	chanID, err := c.channelsRM.Get(m.ServerURI)
	if err != nil {
		return opcua.ErrNotFoundServerURI
	}

	// Get route-map of the OPC-UA NodeID
	thingID, err := c.thingsRM.Get(m.NodeID)
	if err != nil {
		return opcua.ErrNotFoundNodeID
	}

	// Check connection between ServerURI and NodeID
	cKey := fmt.Sprintf("%s:%s", chanID, thingID)
	if _, err := c.connectRM.Get(cKey); err != nil {
		return opcua.ErrNotFoundConn
	}

	// Publish on Mainflux NATS broker
	SenML := fmt.Sprintf(`[{"n":"%s","v":%v}]`, m.Type, m.Data)
	payload := []byte(SenML)
	msg := mainflux.Message{
		Publisher:   thingID,
		Protocol:    "opcua",
		ContentType: "Content-Type",
		Channel:     chanID,
		Payload:     payload,
	}

	if err := c.publisher.Publish(ctx, token, msg); err != nil {
		return err
	}

	c.logger.Info(fmt.Sprintf("publish from server %s and node_id %s with value %v", m.ServerURI, m.NodeID, m.Data))
	return nil
}
