// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package certs_test

import (
	"context"
	"testing"
	"time"

	"github.com/absmach/magistrala/certs"
	"github.com/absmach/magistrala/certs/mocks"
	smqauthn "github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	serialNumber = "20:f4:bd:43:2c:c7:06:82:c7:f2:00:47:51:b6:81:6f:fa:c4:46:0c"
	entityID     = "c1a1daea-ce24-4847-b892-1780bf25b10c"
	domainID     = "domain-id"
	testCertPEM  = "-----BEGIN CERTIFICATE-----\nMIIEMjCCAxqgAwIBAgIUIPS9QyzHBoLH8gBHUbaBb/rERgwwDQYJKoZIhvcNAQEL\nBQAwgaAxDzANBgNVBAYTBkZSQU5DRTEOMAwGA1UECBMFUEFSSVMxDjAMBgNVBAcT\nBVBBUklTMRowGAYDVQQKExFBYnN0cmFjdCBNYWNoaW5lczEaMBgGA1UECxMRQWJz\ndHJhY3QgTWFjaGluZXMxNTAzBgNVBAMTLEFic3RyYWN0IE1hY2hpbmVzIFJvb3Qg\nQ2VydGlmaWNhdGUgQXV0aG9yaXR5MB4XDTI1MDgyNTExNTAyNFoXDTI1MDgyNTIx\nNTA1NFowDzENMAsGA1UEAxMEMDAwMTCCASIwDQYJKoZIhvcNAQEBBQADggEPADCC\nAQoCggEBAMT4eHWFYUVAmQWC0bcgcBuBQjDVWdXD2WJWx8ybeC8vIwsGyCRMEem4\nlveP937ZjM3TTX0Nst4chF0L3WN0FTGTztwlqtpCK67AxcMEdGj54kIlVMAZexLz\nY4mQ5Oe/S4L4elv/ARHDV87BZ0m7oD1b2AC+8CBdm9aWcaD1RZk6qtzLRjs17ouY\nuslj5dN33VuzTYYUlPaTFjCY2nnebK0FLNjJkBVjoIlmT1Oo56uw9SQpLczk4PtL\nlVzeNKHGh0mx3g13tyNOAjKrMvxb7GTQ3tKsL6zZfiWggw4gROqjGQuCejAibfrr\nftN77YndLF4JYqiUZRCsZlRMSkpcSWMCAwEAAaOB8zCB8DAOBgNVHQ8BAf8EBAMC\nA6gwHQYDVR0lBBYwFAYIKwYBBQUHAwEGCCsGAQUFBwMCMB0GA1UdDgQWBBSEDX9D\nU9O6ORjZOJzceZmE2yC93DAfBgNVHSMEGDAWgBSZCSNs3yScbg5YSiuN1VuS6o3g\nyTA7BggrBgEFBQcBAQQvMC0wKwYIKwYBBQUHMAKGH2h0dHA6Ly8xMjcuMC4wLjE6\nODIwMC92MS9wa2kvY2EwDwYDVR0RBAgwBocEwKhkFDAxBgNVHR8EKjAoMCagJKAi\nhiBodHRwOi8vMTI3LjAuMC4xOjgyMDAvdjEvcGtpL2NybDANBgkqhkiG9w0BAQsF\nAAOCAQEAK5fOOweOOJzWmjC0/6A9T/xnTOeXcwdp3gBmMNkaCs/qlh+3Dofo9vHS\nX1vitXbcqbMmJnXuRLkA+qTTlJvhVD8fa4RtixJZ5N0uDMPJ5FVv9tipSoqcnQH8\nwR4iPvrlQQr5hiBt/nfsaTLuDLZgMcKs5N30yHslJXfeLcWrawaQHpIddgavbgqM\n/9L/PoWM2hJknUyg7kis5SNejUGwOh/U1MUf1b18kaUKeK3Q4vhVHVz4foiRZ9M0\niw9xTj2rJJdOE/omE6qJFIfWIF0DuOCYt7z8TKhqKuTfNjmmiqlcgT14P6hniFkK\nl/5upJw86TWS8J0RXQJ1Nbw68EMEuQ==\n-----END CERTIFICATE-----"
)

var (
	certValidityPeriod = time.Hour * 24 * 30
	testSession        = smqauthn.Session{
		DomainUserID: entityID,
		UserID:       entityID,
		DomainID:     domainID,
	}
)

func TestIssueCert(t *testing.T) {
	agent := new(mocks.Agent)
	repo := new(mocks.Repository)
	svc, err := certs.NewService(context.Background(), agent, repo)
	require.NoError(t, err)

	testCases := []struct {
		desc         string
		entityID     string
		ttl          string
		cert         certs.Certificate
		err          error
		agentErr     error
		repoErr      error
		expectedCert certs.Certificate
	}{
		{
			desc:     "issue cert successfully",
			entityID: "entityID",
			ttl:      "1h",
			cert: certs.Certificate{
				SerialNumber: serialNumber,
			},
			expectedCert: certs.Certificate{
				SerialNumber: serialNumber,
				EntityID:     "entityID",
			},
			err: nil,
		},
		{
			desc:     "failed agent issue cert",
			entityID: "entityID",
			ttl:      "1h",
			cert:     certs.Certificate{},
			agentErr: errors.New("agent error"),
			err:      certs.ErrFailedCertCreation,
		},
		{
			desc:     "failed repository save mapping",
			entityID: "entityID",
			ttl:      "1h",
			cert: certs.Certificate{
				SerialNumber: serialNumber,
			},
			repoErr: errors.New("repo error"),
			err:     certs.ErrFailedCertCreation,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			options := certs.SubjectOptions{
				CommonName: tc.entityID,
			}
			agentCall := agent.On("Issue", tc.ttl, []string{}, options).Return(tc.cert, tc.agentErr)
			repoCall := repo.On("SaveCertEntityMapping", mock.Anything, tc.cert.SerialNumber, tc.entityID).Return(tc.repoErr)

			cert, err := svc.IssueCert(context.Background(), testSession, tc.entityID, tc.ttl, []string{}, options)
			if tc.err != nil {
				require.True(t, errors.Contains(err, tc.err), "expected error %v, got %v", tc.err, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedCert, cert)
			}

			agentCall.Unset()
			repoCall.Unset()
		})
	}
}

func TestRevokeBySerial(t *testing.T) {
	agent := new(mocks.Agent)
	repo := new(mocks.Repository)
	svc, err := certs.NewService(context.Background(), agent, repo)
	require.NoError(t, err)

	testCases := []struct {
		desc     string
		serial   string
		agentErr error
		err      error
	}{
		{
			desc:   "revoke cert by serial successfully",
			serial: serialNumber,
			err:    nil,
		},
		{
			desc:     "failed agent revoke",
			serial:   serialNumber,
			agentErr: errors.New("agent error"),
			err:      certs.ErrUpdateEntity,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			agentCall := agent.On("Revoke", tc.serial).Return(tc.agentErr)

			err = svc.RevokeBySerial(context.Background(), testSession, tc.serial)
			if tc.err != nil {
				require.True(t, errors.Contains(err, tc.err), "expected error %v, got %v", tc.err, err)
			} else {
				require.NoError(t, err)
			}

			agentCall.Unset()
		})
	}
}

func TestRenewCert(t *testing.T) {
	agent := new(mocks.Agent)
	repo := new(mocks.Repository)
	svc, err := certs.NewService(context.Background(), agent, repo)
	require.NoError(t, err)

	newCert := certs.Certificate{
		SerialNumber: serialNumber,
		EntityID:     entityID,
		Certificate:  []byte(testCertPEM),
		ExpiryTime:   time.Now().Add(30 * 24 * time.Hour),
	}

	testCases := []struct {
		desc        string
		serial      string
		viewErr     error
		renewErr    error
		newCert     certs.Certificate
		revoked     bool
		expectedErr error
	}{
		{
			desc:        "renew cert successfully",
			serial:      serialNumber,
			newCert:     newCert,
			expectedErr: nil,
		},
		{
			desc:        "failed agent renew",
			serial:      serialNumber,
			renewErr:    certs.ErrUpdateEntity,
			newCert:     certs.Certificate{},
			expectedErr: certs.ErrUpdateEntity,
		},
		{
			desc:        "failed agent view",
			serial:      serialNumber,
			viewErr:     certs.ErrViewEntity,
			newCert:     certs.Certificate{},
			expectedErr: certs.ErrViewEntity,
		},
		{
			desc:        "revoked certificate cannot be renewed",
			serial:      serialNumber,
			newCert:     certs.Certificate{},
			revoked:     true,
			expectedErr: certs.ErrCertRevoked,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			agentCall := agent.On("Renew", mock.Anything, certValidityPeriod.String()).Return(tc.newCert, tc.renewErr)
			agentCall1 := agent.On("View", tc.serial).Return(certs.Certificate{Certificate: []byte(testCertPEM), Revoked: tc.revoked}, tc.viewErr)

			renewedCert, err := svc.RenewCert(context.Background(), testSession, tc.serial)
			require.True(t, errors.Contains(err, tc.expectedErr), "expected error %v, got %v", tc.expectedErr, err)
			if tc.expectedErr == nil {
				require.Equal(t, tc.newCert, renewedCert)
			}
			agentCall1.Unset()
			agentCall.Unset()
		})
	}
}

func TestGetEntityID(t *testing.T) {
	agent := new(mocks.Agent)
	repo := new(mocks.Repository)
	svc, err := certs.NewService(context.Background(), agent, repo)
	require.NoError(t, err)

	testCases := []struct {
		desc     string
		serial   string
		entityID string
		repoErr  error
		err      error
	}{
		{
			desc:     "get entity ID successfully",
			serial:   serialNumber,
			entityID: "entity-123",
			err:      nil,
		},
		{
			desc:    "error retrieving from repository",
			serial:  serialNumber,
			repoErr: errors.New("not found"),
			err:     certs.ErrViewEntity,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("GetEntityIDBySerial", mock.Anything, tc.serial).Return(tc.entityID, tc.repoErr)

			entityID, err := svc.GetEntityID(context.Background(), tc.serial)
			if tc.err != nil {
				require.True(t, errors.Contains(err, tc.err), "expected error %v, got %v", tc.err, err)
				require.Empty(t, entityID)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.entityID, entityID)
			}

			repoCall.Unset()
		})
	}
}

func TestListCerts(t *testing.T) {
	agent := new(mocks.Agent)
	repo := new(mocks.Repository)
	svc, err := certs.NewService(context.Background(), agent, repo)
	require.NoError(t, err)

	pageMetadata := certs.PageMetadata{Limit: 10, Offset: 0}
	pageMetadataWithEntity := certs.PageMetadata{Limit: 10, Offset: 0, EntityID: "entity-123"}

	expectedCertPage := certs.CertificatePage{
		Certificates: []certs.Certificate{
			{SerialNumber: "123"},
			{SerialNumber: "456"},
		},
		PageMetadata: pageMetadata,
	}

	testCases := []struct {
		desc           string
		pm             certs.PageMetadata
		certPage       certs.CertificatePage
		serialNumbers  []string
		agentErr       error
		repoErr        error
		expectedResult certs.CertificatePage
		err            error
	}{
		{
			desc:           "list certs successfully without entity filter",
			pm:             pageMetadata,
			certPage:       expectedCertPage,
			expectedResult: expectedCertPage,
			err:            nil,
		},
		{
			desc:          "list certs successfully with entity filter",
			pm:            pageMetadataWithEntity,
			serialNumbers: []string{"123", "456"},
			expectedResult: certs.CertificatePage{
				Certificates: []certs.Certificate{
					{SerialNumber: "123", EntityID: "entity-123"},
					{SerialNumber: "456", EntityID: "entity-123"},
				},
				PageMetadata: certs.PageMetadata{
					Limit:    10,
					Offset:   0,
					EntityID: "entity-123",
					Total:    2, // Set the total count
				},
			},
			err: nil,
		},
		{
			desc:     "error listing certs from agent",
			pm:       pageMetadata,
			certPage: certs.CertificatePage{},
			agentErr: errors.New("agent error"),
			err:      certs.ErrViewEntity,
		},
		{
			desc:    "error listing certs by entity from repo",
			pm:      pageMetadataWithEntity,
			repoErr: errors.New("repo error"),
			err:     certs.ErrViewEntity,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			var agentCall, repoCall *mock.Call
			var agentViewCalls []*mock.Call

			if tc.pm.EntityID != "" {
				repoCall = repo.On("ListCertsByEntityID", mock.Anything, tc.pm.EntityID).Return(tc.serialNumbers, tc.repoErr)
				if tc.repoErr == nil && len(tc.serialNumbers) > 0 {
					for _, serial := range tc.serialNumbers {
						viewCall := agent.On("View", serial).Return(certs.Certificate{SerialNumber: serial}, nil)
						agentViewCalls = append(agentViewCalls, viewCall)
					}
				}
			} else {
				agentCall = agent.On("ListCerts", tc.pm).Return(tc.certPage, tc.agentErr)
				if tc.agentErr == nil {
					for _, cert := range tc.certPage.Certificates {
						repo.On("GetEntityIDBySerial", mock.Anything, cert.SerialNumber).Return("", errors.New("not found"))
					}
				}
			}

			certPage, err := svc.ListCerts(context.Background(), testSession, tc.pm)
			if tc.err != nil {
				require.True(t, errors.Contains(err, tc.err), "expected error %v, got %v", tc.err, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedResult.Total, certPage.Total)
				require.Len(t, certPage.Certificates, len(tc.expectedResult.Certificates))
			}

			if agentCall != nil {
				agentCall.Unset()
			}
			if repoCall != nil {
				repoCall.Unset()
			}
			for _, viewCall := range agentViewCalls {
				viewCall.Unset()
			}
		})
	}
}

func TestRevokeAll(t *testing.T) {
	agent := new(mocks.Agent)
	repo := new(mocks.Repository)
	svc, err := certs.NewService(context.Background(), agent, repo)
	require.NoError(t, err)

	testCases := []struct {
		desc          string
		entityID      string
		serialNumbers []string
		repoErr       error
		agentErr      error
		removeErr     error
		err           error
	}{
		{
			desc:          "revoke all certs successfully",
			entityID:      "entity-123",
			serialNumbers: []string{"123", "456"},
			err:           nil,
		},
		{
			desc:     "error listing certs by entity",
			entityID: "entity-123",
			repoErr:  errors.New("repo error"),
			err:      certs.ErrViewEntity,
		},
		{
			desc:          "error revoking cert",
			entityID:      "entity-123",
			serialNumbers: []string{"123"},
			agentErr:      errors.New("agent error"),
			err:           certs.ErrUpdateEntity,
		},
		{
			desc:     "no certificates found for entity",
			entityID: "entity-123",
			err:      certs.ErrNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("ListCertsByEntityID", mock.Anything, tc.entityID).Return(tc.serialNumbers, tc.repoErr)

			var agentCalls, removeCalls []*mock.Call
			if tc.repoErr == nil && len(tc.serialNumbers) > 0 {
				for _, serial := range tc.serialNumbers {
					agentCall := agent.On("Revoke", serial).Return(tc.agentErr)
					agentCalls = append(agentCalls, agentCall)

					if tc.agentErr == nil {
						removeCall := repo.On("RemoveCertEntityMapping", mock.Anything, serial).Return(tc.removeErr)
						removeCalls = append(removeCalls, removeCall)
					}
				}
			}

			err := svc.RevokeAll(context.Background(), testSession, tc.entityID)
			if tc.err != nil {
				require.True(t, errors.Contains(err, tc.err), "expected error %v, got %v", tc.err, err)
			} else {
				require.NoError(t, err)
			}

			// Clean up mocks
			repoCall.Unset()
			for _, call := range agentCalls {
				call.Unset()
			}
			for _, call := range removeCalls {
				call.Unset()
			}
		})
	}
}

func TestIssueFromCSR(t *testing.T) {
	agent := new(mocks.Agent)
	repo := new(mocks.Repository)
	svc, err := certs.NewService(context.Background(), agent, repo)
	require.NoError(t, err)

	testCSR := certs.CSR{
		CSR: []byte("test-csr-data"),
	}

	testCases := []struct {
		desc         string
		entityID     string
		ttl          string
		csr          certs.CSR
		cert         certs.Certificate
		expectedCert certs.Certificate
		agentErr     error
		repoErr      error
		err          error
	}{
		{
			desc:     "issue cert from CSR successfully",
			entityID: "entity-123",
			ttl:      "1h",
			csr:      testCSR,
			cert: certs.Certificate{
				SerialNumber: serialNumber,
			},
			expectedCert: certs.Certificate{
				SerialNumber: serialNumber,
				EntityID:     "entity-123",
			},
			err: nil,
		},
		{
			desc:     "failed agent sign CSR",
			entityID: "entity-123",
			ttl:      "1h",
			csr:      testCSR,
			cert:     certs.Certificate{},
			agentErr: errors.New("agent error"),
			err:      certs.ErrFailedCertCreation,
		},
		{
			desc:     "failed repository save mapping",
			entityID: "entity-123",
			ttl:      "1h",
			csr:      testCSR,
			cert: certs.Certificate{
				SerialNumber: serialNumber,
			},
			repoErr: errors.New("repo error"),
			err:     certs.ErrFailedCertCreation,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			agentCall := agent.On("SignCSR", tc.csr.CSR, tc.ttl).Return(tc.cert, tc.agentErr)
			var repoCall *mock.Call
			if tc.agentErr == nil {
				repoCall = repo.On("SaveCertEntityMapping", mock.Anything, tc.cert.SerialNumber, tc.entityID).Return(tc.repoErr)
			}

			cert, err := svc.IssueFromCSR(context.Background(), testSession, tc.entityID, tc.ttl, tc.csr)
			if tc.err != nil {
				require.True(t, errors.Contains(err, tc.err), "expected error %v, got %v", tc.err, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedCert, cert)
			}

			agentCall.Unset()
			if repoCall != nil {
				repoCall.Unset()
			}
		})
	}
}

func TestGenerateCRL(t *testing.T) {
	agent := new(mocks.Agent)
	repo := new(mocks.Repository)
	svc, err := certs.NewService(context.Background(), agent, repo)
	require.NoError(t, err)

	testCases := []struct {
		desc     string
		crlBytes []byte
		agentErr error
		err      error
	}{
		{
			desc:     "generate CRL successfully",
			crlBytes: []byte("test-crl-data"),
			err:      nil,
		},
		{
			desc:     "failed with agent error",
			crlBytes: nil,
			agentErr: errors.New("agent error"),
			err:      certs.ErrFailedCertCreation,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			agentCall := agent.On("GetCRL").Return(tc.crlBytes, tc.agentErr)

			crlBytes, err := svc.GenerateCRL(context.Background())
			if tc.err != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.crlBytes, crlBytes)
			}

			agentCall.Unset()
		})
	}
}
