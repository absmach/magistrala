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
	mgxsdk "github.com/absmach/magistrala/pkg/sdk/go"
	"github.com/spf13/cobra"
)

const (
	jsonExt = ".json"
	csvExt  = ".csv"
)

var (
	msgFormat      = `[{"bn":"provision:", "bu":"V", "t": %d, "bver":5, "n":"voltage", "u":"V", "v":%d}]`
	namesgenerator = namegenerator.NewGenerator()
)

var cmdProvision = []cobra.Command{
	{
		Use:   "things <things_file> <domain_id> <user_token>",
		Short: "Provision things",
		Long:  `Bulk create things`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			if _, err := os.Stat(args[0]); os.IsNotExist(err) {
				logErrorCmd(*cmd, err)
				return
			}

			things, err := thingsFromFile(args[0])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			things, err = sdk.CreateThings(things, args[1], args[2])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logJSONCmd(*cmd, things)
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

			var chs []mgxsdk.Channel
			for _, c := range channels {
				c, err = sdk.CreateChannel(c, args[1], args[2])
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
		Long:  `Bulk connect things to channels`,
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
				if err := sdk.Connect(conn, args[1], args[2]); err != nil {
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
		Long: `Provisions test setup: one test user, two things and two channels. \
						Connect both things to one of the channels, \
						and only on thing to other channel.`,
		Run: func(cmd *cobra.Command, args []string) {
			numThings := 2
			numChan := 2
			things := []mgxsdk.Thing{}
			channels := []mgxsdk.Channel{}

			if len(args) != 0 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			// Create test user
			name := namesgenerator.Generate()
			user := mgxsdk.User{
				FirstName: name,
				Email:     fmt.Sprintf("%s@email.com", name),
				Credentials: mgxsdk.Credentials{
					Username: name,
					Secret:   "12345678",
				},
				Status: mgxsdk.EnabledStatus,
			}
			user, err := sdk.CreateUser(user, "")
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			user.Credentials.Secret = "12345678"
			ut, err := sdk.CreateToken(mgxsdk.Login{Email: user.Email, Secret: user.Credentials.Secret})
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			// create domain
			domain := mgxsdk.Domain{
				Name:   fmt.Sprintf("%s-domain", name),
				Status: mgxsdk.EnabledStatus,
			}
			domain, err = sdk.CreateDomain(domain, ut.AccessToken)
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			// domain login
			ut, err = sdk.CreateToken(mgxsdk.Login{Email: user.Email, Secret: user.Credentials.Secret, DomainID: domain.ID})
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			// Create things
			for i := 0; i < numThings; i++ {
				t := mgxsdk.Thing{
					Name:   fmt.Sprintf("%s-thing-%d", name, i),
					Status: mgxsdk.EnabledStatus,
				}

				things = append(things, t)
			}
			things, err = sdk.CreateThings(things, domain.ID, ut.AccessToken)
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			// Create channels
			for i := 0; i < numChan; i++ {
				c := mgxsdk.Channel{
					Name:   fmt.Sprintf("%s-channel-%d", name, i),
					Status: mgxsdk.EnabledStatus,
				}
				c, err = sdk.CreateChannel(c, domain.ID, ut.AccessToken)
				if err != nil {
					logErrorCmd(*cmd, err)
					return
				}

				channels = append(channels, c)
			}

			// Connect things to channels - first thing to both channels, second only to first
			conIDs := mgxsdk.Connection{
				ChannelID: channels[0].ID,
				ThingID:   things[0].ID,
			}
			if err := sdk.Connect(conIDs, domain.ID, ut.AccessToken); err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			conIDs = mgxsdk.Connection{
				ChannelID: channels[1].ID,
				ThingID:   things[0].ID,
			}
			if err := sdk.Connect(conIDs, domain.ID, ut.AccessToken); err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			conIDs = mgxsdk.Connection{
				ChannelID: channels[0].ID,
				ThingID:   things[1].ID,
			}
			if err := sdk.Connect(conIDs, domain.ID, ut.AccessToken); err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			// send message to test connectivity
			if err := sdk.SendMessage(channels[0].ID, fmt.Sprintf(msgFormat, time.Now().Unix(), rand.Int()), things[0].Credentials.Secret); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			if err := sdk.SendMessage(channels[0].ID, fmt.Sprintf(msgFormat, time.Now().Unix(), rand.Int()), things[1].Credentials.Secret); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			if err := sdk.SendMessage(channels[1].ID, fmt.Sprintf(msgFormat, time.Now().Unix(), rand.Int()), things[0].Credentials.Secret); err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logJSONCmd(*cmd, user, ut, things, channels)
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
