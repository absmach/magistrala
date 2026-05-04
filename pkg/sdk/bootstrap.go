// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	apiutil "github.com/absmach/magistrala/api/http/util"
	"github.com/absmach/magistrala/pkg/errors"
)

const (
	configsEndpoint          = "clients/configs"
	bootstrapEndpoint        = "clients/bootstrap"
	bootstrapCertsEndpoint   = "clients/configs/certs"
	bootstrapProfilesPath    = "clients/bootstrap/profiles"
	bootstrapEnrollmentsPath = "clients/bootstrap/enrollments"
	secureEndpoint           = "secure"
)

var (
	errInvalidBootstrapStatus       = errors.New("invalid bootstrap status")
	errBootstrapConnectionsDisabled = errors.New("bootstrap connection updates are no longer supported")
)

type BootstrapStatus string

const (
	BootstrapDisabledStatus BootstrapStatus = DisabledStatus
	BootstrapEnabledStatus  BootstrapStatus = EnabledStatus
)

func (s BootstrapStatus) String() string {
	return string(s)
}

func (s BootstrapStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(s))
}

func (s *BootstrapStatus) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}

	if data[0] != '"' {
		var n int
		if err := json.Unmarshal(data, &n); err != nil {
			return err
		}
		switch n {
		case 0:
			*s = BootstrapDisabledStatus
			return nil
		case 1:
			*s = BootstrapEnabledStatus
			return nil
		default:
			return errInvalidBootstrapStatus
		}
	}

	var status string
	if err := json.Unmarshal(data, &status); err != nil {
		return err
	}

	switch strings.ToLower(status) {
	case DisabledStatus:
		*s = BootstrapDisabledStatus
		return nil
	case EnabledStatus:
		*s = BootstrapEnabledStatus
		return nil
	default:
		return errInvalidBootstrapStatus
	}
}

// BootstrapConfig represents a bootstrap enrollment.
type BootstrapConfig struct {
	ID            string          `json:"id,omitempty"`
	ExternalID    string          `json:"external_id,omitempty"`
	ExternalKey   string          `json:"external_key,omitempty"`
	Name          string          `json:"name,omitempty"`
	ClientCert    string          `json:"client_cert,omitempty"`
	ClientKey     string          `json:"client_key,omitempty"`
	CACert        string          `json:"ca_cert,omitempty"`
	Content       string          `json:"content,omitempty"`
	Status        BootstrapStatus `json:"status,omitempty"`
	ProfileID     string          `json:"profile_id,omitempty"`
	RenderContext map[string]any  `json:"render_context,omitempty"`
}

// BootstrapProfile represents a bootstrap profile template.
type BootstrapProfile struct {
	ID              string         `json:"id,omitempty"`
	DomainID        string         `json:"domain_id,omitempty"`
	Name            string         `json:"name,omitempty"`
	Description     string         `json:"description,omitempty"`
	TemplateFormat  string         `json:"template_format,omitempty"`
	ContentTemplate string         `json:"content_template,omitempty"`
	Defaults        map[string]any `json:"defaults,omitempty"`
	BindingSlots    []BindingSlot  `json:"binding_slots,omitempty"`
	Version         int            `json:"version,omitempty"`
	CreatedAt       time.Time      `json:"created_at,omitempty"`
	UpdatedAt       time.Time      `json:"updated_at,omitempty"`
}

// BindingSlot declares a named resource placeholder for a bootstrap profile.
type BindingSlot struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Required bool     `json:"required"`
	Fields   []string `json:"fields,omitempty"`
}

// BootstrapBindingRequest binds a profile slot to a concrete resource.
type BootstrapBindingRequest struct {
	Slot       string `json:"slot"`
	Type       string `json:"type"`
	ResourceID string `json:"resource_id"`
}

// BootstrapBindingSnapshot contains a stored enrollment binding snapshot.
type BootstrapBindingSnapshot struct {
	ConfigID       string         `json:"config_id"`
	Slot           string         `json:"slot"`
	Type           string         `json:"type"`
	ResourceID     string         `json:"resource_id"`
	Snapshot       map[string]any `json:"snapshot,omitempty"`
	SecretSnapshot map[string]any `json:"secret_snapshot,omitempty"`
	UpdatedAt      time.Time      `json:"updated_at,omitempty"`
}

type bootstrapBindingsRes struct {
	Bindings []BootstrapBindingSnapshot `json:"bindings"`
}

func (sdk mgSDK) AddBootstrap(ctx context.Context, cfg BootstrapConfig, domainID, token string) (string, errors.SDKError) {
	data, err := json.Marshal(cfg)
	if err != nil {
		return "", errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s", sdk.bootstrapURL, domainID, configsEndpoint)

	headers, _, sdkerr := sdk.processRequest(ctx, http.MethodPost, url, token, data, nil, http.StatusOK, http.StatusCreated)
	if sdkerr != nil {
		return "", sdkerr
	}

	id := strings.TrimPrefix(headers.Get("Location"), "/clients/configs/")

	return id, nil
}

func (sdk mgSDK) CreateBootstrapProfile(ctx context.Context, profile BootstrapProfile, domainID, token string) (BootstrapProfile, errors.SDKError) {
	data, err := json.Marshal(profile)
	if err != nil {
		return BootstrapProfile{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s", sdk.bootstrapURL, domainID, bootstrapProfilesPath)
	_, body, sdkerr := sdk.processRequest(ctx, http.MethodPost, url, token, data, nil, http.StatusOK, http.StatusCreated)
	if sdkerr != nil {
		return BootstrapProfile{}, sdkerr
	}

	var saved BootstrapProfile
	if err := json.Unmarshal(body, &saved); err != nil {
		return BootstrapProfile{}, errors.NewSDKError(err)
	}

	return saved, nil
}

func (sdk mgSDK) Bootstraps(ctx context.Context, pm PageMetadata, domainID, token string) (BootstrapPage, errors.SDKError) {
	endpoint := fmt.Sprintf("%s/%s", domainID, configsEndpoint)
	url, err := sdk.withQueryParams(sdk.bootstrapURL, endpoint, pm)
	if err != nil {
		return BootstrapPage{}, errors.NewSDKError(err)
	}

	_, body, sdkerr := sdk.processRequest(ctx, http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return BootstrapPage{}, sdkerr
	}

	var bb BootstrapPage
	if err = json.Unmarshal(body, &bb); err != nil {
		return BootstrapPage{}, errors.NewSDKError(err)
	}

	return bb, nil
}

func (sdk mgSDK) BootstrapProfiles(ctx context.Context, pm PageMetadata, domainID, token string) (BootstrapProfilesPage, errors.SDKError) {
	endpoint := fmt.Sprintf("%s/%s", domainID, bootstrapProfilesPath)
	url, err := sdk.withQueryParams(sdk.bootstrapURL, endpoint, pm)
	if err != nil {
		return BootstrapProfilesPage{}, errors.NewSDKError(err)
	}

	_, body, sdkerr := sdk.processRequest(ctx, http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return BootstrapProfilesPage{}, sdkerr
	}

	var page BootstrapProfilesPage
	if err := json.Unmarshal(body, &page); err != nil {
		return BootstrapProfilesPage{}, errors.NewSDKError(err)
	}

	return page, nil
}

func (sdk mgSDK) Whitelist(ctx context.Context, id string, status BootstrapStatus, domainID, token string) errors.SDKError {
	if id == "" {
		return errors.NewSDKError(apiutil.ErrMissingID)
	}

	var action string
	switch status {
	case BootstrapEnabledStatus:
		action = enableEndpoint
	case BootstrapDisabledStatus:
		action = disableEndpoint
	default:
		return errors.NewSDKErrorWithStatus(errInvalidBootstrapStatus, http.StatusBadRequest)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/%s", sdk.bootstrapURL, domainID, configsEndpoint, id, action)

	_, _, sdkerr := sdk.processRequest(ctx, http.MethodPost, url, token, nil, nil, http.StatusOK)

	return sdkerr
}

func (sdk mgSDK) ViewBootstrap(ctx context.Context, id, domainID, token string) (BootstrapConfig, errors.SDKError) {
	if id == "" {
		return BootstrapConfig{}, errors.NewSDKError(apiutil.ErrMissingID)
	}
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.bootstrapURL, domainID, configsEndpoint, id)

	_, body, err := sdk.processRequest(ctx, http.MethodGet, url, token, nil, nil, http.StatusOK)
	if err != nil {
		return BootstrapConfig{}, err
	}

	var bc BootstrapConfig
	if err := json.Unmarshal(body, &bc); err != nil {
		return BootstrapConfig{}, errors.NewSDKError(err)
	}

	return bc, nil
}

func (sdk mgSDK) ViewBootstrapProfile(ctx context.Context, id, domainID, token string) (BootstrapProfile, errors.SDKError) {
	if id == "" {
		return BootstrapProfile{}, errors.NewSDKError(apiutil.ErrMissingID)
	}

	url := fmt.Sprintf("%s/%s/%s/%s", sdk.bootstrapURL, domainID, bootstrapProfilesPath, id)
	_, body, sdkerr := sdk.processRequest(ctx, http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return BootstrapProfile{}, sdkerr
	}

	var profile BootstrapProfile
	if err := json.Unmarshal(body, &profile); err != nil {
		return BootstrapProfile{}, errors.NewSDKError(err)
	}

	return profile, nil
}

func (sdk mgSDK) UpdateBootstrap(ctx context.Context, cfg BootstrapConfig, domainID, token string) errors.SDKError {
	if cfg.ID == "" {
		return errors.NewSDKError(apiutil.ErrMissingID)
	}
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.bootstrapURL, domainID, configsEndpoint, cfg.ID)

	data, err := json.Marshal(cfg)
	if err != nil {
		return errors.NewSDKError(err)
	}

	_, _, sdkerr := sdk.processRequest(ctx, http.MethodPatch, url, token, data, nil, http.StatusOK)

	return sdkerr
}

func (sdk mgSDK) UpdateBootstrapProfile(ctx context.Context, profile BootstrapProfile, domainID, token string) errors.SDKError {
	if profile.ID == "" {
		return errors.NewSDKError(apiutil.ErrMissingID)
	}

	url := fmt.Sprintf("%s/%s/%s/%s", sdk.bootstrapURL, domainID, bootstrapProfilesPath, profile.ID)
	data, err := json.Marshal(profile)
	if err != nil {
		return errors.NewSDKError(err)
	}

	_, _, sdkerr := sdk.processRequest(ctx, http.MethodPatch, url, token, data, nil, http.StatusOK)
	return sdkerr
}

func (sdk mgSDK) UpdateBootstrapCerts(ctx context.Context, id, clientCert, clientKey, ca, domainID, token string) (BootstrapConfig, errors.SDKError) {
	if id == "" {
		return BootstrapConfig{}, errors.NewSDKError(apiutil.ErrMissingID)
	}
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.bootstrapURL, domainID, bootstrapCertsEndpoint, id)
	request := BootstrapConfig{
		ClientCert: clientCert,
		ClientKey:  clientKey,
		CACert:     ca,
	}

	data, err := json.Marshal(request)
	if err != nil {
		return BootstrapConfig{}, errors.NewSDKError(err)
	}

	_, body, sdkerr := sdk.processRequest(ctx, http.MethodPatch, url, token, data, nil, http.StatusOK)
	if sdkerr != nil {
		return BootstrapConfig{}, sdkerr
	}

	var bc BootstrapConfig
	if err := json.Unmarshal(body, &bc); err != nil {
		return BootstrapConfig{}, errors.NewSDKError(err)
	}

	return bc, nil
}

func (sdk mgSDK) UpdateBootstrapConnection(ctx context.Context, id string, channels []string, domainID, token string) errors.SDKError {
	if id == "" {
		return errors.NewSDKError(apiutil.ErrMissingID)
	}
	_ = ctx
	_ = channels
	_ = domainID
	_ = token

	return errors.NewSDKError(errBootstrapConnectionsDisabled)
}

func (sdk mgSDK) RemoveBootstrap(ctx context.Context, id, domainID, token string) errors.SDKError {
	if id == "" {
		return errors.NewSDKError(apiutil.ErrMissingID)
	}
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.bootstrapURL, domainID, configsEndpoint, id)

	_, _, err := sdk.processRequest(ctx, http.MethodDelete, url, token, nil, nil, http.StatusNoContent)
	return err
}

func (sdk mgSDK) RemoveBootstrapProfile(ctx context.Context, id, domainID, token string) errors.SDKError {
	if id == "" {
		return errors.NewSDKError(apiutil.ErrMissingID)
	}

	url := fmt.Sprintf("%s/%s/%s/%s", sdk.bootstrapURL, domainID, bootstrapProfilesPath, id)
	_, _, sdkerr := sdk.processRequest(ctx, http.MethodDelete, url, token, nil, nil, http.StatusNoContent)
	return sdkerr
}

func (sdk mgSDK) AssignBootstrapProfile(ctx context.Context, configID, profileID, domainID, token string) errors.SDKError {
	if configID == "" || profileID == "" {
		return errors.NewSDKError(apiutil.ErrMissingID)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/profile", sdk.bootstrapURL, domainID, bootstrapEnrollmentsPath, configID)
	request := struct {
		ProfileID string `json:"profile_id"`
	}{
		ProfileID: profileID,
	}
	data, err := json.Marshal(request)
	if err != nil {
		return errors.NewSDKError(err)
	}

	_, _, sdkerr := sdk.processRequest(ctx, http.MethodPatch, url, token, data, nil, http.StatusNoContent)
	return sdkerr
}

func (sdk mgSDK) BindBootstrapResources(ctx context.Context, configID string, bindings []BootstrapBindingRequest, domainID, token string) errors.SDKError {
	if configID == "" {
		return errors.NewSDKError(apiutil.ErrMissingID)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/bindings", sdk.bootstrapURL, domainID, bootstrapEnrollmentsPath, configID)
	request := struct {
		Bindings []BootstrapBindingRequest `json:"bindings"`
	}{
		Bindings: bindings,
	}
	data, err := json.Marshal(request)
	if err != nil {
		return errors.NewSDKError(err)
	}

	_, _, sdkerr := sdk.processRequest(ctx, http.MethodPut, url, token, data, nil, http.StatusNoContent)
	return sdkerr
}

func (sdk mgSDK) BootstrapBindings(ctx context.Context, configID, domainID, token string) ([]BootstrapBindingSnapshot, errors.SDKError) {
	if configID == "" {
		return nil, errors.NewSDKError(apiutil.ErrMissingID)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/bindings", sdk.bootstrapURL, domainID, bootstrapEnrollmentsPath, configID)
	_, body, sdkerr := sdk.processRequest(ctx, http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return nil, sdkerr
	}

	var res bootstrapBindingsRes
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, errors.NewSDKError(err)
	}
	if res.Bindings == nil {
		return []BootstrapBindingSnapshot{}, nil
	}

	return res.Bindings, nil
}

func (sdk mgSDK) RefreshBootstrapBindings(ctx context.Context, configID, domainID, token string) errors.SDKError {
	if configID == "" {
		return errors.NewSDKError(apiutil.ErrMissingID)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/bindings/refresh", sdk.bootstrapURL, domainID, bootstrapEnrollmentsPath, configID)
	_, _, sdkerr := sdk.processRequest(ctx, http.MethodPost, url, token, nil, nil, http.StatusNoContent)
	return sdkerr
}

func (sdk mgSDK) Bootstrap(ctx context.Context, externalID, externalKey string) (BootstrapConfig, errors.SDKError) {
	if externalID == "" {
		return BootstrapConfig{}, errors.NewSDKError(apiutil.ErrMissingID)
	}
	url := fmt.Sprintf("%s/%s/%s", sdk.bootstrapURL, bootstrapEndpoint, externalID)

	_, body, err := sdk.processRequest(ctx, http.MethodGet, url, ClientPrefix+externalKey, nil, nil, http.StatusOK)
	if err != nil {
		return BootstrapConfig{}, err
	}

	var bc BootstrapConfig
	if err := json.Unmarshal(body, &bc); err != nil {
		return BootstrapConfig{}, errors.NewSDKError(err)
	}

	return bc, nil
}

func (sdk mgSDK) BootstrapSecure(ctx context.Context, externalID, externalKey, cryptoKey string) (BootstrapConfig, errors.SDKError) {
	if externalID == "" {
		return BootstrapConfig{}, errors.NewSDKError(apiutil.ErrMissingID)
	}
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.bootstrapURL, bootstrapEndpoint, secureEndpoint, externalID)

	encExtKey, err := bootstrapEncrypt([]byte(externalKey), cryptoKey)
	if err != nil {
		return BootstrapConfig{}, errors.NewSDKError(err)
	}

	_, body, sdkErr := sdk.processRequest(ctx, http.MethodGet, url, ClientPrefix+encExtKey, nil, nil, http.StatusOK)
	if sdkErr != nil {
		return BootstrapConfig{}, sdkErr
	}

	decBody, decErr := bootstrapDecrypt(body, cryptoKey)
	if decErr != nil {
		return BootstrapConfig{}, errors.NewSDKError(decErr)
	}
	var bc BootstrapConfig
	if err := json.Unmarshal(decBody, &bc); err != nil {
		return BootstrapConfig{}, errors.NewSDKError(err)
	}

	return bc, nil
}

func bootstrapEncrypt(in []byte, cryptoKey string) (string, error) {
	block, err := aes.NewCipher([]byte(cryptoKey))
	if err != nil {
		return "", err
	}
	ciphertext := make([]byte, aes.BlockSize+len(in))
	iv := ciphertext[:aes.BlockSize]

	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], in)
	return hex.EncodeToString(ciphertext), nil
}

func bootstrapDecrypt(in []byte, cryptoKey string) ([]byte, error) {
	ciphertext := in

	block, err := aes.NewCipher([]byte(cryptoKey))
	if err != nil {
		return nil, err
	}
	if len(ciphertext) < aes.BlockSize {
		return nil, err
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]
	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)
	return ciphertext, nil
}
