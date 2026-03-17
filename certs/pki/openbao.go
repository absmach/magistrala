// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package pki wraps OpenBao client for PKI operations
package pki

import (
	"context"
	"crypto"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/absmach/supermq/certs"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/mitchellh/mapstructure"
	"github.com/openbao/openbao/api/v2"
	"golang.org/x/crypto/ocsp"
)

const (
	issue        = "issue"
	sign         = "sign"
	cert         = "cert"
	revoke       = "revoke"
	ca           = "ca"
	caChain      = "ca_chain"
	crl          = "crl"
	ocspPath     = "ocsp"
	certsList    = "certs"
	signVerbatim = "sign-verbatim"

	defaultRenewThreshold      = 24 * time.Hour
	defaultSecretIDTTL         = 72 * time.Hour
	defaultSecretCheckInterval = 30 * time.Second
)

var (
	errFailedToLogin = errors.New("failed to login to OpenBao")
	errNoAuthInfo    = errors.New("no auth information from OpenBao")
	errRenewWatcher  = errors.New("unable to initialize new lifetime watcher for renewing auth token")
)

type openbaoPKIAgent struct {
	appRole             string
	appSecret           string
	namespace           string
	path                string
	intermediatePath    string
	role                string
	host                string
	issueURL            string
	signURL             string
	signVerbatimURL     string
	readURL             string
	revokeURL           string
	caURL               string
	caChainURL          string
	rootCAURL           string
	rootCAChainURL      string
	crlURL              string
	ocspURL             string
	certsURL            string
	secretIDPath        string
	client              *api.Client
	secret              *api.Secret
	logger              *slog.Logger
	serviceToken        string
	renewThreshold      time.Duration
	secretCheckInterval time.Duration
	mu                  sync.RWMutex
	secretAccessor      string
	secretCreatedAt     time.Time
	secretTTL           time.Duration
}

// NewAgent instantiates an OpenBao PKI client that implements certs.Agent.
func NewAgent(appRole, appSecret, host, namespace, path, role, serviceToken, renewThreshold, secretIDTTL, secretCheckInterval string, logger *slog.Logger) (certs.Agent, error) {
	conf := api.DefaultConfig()
	conf.Address = host

	client, err := api.NewClient(conf)
	if err != nil {
		return nil, err
	}
	if namespace != "" {
		client.SetNamespace(namespace)
	}

	intermediatePath := path + "_int"

	renewDuration, err := time.ParseDuration(renewThreshold)
	if err != nil {
		logger.Warn("Invalid renew threshold duration, using default 24h", "error", err, "provided", renewThreshold)
		renewDuration = defaultRenewThreshold
	}

	ttlDuration, err := time.ParseDuration(secretIDTTL)
	if err != nil {
		logger.Warn("Invalid secret ID TTL duration, using default 72h", "error", err, "provided", secretIDTTL)
		ttlDuration = defaultSecretIDTTL
	}

	checkInterval, err := time.ParseDuration(secretCheckInterval)
	if err != nil {
		logger.Warn("Invalid secret check interval duration, using default 30s", "error", err, "provided", secretCheckInterval)
		checkInterval = defaultSecretCheckInterval
	}

	p := openbaoPKIAgent{
		appRole:             appRole,
		appSecret:           appSecret,
		host:                host,
		namespace:           namespace,
		role:                role,
		path:                path,
		intermediatePath:    intermediatePath,
		client:              client,
		logger:              logger,
		serviceToken:        serviceToken,
		renewThreshold:      renewDuration,
		secretCheckInterval: checkInterval,
		secretTTL:           ttlDuration,
		issueURL:            fmt.Sprintf("%s/%s/%s", intermediatePath, issue, role),
		signURL:             fmt.Sprintf("%s/%s/%s", intermediatePath, sign, role),
		signVerbatimURL:     fmt.Sprintf("%s/%s/%s", intermediatePath, signVerbatim, role),
		readURL:             fmt.Sprintf("%s/%s/", intermediatePath, cert),
		revokeURL:           fmt.Sprintf("%s/%s", intermediatePath, revoke),
		caURL:               fmt.Sprintf("%s/%s", intermediatePath, ca),
		caChainURL:          fmt.Sprintf("%s/%s", intermediatePath, caChain),
		rootCAURL:           fmt.Sprintf("%s/%s", path, ca),
		rootCAChainURL:      fmt.Sprintf("%s/%s", path, caChain),
		crlURL:              fmt.Sprintf("%s/%s", intermediatePath, crl),
		ocspURL:             fmt.Sprintf("%s/%s", intermediatePath, ocspPath),
		certsURL:            fmt.Sprintf("%s/%s", intermediatePath, certsList),
		secretIDPath:        fmt.Sprintf("auth/approle/role/%s/secret-id", role),
	}
	return &p, nil
}

func (agent *openbaoPKIAgent) getIntermediateCADefaultSANs() ([]string, []string, error) {
	err := agent.LoginAndRenew()
	if err != nil {
		return nil, nil, err
	}

	certData, err := agent.GetCA()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get intermediate CA certificate: %w", err)
	}

	block, _ := pem.Decode(certData)
	if block == nil {
		return nil, nil, fmt.Errorf("failed to decode intermediate CA certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse intermediate CA certificate: %w", err)
	}

	var ipSANs []string
	for _, ip := range cert.IPAddresses {
		ipSANs = append(ipSANs, ip.String())
	}

	return cert.DNSNames, ipSANs, nil
}

func (agent *openbaoPKIAgent) Issue(ttl string, ipAddrs []string, options certs.SubjectOptions) (certs.Certificate, error) {
	err := agent.LoginAndRenew()
	if err != nil {
		return certs.Certificate{}, err
	}

	secretValues := map[string]any{
		"common_name":          options.CommonName,
		"ttl":                  ttl,
		"exclude_cn_from_sans": true,
	}

	if len(options.Organization) > 0 {
		secretValues["organization"] = options.Organization
	}
	if len(options.OrganizationalUnit) > 0 {
		secretValues["ou"] = options.OrganizationalUnit
	}
	if len(options.Country) > 0 {
		secretValues["country"] = options.Country
	}
	if len(options.Province) > 0 {
		secretValues["province"] = options.Province
	}
	if len(options.Locality) > 0 {
		secretValues["locality"] = options.Locality
	}
	if len(options.StreetAddress) > 0 {
		secretValues["street_address"] = options.StreetAddress
	}
	if len(options.PostalCode) > 0 {
		secretValues["postal_code"] = options.PostalCode
	}

	allDNSNames := make([]string, 0)
	allDNSNames = append(allDNSNames, options.DnsNames...)

	defaultDNSNames, defaultIPSANs, err := agent.getIntermediateCADefaultSANs()
	if err != nil {
		agent.logger.Warn("failed to get default SANs from intermediate CA", "error", err)
	} else {
		for _, defaultDNS := range defaultDNSNames {
			found := false
			for _, existing := range allDNSNames {
				if existing == defaultDNS {
					found = true
					break
				}
			}
			if !found {
				allDNSNames = append(allDNSNames, defaultDNS)
			}
		}
	}

	if len(allDNSNames) > 0 {
		altNamesValue := strings.Join(allDNSNames, ",")
		secretValues["alt_names"] = altNamesValue
	}

	allIPs := make([]string, 0)
	allIPs = append(allIPs, ipAddrs...)
	for _, ip := range options.IpAddresses {
		allIPs = append(allIPs, ip.String())
	}

	for _, defaultIP := range defaultIPSANs {
		found := false
		for _, existing := range allIPs {
			if existing == defaultIP {
				found = true
				break
			}
		}
		if !found {
			allIPs = append(allIPs, defaultIP)
		}
	}

	if len(allIPs) > 0 {
		ipSansValue := strings.Join(allIPs, ",")
		secretValues["ip_sans"] = ipSansValue
	}

	secret, err := agent.client.Logical().Write(agent.issueURL, secretValues)
	if err != nil {
		return certs.Certificate{}, err
	}

	if secret == nil || secret.Data == nil {
		return certs.Certificate{}, fmt.Errorf("no certificate data returned from OpenBao")
	}

	cert := certs.Certificate{}

	if certData, ok := secret.Data["certificate"].(string); ok {
		cert.Certificate = []byte(certData)
	}

	if keyData, ok := secret.Data["private_key"].(string); ok {
		cert.Key = []byte(keyData)
	}

	if serialNumber, ok := secret.Data["serial_number"].(string); ok {
		cert.SerialNumber = serialNumber
	}

	if expirationInterface, ok := secret.Data["expiration"]; ok {
		switch exp := expirationInterface.(type) {
		case int64:
			cert.ExpiryTime = time.Unix(exp, 0)
		case float64:
			cert.ExpiryTime = time.Unix(int64(exp), 0)
		case json.Number:
			if expInt, err := exp.Int64(); err == nil {
				cert.ExpiryTime = time.Unix(expInt, 0)
			}
		}
	}

	return cert, nil
}

func (agent *openbaoPKIAgent) View(serialNumber string) (certs.Certificate, error) {
	err := agent.LoginAndRenew()
	if err != nil {
		return certs.Certificate{}, err
	}

	secret, err := agent.client.Logical().Read(fmt.Sprintf("%s%s", agent.readURL, serialNumber))
	if err != nil {
		return certs.Certificate{}, err
	}

	if secret == nil || secret.Data == nil {
		return certs.Certificate{}, fmt.Errorf("certificate not found")
	}

	cert := certs.Certificate{
		SerialNumber: serialNumber,
	}

	if certData, ok := secret.Data["certificate"].(string); ok {
		cert.Certificate = []byte(certData)
	}

	cert.Revoked = false
	if revokedTimeStr, ok := secret.Data["revocation_time_rfc3339"].(string); ok && revokedTimeStr != "" {
		cert.Revoked = true
	}

	if len(cert.Certificate) > 0 {
		if expiry, err := agent.parseCertificateExpiry(string(cert.Certificate)); err == nil {
			cert.ExpiryTime = expiry
		}
	}
	return cert, nil
}

func (agent *openbaoPKIAgent) parseCertificateExpiry(certPEM string) (time.Time, error) {
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return time.Time{}, fmt.Errorf("failed to decode PEM certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse X509 certificate: %w", err)
	}

	return cert.NotAfter, nil
}

func (agent *openbaoPKIAgent) Renew(existingCert certs.Certificate, increment string) (certs.Certificate, error) {
	err := agent.LoginAndRenew()
	if err != nil {
		return certs.Certificate{}, err
	}

	block, _ := pem.Decode(existingCert.Certificate)
	if block == nil {
		return certs.Certificate{}, fmt.Errorf("failed to decode existing certificate PEM")
	}

	x509Cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return certs.Certificate{}, fmt.Errorf("failed to parse existing certificate: %w", err)
	}

	options := certs.SubjectOptions{
		CommonName: x509Cert.Subject.CommonName,
		DnsNames:   x509Cert.DNSNames,
	}

	options.IpAddresses = append(options.IpAddresses, x509Cert.IPAddresses...)

	if len(x509Cert.Subject.Organization) > 0 {
		options.Organization = x509Cert.Subject.Organization
	}
	if len(x509Cert.Subject.OrganizationalUnit) > 0 {
		options.OrganizationalUnit = x509Cert.Subject.OrganizationalUnit
	}
	if len(x509Cert.Subject.Country) > 0 {
		options.Country = x509Cert.Subject.Country
	}
	if len(x509Cert.Subject.Province) > 0 {
		options.Province = x509Cert.Subject.Province
	}
	if len(x509Cert.Subject.Locality) > 0 {
		options.Locality = x509Cert.Subject.Locality
	}
	if len(x509Cert.Subject.StreetAddress) > 0 {
		options.StreetAddress = x509Cert.Subject.StreetAddress
	}
	if len(x509Cert.Subject.PostalCode) > 0 {
		options.PostalCode = x509Cert.Subject.PostalCode
	}

	var ipAddrs []string
	for _, ip := range x509Cert.IPAddresses {
		ipAddrs = append(ipAddrs, ip.String())
	}

	newCert, err := agent.Issue(increment, ipAddrs, options)
	if err != nil {
		return certs.Certificate{}, fmt.Errorf("failed to issue renewed certificate: %w", err)
	}

	return newCert, nil
}

func (agent *openbaoPKIAgent) Revoke(serialNumber string) error {
	err := agent.LoginAndRenew()
	if err != nil {
		return err
	}

	secretValues := map[string]any{
		"serial_number": serialNumber,
	}

	_, err = agent.client.Logical().Write(agent.revokeURL, secretValues)
	if err != nil {
		return err
	}
	return nil
}

func (agent *openbaoPKIAgent) ListCerts(pm certs.PageMetadata) (certs.CertificatePage, error) {
	err := agent.LoginAndRenew()
	if err != nil {
		return certs.CertificatePage{}, err
	}

	secret, err := agent.client.Logical().List(agent.certsURL)
	if err != nil {
		return certs.CertificatePage{}, err
	}

	certPage := certs.CertificatePage{
		Certificates: []certs.Certificate{},
		PageMetadata: pm,
	}

	if secret == nil || secret.Data == nil {
		return certPage, nil
	}

	keysInterface, ok := secret.Data["keys"]
	if !ok {
		return certPage, nil
	}

	var serialNumbers []string
	if err := mapstructure.Decode(keysInterface, &serialNumbers); err != nil {
		return certPage, fmt.Errorf("failed to decode certificate serial numbers: %w", err)
	}

	var allCerts []certs.Certificate
	for _, serialNumber := range serialNumbers {
		cert, err := agent.View(serialNumber)
		if err != nil {
			agent.logger.Warn("failed to retrieve certificate details", "serial", serialNumber, "error", err)
			continue
		}

		allCerts = append(allCerts, cert)
	}

	certPage.Total = uint64(len(allCerts))

	start := pm.Offset
	end := pm.Offset + pm.Limit
	if pm.Limit == 0 {
		end = uint64(len(allCerts))
	}
	if start >= uint64(len(allCerts)) {
		return certPage, nil
	}
	if end > uint64(len(allCerts)) {
		end = uint64(len(allCerts))
	}

	for i := start; i < end; i++ {
		certPage.Certificates = append(certPage.Certificates, allCerts[i])
	}

	return certPage, nil
}

func (agent *openbaoPKIAgent) LoginAndRenew() error {
	agent.mu.RLock()
	hasValidToken := agent.secret != nil && agent.secret.Auth != nil && agent.secret.Auth.ClientToken != ""
	agent.mu.RUnlock()

	if hasValidToken {
		_, err := agent.client.Auth().Token().LookupSelf()
		if err == nil {
			return nil
		}
		agent.logger.Warn("Token validation failed, re-authenticating", "error", err)
	}

	agent.mu.RLock()
	roleID := agent.appRole
	secretID := agent.appSecret
	agent.mu.RUnlock()

	authData := map[string]any{
		"role_id":   roleID,
		"secret_id": secretID,
	}

	authResp, err := agent.client.Logical().Write("auth/approle/login", authData)
	if err != nil {
		return fmt.Errorf("%s: %w", errFailedToLogin, err)
	}

	if authResp == nil || authResp.Auth == nil {
		return errNoAuthInfo
	}

	agent.mu.Lock()
	agent.secret = authResp
	agent.client.SetToken(authResp.Auth.ClientToken)
	agent.mu.Unlock()

	if authResp.Auth.Renewable {
		watcher, err := agent.client.NewLifetimeWatcher(&api.LifetimeWatcherInput{
			Secret: authResp,
		})
		if err != nil {
			return fmt.Errorf("%s: %w", errRenewWatcher, err)
		}

		go agent.renewToken(watcher)
	}

	return nil
}

func (agent *openbaoPKIAgent) renewToken(watcher *api.LifetimeWatcher) {
	defer watcher.Stop()

	watcher.Start()
	for {
		select {
		case err := <-watcher.DoneCh():
			if err != nil {
				agent.logger.Error("token renewal failed", "error", err)
			}
			return
		case renewal := <-watcher.RenewCh():
			agent.logger.Info("token renewed successfully", "lease_duration", renewal.Secret.LeaseDuration)
		}
	}
}

func (agent *openbaoPKIAgent) GetCA() ([]byte, error) {
	err := agent.LoginAndRenew()
	if err != nil {
		return nil, err
	}

	secret, err := agent.client.Logical().ReadRaw(agent.caURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get CA certificate: %w", err)
	}
	defer secret.Body.Close()

	if secret.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(secret.Body)
		return nil, fmt.Errorf("failed to get CA certificate: HTTP %d - %s", secret.StatusCode, string(body))
	}

	certData, err := io.ReadAll(secret.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA response: %w", err)
	}

	if len(certData) == 0 {
		return nil, fmt.Errorf("CA certificate response is empty - PKI may not be initialized")
	}

	_, err = x509.ParseCertificate(certData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DER certificate: %w", err)
	}

	pemBlock := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certData,
	}

	pemData := pem.EncodeToMemory(pemBlock)
	return pemData, nil
}

func (agent *openbaoPKIAgent) GetCAChain() ([]byte, error) {
	err := agent.LoginAndRenew()
	if err != nil {
		return nil, err
	}

	secret, err := agent.client.Logical().ReadRaw(agent.caChainURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get CA chain: %w", err)
	}
	defer secret.Body.Close()

	if secret.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get CA chain: HTTP %d", secret.StatusCode)
	}

	chainData, err := io.ReadAll(secret.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA chain response: %w", err)
	}

	return chainData, nil
}

func (agent *openbaoPKIAgent) GetCRL() ([]byte, error) {
	err := agent.LoginAndRenew()
	if err != nil {
		return nil, err
	}

	secret, err := agent.client.Logical().ReadRaw(agent.crlURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get CRL: %w", err)
	}
	defer secret.Body.Close()

	if secret.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get CRL: HTTP %d", secret.StatusCode)
	}

	crlData, err := io.ReadAll(secret.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read CRL response: %w", err)
	}

	return crlData, nil
}

func (agent *openbaoPKIAgent) SignCSR(csr []byte, ttl string) (certs.Certificate, error) {
	err := agent.LoginAndRenew()
	if err != nil {
		return certs.Certificate{}, err
	}

	block, _ := pem.Decode(csr)
	if block == nil {
		return certs.Certificate{}, fmt.Errorf("failed to decode CSR PEM")
	}

	csrData, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		return certs.Certificate{}, fmt.Errorf("failed to parse CSR: %w", err)
	}

	secretValues := map[string]any{
		"csr":            string(csr),
		"ttl":            ttl,
		"use_csr_values": true,
	}

	existingDNSNames := csrData.DNSNames
	var existingIPs []string
	for _, ip := range csrData.IPAddresses {
		existingIPs = append(existingIPs, ip.String())
	}

	defaultDNSNames, defaultIPSANs, err := agent.getIntermediateCADefaultSANs()
	if err != nil {
		defaultDNSNames = []string{}
		defaultIPSANs = []string{}
	}

	allDNSNames := make([]string, 0)
	allDNSNames = append(allDNSNames, existingDNSNames...)

	for _, defaultDNS := range defaultDNSNames {
		found := false
		for _, existing := range allDNSNames {
			if existing == defaultDNS {
				found = true
				break
			}
		}
		if !found {
			allDNSNames = append(allDNSNames, defaultDNS)
		}
	}

	allIPs := make([]string, 0)
	allIPs = append(allIPs, existingIPs...)

	for _, defaultIP := range defaultIPSANs {
		found := false
		for _, existing := range allIPs {
			if existing == defaultIP {
				found = true
				break
			}
		}
		if !found {
			allIPs = append(allIPs, defaultIP)
		}
	}

	if len(allDNSNames) > 0 {
		altNamesValue := strings.Join(allDNSNames, ",")
		secretValues["alt_names"] = altNamesValue
	}

	if len(allIPs) > 0 {
		ipSansValue := strings.Join(allIPs, ",")
		secretValues["ip_sans"] = ipSansValue
	}

	path := agent.signVerbatimURL

	secret, err := agent.client.Logical().Write(path, secretValues)
	if err != nil {
		return certs.Certificate{}, err
	}

	if secret == nil || secret.Data == nil {
		return certs.Certificate{}, fmt.Errorf("no certificate data returned from OpenBao")
	}

	cert := certs.Certificate{}

	if certData, ok := secret.Data["certificate"].(string); ok {
		cert.Certificate = []byte(certData)
	}

	if serialNumber, ok := secret.Data["serial_number"].(string); ok {
		cert.SerialNumber = serialNumber
	}

	if expirationInterface, ok := secret.Data["expiration"]; ok {
		switch exp := expirationInterface.(type) {
		case int64:
			cert.ExpiryTime = time.Unix(exp, 0)
		case float64:
			cert.ExpiryTime = time.Unix(int64(exp), 0)
		case json.Number:
			if expInt, err := exp.Int64(); err == nil {
				cert.ExpiryTime = time.Unix(expInt, 0)
			}
		}
	}

	return cert, nil
}

func (agent *openbaoPKIAgent) OCSP(serialNumber string, ocspRequestDER []byte) ([]byte, error) {
	err := agent.LoginAndRenew()
	if err != nil {
		return nil, err
	}

	var requestDER []byte

	if len(ocspRequestDER) > 0 {
		requestDER = ocspRequestDER
	} else {
		issuerCert, err := agent.getIssuerCertificate()
		if err != nil {
			return nil, fmt.Errorf("failed to get issuer certificate for OCSP: %w", err)
		}

		cert, err := agent.View(serialNumber)
		if err != nil {
			return nil, fmt.Errorf("failed to get certificate for OCSP: %w", err)
		}

		block, _ := pem.Decode(cert.Certificate)
		if block == nil {
			return nil, fmt.Errorf("failed to decode certificate PEM")
		}

		subject, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse certificate: %w", err)
		}

		requestDER, err = ocsp.CreateRequest(subject, issuerCert, &ocsp.RequestOptions{
			Hash: crypto.SHA1,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create OCSP request: %w", err)
		}
	}

	url := fmt.Sprintf("%s/v1/%s/ocsp", agent.host, agent.intermediatePath)
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(string(requestDER)))
	if err != nil {
		return nil, fmt.Errorf("failed to create OCSP POST request: %w", err)
	}

	req.Header.Set("Content-Type", "application/ocsp-request")
	if agent.secret != nil && agent.secret.Auth != nil && agent.secret.Auth.ClientToken != "" {
		req.Header.Set("X-Vault-Token", agent.secret.Auth.ClientToken)
	}
	if agent.namespace != "" {
		req.Header.Set("X-Vault-Namespace", agent.namespace)
	}

	httpClient := agent.client.CloneConfig().HttpClient
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query OpenBao OCSP: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OCSP request failed: HTTP %d - %s", resp.StatusCode, string(body))
	}

	der, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read OCSP response: %w", err)
	}

	_, err = ocsp.ParseResponse(der, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid OCSP response from OpenBao: %w", err)
	}

	return der, nil
}

func (agent *openbaoPKIAgent) getIssuerCertificate() (*x509.Certificate, error) {
	certData, err := agent.GetCA()
	if err != nil {
		return nil, fmt.Errorf("failed to get CA certificate: %w", err)
	}

	block, _ := pem.Decode(certData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse X509 certificate: %w", err)
	}

	return cert, nil
}

func (agent *openbaoPKIAgent) StartSecretRenewal(ctx context.Context) error {
	if agent.serviceToken == "" {
		agent.logger.Info("No service token provided, secret renewal is disabled")
		return nil
	}

	if err := agent.LoginAndRenew(); err != nil {
		return fmt.Errorf("initial login failed: %w", err)
	}

	if err := agent.lookupSecretMetadata(); err != nil {
		agent.logger.Warn("Failed to lookup secret metadata, secret renewal may not work properly", "error", err)
	}

	go agent.monitorSecretExpiration(ctx)

	return nil
}

func (agent *openbaoPKIAgent) lookupSecretMetadata() error {
	tempClient, err := api.NewClient(agent.client.CloneConfig())
	if err != nil {
		return fmt.Errorf("failed to create temp client: %w", err)
	}
	if agent.namespace != "" {
		tempClient.SetNamespace(agent.namespace)
	}
	tempClient.SetToken(agent.serviceToken)

	agent.mu.Lock()
	agent.secretCreatedAt = time.Now().UTC()
	agent.mu.Unlock()

	agent.logger.Info("Secret metadata initialized", "created_at", agent.secretCreatedAt, "ttl", agent.secretTTL.String())
	return nil
}

func (agent *openbaoPKIAgent) monitorSecretExpiration(ctx context.Context) {
	ticker := time.NewTicker(agent.secretCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			agent.logger.Info("Secret renewal monitoring stopped")
			return
		case <-ticker.C:
			agent.mu.RLock()
			createdAt := agent.secretCreatedAt
			ttl := agent.secretTTL
			agent.mu.RUnlock()

			if createdAt.IsZero() {
				continue
			}

			expiryTime := createdAt.Add(ttl)
			timeUntilExpiry := time.Until(expiryTime)

			if timeUntilExpiry <= agent.renewThreshold {
				agent.logger.Warn("Secret ID approaching expiration and will be renewed", "time_until_expiry", timeUntilExpiry, "renew_threshold", agent.renewThreshold, "expiry_time", expiryTime)

				if err := agent.renewSecretID(); err != nil {
					agent.logger.Error("Failed to renew secret ID", "error", err)
					continue
				}

				agent.logger.Info("Successfully renewed secret ID")
			} else {
				agent.logger.Debug("Secret ID still valid", "time_until_expiry", timeUntilExpiry, "expiry_time", expiryTime)
			}
		}
	}
}

func (agent *openbaoPKIAgent) renewSecretID() error {
	tempClient, err := api.NewClient(agent.client.CloneConfig())
	if err != nil {
		return fmt.Errorf("failed to create temp client: %w", err)
	}
	if agent.namespace != "" {
		tempClient.SetNamespace(agent.namespace)
	}
	tempClient.SetToken(agent.serviceToken)

	if err := agent.renewServiceToken(tempClient); err != nil {
		agent.logger.Warn("Failed to renew service token", "error", err)
	}

	secret, err := tempClient.Logical().Write(agent.secretIDPath, map[string]interface{}{})
	if err != nil {
		return fmt.Errorf("failed to generate new secret ID: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return fmt.Errorf("no data returned when generating secret ID")
	}

	newSecretID, ok := secret.Data["secret_id"].(string)
	if !ok || newSecretID == "" {
		return fmt.Errorf("secret_id not found in response")
	}

	secretAccessor, _ := secret.Data["secret_id_accessor"].(string)

	agent.mu.Lock()
	agent.appSecret = newSecretID
	agent.secretAccessor = secretAccessor
	agent.secretCreatedAt = time.Now().UTC()
	agent.secret = nil
	agent.mu.Unlock()

	if err := agent.LoginAndRenew(); err != nil {
		return fmt.Errorf("authentication failed with new secret: %w", err)
	}

	return nil
}

func (agent *openbaoPKIAgent) renewServiceToken(client *api.Client) error {
	renewable, err := client.Auth().Token().RenewSelf(0)
	if err != nil {
		return fmt.Errorf("failed to renew service token: %w", err)
	}

	if renewable == nil || renewable.Auth == nil {
		return fmt.Errorf("no auth information returned from token renewal")
	}

	return nil
}
