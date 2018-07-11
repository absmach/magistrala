package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var cmdUsers = []cobra.Command{
	cobra.Command{
		Use:   "create",
		Short: "create <username> <password>",
		Long:  `Creates new user`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				LogUsage(cmd.Short)
				return
			}
			CreateUser(args[0], args[1])
		},
	},
	cobra.Command{
		Use:   "token",
		Short: "token <username> <password>",
		Long:  `Creates new token`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				LogUsage(cmd.Short)
				return
			}
			CreateToken(args[0], args[1])
		},
	},
}

func NewUsersCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "users",
		Short: "users create/token <email> <password>",
		Long:  `Manages users in the system (create account or token)`,
		Run: func(cmd *cobra.Command, args []string) {
			LogUsage(cmd.Short)
		},
	}

	for i, _ := range cmdUsers {
		cmd.AddCommand(&cmdUsers[i])
	}

	return &cmd
}

// CreateUser - create user
func CreateUser(user, pwd string) {
	msg := fmt.Sprintf(`{"email": "%s", "password": "%s"}`, user, pwd)
	url := fmt.Sprintf("%s/users", serverAddr)
	resp, err := httpClient.Post(url, contentType, strings.NewReader(msg))
	FormatResLog(resp, err)
}

// CreateToken - create user token
func CreateToken(user, pwd string) {
	msg := fmt.Sprintf(`{"email": "%s", "password": "%s"}`, user, pwd)
	url := fmt.Sprintf("%s/tokens", serverAddr)
	resp, err := httpClient.Post(url, contentType, strings.NewReader(msg))
	FormatResLog(resp, err)
}
