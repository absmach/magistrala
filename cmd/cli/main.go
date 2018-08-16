package main

import (
	"log"

	"github.com/mainflux/mainflux/cli"
	"github.com/spf13/cobra"
)

func main() {

	conf := struct {
		Host string
		Port int
	}{
		"localhost",
		0,
	}

	// Root
	var rootCmd = &cobra.Command{
		Use: "mainflux-cli",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Set HTTP server address
			cli.SetServerAddr(conf.Host, conf.Port)
		},
	}

	// API commands
	versionCmd := cli.NewVersionCmd()
	usersCmd := cli.NewUsersCmd()
	thingsCmd := cli.NewThingsCmd()
	channelsCmd := cli.NewChannelsCmd()
	messagesCmd := cli.NewMessagesCmd()

	// Root Commands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(usersCmd)
	rootCmd.AddCommand(thingsCmd)
	rootCmd.AddCommand(channelsCmd)
	rootCmd.AddCommand(messagesCmd)

	// Root Flags
	rootCmd.PersistentFlags().StringVarP(
		&conf.Host, "host", "m", conf.Host, "HTTP Host address")
	rootCmd.PersistentFlags().IntVarP(
		&conf.Port, "port", "p", conf.Port, "HTTP Host Port")

	// Client and Channels Flags
	rootCmd.PersistentFlags().IntVarP(
		&cli.Limit, "limit", "l", 100, "limit query parameter")
	rootCmd.PersistentFlags().IntVarP(
		&cli.Offset, "offset", "o", 0, "offset query parameter")

	// Set TLS certificates
	cli.SetCerts()

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
