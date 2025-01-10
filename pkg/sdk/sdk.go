// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/absmach/supermq/pkg/errors"
	smqSDK "github.com/absmach/supermq/pkg/sdk"
	"moul.io/http2curl"
)

var _ SDK = (*mgSDK)(nil)

type Metadata map[string]interface{}

type PageMetadata struct {
	Total    uint64   `json:"total"`
	Offset   uint64   `json:"offset"`
	Limit    uint64   `json:"limit"`
	Metadata Metadata `json:"metadata,omitempty"`
	Topic    string   `json:"topic,omitempty"`
	Contact  string   `json:"contact,omitempty"`
	DomainID string   `json:"domain_id,omitempty"`
	Level    uint64   `json:"level,omitempty"`
}

type MessagePageMetadata struct {
	PageMetadata
	Subtopic    string  `json:"subtopic,omitempty"`
	Publisher   string  `json:"publisher,omitempty"`
	Comparator  string  `json:"comparator,omitempty"`
	BoolValue   *bool   `json:"vb,omitempty"`
	StringValue string  `json:"vs,omitempty"`
	DataValue   string  `json:"vd,omitempty"`
	From        float64 `json:"from,omitempty"`
	To          float64 `json:"to,omitempty"`
	Aggregation string  `json:"aggregation,omitempty"`
	Interval    string  `json:"interval,omitempty"`
	Value       float64 `json:"value,omitempty"`
	Protocol    string  `json:"protocol,omitempty"`
}

// SDK contains Magistrala API.
//
//go:generate mockery --name SDK --output=./mocks --filename sdk.go --quiet --note "Copyright (c) Abstract Machines"
type SDK interface {
	smqSDK.SDK

	// AddBootstrap add bootstrap configuration
	//
	// example:
	//  cfg := sdk.BootstrapConfig{
	//    ClientID: "clientID",
	//    Name: "bootstrap",
	//    ExternalID: "externalID",
	//    ExternalKey: "externalKey",
	//    Channels: []string{"channel1", "channel2"},
	//  }
	//  id, _ := sdk.AddBootstrap(cfg, "domainID", "token")
	//  fmt.Println(id)
	AddBootstrap(cfg BootstrapConfig, domainID, token string) (string, errors.SDKError)

	// View returns Client Config with given ID belonging to the user identified by the given token.
	//
	// example:
	//  bootstrap, _ := sdk.ViewBootstrap("id", "domainID", "token")
	//  fmt.Println(bootstrap)
	ViewBootstrap(id, domainID, token string) (BootstrapConfig, errors.SDKError)

	// Update updates editable fields of the provided Config.
	//
	// example:
	//  cfg := sdk.BootstrapConfig{
	//    ClientID: "clientID",
	//    Name: "bootstrap",
	//    ExternalID: "externalID",
	//    ExternalKey: "externalKey",
	//    Channels: []string{"channel1", "channel2"},
	//  }
	//  err := sdk.UpdateBootstrap(cfg, "domainID", "token")
	//  fmt.Println(err)
	UpdateBootstrap(cfg BootstrapConfig, domainID, token string) errors.SDKError

	// Update bootstrap config certificates.
	//
	// example:
	//  err := sdk.UpdateBootstrapCerts("id", "clientCert", "clientKey", "ca", "domainID", "token")
	//  fmt.Println(err)
	UpdateBootstrapCerts(id string, clientCert, clientKey, ca string, domainID, token string) (BootstrapConfig, errors.SDKError)

	// UpdateBootstrapConnection updates connections performs update of the channel list corresponding Client is connected to.
	//
	// example:
	//  err := sdk.UpdateBootstrapConnection("id", []string{"channel1", "channel2"}, "domainID", "token")
	//  fmt.Println(err)
	UpdateBootstrapConnection(id string, channels []string, domainID, token string) errors.SDKError

	// Remove removes Config with specified token that belongs to the user identified by the given token.
	//
	// example:
	//  err := sdk.RemoveBootstrap("id", "domainID", "token")
	//  fmt.Println(err)
	RemoveBootstrap(id, domainID, token string) errors.SDKError

	// Bootstrap returns Config to the Client with provided external ID using external key.
	//
	// example:
	//  bootstrap, _ := sdk.Bootstrap("externalID", "externalKey")
	//  fmt.Println(bootstrap)
	Bootstrap(externalID, externalKey string) (BootstrapConfig, errors.SDKError)

	// BootstrapSecure retrieves a configuration with given external ID and encrypted external key.
	//
	// example:
	//  bootstrap, _ := sdk.BootstrapSecure("externalID", "externalKey", "cryptoKey")
	//  fmt.Println(bootstrap)
	BootstrapSecure(externalID, externalKey, cryptoKey string) (BootstrapConfig, errors.SDKError)

	// Bootstraps retrieves a list of managed configs.
	//
	// example:
	//  pm := sdk.PageMetadata{
	//    Offset: 0,
	//    Limit:  10,
	//  }
	//  bootstraps, _ := sdk.Bootstraps(pm, "domainID", "token")
	//  fmt.Println(bootstraps)
	Bootstraps(pm PageMetadata, domainID, token string) (BootstrapPage, errors.SDKError)

	// Whitelist updates Client state Config with given ID belonging to the user identified by the given token.
	//
	// example:
	//  err := sdk.Whitelist("clientID", 1, "domainID", "token")
	//  fmt.Println(err)
	Whitelist(clientID string, state int, domainID, token string) errors.SDKError

	// ReadMessages read messages of specified channel.
	//
	// example:
	//  pm := sdk.MessagePageMetadata{
	//    Offset: 0,
	//    Limit:  10,
	//  }
	//  msgs, _ := sdk.ReadMessages(pm,"channelID", "domainID", "token")
	//  fmt.Println(msgs)
	ReadMessages(pm MessagePageMetadata, chanID, domainID, token string) (MessagesPage, errors.SDKError)

	// CreateSubscription creates a new subscription
	//
	// example:
	//  subscription, _ := sdk.CreateSubscription("topic", "contact", "token")
	//  fmt.Println(subscription)
	CreateSubscription(topic, contact, token string) (string, errors.SDKError)

	// ListSubscriptions list subscriptions given list parameters.
	//
	// example:
	//  pm := sdk.PageMetadata{
	//    Offset: 0,
	//    Limit:  10,
	//  }
	//  subscriptions, _ := sdk.ListSubscriptions(pm, "token")
	//  fmt.Println(subscriptions)
	ListSubscriptions(pm PageMetadata, token string) (SubscriptionPage, errors.SDKError)

	// ViewSubscription retrieves a subscription with the provided id.
	//
	// example:
	//  subscription, _ := sdk.ViewSubscription("id", "token")
	//  fmt.Println(subscription)
	ViewSubscription(id, token string) (Subscription, errors.SDKError)

	// DeleteSubscription removes a subscription with the provided id.
	//
	// example:
	//  err := sdk.DeleteSubscription("id", "token")
	//  fmt.Println(err)
	DeleteSubscription(id, token string) errors.SDKError
}

type mgSDK struct {
	bootstrapURL   string
	readersURL     string
	usersURL       string
	client         *http.Client
	curlFlag       bool
	msgContentType smqSDK.ContentType

	smqSDK.SDK
}

// Config contains sdk configuration parameters.
type Config struct {
	BootstrapURL   string
	CertsURL       string
	HTTPAdapterURL string
	ReaderURL      string
	ClientsURL     string
	UsersURL       string
	GroupsURL      string
	ChannelsURL    string
	DomainsURL     string
	InvitationsURL string
	JournalURL     string
	HostURL        string

	MsgContentType  smqSDK.ContentType
	TLSVerification bool
	CurlFlag        bool
}

// NewSDK returns new supermq SDK instance.
func NewSDK(conf Config) SDK {
	smqSDK := smqSDK.NewSDK(smqSDK.Config{
		CertsURL:       conf.CertsURL,
		HTTPAdapterURL: conf.HTTPAdapterURL,
		ClientsURL:     conf.ClientsURL,
		UsersURL:       conf.UsersURL,
		GroupsURL:      conf.GroupsURL,
		ChannelsURL:    conf.ChannelsURL,
		DomainsURL:     conf.DomainsURL,
		InvitationsURL: conf.InvitationsURL,
		JournalURL:     conf.JournalURL,
		HostURL:        conf.HostURL,

		MsgContentType:  conf.MsgContentType,
		TLSVerification: conf.TLSVerification,
		CurlFlag:        conf.CurlFlag,
	})

	return &mgSDK{
		bootstrapURL:   conf.BootstrapURL,
		readersURL:     conf.ReaderURL,
		usersURL:       conf.UsersURL,
		msgContentType: conf.MsgContentType,

		client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: !conf.TLSVerification,
				},
			},
		},
		curlFlag: conf.CurlFlag,
		SDK:      smqSDK,
	}
}

// processRequest creates and send a new HTTP request, and checks for errors in the HTTP response.
// It then returns the response headers, the response body, and the associated error(s) (if any).
func (sdk mgSDK) processRequest(method, reqUrl, token string, data []byte, headers map[string]string, expectedRespCodes ...int) (http.Header, []byte, errors.SDKError) {
	req, err := http.NewRequest(method, reqUrl, bytes.NewReader(data))
	if err != nil {
		return make(http.Header), []byte{}, errors.NewSDKError(err)
	}

	// Sets a default value for the Content-Type.
	// Overridden if Content-Type is passed in the headers arguments.
	req.Header.Add("Content-Type", string(smqSDK.CTJSON))

	for key, value := range headers {
		req.Header.Add(key, value)
	}

	if token != "" {
		if !strings.Contains(token, smqSDK.ClientPrefix) {
			token = smqSDK.BearerPrefix + token
		}
		req.Header.Set("Authorization", token)
	}

	if sdk.curlFlag {
		curlCommand, err := http2curl.GetCurlCommand(req)
		if err != nil {
			return nil, nil, errors.NewSDKError(err)
		}
		log.Println(curlCommand.String())
	}

	resp, err := sdk.client.Do(req)
	if err != nil {
		return make(http.Header), []byte{}, errors.NewSDKError(err)
	}
	defer resp.Body.Close()

	sdkerr := errors.CheckError(resp, expectedRespCodes...)
	if sdkerr != nil {
		return make(http.Header), []byte{}, sdkerr
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return make(http.Header), []byte{}, errors.NewSDKError(err)
	}

	return resp.Header, body, nil
}

func (sdk mgSDK) withQueryParams(baseURL, endpoint string, pm PageMetadata) (string, error) {
	q, err := pm.query()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/%s?%s", baseURL, endpoint, q), nil
}

func (pm PageMetadata) query() (string, error) {
	q := url.Values{}
	if pm.Offset != 0 {
		q.Add("offset", strconv.FormatUint(pm.Offset, 10))
	}
	if pm.Limit != 0 {
		q.Add("limit", strconv.FormatUint(pm.Limit, 10))
	}
	if pm.Total != 0 {
		q.Add("total", strconv.FormatUint(pm.Total, 10))
	}
	if pm.Metadata != nil {
		md, err := json.Marshal(pm.Metadata)
		if err != nil {
			return "", errors.NewSDKError(err)
		}
		q.Add("metadata", string(md))
	}
	if pm.Topic != "" {
		q.Add("topic", pm.Topic)
	}
	if pm.Contact != "" {
		q.Add("contact", pm.Contact)
	}
	if pm.DomainID != "" {
		q.Add("domain_id", pm.DomainID)
	}
	if pm.Level != 0 {
		q.Add("level", strconv.FormatUint(pm.Level, 10))
	}

	return q.Encode(), nil
}
