// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/pkg/errors"
)

const channelParts = 2

type publishRequest struct {
	Topic   string `json:"topic"`
	Payload []byte `json:"payload"`
	QoS     byte   `json:"qos"`
	Retain  bool   `json:"retain"`
}

func (sdk mgSDK) SendMessage(ctx context.Context, domainID, topic, msg, secret string) errors.SDKError {
	chanNameParts := strings.SplitN(topic, "/", channelParts)
	chanID := chanNameParts[0]
	brokerTopic := fmt.Sprintf("m/%s/c/%s", domainID, chanID)
	if len(chanNameParts) == channelParts {
		brokerTopic = fmt.Sprintf("%s/%s", brokerTopic, chanNameParts[1])
	}
	data, err := json.Marshal(publishRequest{
		Topic:   brokerTopic,
		Payload: []byte(msg),
	})
	if err != nil {
		return errors.NewSDKError(err)
	}

	headers := map[string]string{
		"X-FluxMQ-Username": domainID,
	}

	reqURL := fmt.Sprintf("%s/publish", sdk.httpAdapterURL)
	_, _, sdkErr := sdk.processRequest(ctx, http.MethodPost, reqURL, secret, data, headers, http.StatusOK)

	return sdkErr
}

func (sdk *mgSDK) SetContentType(ct ContentType) errors.SDKError {
	if ct != CTJSON && ct != CTJSONSenML && ct != CTBinary {
		return errors.NewSDKError(apiutil.ErrUnsupportedContentType)
	}

	sdk.msgContentType = ct

	return nil
}
