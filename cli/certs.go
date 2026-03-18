// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"
	"encoding/json"
	"os"

	"github.com/absmach/supermq/certs"
	smqsdk "github.com/absmach/supermq/pkg/sdk"
	"github.com/spf13/cobra"
)

var cmdCerts = []cobra.Command{
	{
		Use:   "get [all | <entity_id>] <domain_id> <token>",
		Short: "Get certificate",
		Long:  `Gets a certificate for a given entity ID or all certificates.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			if args[0] == all {
				pm := smqsdk.PageMetadata{
					Limit:  Limit,
					Offset: Offset,
				}
				page, err := sdk.ListCerts(context.Background(), pm, args[1], args[2])
				if err != nil {
					logErrorCmd(*cmd, err)
					return
				}
				logJSONCmd(*cmd, page)
				return
			}
			pm := smqsdk.PageMetadata{
				EntityID: args[0],
				Limit:    Limit,
				Offset:   Offset,
			}
			page, err := sdk.ListCerts(context.Background(), pm, args[1], args[2])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logJSONCmd(*cmd, page)
		},
	},
	{
		Use:   "revoke <serial_number> <domain_id> <token>",
		Short: "Revoke certificate",
		Long:  `Revokes a certificate for a given serial number.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			err := sdk.RevokeCert(context.Background(), args[0], args[1], args[2])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logOKCmd(*cmd)
		},
	},
	{
		Use:   "delete <entity_id> <domain_id> <token>",
		Short: "Delete certificate",
		Long:  `Deletes certificates for a given entity id.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			err := sdk.DeleteCert(context.Background(), args[0], args[1], args[2])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logOKCmd(*cmd)
		},
	},
	{
		Use:   "renew <serial_number> <domain_id> <token>",
		Short: "Renew certificate",
		Long:  `Renews a certificate for a given serial number.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			_, err := sdk.RenewCert(context.Background(), args[0], args[1], args[2])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logOKCmd(*cmd)
		},
	},
	{
		Use:   "ocsp <serial_number_or_certificate_path>",
		Short: "OCSP",
		Long:  `OCSP for a given serial number or certificate.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			var serialNumber, certContent string
			if _, statErr := os.Stat(args[0]); statErr == nil {
				certBytes, err := os.ReadFile(args[0])
				if err != nil {
					logErrorCmd(*cmd, err)
					return
				}
				certContent = string(certBytes)
			} else {
				serialNumber = args[0]
			}
			response, err := sdk.OCSP(context.Background(), serialNumber, certContent)
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logJSONCmd(*cmd, response)
		},
	},
	{
		Use:   "view <serial_number> <domain_id> <token>",
		Short: "View certificate",
		Long:  `Views a certificate for a given serial number.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			cert, err := sdk.ViewCert(context.Background(), args[0], args[1], args[2])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logJSONCmd(*cmd, cert)
		},
	},
	{
		Use:   "view-ca",
		Short: "View-ca certificate",
		Long:  `Views ca certificate.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 0 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			cert, err := sdk.ViewCA(context.Background())
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logJSONCmd(*cmd, cert)
		},
	},
	{
		Use:   "download-ca",
		Short: "Download signing CA",
		Long:  `Download intermediate cert and ca.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 0 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			bundle, err := sdk.DownloadCA(context.Background())
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logSaveCAFiles(*cmd, bundle)
		},
	},
	{
		Use:   "csr <metadata> <private_key_path>",
		Short: "Create CSR",
		Long:  `Creates a CSR.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			var pm certs.CSRMetadata
			if err := json.Unmarshal([]byte(args[0]), &pm); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			data, err := os.ReadFile(args[1])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			csr, err := sdk.CreateCSR(context.Background(), pm, data)
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logSaveCSRFiles(*cmd, csr)
		},
	},
	{
		Use:   "issue-csr <entity_id> <ttl> <path_to_csr> <domain_id> <token>",
		Short: "Issue from CSR",
		Long:  `issues a certificate for a given csr.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 5 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			csrData, err := os.ReadFile(args[2])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			cert, err := sdk.IssueFromCSR(context.Background(), args[0], args[1], string(csrData), args[3], args[4])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logJSONCmd(*cmd, cert)
			logSaveCertFiles(*cmd, cert)
		},
	},
	{
		Use:   "issue-csr-internal <entity_id> <ttl> <path_to_csr> <agent_token>",
		Short: "Issue from CSR Internal (Agent)",
		Long:  `Issues a certificate for a given CSR using agent authentication.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 4 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			csrData, err := os.ReadFile(args[2])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			cert, err := sdk.IssueFromCSRInternal(context.Background(), args[0], args[1], string(csrData), args[3])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logJSONCmd(*cmd, cert)
			logSaveCertFiles(*cmd, cert)
		},
	},
	{
		Use:   "crl",
		Short: "Generate CRL",
		Long:  `Generates a Certificate Revocation List (CRL).`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 0 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			crlBytes, err := sdk.GenerateCRL(context.Background())
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logSaveCRLFile(*cmd, crlBytes)
		},
	},
	{
		Use:   "entity-id <serial_number> <domain_id> <token>",
		Short: "Get entity ID by serial number",
		Long:  `Gets the entity ID for a certificate by its serial number.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			entityID, err := sdk.EntityID(context.Background(), args[0], args[1], args[2])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logJSONCmd(*cmd, map[string]string{"entity_id": entityID})
		},
	},
}

// NewCertsCmd returns certificate command.
func NewCertsCmd() *cobra.Command {
	var ttl string
	issueCmd := cobra.Command{
		Use:   "issue <entity_id> <common_name> <ip_addrs_json> [<options_json>] <domain_id> <token> [--ttl=8760h]",
		Short: "Issue certificate",
		Long:  `Issues a certificate for a given entity ID.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 5 || len(args) > 6 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			var ipAddrs []string
			if err := json.Unmarshal([]byte(args[2]), &ipAddrs); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			var option smqsdk.Options
			option.CommonName = args[1]
			var domainID, token string
			if len(args) == 5 {
				domainID = args[3]
				token = args[4]
			} else {
				if err := json.Unmarshal([]byte(args[3]), &option); err != nil {
					logErrorCmd(*cmd, err)
					return
				}
				domainID = args[4]
				token = args[5]
			}
			cert, err := sdk.IssueCert(context.Background(), args[0], ttl, ipAddrs, option, domainID, token)
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logJSONCmd(*cmd, cert)
			logSaveCertFiles(*cmd, cert)
		},
	}
	issueCmd.Flags().StringVar(&ttl, "ttl", "8760h", "certificate time to live in duration")

	cmd := cobra.Command{
		Use:   "certs [issue | get | revoke | renew | ocsp | view | download-ca | view-ca | csr | issue-csr | issue-csr-internal | crl | entity-id]",
		Short: "Certificates management",
		Long:  `Certificates management: issue, get all, get by entity ID, revoke, renew, OCSP, view, CRL generation, entity ID lookup, agent CSR issuing, and CA operations.`,
	}
	cmd.AddCommand(&issueCmd)
	for i := range cmdCerts {
		cmd.AddCommand(&cmdCerts[i])
	}
	return &cmd
}
