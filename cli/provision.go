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

	mgxsdk "github.com/absmach/magistrala/pkg/sdk/go"
	"github.com/docker/docker/pkg/namesgenerator"
	"github.com/spf13/cobra"
)

const (
	jsonExt = ".json"
	csvExt  = ".csv"
)

var cmdProvision = []cobra.Command{
	{
		Use:   "things <things_file> <user_token>",
		Short: "Provision things",
		Long:  `Bulk create things`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			if _, err := os.Stat(args[0]); os.IsNotExist(err) {
				logError(err)
				return
			}

			things, err := thingsFromFile(args[0])
			if err != nil {
				logError(err)
				return
			}

			things, err = sdk.CreateThings(things, args[1])
			if err != nil {
				logError(err)
				return
			}

			logJSON(things)
		},
	},
	{
		Use:   "channels <channels_file> <user_token>",
		Short: "Provision channels",
		Long:  `Bulk create channels`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			channels, err := channelsFromFile(args[0])
			if err != nil {
				logError(err)
				return
			}

			channels, err = sdk.CreateChannels(channels, args[1])
			if err != nil {
				logError(err)
				return
			}

			logJSON(channels)
		},
	},
	{
		Use:   "connect <connections_file> <user_token>",
		Short: "Provision connections",
		Long:  `Bulk connect things to channels`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			connIDs, err := connectionsFromFile(args[0])
			if err != nil {
				logError(err)
				return
			}
			for _, conn := range connIDs {
				if err := sdk.Connect(conn, args[1]); err != nil {
					logError(err)
					return
				}
			}

			logOK()
		},
	},
	{
		Use:   "test",
		Short: "test",
		Long: `Provisions test setup: one test user, two things and two channels. \
						Connect both things to one of the channels, \
						and only on thing to other channel.`,
		Run: func(cmd *cobra.Command, args []string) {
			numThings := 2
			numChan := 2
			things := []mgxsdk.Thing{}
			channels := []mgxsdk.Channel{}

			if len(args) != 0 {
				logUsage(cmd.Use)
				return
			}

			rand.Seed(time.Now().UnixNano())
			name := namesgenerator.GetRandomName(0)
			// Create test user
			user := mgxsdk.User{
				Name: name,
				Credentials: mgxsdk.Credentials{
					Identity: fmt.Sprintf("%s@email.com", name),
					Secret:   "12345678",
				},
				Status: mgxsdk.EnabledStatus,
			}
			user, err := sdk.CreateUser(user, "")
			if err != nil {
				logError(err)
				return
			}

			user.Credentials.Secret = "12345678"
			ut, err := sdk.CreateToken(mgxsdk.Login{Identity: user.Credentials.Identity, Secret: user.Credentials.Secret})
			if err != nil {
				logError(err)
				return
			}

			// Create things
			for i := 0; i < numThings; i++ {
				n := fmt.Sprintf("d%d", i)
				t := mgxsdk.Thing{
					Name:   n,
					Status: mgxsdk.EnabledStatus,
				}

				things = append(things, t)
			}
			things, err = sdk.CreateThings(things, ut.AccessToken)
			if err != nil {
				logError(err)
				return
			}

			// Create channels
			for i := 0; i < numChan; i++ {
				n := fmt.Sprintf("c%d", i)

				c := mgxsdk.Channel{
					Name:   n,
					Status: mgxsdk.EnabledStatus,
				}

				channels = append(channels, c)
			}
			channels, err = sdk.CreateChannels(channels, ut.AccessToken)
			if err != nil {
				logError(err)
				return
			}

			// Connect things to channels - first thing to both channels, second only to first
			conIDs := mgxsdk.Connection{
				ChannelID: channels[0].ID,
				ThingID:   things[0].ID,
			}
			if err := sdk.Connect(conIDs, ut.AccessToken); err != nil {
				logError(err)
				return
			}

			conIDs = mgxsdk.Connection{
				ChannelID: channels[1].ID,
				ThingID:   things[0].ID,
			}
			if err := sdk.Connect(conIDs, ut.AccessToken); err != nil {
				logError(err)
				return
			}

			conIDs = mgxsdk.Connection{
				ChannelID: channels[0].ID,
				ThingID:   things[1].ID,
			}
			if err := sdk.Connect(conIDs, ut.AccessToken); err != nil {
				logError(err)
				return
			}

			logJSON(user, ut, things, channels)
		},
	},
}

// NewProvisionCmd returns provision command.
func NewProvisionCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "provision [things | channels | connect | test]",
		Short: "Provision things and channels from a config file",
		Long:  `Provision things and channels: use json or csv file to bulk provision things and channels`,
	}

	for i := range cmdProvision {
		cmd.AddCommand(&cmdProvision[i])
	}

	return &cmd
}

func thingsFromFile(path string) ([]mgxsdk.Thing, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []mgxsdk.Thing{}, err
	}

	file, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return []mgxsdk.Thing{}, err
	}
	defer file.Close()

	things := []mgxsdk.Thing{}
	switch filepath.Ext(path) {
	case csvExt:
		reader := csv.NewReader(file)

		for {
			l, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				return []mgxsdk.Thing{}, err
			}

			if len(l) < 1 {
				return []mgxsdk.Thing{}, errors.New("empty line found in file")
			}

			thing := mgxsdk.Thing{
				Name: l[0],
			}

			things = append(things, thing)
		}
	case jsonExt:
		err := json.NewDecoder(file).Decode(&things)
		if err != nil {
			return []mgxsdk.Thing{}, err
		}
	default:
		return []mgxsdk.Thing{}, err
	}

	return things, nil
}

func channelsFromFile(path string) ([]mgxsdk.Channel, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []mgxsdk.Channel{}, err
	}

	file, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return []mgxsdk.Channel{}, err
	}
	defer file.Close()

	channels := []mgxsdk.Channel{}
	switch filepath.Ext(path) {
	case csvExt:
		reader := csv.NewReader(file)

		for {
			l, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				return []mgxsdk.Channel{}, err
			}

			if len(l) < 1 {
				return []mgxsdk.Channel{}, errors.New("empty line found in file")
			}

			channel := mgxsdk.Channel{
				Name: l[0],
			}

			channels = append(channels, channel)
		}
	case jsonExt:
		err := json.NewDecoder(file).Decode(&channels)
		if err != nil {
			return []mgxsdk.Channel{}, err
		}
	default:
		return []mgxsdk.Channel{}, err
	}

	return channels, nil
}

func connectionsFromFile(path string) ([]mgxsdk.Connection, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []mgxsdk.Connection{}, err
	}

	file, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return []mgxsdk.Connection{}, err
	}
	defer file.Close()

	connections := []mgxsdk.Connection{}
	switch filepath.Ext(path) {
	case csvExt:
		reader := csv.NewReader(file)

		for {
			l, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				return []mgxsdk.Connection{}, err
			}

			if len(l) < 1 {
				return []mgxsdk.Connection{}, errors.New("empty line found in file")
			}
			connections = append(connections, mgxsdk.Connection{
				ThingID:   l[0],
				ChannelID: l[1],
			})
		}
	case jsonExt:
		err := json.NewDecoder(file).Decode(&connections)
		if err != nil {
			return []mgxsdk.Connection{}, err
		}
	default:
		return []mgxsdk.Connection{}, err
	}

	return connections, nil
}
