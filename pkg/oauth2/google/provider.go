// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package google

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	mfclients "github.com/absmach/magistrala/pkg/clients"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	mgoauth2 "github.com/absmach/magistrala/pkg/oauth2"
	"golang.org/x/oauth2"
	googleoauth2 "golang.org/x/oauth2/google"
)

const (
	providerName = "google"
	defTimeout   = 1 * time.Minute
	userInfoURL  = "https://www.googleapis.com/oauth2/v2/userinfo?access_token="
	tokenInfoURL = "https://oauth2.googleapis.com/tokeninfo?access_token="
)

var scopes = []string{
	"https://www.googleapis.com/auth/userinfo.email",
	"https://www.googleapis.com/auth/userinfo.profile",
}

var _ mgoauth2.Provider = (*config)(nil)

type config struct {
	config        *oauth2.Config
	state         string
	uiRedirectURL string
	errorURL      string
}

// NewProvider returns a new Google OAuth provider.
func NewProvider(cfg mgoauth2.Config, uiRedirectURL, errorURL string) mgoauth2.Provider {
	return &config{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			Endpoint:     googleoauth2.Endpoint,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       scopes,
		},
		state:         cfg.State,
		uiRedirectURL: uiRedirectURL,
		errorURL:      errorURL,
	}
}

func (cfg *config) Name() string {
	return providerName
}

func (cfg *config) State() string {
	return cfg.state
}

func (cfg *config) RedirectURL() string {
	return cfg.uiRedirectURL
}

func (cfg *config) ErrorURL() string {
	return cfg.errorURL
}

func (cfg *config) IsEnabled() bool {
	return cfg.config.ClientID != "" && cfg.config.ClientSecret != ""
}

func (cfg *config) UserDetails(ctx context.Context, code string) (mfclients.Client, oauth2.Token, error) {
	token, err := cfg.config.Exchange(ctx, code)
	if err != nil {
		return mfclients.Client{}, oauth2.Token{}, err
	}
	if token.RefreshToken == "" {
		return mfclients.Client{}, oauth2.Token{}, svcerr.ErrAuthentication
	}

	resp, err := http.Get(userInfoURL + url.QueryEscape(token.AccessToken))
	if err != nil {
		return mfclients.Client{}, oauth2.Token{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return mfclients.Client{}, oauth2.Token{}, svcerr.ErrAuthentication
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return mfclients.Client{}, oauth2.Token{}, err
	}

	var user struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Email   string `json:"email"`
		Picture string `json:"picture"`
	}
	if err := json.Unmarshal(data, &user); err != nil {
		return mfclients.Client{}, oauth2.Token{}, err
	}

	if user.ID == "" || user.Name == "" || user.Email == "" {
		return mfclients.Client{}, oauth2.Token{}, svcerr.ErrAuthentication
	}

	client := mfclients.Client{
		ID:   user.ID,
		Name: user.Name,
		Credentials: mfclients.Credentials{
			Identity: user.Email,
		},
		Metadata: map[string]interface{}{
			"oauth_provider":  providerName,
			"profile_picture": user.Picture,
		},
		Status: mfclients.EnabledStatus,
	}

	return client, *token, nil
}

func (cfg *config) Validate(ctx context.Context, token string) error {
	client := &http.Client{
		Timeout: defTimeout,
	}
	req, err := http.NewRequest(http.MethodGet, tokenInfoURL+token, http.NoBody)
	if err != nil {
		return svcerr.ErrAuthentication
	}
	res, err := client.Do(req)
	if err != nil {
		return svcerr.ErrAuthentication
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return svcerr.ErrAuthentication
	}

	return nil
}

func (cfg *config) Refresh(ctx context.Context, token string) (oauth2.Token, error) {
	payload := strings.NewReader(fmt.Sprintf("grant_type=refresh_token&refresh_token=%s&client_id=%s&client_secret=%s", token, cfg.config.ClientID, cfg.config.ClientSecret))
	client := &http.Client{
		Timeout: defTimeout,
	}
	req, err := http.NewRequest(http.MethodPost, cfg.config.Endpoint.TokenURL, payload)
	if err != nil {
		return oauth2.Token{}, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := client.Do(req)
	if err != nil {
		return oauth2.Token{}, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return oauth2.Token{}, err
	}
	var tokenData oauth2.Token
	if err := json.Unmarshal(body, &tokenData); err != nil {
		return oauth2.Token{}, err
	}
	tokenData.RefreshToken = token

	return tokenData, nil
}
