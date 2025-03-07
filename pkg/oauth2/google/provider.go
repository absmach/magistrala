// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package google

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"

	svcerr "github.com/absmach/supermq/pkg/errors/service"
	mgoauth2 "github.com/absmach/supermq/pkg/oauth2"
	uclient "github.com/absmach/supermq/users"
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

func (cfg *config) Exchange(ctx context.Context, code string) (oauth2.Token, error) {
	token, err := cfg.config.Exchange(ctx, code)
	if err != nil {
		return oauth2.Token{}, err
	}

	return *token, nil
}

func (cfg *config) UserInfo(accessToken string) (uclient.User, error) {
	resp, err := http.Get(userInfoURL + url.QueryEscape(accessToken))
	if err != nil {
		return uclient.User{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return uclient.User{}, svcerr.ErrAuthentication
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return uclient.User{}, err
	}

	var user struct {
		ID        string `json:"id"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Username  string `json:"username"`
		Email     string `json:"email"`
		Picture   string `json:"picture"`
	}
	if err := json.Unmarshal(data, &user); err != nil {
		return uclient.User{}, err
	}

	if user.ID == "" || user.FirstName == "" || user.LastName == "" || user.Email == "" {
		return uclient.User{}, svcerr.ErrAuthentication
	}

	client := uclient.User{
		ID:        user.ID,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     user.Email,
		Metadata: map[string]interface{}{
			"oauth_provider":  providerName,
			"profile_picture": user.Picture,
		},
		Status: uclient.EnabledStatus,
	}

	return client, nil
}
