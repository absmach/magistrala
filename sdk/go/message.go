//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package sdk

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

func (sdk mfSDK) SendMessage(chanName, msg, token string) error {

	chanNameParts := strings.SplitN(chanName, ".", 2)
	chanID := chanNameParts[0]
	subtopicPart := ""
	if len(chanNameParts) == 2 {
		subtopicPart = fmt.Sprintf("/%s", strings.Replace(chanNameParts[1], ".", "/", -1))
	}

	endpoint := fmt.Sprintf("channels/%s/messages%s", chanID, subtopicPart)
	url := createURL(sdk.baseURL, sdk.httpAdapterPrefix, endpoint)

	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(msg))
	if err != nil {
		return err
	}

	resp, err := sdk.sendRequest(req, token, string(sdk.msgContentType))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusAccepted {
		switch resp.StatusCode {
		case http.StatusBadRequest:
			return ErrInvalidArgs
		case http.StatusForbidden:
			return ErrUnauthorized
		default:
			return ErrFailedPublish
		}
	}

	return nil
}

func (sdk mfSDK) ReadMessages(chanName, token string) (MessagesPage, error) {
	chanNameParts := strings.SplitN(chanName, ".", 2)
	chanID := chanNameParts[0]
	subtopicPart := ""
	if len(chanNameParts) == 2 {
		subtopicPart = fmt.Sprintf("?subtopic=%s", strings.Replace(chanNameParts[1], ".", "/", -1))
	}

	endpoint := fmt.Sprintf("channels/%s/messages%s", chanID, subtopicPart)
	url := createURL(sdk.readerURL, "", endpoint)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return MessagesPage{}, err
	}

	resp, err := sdk.sendRequest(req, token, string(sdk.msgContentType))
	if err != nil {
		return MessagesPage{}, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return MessagesPage{}, err
	}

	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case http.StatusBadRequest:
			return MessagesPage{}, ErrInvalidArgs
		case http.StatusForbidden:
			return MessagesPage{}, ErrUnauthorized
		default:
			return MessagesPage{}, ErrFailedRead
		}
	}

	mp := messagesPageRes{}
	if err := json.Unmarshal(body, &mp); err != nil {
		return MessagesPage{}, err
	}

	return MessagesPage{
		Total:    mp.Total,
		Offset:   mp.Offset,
		Limit:    mp.Limit,
		Messages: mp.Messages,
	}, nil
}

func (sdk *mfSDK) SetContentType(ct ContentType) error {
	if ct != CTJSON && ct != CTJSONSenML && ct != CTBinary {
		return ErrInvalidContentType
	}

	sdk.msgContentType = ct

	return nil
}
