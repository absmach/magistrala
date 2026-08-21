// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/absmach/magistrala/pkg/errors"
)

func (sdk mgSDK) ReadMessages(ctx context.Context, pm MessagePageMetadata, chanName, workspaceID, token string) (MessagesPage, errors.SDKError) {
	chanNameParts := strings.SplitN(chanName, "/", channelParts)
	chanID := chanNameParts[0]
	subtopicPart := ""
	if len(chanNameParts) == channelParts {
		subtopicPart = fmt.Sprintf("?subtopic=%s", chanNameParts[1])
	}

	msgURL, err := sdk.withMessageQueryParams(sdk.readersURL, fmt.Sprintf("%s/channels/%s/messages%s", workspaceID, chanID, subtopicPart), pm)
	if err != nil {
		return MessagesPage{}, errors.NewSDKError(err)
	}

	header := make(map[string]string)
	header["Content-Type"] = string(sdk.msgContentType)

	_, body, sdkerr := sdk.processRequest(ctx, http.MethodGet, msgURL, token, nil, header, http.StatusOK)
	if sdkerr != nil {
		return MessagesPage{}, sdkerr
	}

	var mp MessagesPage
	if err := json.Unmarshal(body, &mp); err != nil {
		return MessagesPage{}, errors.NewSDKError(err)
	}

	return mp, nil
}

// ListGatewayDevices lists the devices observed publishing through a gateway
// on a channel (MG-15).
func (sdk mgSDK) ListGatewayDevices(ctx context.Context, chanID, publisherID string, pm DeviceViewPageMetadata, workspaceID, token string) (GatewayDevicesPage, errors.SDKError) {
	msgURL, err := sdk.withDeviceViewQueryParams(sdk.readersURL, fmt.Sprintf("%s/channels/%s/devices", workspaceID, chanID), "publisher", publisherID, pm)
	if err != nil {
		return GatewayDevicesPage{}, errors.NewSDKError(err)
	}

	header := make(map[string]string)
	header["Content-Type"] = string(sdk.msgContentType)

	_, body, sdkerr := sdk.processRequest(ctx, http.MethodGet, msgURL, token, nil, header, http.StatusOK)
	if sdkerr != nil {
		return GatewayDevicesPage{}, sdkerr
	}

	var page GatewayDevicesPage
	if err := json.Unmarshal(body, &page); err != nil {
		return GatewayDevicesPage{}, errors.NewSDKError(err)
	}

	return page, nil
}

// ListDeviceGateways lists the gateways observed relaying for a device on a
// channel (MG-15).
func (sdk mgSDK) ListDeviceGateways(ctx context.Context, chanID, deviceID string, pm DeviceViewPageMetadata, workspaceID, token string) (DeviceGatewaysPage, errors.SDKError) {
	msgURL, err := sdk.withDeviceViewQueryParams(sdk.readersURL, fmt.Sprintf("%s/channels/%s/publishers", workspaceID, chanID), "device_id", deviceID, pm)
	if err != nil {
		return DeviceGatewaysPage{}, errors.NewSDKError(err)
	}

	header := make(map[string]string)
	header["Content-Type"] = string(sdk.msgContentType)

	_, body, sdkerr := sdk.processRequest(ctx, http.MethodGet, msgURL, token, nil, header, http.StatusOK)
	if sdkerr != nil {
		return DeviceGatewaysPage{}, sdkerr
	}

	var page DeviceGatewaysPage
	if err := json.Unmarshal(body, &page); err != nil {
		return DeviceGatewaysPage{}, errors.NewSDKError(err)
	}

	return page, nil
}

// withDeviceViewQueryParams builds the query string for one of the MG-15
// observed-device endpoints: idKey/idVal is the anchoring gateway publisher
// id or device serial, sent as a query parameter rather than a URL path
// segment because a device serial is format-unconstrained (MG-09) and may
// contain characters, such as `/`, that a path segment cannot carry
// verbatim.
func (sdk mgSDK) withDeviceViewQueryParams(baseURL, endpoint, idKey, idVal string, pm DeviceViewPageMetadata) (string, error) {
	b, err := json.Marshal(pm)
	if err != nil {
		return "", err
	}
	q := map[string]any{}
	if err := json.Unmarshal(b, &q); err != nil {
		return "", err
	}
	ret := url.Values{}
	ret.Add(idKey, idVal)
	for k, v := range q {
		switch t := v.(type) {
		case string:
			ret.Add(k, t)
		case float64:
			ret.Add(k, strconv.FormatFloat(t, 'f', -1, 64))
		}
	}
	qs := ret.Encode()

	return fmt.Sprintf("%s/%s?%s", baseURL, endpoint, qs), nil
}

func (sdk mgSDK) withMessageQueryParams(baseURL, endpoint string, mpm MessagePageMetadata) (string, error) {
	b, err := json.Marshal(mpm)
	if err != nil {
		return "", err
	}
	q := map[string]any{}
	if err := json.Unmarshal(b, &q); err != nil {
		return "", err
	}
	ret := url.Values{}
	for k, v := range q {
		switch t := v.(type) {
		case string:
			ret.Add(k, t)
		case float64:
			ret.Add(k, strconv.FormatFloat(t, 'f', -1, 64))
		case uint64:
			ret.Add(k, strconv.FormatUint(t, 10))
		case int64:
			ret.Add(k, strconv.FormatInt(t, 10))
		case json.Number:
			ret.Add(k, t.String())
		case bool:
			ret.Add(k, strconv.FormatBool(t))
		case []any:
			// List-valued filters (publishers, device_ids) go out as repeated
			// query parameters so each value survives verbatim.
			for _, e := range t {
				if s, ok := e.(string); ok {
					ret.Add(k, s)
				}
			}
		}
	}
	qs := ret.Encode()

	return fmt.Sprintf("%s/%s?%s", baseURL, endpoint, qs), nil
}
