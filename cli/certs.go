package cli

import (
	"strconv"

	"github.com/spf13/cobra"
)

// NewCertsCmd returns certificate command.
func NewCertsCmd() *cobra.Command {
	var keySize uint16
	var keyType string
	var ttl uint32

	issueCmd := cobra.Command{
		Use:   "issue <thing_id> <user_auth_token> [--keysize=2048] [--keytype=rsa] [--ttl=8760]",
		Short: "Issue certificate",
		Long:  `Issues new certificate for a thing`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			thingID := args[0]
			valid := strconv.FormatUint(uint64(ttl), 10)

			c, err := sdk.IssueCert(thingID, int(keySize), keyType, valid, args[1])
			if err != nil {
				logError(err)
				return
			}
			logJSON(c)
		},
	}

	issueCmd.Flags().Uint16Var(&keySize, "keysize", 2048, "certificate key strength in bits: 2048, 4096 (RSA) or 224, 256, 384, 512 (EC)")
	issueCmd.Flags().StringVar(&keyType, "keytype", "rsa", "certificate key type: RSA or EC")
	issueCmd.Flags().Uint32Var(&ttl, "ttl", 8760, "certificate time to live in hours")

	cmd := cobra.Command{
		Use:   "certs [issue | get]",
		Short: "Certificates management",
		Long:  `Certificates management: create certificates for things"`,
	}

	cmdCerts := []cobra.Command{
		issueCmd,
	}

	for i := range cmdCerts {
		cmd.AddCommand(&cmdCerts[i])
	}

	return &cmd
}
