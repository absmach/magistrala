// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package producer

import (
	"github.com/absmach/magistrala/bootstrap"
	"github.com/absmach/magistrala/pkg/events"
)

const (
	configPrefix    = "bootstrap.config."
	configCreate    = configPrefix + "create"
	configUpdate    = configPrefix + "update"
	configRemove    = configPrefix + "remove"
	configView      = configPrefix + "view"
	configList      = configPrefix + "list"
	clientPrefix    = "bootstrap.client."
	clientBootstrap = clientPrefix + "bootstrap"
	configEnable    = configPrefix + "enable"
	configDisable   = configPrefix + "disable"
	certUpdate      = "bootstrap.cert.update"

	profilePrefix   = "bootstrap.profile."
	profileCreate   = profilePrefix + "create"
	profileView     = profilePrefix + "view"
	profileUpdate   = profilePrefix + "update"
	profileList     = profilePrefix + "list"
	profileDelete   = profilePrefix + "delete"
	profileAssign   = profilePrefix + "assign"
	bindingsPrefix  = "bootstrap.bindings."
	bindingsBind    = bindingsPrefix + "bind"
	bindingsList    = bindingsPrefix + "list"
	bindingsRefresh = bindingsPrefix + "refresh"
)

var (
	_ events.Event = (*configEvent)(nil)
	_ events.Event = (*removeConfigEvent)(nil)
	_ events.Event = (*bootstrapEvent)(nil)
	_ events.Event = (*enableConfigEvent)(nil)
	_ events.Event = (*disableConfigEvent)(nil)
	_ events.Event = (*updateCertEvent)(nil)
	_ events.Event = (*listConfigsEvent)(nil)
	_ events.Event = (*profileEvent)(nil)
	_ events.Event = (*deleteProfileEvent)(nil)
	_ events.Event = (*assignProfileEvent)(nil)
	_ events.Event = (*bindResourcesEvent)(nil)
	_ events.Event = (*listBindingsEvent)(nil)
	_ events.Event = (*refreshBindingsEvent)(nil)
)

type configEvent struct {
	bootstrap.Config
	operation string
}

func (ce configEvent) Encode() (map[string]any, error) {
	val := map[string]any{
		"status":    ce.Status.String(),
		"operation": ce.operation,
	}
	if ce.ID != "" {
		val["config_id"] = ce.ID
	}
	if ce.WorkspaceID != "" {
		val["workspace_id"] = ce.WorkspaceID
	}
	if ce.Name != "" {
		val["name"] = ce.Name
	}
	if ce.ExternalID != "" {
		val["external_id"] = ce.ExternalID
	}
	// Certificates are public material and stay on the event. The private
	// key and the config content do not: content is the rendered profile,
	// which interpolates decrypted binding secrets, and both would otherwise
	// reach every consumer of the bootstrap event stream in the clear.
	if ce.ClientCert != "" {
		val["client_cert"] = ce.ClientCert
	}
	if ce.CACert != "" {
		val["ca_cert"] = ce.CACert
	}

	return val, nil
}

type removeConfigEvent struct {
	config string
}

func (rce removeConfigEvent) Encode() (map[string]any, error) {
	return map[string]any{
		"config_id": rce.config,
		"operation": configRemove,
	}, nil
}

type listConfigsEvent struct {
	offset       uint64
	limit        uint64
	fullMatch    map[string]string
	partialMatch map[string]string
}

func (rce listConfigsEvent) Encode() (map[string]any, error) {
	val := map[string]any{
		"offset":    rce.offset,
		"limit":     rce.limit,
		"operation": configList,
	}
	if len(rce.fullMatch) > 0 {
		val["full_match"] = rce.fullMatch
	}

	if len(rce.partialMatch) > 0 {
		val["partial_match"] = rce.partialMatch
	}
	return val, nil
}

type bootstrapEvent struct {
	bootstrap.Config
	externalID string
	success    bool
}

func (be bootstrapEvent) Encode() (map[string]any, error) {
	val := map[string]any{
		"external_id": be.externalID,
		"success":     be.success,
		"operation":   clientBootstrap,
	}

	if be.ID != "" {
		val["config_id"] = be.ID
	}
	if be.WorkspaceID != "" {
		val["workspace_id"] = be.WorkspaceID
	}
	if be.Name != "" {
		val["name"] = be.Name
	}
	if be.ExternalID != "" {
		val["external_id"] = be.ExternalID
	}
	// See configEvent.Encode: the rendered content and the device private
	// key are deliberately not published.
	if be.ClientCert != "" {
		val["client_cert"] = be.ClientCert
	}
	if be.CACert != "" {
		val["ca_cert"] = be.CACert
	}
	return val, nil
}

type enableConfigEvent struct {
	configID string
}

func (e enableConfigEvent) Encode() (map[string]any, error) {
	return map[string]any{
		"config_id": e.configID,
		"operation": configEnable,
	}, nil
}

type disableConfigEvent struct {
	configID string
}

func (e disableConfigEvent) Encode() (map[string]any, error) {
	return map[string]any{
		"config_id": e.configID,
		"operation": configDisable,
	}, nil
}

type updateCertEvent struct {
	configID   string
	clientCert string
	caCert     string
}

func (uce updateCertEvent) Encode() (map[string]any, error) {
	// clientKey is the device private key and is never published; see
	// configEvent.Encode.
	return map[string]any{
		"config_id":   uce.configID,
		"client_cert": uce.clientCert,
		"ca_cert":     uce.caCert,
		"operation":   certUpdate,
	}, nil
}

type profileEvent struct {
	bootstrap.Profile
	operation string
}

func (pe profileEvent) Encode() (map[string]any, error) {
	val := map[string]any{
		"operation": pe.operation,
	}
	if pe.ID != "" {
		val["profile_id"] = pe.ID
	}
	if pe.WorkspaceID != "" {
		val["workspace_id"] = pe.WorkspaceID
	}
	if pe.Name != "" {
		val["name"] = pe.Name
	}
	return val, nil
}

type deleteProfileEvent struct {
	profileID string
}

func (dpe deleteProfileEvent) Encode() (map[string]any, error) {
	return map[string]any{
		"profile_id": dpe.profileID,
		"operation":  profileDelete,
	}, nil
}

type assignProfileEvent struct {
	configID  string
	profileID string
}

func (ape assignProfileEvent) Encode() (map[string]any, error) {
	return map[string]any{
		"config_id":  ape.configID,
		"profile_id": ape.profileID,
		"operation":  profileAssign,
	}, nil
}

type bindResourcesEvent struct {
	configID string
	slots    []string
}

func (bre bindResourcesEvent) Encode() (map[string]any, error) {
	return map[string]any{
		"config_id": bre.configID,
		"slots":     bre.slots,
		"operation": bindingsBind,
	}, nil
}

type listBindingsEvent struct {
	configID string
}

func (lbe listBindingsEvent) Encode() (map[string]any, error) {
	return map[string]any{
		"config_id": lbe.configID,
		"operation": bindingsList,
	}, nil
}

type refreshBindingsEvent struct {
	configID string
}

func (rbe refreshBindingsEvent) Encode() (map[string]any, error) {
	return map[string]any{
		"config_id": rbe.configID,
		"operation": bindingsRefresh,
	}, nil
}
