// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/0x6flab/namegenerator"
	smqsdk "github.com/absmach/supermq/pkg/sdk"
	"github.com/spf13/cobra"
)

const (
	jsonExt       = ".json"
	csvExt        = ".csv"
	PublishType   = "publish"
	SubscribeType = "subscribe"
)

var (
	msgFormat      = `[{"bn":"provision:", "bu":"V", "t": %d, "bver":5, "n":"voltage", "u":"V", "v":%d}]`
	namesgenerator = namegenerator.NewGenerator()
)

var cmdProvision = []cobra.Command{
	{
		Use:   "clients <clients_file> <domain_id> <user_token>",
		Short: "Provision clients",
		Long:  `Bulk create clients`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			if _, err := os.Stat(args[0]); os.IsNotExist(err) {
				logErrorCmd(*cmd, err)
				return
			}

			clients, err := clientsFromFile(args[0])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			clients, err = sdk.CreateClients(context.Background(), clients, args[1], args[2])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logJSONCmd(*cmd, clients)
		},
	},
	{
		Use:   "channels <channels_file> <domain_id> <user_token>",
		Short: "Provision channels",
		Long:  `Bulk create channels`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			channels, err := channelsFromFile(args[0])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			var chs []smqsdk.Channel
			for _, c := range channels {
				c, err = sdk.CreateChannel(context.Background(), c, args[1], args[2])
				if err != nil {
					logErrorCmd(*cmd, err)
					return
				}
				chs = append(chs, c)
			}
			channels = chs

			logJSONCmd(*cmd, channels)
		},
	},
	{
		Use:   "connect <connections_file> <domain_id> <user_token>",
		Short: "Provision connections",
		Long:  `Bulk connect clients to channels`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			connIDs, err := connectionsFromFile(args[0])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			for _, conn := range connIDs {
				if err := sdk.Connect(context.Background(), conn, args[1], args[2]); err != nil {
					logErrorCmd(*cmd, err)
					return
				}
			}

			logOKCmd(*cmd)
		},
	},
	{
		Use:   "test",
		Short: "test",
		Long: `Provisions test setup: one test user, two clients and two channels. \
						Connect both clients to one of the channels, \
						and only on client to other channel.`,
		Run: func(cmd *cobra.Command, args []string) {
			numClients := 2
			numChan := 2
			clients := []smqsdk.Client{}
			channels := []smqsdk.Channel{}

			if len(args) != 0 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			// Create test user
			name := namesgenerator.Generate()
			user := smqsdk.User{
				FirstName: name,
				Email:     fmt.Sprintf("%s@email.com", name),
				Credentials: smqsdk.Credentials{
					Username: name,
					Secret:   "12345678",
				},
				Status: smqsdk.EnabledStatus,
			}
			user, err := sdk.CreateUser(context.Background(), user, "")
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			ut, err := sdk.CreateToken(context.Background(), smqsdk.Login{Username: user.Credentials.Username, Password: user.Credentials.Secret})
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			// create domain
			domain := smqsdk.Domain{
				Name:   fmt.Sprintf("%s-domain", name),
				Status: smqsdk.EnabledStatus,
			}
			domain, err = sdk.CreateDomain(context.Background(), domain, ut.AccessToken)
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			ut, err = sdk.CreateToken(context.Background(), smqsdk.Login{Username: user.Email, Password: user.Credentials.Secret})
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			// Create clients
			for i := 0; i < numClients; i++ {
				t := smqsdk.Client{
					Name:   fmt.Sprintf("%s-client-%d", name, i),
					Status: smqsdk.EnabledStatus,
				}

				clients = append(clients, t)
			}
			clients, err = sdk.CreateClients(context.Background(), clients, domain.ID, ut.AccessToken)
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			// Create channels
			for i := 0; i < numChan; i++ {
				c := smqsdk.Channel{
					Name:   fmt.Sprintf("%s-channel-%d", name, i),
					Status: smqsdk.EnabledStatus,
				}
				c, err = sdk.CreateChannel(context.Background(), c, domain.ID, ut.AccessToken)
				if err != nil {
					logErrorCmd(*cmd, err)
					return
				}

				channels = append(channels, c)
			}

			// Connect clients to channels - first client to both channels, second only to first
			conIDs := smqsdk.Connection{
				ChannelIDs: []string{channels[0].ID},
				ClientIDs:  []string{clients[0].ID},
				Types:      []string{PublishType, SubscribeType},
			}
			if err := sdk.Connect(context.Background(), conIDs, domain.ID, ut.AccessToken); err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			conIDs = smqsdk.Connection{
				ChannelIDs: []string{channels[1].ID},
				ClientIDs:  []string{clients[0].ID},
				Types:      []string{PublishType, SubscribeType},
			}
			if err := sdk.Connect(context.Background(), conIDs, domain.ID, ut.AccessToken); err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			conIDs = smqsdk.Connection{
				ChannelIDs: []string{channels[0].ID},
				ClientIDs:  []string{clients[1].ID},
				Types:      []string{PublishType, SubscribeType},
			}
			if err := sdk.Connect(context.Background(), conIDs, domain.ID, ut.AccessToken); err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			// send message to test connectivity
			if err := sdk.SendMessage(context.Background(), channels[0].ID, fmt.Sprintf(msgFormat, time.Now().Unix(), rand.Int()), clients[0].Credentials.Secret); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			if err := sdk.SendMessage(context.Background(), channels[0].ID, fmt.Sprintf(msgFormat, time.Now().Unix(), rand.Int()), clients[1].Credentials.Secret); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			if err := sdk.SendMessage(context.Background(), channels[1].ID, fmt.Sprintf(msgFormat, time.Now().Unix(), rand.Int()), clients[0].Credentials.Secret); err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logJSONCmd(*cmd, user, ut, clients, channels)
		},
	},
}

// NewProvisionCmd returns provision command.
func NewProvisionCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "provision [clients | channels | connect | test]",
		Short: "Provision clients and channels from a config file",
		Long:  `Provision clients and channels: use json or csv file to bulk provision clients and channels`,
	}

	for i := range cmdProvision {
		cmd.AddCommand(&cmdProvision[i])
	}

	return &cmd
}

func clientsFromFile(path string) ([]smqsdk.Client, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []smqsdk.Client{}, err
	}

	file, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return []smqsdk.Client{}, err
	}
	defer file.Close()

	clients := []smqsdk.Client{}
	switch filepath.Ext(path) {
	case csvExt:
		reader := csv.NewReader(file)

		for {
			l, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				return []smqsdk.Client{}, err
			}

			if len(l) < 1 {
				return []smqsdk.Client{}, errors.New("empty line found in file")
			}

			client := smqsdk.Client{
				Name: l[0],
			}

			clients = append(clients, client)
		}
	case jsonExt:
		err := json.NewDecoder(file).Decode(&clients)
		if err != nil {
			return []smqsdk.Client{}, err
		}
	default:
		return []smqsdk.Client{}, err
	}

	return clients, nil
}

func channelsFromFile(path string) ([]smqsdk.Channel, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []smqsdk.Channel{}, err
	}

	file, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return []smqsdk.Channel{}, err
	}
	defer file.Close()

	channels := []smqsdk.Channel{}
	switch filepath.Ext(path) {
	case csvExt:
		reader := csv.NewReader(file)

		for {
			l, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				return []smqsdk.Channel{}, err
			}

			if len(l) < 1 {
				return []smqsdk.Channel{}, errors.New("empty line found in file")
			}

			channel := smqsdk.Channel{
				Name: l[0],
			}

			channels = append(channels, channel)
		}
	case jsonExt:
		err := json.NewDecoder(file).Decode(&channels)
		if err != nil {
			return []smqsdk.Channel{}, err
		}
	default:
		return []smqsdk.Channel{}, err
	}

	return channels, nil
}

func connectionsFromFile(path string) ([]smqsdk.Connection, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []smqsdk.Connection{}, err
	}

	file, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return []smqsdk.Connection{}, err
	}
	defer file.Close()

	connections := []smqsdk.Connection{}
	switch filepath.Ext(path) {
	case csvExt:
		reader := csv.NewReader(file)

		for {
			l, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				return []smqsdk.Connection{}, err
			}

			if len(l) < 1 {
				return []smqsdk.Connection{}, errors.New("empty line found in file")
			}
			connections = append(connections, smqsdk.Connection{
				ClientIDs:  []string{l[0]},
				ChannelIDs: []string{l[1]},
				Types:      []string{PublishType, SubscribeType},
			})
		}
	case jsonExt:
		err := json.NewDecoder(file).Decode(&connections)
		if err != nil {
			return []smqsdk.Connection{}, err
		}
	default:
		return []smqsdk.Connection{}, err
	}

	return connections, nil
}
