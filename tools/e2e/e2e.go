// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"time"

	"github.com/0x6flab/namegenerator"
	sdk "github.com/absmach/magistrala/pkg/sdk/go"
	"github.com/gookit/color"
	"github.com/gorilla/websocket"
	"golang.org/x/sync/errgroup"
)

const (
	defPass     = "12345678"
	defWSPort   = "8186"
	numAdapters = 4
	batchSize   = 99
	usersPort   = "9002"
	clientsPort = "9000"
	domainsPort = "8189"
)

var (
	namesgenerator = namegenerator.NewGenerator()
	msgFormat      = `[{"bn":"demo", "bu":"V", "t": %d, "bver":5, "n":"voltage", "u":"V", "v":%d}]`
)

// Config - test configuration.
type Config struct {
	Host     string
	Num      uint64
	NumOfMsg uint64
	SSL      bool
	CA       string
	CAKey    string
	Prefix   string
}

// Test - function that does actual end to end testing.
// The operations are:
// - Create a user
// - Create other users
// - Do Read, Update and Change of Status operations on users.

// - Create groups using hierarchy
// - Do Read, Update and Change of Status operations on groups.

// - Create clients
// - Do Read, Update and Change of Status operations on clients.

// - Create channels
// - Do Read, Update and Change of Status operations on channels.

// - Connect client to channel
// - Publish message from HTTP, MQTT, WS and CoAP Adapters.
func Test(conf Config) {
	sdkConf := sdk.Config{
		ClientsURL:      fmt.Sprintf("http://%s:%s", conf.Host, clientsPort),
		UsersURL:        fmt.Sprintf("http://%s:%s", conf.Host, usersPort),
		DomainsURL:      fmt.Sprintf("http://%s:%s", conf.Host, domainsPort),
		HTTPAdapterURL:  fmt.Sprintf("http://%s/http", conf.Host),
		MsgContentType:  sdk.CTJSONSenML,
		TLSVerification: false,
	}

	s := sdk.NewSDK(sdkConf)

	magenta := color.FgLightMagenta.Render

	domainID, token, err := createUser(s, conf)
	if err != nil {
		errExit(fmt.Errorf("unable to create user: %w", err))
	}
	color.Success.Printf("created user with token %s\n", magenta(token))

	users, err := createUsers(s, conf, token)
	if err != nil {
		errExit(fmt.Errorf("unable to create users: %w", err))
	}
	color.Success.Printf("created users of ids:\n%s\n", magenta(getIDS(users)))

	groups, err := createGroups(s, conf, domainID, token)
	if err != nil {
		errExit(fmt.Errorf("unable to create groups: %w", err))
	}
	color.Success.Printf("created groups of ids:\n%s\n", magenta(getIDS(groups)))

	clients, err := createClients(s, conf, domainID, token)
	if err != nil {
		errExit(fmt.Errorf("unable to create clients: %w", err))
	}
	color.Success.Printf("created clients of ids:\n%s\n", magenta(getIDS(clients)))

	channels, err := createChannels(s, conf, domainID, token)
	if err != nil {
		errExit(fmt.Errorf("unable to create channels: %w", err))
	}
	color.Success.Printf("created channels of ids:\n%s\n", magenta(getIDS(channels)))

	// List users, groups, clients and channels
	if err := read(s, conf, domainID, token, users, groups, clients, channels); err != nil {
		errExit(fmt.Errorf("unable to read users, groups, clients and channels: %w", err))
	}
	color.Success.Println("viewed users, groups, clients and channels")

	// Update users, groups, clients and channels
	if err := update(s, domainID, token, users, groups, clients, channels); err != nil {
		errExit(fmt.Errorf("unable to update users, groups, clients and channels: %w", err))
	}
	color.Success.Println("updated users, groups, clients and channels")

	// Send messages to channels
	if err := messaging(s, conf, domainID, token, clients, channels); err != nil {
		errExit(fmt.Errorf("unable to send messages to channels: %w", err))
	}
	color.Success.Println("sent messages to channels")
}

func errExit(err error) {
	color.Error.Println(err.Error())
	os.Exit(1)
}

func createUser(s sdk.SDK, conf Config) (string, string, error) {
	user := sdk.User{
		FirstName: fmt.Sprintf("%s%s", conf.Prefix, namesgenerator.Generate()),
		LastName:  fmt.Sprintf("%s%s", conf.Prefix, namesgenerator.Generate()),
		Email:     fmt.Sprintf("%s%s@email.com", conf.Prefix, namesgenerator.Generate()),
		Credentials: sdk.Credentials{
			Username: fmt.Sprintf("%s%s", conf.Prefix, namesgenerator.Generate()),
			Secret:   defPass,
		},
		Status: sdk.EnabledStatus,
		Role:   "admin",
	}

	if _, err := s.CreateUser(user, ""); err != nil {
		return "", "", fmt.Errorf("unable to create user: %w", err)
	}

	login := sdk.Login{
		Identity: user.Credentials.Username,
		Secret:   user.Credentials.Secret,
	}
	token, err := s.CreateToken(login)
	if err != nil {
		return "", "", fmt.Errorf("unable to login user: %w", err)
	}

	dname := fmt.Sprintf("%s%s", conf.Prefix, namesgenerator.Generate())
	domain := sdk.Domain{
		Name:       dname,
		Alias:      strings.ToLower(dname),
		Permission: "admin",
	}

	domain, err = s.CreateDomain(domain, token.AccessToken)
	if err != nil {
		return "", "", fmt.Errorf("unable to create domain: %w", err)
	}

	login = sdk.Login{
		Identity: user.Credentials.Username,
		Secret:   user.Credentials.Secret,
	}
	token, err = s.CreateToken(login)
	if err != nil {
		return "", "", fmt.Errorf("unable to login user: %w", err)
	}

	return domain.ID, token.AccessToken, nil
}

func createUsers(s sdk.SDK, conf Config, token string) ([]sdk.User, error) {
	var err error
	users := []sdk.User{}

	for i := uint64(0); i < conf.Num; i++ {
		user := sdk.User{
			FirstName: fmt.Sprintf("%s%s", conf.Prefix, namesgenerator.Generate()),
			LastName:  fmt.Sprintf("%s%s", conf.Prefix, namesgenerator.Generate()),
			Email:     fmt.Sprintf("%s%s@email.com", conf.Prefix, namesgenerator.Generate()),
			Credentials: sdk.Credentials{
				Username: fmt.Sprintf("%s%s", conf.Prefix, namesgenerator.Generate()),
				Secret:   defPass,
			},
			Status: sdk.EnabledStatus,
		}

		user, err = s.CreateUser(user, token)
		if err != nil {
			return []sdk.User{}, fmt.Errorf("failed to create the users: %w", err)
		}
		users = append(users, user)
	}

	return users, nil
}

func createGroups(s sdk.SDK, conf Config, domainID, token string) ([]sdk.Group, error) {
	var err error
	groups := []sdk.Group{}

	for i := uint64(0); i < conf.Num; i++ {
		group := sdk.Group{
			Name:   fmt.Sprintf("%s%s", conf.Prefix, namesgenerator.Generate()),
			Status: sdk.EnabledStatus,
		}

		group, err = s.CreateGroup(group, domainID, token)
		if err != nil {
			return []sdk.Group{}, fmt.Errorf("failed to create the group: %w", err)
		}
		groups = append(groups, group)
	}

	return groups, nil
}

func createClientsInBatch(s sdk.SDK, conf Config, domainID, token string, num uint64) ([]sdk.Client, error) {
	var err error
	clients := make([]sdk.Client, num)

	for i := uint64(0); i < num; i++ {
		clients[i] = sdk.Client{
			Name: fmt.Sprintf("%s%s", conf.Prefix, namesgenerator.Generate()),
		}
	}

	clients, err = s.CreateClients(clients, domainID, token)
	if err != nil {
		return []sdk.Client{}, fmt.Errorf("failed to create the clients: %w", err)
	}

	return clients, nil
}

func createClients(s sdk.SDK, conf Config, domainID, token string) ([]sdk.Client, error) {
	clients := []sdk.Client{}

	if conf.Num > batchSize {
		batches := int(conf.Num) / batchSize
		for i := 0; i < batches; i++ {
			ths, err := createClientsInBatch(s, conf, domainID, token, batchSize)
			if err != nil {
				return []sdk.Client{}, fmt.Errorf("failed to create the clients: %w", err)
			}
			clients = append(clients, ths...)
		}
		ths, err := createClientsInBatch(s, conf, domainID, token, conf.Num%uint64(batchSize))
		if err != nil {
			return []sdk.Client{}, fmt.Errorf("failed to create the clients: %w", err)
		}
		clients = append(clients, ths...)
	} else {
		ths, err := createClientsInBatch(s, conf, domainID, token, conf.Num)
		if err != nil {
			return []sdk.Client{}, fmt.Errorf("failed to create the clients: %w", err)
		}
		clients = append(clients, ths...)
	}

	return clients, nil
}

func createChannelsInBatch(s sdk.SDK, conf Config, domainID, token string, num uint64) ([]sdk.Channel, error) {
	var err error
	channels := make([]sdk.Channel, num)

	for i := uint64(0); i < num; i++ {
		channels[i] = sdk.Channel{
			Name: fmt.Sprintf("%s%s", conf.Prefix, namesgenerator.Generate()),
		}
		channels[i], err = s.CreateChannel(channels[i], domainID, token)
		if err != nil {
			return []sdk.Channel{}, fmt.Errorf("failed to create the channels: %w", err)
		}
	}

	return channels, nil
}

func createChannels(s sdk.SDK, conf Config, domainID, token string) ([]sdk.Channel, error) {
	channels := []sdk.Channel{}

	if conf.Num > batchSize {
		batches := int(conf.Num) / batchSize
		for i := 0; i < batches; i++ {
			chs, err := createChannelsInBatch(s, conf, token, domainID, batchSize)
			if err != nil {
				return []sdk.Channel{}, fmt.Errorf("failed to create the channels: %w", err)
			}
			channels = append(channels, chs...)
		}
		chs, err := createChannelsInBatch(s, conf, domainID, token, conf.Num%uint64(batchSize))
		if err != nil {
			return []sdk.Channel{}, fmt.Errorf("failed to create the channels: %w", err)
		}
		channels = append(channels, chs...)
	} else {
		chs, err := createChannelsInBatch(s, conf, domainID, token, conf.Num)
		if err != nil {
			return []sdk.Channel{}, fmt.Errorf("failed to create the channels: %w", err)
		}
		channels = append(channels, chs...)
	}

	return channels, nil
}

func read(s sdk.SDK, conf Config, domainID, token string, users []sdk.User, groups []sdk.Group, clients []sdk.Client, channels []sdk.Channel) error {
	for _, user := range users {
		if _, err := s.User(user.ID, token); err != nil {
			return fmt.Errorf("failed to get user %w", err)
		}
	}
	up, err := s.Users(sdk.PageMetadata{}, token)
	if err != nil {
		return fmt.Errorf("failed to get users %w", err)
	}
	if up.Total < conf.Num {
		return fmt.Errorf("returned users %d less than created users %d", up.Total, conf.Num)
	}
	for _, group := range groups {
		if _, err := s.Group(group.ID, domainID, token); err != nil {
			return fmt.Errorf("failed to get group %w", err)
		}
	}
	gp, err := s.Groups(sdk.PageMetadata{}, domainID, token)
	if err != nil {
		return fmt.Errorf("failed to get groups %w", err)
	}
	if gp.Total < conf.Num {
		return fmt.Errorf("returned groups %d less than created groups %d", gp.Total, conf.Num)
	}
	for _, c := range clients {
		if _, err := s.Client(c.ID, domainID, token); err != nil {
			return fmt.Errorf("failed to get client %w", err)
		}
	}
	tp, err := s.Clients(sdk.PageMetadata{}, domainID, token)
	if err != nil {
		return fmt.Errorf("failed to get clients %w", err)
	}
	if tp.Total < conf.Num {
		return fmt.Errorf("returned clients %d less than created clients %d", tp.Total, conf.Num)
	}
	for _, channel := range channels {
		if _, err := s.Channel(channel.ID, domainID, token); err != nil {
			return fmt.Errorf("failed to get channel %w", err)
		}
	}
	cp, err := s.Channels(sdk.PageMetadata{}, domainID, token)
	if err != nil {
		return fmt.Errorf("failed to get channels %w", err)
	}
	if cp.Total < conf.Num {
		return fmt.Errorf("returned channels %d less than created channels %d", cp.Total, conf.Num)
	}

	return nil
}

func update(s sdk.SDK, domainID, token string, users []sdk.User, groups []sdk.Group, clients []sdk.Client, channels []sdk.Channel) error {
	for _, user := range users {
		user.FirstName = namesgenerator.Generate()
		user.Metadata = sdk.Metadata{"Update": namesgenerator.Generate()}
		rUser, err := s.UpdateUser(user, token)
		if err != nil {
			return fmt.Errorf("failed to update user %w", err)
		}
		if rUser.FirstName != user.FirstName {
			return fmt.Errorf("failed to update user name before %s after %s", user.FirstName, rUser.FirstName)
		}
		if rUser.Metadata["Update"] != user.Metadata["Update"] {
			return fmt.Errorf("failed to update user metadata before %s after %s", user.Metadata["Update"], rUser.Metadata["Update"])
		}
		user = rUser
		user.Credentials.Username = namesgenerator.Generate()
		rUser, err = s.UpdateUsername(user, token)
		if err != nil {
			return fmt.Errorf("failed to update username %w", err)
		}
		if rUser.Credentials.Username != user.Credentials.Username {
			return fmt.Errorf("failed to update user name before %s after %s", user.Credentials.Username, rUser.Credentials.Username)
		}
		user = rUser
		rUser, err = s.UpdateUserEmail(user, token)
		if err != nil {
			return fmt.Errorf("failed to update user identity %w", err)
		}
		if rUser.Email != user.Email {
			return fmt.Errorf("failed to update user identity before %s after %s", user.Email, rUser.Email)
		}
		user = rUser
		user.Tags = []string{namesgenerator.Generate()}
		rUser, err = s.UpdateUserTags(user, token)
		if err != nil {
			return fmt.Errorf("failed to update user tags %w", err)
		}
		if rUser.Tags[0] != user.Tags[0] {
			return fmt.Errorf("failed to update user tags before %s after %s", user.Tags[0], rUser.Tags[0])
		}
		user = rUser
		rUser, err = s.DisableUser(user.ID, token)
		if err != nil {
			return fmt.Errorf("failed to disable user %w", err)
		}
		if rUser.Status != sdk.DisabledStatus {
			return fmt.Errorf("failed to disable user before %s after %s", user.Status, rUser.Status)
		}
		user = rUser
		rUser, err = s.EnableUser(user.ID, token)
		if err != nil {
			return fmt.Errorf("failed to enable user %w", err)
		}
		if rUser.Status != sdk.EnabledStatus {
			return fmt.Errorf("failed to enable user before %s after %s", user.Status, rUser.Status)
		}
	}
	for _, group := range groups {
		group.Name = namesgenerator.Generate()
		group.Metadata = sdk.Metadata{"Update": namesgenerator.Generate()}
		rGroup, err := s.UpdateGroup(group, domainID, token)
		if err != nil {
			return fmt.Errorf("failed to update group %w", err)
		}
		if rGroup.Name != group.Name {
			return fmt.Errorf("failed to update group name before %s after %s", group.Name, rGroup.Name)
		}
		if rGroup.Metadata["Update"] != group.Metadata["Update"] {
			return fmt.Errorf("failed to update group metadata before %s after %s", group.Metadata["Update"], rGroup.Metadata["Update"])
		}
		group = rGroup
		rGroup, err = s.DisableGroup(group.ID, domainID, token)
		if err != nil {
			return fmt.Errorf("failed to disable group %w", err)
		}
		if rGroup.Status != sdk.DisabledStatus {
			return fmt.Errorf("failed to disable group before %s after %s", group.Status, rGroup.Status)
		}
		group = rGroup
		rGroup, err = s.EnableGroup(group.ID, domainID, token)
		if err != nil {
			return fmt.Errorf("failed to enable group %w", err)
		}
		if rGroup.Status != sdk.EnabledStatus {
			return fmt.Errorf("failed to enable group before %s after %s", group.Status, rGroup.Status)
		}
	}
	for _, t := range clients {
		t.Name = namesgenerator.Generate()
		t.Metadata = sdk.Metadata{"Update": namesgenerator.Generate()}
		rClient, err := s.UpdateClient(t, domainID, token)
		if err != nil {
			return fmt.Errorf("failed to update client %w", err)
		}
		if rClient.Name != t.Name {
			return fmt.Errorf("failed to update client name before %s after %s", t.Name, rClient.Name)
		}
		if rClient.Metadata["Update"] != t.Metadata["Update"] {
			return fmt.Errorf("failed to update client metadata before %s after %s", t.Metadata["Update"], rClient.Metadata["Update"])
		}
		t = rClient
		rClient, err = s.UpdateClientSecret(t.ID, t.Credentials.Secret, domainID, token)
		if err != nil {
			return fmt.Errorf("failed to update client secret %w", err)
		}
		t = rClient
		t.Tags = []string{namesgenerator.Generate()}
		rClient, err = s.UpdateClientTags(t, domainID, token)
		if err != nil {
			return fmt.Errorf("failed to update client tags %w", err)
		}
		if rClient.Tags[0] != t.Tags[0] {
			return fmt.Errorf("failed to update client tags before %s after %s", t.Tags[0], rClient.Tags[0])
		}
		t = rClient
		rClient, err = s.DisableClient(t.ID, domainID, token)
		if err != nil {
			return fmt.Errorf("failed to disable client %w", err)
		}
		if rClient.Status != sdk.DisabledStatus {
			return fmt.Errorf("failed to disable client before %s after %s", t.Status, rClient.Status)
		}
		t = rClient
		rClient, err = s.EnableClient(t.ID, domainID, token)
		if err != nil {
			return fmt.Errorf("failed to enable client %w", err)
		}
		if rClient.Status != sdk.EnabledStatus {
			return fmt.Errorf("failed to enable client before %s after %s", t.Status, rClient.Status)
		}
	}
	for _, channel := range channels {
		channel.Name = namesgenerator.Generate()
		channel.Metadata = sdk.Metadata{"Update": namesgenerator.Generate()}
		rChannel, err := s.UpdateChannel(channel, domainID, token)
		if err != nil {
			return fmt.Errorf("failed to update channel %w", err)
		}
		if rChannel.Name != channel.Name {
			return fmt.Errorf("failed to update channel name before %s after %s", channel.Name, rChannel.Name)
		}
		if rChannel.Metadata["Update"] != channel.Metadata["Update"] {
			return fmt.Errorf("failed to update channel metadata before %s after %s", channel.Metadata["Update"], rChannel.Metadata["Update"])
		}
		channel = rChannel
		rChannel, err = s.DisableChannel(channel.ID, domainID, token)
		if err != nil {
			return fmt.Errorf("failed to disable channel %w", err)
		}
		if rChannel.Status != sdk.DisabledStatus {
			return fmt.Errorf("failed to disable channel before %s after %s", channel.Status, rChannel.Status)
		}
		channel = rChannel
		rChannel, err = s.EnableChannel(channel.ID, domainID, token)
		if err != nil {
			return fmt.Errorf("failed to enable channel %w", err)
		}
		if rChannel.Status != sdk.EnabledStatus {
			return fmt.Errorf("failed to enable channel before %s after %s", channel.Status, rChannel.Status)
		}
	}

	return nil
}

func messaging(s sdk.SDK, conf Config, domainID, token string, clients []sdk.Client, channels []sdk.Channel) error {
	for _, c := range clients {
		for _, channel := range channels {
			conn := sdk.Connection{
				ClientID:  c.ID,
				ChannelID: channel.ID,
			}
			if err := s.Connect(conn, domainID, token); err != nil {
				return fmt.Errorf("failed to connect client %s to channel %s", c.ID, channel.ID)
			}
		}
	}

	g := new(errgroup.Group)

	bt := time.Now().Unix()
	for i := uint64(0); i < conf.NumOfMsg; i++ {
		for _, client := range clients {
			for _, channel := range channels {
				func(num int64, client sdk.Client, channel sdk.Channel) {
					g.Go(func() error {
						msg := fmt.Sprintf(msgFormat, num+1, rand.Int())
						return sendHTTPMessage(s, msg, client, channel.ID)
					})
					g.Go(func() error {
						msg := fmt.Sprintf(msgFormat, num+2, rand.Int())
						return sendCoAPMessage(msg, client, channel.ID)
					})
					g.Go(func() error {
						msg := fmt.Sprintf(msgFormat, num+3, rand.Int())
						return sendMQTTMessage(msg, client, channel.ID)
					})
					g.Go(func() error {
						msg := fmt.Sprintf(msgFormat, num+4, rand.Int())
						return sendWSMessage(conf, msg, client, channel.ID)
					})
				}(bt, client, channel)
				bt += numAdapters
			}
		}
	}

	return g.Wait()
}

func sendHTTPMessage(s sdk.SDK, msg string, client sdk.Client, chanID string) error {
	if err := s.SendMessage(chanID, msg, client.Credentials.Secret); err != nil {
		return fmt.Errorf("HTTP failed to send message from client %s to channel %s: %w", client.ID, chanID, err)
	}

	return nil
}

func sendCoAPMessage(msg string, client sdk.Client, chanID string) error {
	cmd := exec.Command("coap-cli", "post", fmt.Sprintf("channels/%s/messages", chanID), "--auth", client.Credentials.Secret, "-d", msg)
	if _, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("CoAP failed to send message from client %s to channel %s: %w", client.ID, chanID, err)
	}

	return nil
}

func sendMQTTMessage(msg string, client sdk.Client, chanID string) error {
	cmd := exec.Command("mosquitto_pub", "--id-prefix", "magistrala", "-u", client.ID, "-P", client.Credentials.Secret, "-t", fmt.Sprintf("channels/%s/messages", chanID), "-h", "localhost", "-m", msg)
	if _, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("MQTT failed to send message from client %s to channel %s: %w", client.ID, chanID, err)
	}

	return nil
}

func sendWSMessage(conf Config, msg string, client sdk.Client, chanID string) error {
	socketURL := fmt.Sprintf("ws://%s:%s/channels/%s/messages", conf.Host, defWSPort, chanID)
	header := http.Header{"Authorization": []string{client.Credentials.Secret}}
	conn, _, err := websocket.DefaultDialer.Dial(socketURL, header)
	if err != nil {
		return fmt.Errorf("unable to connect to websocket: %w", err)
	}
	defer conn.Close()
	if err := conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
		return fmt.Errorf("WS failed to send message from client %s to channel %s: %w", client.ID, chanID, err)
	}

	return nil
}

// getIDS returns a list of IDs of the given objects.
func getIDS(objects interface{}) string {
	v := reflect.ValueOf(objects)
	if v.Kind() != reflect.Slice {
		panic("objects argument must be a slice")
	}
	ids := make([]string, v.Len())
	for i := 0; i < v.Len(); i++ {
		id := v.Index(i).FieldByName("ID").String()
		ids[i] = id
	}
	idList := strings.Join(ids, "\n")

	return idList
}
