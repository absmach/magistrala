// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import (
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
	smqsdk "github.com/absmach/magistrala/pkg/sdk"
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
		Use:   "devices <devices_file> <domain_id> <user_token>",
		Short: "Provision devices",
		Long:  `Bulk create devices`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			if _, err := os.Stat(args[0]); os.IsNotExist(err) {
				logErrorCmd(*cmd, err)
				return
			}

			devices, err := devicesFromFile(args[0])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			devices, err = sdk.CreateClients(cmd.Context(), devices, args[1], args[2])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logJSONCmd(*cmd, devices)
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
				c, err = sdk.CreateChannel(cmd.Context(), c, args[1], args[2])
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
		Long:  `Bulk connect devices to channels`,
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
				if err := sdk.Connect(cmd.Context(), conn, args[1], args[2]); err != nil {
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
		Long: `Provisions test setup: one test user, two devices and two channels. \
						Connect both devices to one of the channels, \
						and only one device to the other channel.`,
		Run: func(cmd *cobra.Command, args []string) {
			numDevices := 2
			numChan := 2
			devices := []smqsdk.Client{}
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
			user, err := sdk.CreateUser(cmd.Context(), user, "")
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			ut, err := sdk.CreateToken(cmd.Context(), smqsdk.Login{Username: user.Credentials.Username, Password: user.Credentials.Secret})
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			// create domain
			domain := smqsdk.Domain{
				Name:   fmt.Sprintf("%s-domain", name),
				Status: smqsdk.EnabledStatus,
			}
			domain, err = sdk.CreateDomain(cmd.Context(), domain, ut.AccessToken)
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			ut, err = sdk.CreateToken(cmd.Context(), smqsdk.Login{Username: user.Email, Password: user.Credentials.Secret})
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			// Create devices
			for i := 0; i < numDevices; i++ {
				t := smqsdk.Client{
					Name:   fmt.Sprintf("%s-device-%d", name, i),
					Status: smqsdk.EnabledStatus,
				}

				devices = append(devices, t)
			}
			devices, err = sdk.CreateClients(cmd.Context(), devices, domain.ID, ut.AccessToken)
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
				c, err = sdk.CreateChannel(cmd.Context(), c, domain.ID, ut.AccessToken)
				if err != nil {
					logErrorCmd(*cmd, err)
					return
				}

				channels = append(channels, c)
			}

			// Connect devices to channels: first device to both channels, second only to first.
			conIDs := smqsdk.Connection{
				ChannelIDs: []string{channels[0].ID},
				ClientIDs:  []string{devices[0].ID},
				Types:      []string{PublishType, SubscribeType},
			}
			if err := sdk.Connect(cmd.Context(), conIDs, domain.ID, ut.AccessToken); err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			conIDs = smqsdk.Connection{
				ChannelIDs: []string{channels[1].ID},
				ClientIDs:  []string{devices[0].ID},
				Types:      []string{PublishType, SubscribeType},
			}
			if err := sdk.Connect(cmd.Context(), conIDs, domain.ID, ut.AccessToken); err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			conIDs = smqsdk.Connection{
				ChannelIDs: []string{channels[0].ID},
				ClientIDs:  []string{devices[1].ID},
				Types:      []string{PublishType, SubscribeType},
			}
			if err := sdk.Connect(cmd.Context(), conIDs, domain.ID, ut.AccessToken); err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			// send message to test connectivity
			if err := sdk.SendMessage(cmd.Context(), domain.ID, channels[0].ID, devices[0].Credentials.Secret, fmt.Sprintf(msgFormat, time.Now().Unix(), rand.Int())); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			if err := sdk.SendMessage(cmd.Context(), domain.ID, channels[0].ID, devices[1].Credentials.Secret, fmt.Sprintf(msgFormat, time.Now().Unix(), rand.Int())); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			if err := sdk.SendMessage(cmd.Context(), domain.ID, channels[1].ID, devices[0].Credentials.Secret, fmt.Sprintf(msgFormat, time.Now().Unix(), rand.Int())); err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logJSONCmd(*cmd, user, ut, devices, channels)
		},
	},
}

// NewProvisionCmd returns provision command.
func NewProvisionCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "provision [devices | channels | connect | test]",
		Short: "Provision devices and channels from a config file",
		Long:  `Provision devices and channels: use json or csv file to bulk provision devices and channels`,
	}

	for i := range cmdProvision {
		cmd.AddCommand(&cmdProvision[i])
	}

	return &cmd
}

func devicesFromFile(path string) ([]smqsdk.Client, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []smqsdk.Client{}, err
	}

	file, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return []smqsdk.Client{}, err
	}
	defer file.Close()

	devices := []smqsdk.Client{}
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

			device := smqsdk.Client{
				Name: l[0],
			}

			devices = append(devices, device)
		}
	case jsonExt:
		err := json.NewDecoder(file).Decode(&devices)
		if err != nil {
			return []smqsdk.Client{}, err
		}
	default:
		return []smqsdk.Client{}, err
	}

	return devices, nil
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
