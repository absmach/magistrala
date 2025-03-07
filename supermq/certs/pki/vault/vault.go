// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package pki wraps vault client
package pki

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/absmach/supermq/pkg/errors"
	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/api/auth/approle"
	"github.com/mitchellh/mapstructure"
)

const (
	issue  = "issue"
	cert   = "cert"
	revoke = "revoke"
)

var (
	errFailedCertDecoding = errors.New("failed to decode response from vault service")
	errFailedToLogin      = errors.New("failed to login to Vault")
	errFailedAppRole      = errors.New("failed to create vault new app role")
	errNoAuthInfo         = errors.New("no auth information from Vault")
	errNonRenewal         = errors.New("token is not configured to be renewable")
	errRenewWatcher       = errors.New("unable to initialize new lifetime watcher for renewing auth token")
	errFailedRenew        = errors.New("failed to renew token")
	errCouldNotRenew      = errors.New("token can no longer be renewed")
)

type Cert struct {
	ClientCert     string   `json:"client_cert" mapstructure:"certificate"`
	IssuingCA      string   `json:"issuing_ca" mapstructure:"issuing_ca"`
	CAChain        []string `json:"ca_chain" mapstructure:"ca_chain"`
	ClientKey      string   `json:"client_key" mapstructure:"private_key"`
	PrivateKeyType string   `json:"private_key_type" mapstructure:"private_key_type"`
	Serial         string   `json:"serial" mapstructure:"serial_number"`
	Expire         int64    `json:"expire" mapstructure:"expiration"`
}

// Agent represents the Vault PKI interface.
type Agent interface {
	// IssueCert issues certificate on PKI
	IssueCert(cn, ttl string) (Cert, error)

	// Read retrieves certificate from PKI
	Read(serial string) (Cert, error)

	// Revoke revokes certificate from PKI
	Revoke(serial string) (time.Time, error)

	// Login to PKI and renews token
	LoginAndRenew(ctx context.Context) error
}

type pkiAgent struct {
	appRole   string
	appSecret string
	namespace string
	path      string
	role      string
	host      string
	issueURL  string
	readURL   string
	revokeURL string
	client    *api.Client
	secret    *api.Secret
	logger    *slog.Logger
}

type certReq struct {
	CommonName string `json:"common_name"`
	TTL        string `json:"ttl"`
}

type certRevokeReq struct {
	SerialNumber string `json:"serial_number"`
}

// NewVaultClient instantiates a Vault client.
func NewVaultClient(appRole, appSecret, host, namespace, path, role string, logger *slog.Logger) (Agent, error) {
	conf := api.DefaultConfig()
	conf.Address = host

	client, err := api.NewClient(conf)
	if err != nil {
		return nil, err
	}
	if namespace != "" {
		client.SetNamespace(namespace)
	}

	p := pkiAgent{
		appRole:   appRole,
		appSecret: appSecret,
		host:      host,
		namespace: namespace,
		role:      role,
		path:      path,
		client:    client,
		logger:    logger,
		issueURL:  "/" + path + "/" + issue + "/" + role,
		readURL:   "/" + path + "/" + cert + "/",
		revokeURL: "/" + path + "/" + revoke,
	}
	return &p, nil
}

func (p *pkiAgent) IssueCert(cn, ttl string) (Cert, error) {
	cReq := certReq{
		CommonName: cn,
		TTL:        ttl,
	}

	var certIssueReq map[string]interface{}
	data, err := json.Marshal(cReq)
	if err != nil {
		return Cert{}, err
	}
	if err := json.Unmarshal(data, &certIssueReq); err != nil {
		return Cert{}, nil
	}

	s, err := p.client.Logical().Write(p.issueURL, certIssueReq)
	if err != nil {
		return Cert{}, err
	}

	cert := Cert{}
	if err = mapstructure.Decode(s.Data, &cert); err != nil {
		return Cert{}, errors.Wrap(errFailedCertDecoding, err)
	}

	return cert, nil
}

func (p *pkiAgent) Read(serial string) (Cert, error) {
	s, err := p.client.Logical().Read(p.readURL + serial)
	if err != nil {
		return Cert{}, err
	}
	cert := Cert{}
	if err = mapstructure.Decode(s.Data, &cert); err != nil {
		return Cert{}, errors.Wrap(errFailedCertDecoding, err)
	}
	return cert, nil
}

func (p *pkiAgent) Revoke(serial string) (time.Time, error) {
	cReq := certRevokeReq{
		SerialNumber: serial,
	}

	var certRevokeReq map[string]interface{}
	data, err := json.Marshal(cReq)
	if err != nil {
		return time.Time{}, err
	}
	if err := json.Unmarshal(data, &certRevokeReq); err != nil {
		return time.Time{}, nil
	}

	s, err := p.client.Logical().Write(p.revokeURL, certRevokeReq)
	if err != nil {
		return time.Time{}, err
	}

	// Vault will return a response without errors but with a warning if the certificate is expired.
	// The response will not have "revocation_time" in such cases.
	if revokeTime, ok := s.Data["revocation_time"]; ok {
		switch v := revokeTime.(type) {
		case json.Number:
			rev, err := v.Float64()
			if err != nil {
				return time.Time{}, err
			}
			return time.Unix(0, int64(rev)*int64(time.Second)), nil

		default:
			return time.Time{}, fmt.Errorf("unsupported type for revocation_time: %T", v)
		}
	}

	return time.Time{}, nil
}

func (p *pkiAgent) LoginAndRenew(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			p.logger.Info("pki login and renew function stopping")
			return nil
		default:
			err := p.login(ctx)
			if err != nil {
				p.logger.Info("unable to authenticate to Vault", slog.Any("error", err))
				time.Sleep(5 * time.Second)
				break
			}
			tokenErr := p.manageTokenLifecycle()
			if tokenErr != nil {
				p.logger.Info("unable to start managing token lifecycle", slog.Any("error", tokenErr))
				time.Sleep(5 * time.Second)
			}
		}
	}
}

func (p *pkiAgent) login(ctx context.Context) error {
	secretID := &approle.SecretID{FromString: p.appSecret}

	authMethod, err := approle.NewAppRoleAuth(
		p.appRole,
		secretID,
	)
	if err != nil {
		return errors.Wrap(errFailedAppRole, err)
	}
	if p.namespace != "" {
		p.client.SetNamespace(p.namespace)
	}
	secret, err := p.client.Auth().Login(ctx, authMethod)
	if err != nil {
		return errors.Wrap(errFailedToLogin, err)
	}
	if secret == nil {
		return errNoAuthInfo
	}
	p.secret = secret
	return nil
}

func (p *pkiAgent) manageTokenLifecycle() error {
	renew := p.secret.Auth.Renewable
	if !renew {
		return errNonRenewal
	}

	watcher, err := p.client.NewLifetimeWatcher(&api.LifetimeWatcherInput{
		Secret:    p.secret,
		Increment: 3600, // Requesting token for 3600s = 1h, If this is more than token_max_ttl, then response token will have token_max_ttl
	})
	if err != nil {
		return errors.Wrap(errRenewWatcher, err)
	}

	go watcher.Start()
	defer watcher.Stop()

	for {
		select {
		case err := <-watcher.DoneCh():
			if err != nil {
				return errors.Wrap(errFailedRenew, err)
			}
			// This occurs once the token has reached max TTL or if token is disabled for renewal.
			return errCouldNotRenew

		case renewal := <-watcher.RenewCh():
			p.logger.Info("Successfully renewed token", slog.Any("renewed_at", renewal.RenewedAt))
		}
	}
}
