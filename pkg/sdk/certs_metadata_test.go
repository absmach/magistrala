// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPageMetadataQueryWithCertFilters(t *testing.T) {
	pm := PageMetadata{
		EntityID:           "entity-id",
		CommonName:         "device-cn",
		Organization:       []string{"Acme", "QA"},
		OrganizationalUnit: []string{"Platform"},
		Country:            []string{"RS"},
		Province:           []string{"Belgrade"},
		Locality:           []string{"Belgrade"},
		StreetAddress:      []string{"Nemanjina 4"},
		PostalCode:         []string{"11000"},
		DNSNames:           []string{"device.local"},
		IPAddresses:        []string{"127.0.0.1"},
		EmailAddresses:     []string{"device@example.com"},
		TTL:                "24h",
	}

	encoded, err := pm.query()
	require.NoError(t, err)

	values, err := url.ParseQuery(encoded)
	require.NoError(t, err)

	assert.Equal(t, "entity-id", values.Get("entity_id"))
	assert.Equal(t, "device-cn", values.Get("common_name"))
	assert.Equal(t, "24h", values.Get("ttl"))
	assert.ElementsMatch(t, []string{"Acme", "QA"}, values["organization"])
	assert.Equal(t, []string{"Platform"}, values["organizational_unit"])
	assert.Equal(t, []string{"RS"}, values["country"])
	assert.Equal(t, []string{"Belgrade"}, values["province"])
	assert.Equal(t, []string{"Belgrade"}, values["locality"])
	assert.Equal(t, []string{"Nemanjina 4"}, values["street_address"])
	assert.Equal(t, []string{"11000"}, values["postal_code"])
	assert.Equal(t, []string{"device.local"}, values["dns_names"])
	assert.Equal(t, []string{"127.0.0.1"}, values["ip_addresses"])
	assert.Equal(t, []string{"device@example.com"}, values["email_addresses"])
}

func TestCertStatusAliases(t *testing.T) {
	assert.Equal(t, CertValid, Valid)
	assert.Equal(t, CertRevoked, Revoked)
	assert.Equal(t, CertUnknown, Unknown)
}

func TestCertTypeString(t *testing.T) {
	tests := []struct {
		desc     string
		typ      CertType
		expected string
	}{
		{desc: "root", typ: RootCA, expected: "root"},
		{desc: "intermediate", typ: IntermediateCA, expected: "intermediate"},
		{desc: "unknown", typ: CertType(99), expected: "unknown"},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.typ.String())
		})
	}
}
