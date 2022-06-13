// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/mainflux/mainflux/pkg/errors"
)

func (sdk mfSDK) SendMessage(chanName, msg, key string) error {
	chanNameParts := strings.SplitN(chanName, ".", 2)
	chanID := chanNameParts[0]
	subtopicPart := ""
	if len(chanNameParts) == 2 {
		subtopicPart = fmt.Sprintf("/%s", strings.Replace(chanNameParts[1], ".", "/", -1))
	}

	url := fmt.Sprintf("%s/channels/%s/messages/%s", sdk.httpAdapterURL, chanID, subtopicPart)

	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(msg))
	if err != nil {
		return err
	}

	resp, err := sdk.sendThingRequest(req, key, string(sdk.msgContentType))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusAccepted {
		return errors.Wrap(ErrFailedPublish, errors.New(resp.Status))
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

	url := fmt.Sprintf("%s/channels/%s/messages%s", sdk.readerURL, chanID, subtopicPart)
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
		return MessagesPage{}, errors.Wrap(ErrFailedRead, errors.New(resp.Status))
	}

	var mp MessagesPage
	if err := json.Unmarshal(body, &mp); err != nil {
		return MessagesPage{}, err
	}

	return mp, nil
}

func (sdk mfSDK) SetContentType(ct ContentType) error {
	if ct != CTJSON && ct != CTJSONSenML && ct != CTBinary {
		return ErrInvalidContentType
	}

	sdk.msgContentType = ct

	return nil
}
