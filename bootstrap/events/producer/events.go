// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package producer

import (
	"github.com/absmach/magistrala/bootstrap"
	"github.com/absmach/magistrala/pkg/events"
)

const (
	configPrefix        = "bootstrap.config."
	configCreate        = configPrefix + "create"
	configUpdate        = configPrefix + "update"
	configRemove        = configPrefix + "remove"
	configView          = configPrefix + "view"
	configList          = configPrefix + "list"
	configHandlerRemove = configPrefix + "remove_handler"

	clientPrefix            = "bootstrap.client."
	clientBootstrap         = clientPrefix + "bootstrap"
	clientStateChange       = clientPrefix + "change_state"
	clientUpdateConnections = clientPrefix + "update_connections"
	clientConnect           = clientPrefix + "connect"
	clientDisconnect        = clientPrefix + "disconnect"

	channelPrefix        = "bootstrap.channel."
	channelHandlerRemove = channelPrefix + "remove_handler"
	channelUpdateHandler = channelPrefix + "update_handler"

	certUpdate = "bootstrap.cert.update"
)

var (
	_ events.Event = (*configEvent)(nil)
	_ events.Event = (*removeConfigEvent)(nil)
	_ events.Event = (*bootstrapEvent)(nil)
	_ events.Event = (*changeStateEvent)(nil)
	_ events.Event = (*updateConnectionsEvent)(nil)
	_ events.Event = (*updateCertEvent)(nil)
	_ events.Event = (*listConfigsEvent)(nil)
	_ events.Event = (*removeHandlerEvent)(nil)
)

type configEvent struct {
	bootstrap.Config
	operation string
}

func (ce configEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"state":     ce.State.String(),
		"operation": ce.operation,
	}
	if ce.ClientID != "" {
		val["client_id"] = ce.ClientID
	}
	if ce.Content != "" {
		val["content"] = ce.Content
	}
	if ce.DomainID != "" {
		val["domain_id "] = ce.DomainID
	}
	if ce.Name != "" {
		val["name"] = ce.Name
	}
	if ce.ExternalID != "" {
		val["external_id"] = ce.ExternalID
	}
	if len(ce.Channels) > 0 {
		channels := make([]string, len(ce.Channels))
		for i, ch := range ce.Channels {
			channels[i] = ch.ID
		}
		val["channels"] = channels
	}
	if ce.ClientCert != "" {
		val["client_cert"] = ce.ClientCert
	}
	if ce.ClientKey != "" {
		val["client_key"] = ce.ClientKey
	}
	if ce.CACert != "" {
		val["ca_cert"] = ce.CACert
	}
	if ce.Content != "" {
		val["content"] = ce.Content
	}

	return val, nil
}

type removeConfigEvent struct {
	client string
}

func (rce removeConfigEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"client_id": rce.client,
		"operation": configRemove,
	}, nil
}

type listConfigsEvent struct {
	offset       uint64
	limit        uint64
	fullMatch    map[string]string
	partialMatch map[string]string
}

func (rce listConfigsEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"offset":    rce.offset,
		"limit":     rce.limit,
		"operation": configList,
	}
	if len(rce.fullMatch) > 0 {
		val["full_match"] = rce.fullMatch
	}

	if len(rce.partialMatch) > 0 {
		val["full_match"] = rce.partialMatch
	}
	return val, nil
}

type bootstrapEvent struct {
	bootstrap.Config
	externalID string
	success    bool
}

func (be bootstrapEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"external_id": be.externalID,
		"success":     be.success,
		"operation":   clientBootstrap,
	}

	if be.ClientID != "" {
		val["client_id"] = be.ClientID
	}
	if be.Content != "" {
		val["content"] = be.Content
	}
	if be.DomainID != "" {
		val["domain_id "] = be.DomainID
	}
	if be.Name != "" {
		val["name"] = be.Name
	}
	if be.ExternalID != "" {
		val["external_id"] = be.ExternalID
	}
	if len(be.Channels) > 0 {
		channels := make([]string, len(be.Channels))
		for i, ch := range be.Channels {
			channels[i] = ch.ID
		}
		val["channels"] = channels
	}
	if be.ClientCert != "" {
		val["client_cert"] = be.ClientCert
	}
	if be.ClientKey != "" {
		val["client_key"] = be.ClientKey
	}
	if be.CACert != "" {
		val["ca_cert"] = be.CACert
	}
	if be.Content != "" {
		val["content"] = be.Content
	}
	return val, nil
}

type changeStateEvent struct {
	mgClient string
	state    bootstrap.State
}

func (cse changeStateEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"client_id": cse.mgClient,
		"state":     cse.state.String(),
		"operation": clientStateChange,
	}, nil
}

type updateConnectionsEvent struct {
	mgClient   string
	mgChannels []string
}

func (uce updateConnectionsEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"client_id": uce.mgClient,
		"channels":  uce.mgChannels,
		"operation": clientUpdateConnections,
	}, nil
}

type updateCertEvent struct {
	clientID   string
	clientCert string
	clientKey  string
	caCert     string
}

func (uce updateCertEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"client_id":   uce.clientID,
		"client_cert": uce.clientCert,
		"client_key":  uce.clientKey,
		"ca_cert":     uce.caCert,
		"operation":   certUpdate,
	}, nil
}

type removeHandlerEvent struct {
	id        string
	operation string
}

func (rhe removeHandlerEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"config_id": rhe.id,
		"operation": rhe.operation,
	}, nil
}

type updateChannelHandlerEvent struct {
	bootstrap.Channel
}

func (uche updateChannelHandlerEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": channelUpdateHandler,
	}

	if uche.ID != "" {
		val["channel_id"] = uche.ID
	}
	if uche.Name != "" {
		val["name"] = uche.Name
	}
	if uche.Metadata != nil {
		val["metadata"] = uche.Metadata
	}
	return val, nil
}

type connectClientEvent struct {
	clientID  string
	channelID string
}

func (cte connectClientEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"client_id":  cte.clientID,
		"channel_id": cte.channelID,
		"operation":  clientConnect,
	}, nil
}

type disconnectClientEvent struct {
	clientID  string
	channelID string
}

func (dte disconnectClientEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"client_id":  dte.clientID,
		"channel_id": dte.channelID,
		"operation":  clientDisconnect,
	}, nil
}
