package main

import (
	"log"

	bf "github.com/mainflux/mainflux/bashflux"
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
		Use: "bashflux",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Set HTTP server address
			bf.SetServerAddr(conf.Host, conf.Port)
		},
	}

	// API commands
	versionCmd := bf.NewVersionCmd()
	usersCmd := bf.NewUsersCmd()
	thingsCmd := bf.NewThingsCmd()
	channelsCmd := bf.NewChannelsCmd()
	messagesCmd := bf.NewMessagesCmd()

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
		&bf.Limit, "limit", "l", 100, "limit query parameter")
	rootCmd.PersistentFlags().IntVarP(
		&bf.Offset, "offset", "o", 0, "offset query parameter")

	// Set TLS certificates
	bf.SetCerts()

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
