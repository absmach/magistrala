package cli

import (
	"errors"
	"strconv"

	"github.com/spf13/cobra"
)

var cmdCerts = []cobra.Command{
	cobra.Command{
		Use:   "issue",
		Short: "issue <thing_id> <keybits> <keytype> <hoursvalid> <user_auth_token>",
		Long:  `Issues new certificate for a thing`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 5 {
				logUsage(cmd.Short)
				return
			}
			thingID := args[0]
			keyBits, err := strconv.Atoi(args[1])
			if err != nil {
				logError(errors.New("invalid format for keybits"))
				return
			}

			keyType := args[2]
			valid := args[3]
			token := args[4]

			c, err := sdk.IssueCert(thingID, keyBits, keyType, valid, token)
			if err != nil {
				logError(err)
				return
			}
			logJSON(c)
		},
	},
}

// NewCertsCmd returns certificate command.
func NewCertsCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "cert",
		Short: "Certificate management",
		Long:  `Certificate management: create certificates for things"`,
		Run: func(cmd *cobra.Command, args []string) {
			logUsage("cert issue <thing_id> <keybits> <keytype> <hoursvalid> <user_auth_token>")
		},
	}

	for i := range cmdCerts {
		cmd.AddCommand(&cmdCerts[i])
	}

	return &cmd
}
