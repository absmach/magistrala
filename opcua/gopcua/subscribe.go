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
	"github.com/mainflux/mainflux/opcua/db"
)

const protocol = "opcua"
const token = ""

var (
	errNotFoundServerURI = errors.New("route map not found for Server URI")
	errNotFoundNodeID    = errors.New("route map not found for Node ID")
	errNotFoundConn      = errors.New("connection not found")

	errFailedConn          = errors.New("failed to connect")
	errFailedRead          = errors.New("failed to read")
	errFailedSub           = errors.New("failed to subscribe")
	errFailedFindEndpoint  = errors.New("failed to find suitable endpoint")
	errFailedFetchEndpoint = errors.New("failed to fetch OPC-UA server endpoints")
	errFailedParseNodeID   = errors.New("failed to parse NodeID")
	errFailedCreateReq     = errors.New("failed to create request")
	errResponseStatus      = errors.New("response status not OK")
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

// NewSubscriber returns new OPC-UA client instance.
func NewSubscriber(ctx context.Context, pub mainflux.MessagePublisher, thingsRM, channelsRM, connectRM opcua.RouteMapRepository, log logger.Logger) opcua.Subscriber {
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

	// Store subscription details
	if err := db.Save(cfg.ServerURI, cfg.NodeID); err != nil {
		return err
	}

	c.logger.Info(fmt.Sprintf("subscribed to server %s and node_id %s", cfg.ServerURI, cfg.NodeID))

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
						Time:      item.Value.SourceTimestamp.Unix(),
					}

					switch item.Value.Value.Type() {
					case uaGopcua.TypeIDBoolean:
						msg.Data = item.Value.Value.Bool()
					case uaGopcua.TypeIDInt64, uaGopcua.TypeIDInt32, uaGopcua.TypeIDInt16:
						msg.Data = item.Value.Value.Int()
					case uaGopcua.TypeIDUint64:
						msg.Data = item.Value.Value.Uint()
					case uaGopcua.TypeIDFloat, uaGopcua.TypeIDDouble:
						msg.Data = item.Value.Value.Float()
					case uaGopcua.TypeIDString:
						msg.Data = item.Value.Value.String()
					case uaGopcua.TypeIDByte:
						msg.Data = item.Value.Value.EncodingMask()
					default:
						msg.Data = 0
					}

					if err := c.publish(token, msg); err != nil {
						c.logger.Warn(fmt.Sprintf("failed to publish: %s", err))
					}
				}

			default:
				c.logger.Info(fmt.Sprintf("unknown publish result: %T", res.Value))
			}
		}
	}
}

// Publish forwards messages from the OPC-UA Server to Mainflux NATS broker
func (c client) publish(token string, m opcua.Message) error {
	// Get route-map of the OPC-UA ServerURI
	chanID, err := c.channelsRM.Get(m.ServerURI)
	if err != nil {
		return fmt.Errorf("%s %s", errNotFoundServerURI, m.ServerURI)
	}

	// Get route-map of the OPC-UA NodeID
	thingID, err := c.thingsRM.Get(m.NodeID)
	if err != nil {
		return fmt.Errorf("%s %s", errNotFoundNodeID, m.NodeID)
	}

	// Check connection between ServerURI and NodeID
	cKey := fmt.Sprintf("%s:%s", chanID, thingID)
	if _, err := c.connectRM.Get(cKey); err != nil {
		return fmt.Errorf("%s between channel %s and thing %s", errNotFoundConn, chanID, thingID)
	}

	// Publish on Mainflux NATS broker
	SenML := fmt.Sprintf(`[{"n":"%s", "t": %d, "v":%v}]`, m.Type, m.Time, m.Data)
	payload := []byte(SenML)
	msg := mainflux.Message{
		Publisher:   thingID,
		Protocol:    protocol,
		ContentType: "Content-Type",
		Channel:     chanID,
		Payload:     payload,
		Subtopic:    m.NodeID,
	}

	if err := c.publisher.Publish(c.ctx, token, msg); err != nil {
		return err
	}

	c.logger.Info(fmt.Sprintf("publish from server %s and node_id %s with value %v", m.ServerURI, m.NodeID, m.Data))
	return nil
}
