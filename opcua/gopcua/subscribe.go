// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package gopcua

import (
	"context"
	"fmt"
	"strconv"
	"time"

	opcuaGopcua "github.com/gopcua/opcua"
	uaGopcua "github.com/gopcua/opcua/ua"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/opcua"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/messaging"
)

const protocol = "opcua"
const token = ""

var (
	errNotFoundServerURI = errors.New("route map not found for Server URI")
	errNotFoundNodeID    = errors.New("route map not found for Node ID")
	errNotFoundConn      = errors.New("connection not found")

	errFailedConn          = errors.New("failed to connect")
	errFailedParseInterval = errors.New("failed to parse subscription interval")
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
	publisher  messaging.Publisher
	thingsRM   opcua.RouteMapRepository
	channelsRM opcua.RouteMapRepository
	connectRM  opcua.RouteMapRepository
	logger     logger.Logger
}

type message struct {
	ServerURI string
	NodeID    string
	Type      string
	Time      int64
	DataKey   string
	Data      interface{}
}

// NewSubscriber returns new OPC-UA client instance.
func NewSubscriber(ctx context.Context, publisher messaging.Publisher, thingsRM, channelsRM, connectRM opcua.RouteMapRepository, log logger.Logger) opcua.Subscriber {
	return client{
		ctx:        ctx,
		publisher:  publisher,
		thingsRM:   thingsRM,
		channelsRM: channelsRM,
		connectRM:  connectRM,
		logger:     log,
	}
}

// Subscribe subscribes to the OPC-UA Server.
func (c client) Subscribe(ctx context.Context, cfg opcua.Config) error {
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

	i, err := strconv.Atoi(cfg.Interval)
	if err != nil {
		return errors.Wrap(errFailedParseInterval, err)
	}

	sub, err := oc.Subscribe(&opcuaGopcua.SubscriptionParameters{
		Interval: time.Duration(i) * time.Millisecond,
	})
	if err != nil {
		return errors.Wrap(errFailedSub, err)
	}
	defer sub.Cancel()

	if err := c.runHandler(ctx, sub, cfg.ServerURI, cfg.NodeID); err != nil {
		c.logger.Warn(fmt.Sprintf("Unsubscribed from OPC-UA node %s.%s: %s", cfg.ServerURI, cfg.NodeID, err))
	}

	return nil
}

func (c client) runHandler(ctx context.Context, sub *opcuaGopcua.Subscription, uri, node string) error {
	nodeID, err := uaGopcua.ParseNodeID(node)
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

	c.logger.Info(fmt.Sprintf("subscribed to server %s and node_id %s", uri, node))

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
					msg := message{
						ServerURI: uri,
						NodeID:    node,
						Type:      item.Value.Value.Type().String(),
						Time:      item.Value.SourceTimestamp.Unix(),
						DataKey:   "v",
					}

					switch item.Value.Value.Type() {
					case uaGopcua.TypeIDBoolean:
						msg.DataKey = "vb"
						msg.Data = item.Value.Value.Bool()
					case uaGopcua.TypeIDString, uaGopcua.TypeIDByteString:
						msg.DataKey = "vs"
						msg.Data = item.Value.Value.String()
					case uaGopcua.TypeIDDataValue:
						msg.DataKey = "vd"
						msg.Data = item.Value.Value.String()
					case uaGopcua.TypeIDInt64, uaGopcua.TypeIDInt32, uaGopcua.TypeIDInt16:
						msg.Data = float64(item.Value.Value.Int())
					case uaGopcua.TypeIDUint64, uaGopcua.TypeIDUint32, uaGopcua.TypeIDUint16:
						msg.Data = float64(item.Value.Value.Uint())
					case uaGopcua.TypeIDFloat, uaGopcua.TypeIDDouble:
						msg.Data = item.Value.Value.Float()
					case uaGopcua.TypeIDByte:
						msg.Data = float64(item.Value.Value.Uint())
					case uaGopcua.TypeIDDateTime:
						msg.Data = item.Value.Value.Time().Unix()
					default:
						msg.Data = 0
					}

					if err := c.publish(ctx, token, msg); err != nil {
						switch err {
						case errNotFoundServerURI, errNotFoundNodeID, errNotFoundConn:
							return err
						default:
							c.logger.Error(fmt.Sprintf("Failed to publish: %s", err))
						}
					}
				}

			default:
				c.logger.Info(fmt.Sprintf("unknown publish result: %T", res.Value))
			}
		}
	}
}

// Publish forwards messages from the OPC-UA Server to Mainflux NATS broker
func (c client) publish(ctx context.Context, token string, m message) error {
	// Get route-map of the OPC-UA ServerURI
	chanID, err := c.channelsRM.Get(ctx, m.ServerURI)
	if err != nil {
		return errNotFoundServerURI
	}

	// Get route-map of the OPC-UA NodeID
	thingID, err := c.thingsRM.Get(ctx, m.NodeID)
	if err != nil {
		return errNotFoundNodeID
	}

	// Check connection between ServerURI and NodeID
	cKey := fmt.Sprintf("%s:%s", chanID, thingID)
	if _, err := c.connectRM.Get(ctx, cKey); err != nil {
		return fmt.Errorf("%s between channel %s and thing %s", errNotFoundConn, chanID, thingID)
	}

	// Publish on Mainflux NATS broker
	SenML := fmt.Sprintf(`[{"n":"%s", "t": %d, "%s":%v}]`, m.Type, m.Time, m.DataKey, m.Data)
	payload := []byte(SenML)

	msg := messaging.Message{
		Publisher: thingID,
		Protocol:  protocol,
		Channel:   chanID,
		Payload:   payload,
		Subtopic:  m.NodeID,
		Created:   time.Now().UnixNano(),
	}

	if err := c.publisher.Publish(msg.Channel, msg); err != nil {
		return err
	}

	c.logger.Info(fmt.Sprintf("publish from server %s and node_id %s with value %v", m.ServerURI, m.NodeID, m.Data))
	return nil
}
