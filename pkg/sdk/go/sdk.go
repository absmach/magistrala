// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/absmach/magistrala/pkg/errors"
	"moul.io/http2curl"
)

const (
	// CTJSON represents JSON content type.
	CTJSON ContentType = "application/json"

	// CTJSONSenML represents JSON SenML content type.
	CTJSONSenML ContentType = "application/senml+json"

	// CTBinary represents binary content type.
	CTBinary ContentType = "application/octet-stream"

	// EnabledStatus represents enable status for a client.
	EnabledStatus = "enabled"

	// DisabledStatus represents disabled status for a client.
	DisabledStatus = "disabled"

	BearerPrefix = "Bearer "

	ThingPrefix = "Thing "
)

// ContentType represents all possible content types.
type ContentType string

var _ SDK = (*mgSDK)(nil)

var (
	// ErrFailedCreation indicates that entity creation failed.
	ErrFailedCreation = errors.New("failed to create entity in the db")

	// ErrFailedList indicates that entities list failed.
	ErrFailedList = errors.New("failed to list entities")

	// ErrFailedUpdate indicates that entity update failed.
	ErrFailedUpdate = errors.New("failed to update entity")

	// ErrFailedFetch indicates that fetching of entity data failed.
	ErrFailedFetch = errors.New("failed to fetch entity")

	// ErrFailedRemoval indicates that entity removal failed.
	ErrFailedRemoval = errors.New("failed to remove entity")

	// ErrFailedEnable indicates that client enable failed.
	ErrFailedEnable = errors.New("failed to enable client")

	// ErrFailedDisable indicates that client disable failed.
	ErrFailedDisable = errors.New("failed to disable client")

	ErrInvalidJWT = errors.New("invalid JWT")
)

type MessagePageMetadata struct {
	PageMetadata
	Subtopic    string  `json:"subtopic,omitempty"`
	Publisher   string  `json:"publisher,omitempty"`
	Comparator  string  `json:"comparator,omitempty"`
	BoolValue   *bool   `json:"vb,omitempty"`
	StringValue string  `json:"vs,omitempty"`
	DataValue   string  `json:"vd,omitempty"`
	From        float64 `json:"from,omitempty"`
	To          float64 `json:"to,omitempty"`
	Aggregation string  `json:"aggregation,omitempty"`
	Interval    string  `json:"interval,omitempty"`
	Value       float64 `json:"value,omitempty"`
	Protocol    string  `json:"protocol,omitempty"`
}

type PageMetadata struct {
	Total           uint64   `json:"total"`
	Offset          uint64   `json:"offset"`
	Limit           uint64   `json:"limit"`
	Order           string   `json:"order,omitempty"`
	Direction       string   `json:"direction,omitempty"`
	Level           uint64   `json:"level,omitempty"`
	Identity        string   `json:"identity,omitempty"`
	Name            string   `json:"name,omitempty"`
	Type            string   `json:"type,omitempty"`
	Metadata        Metadata `json:"metadata,omitempty"`
	Status          string   `json:"status,omitempty"`
	Action          string   `json:"action,omitempty"`
	Subject         string   `json:"subject,omitempty"`
	Object          string   `json:"object,omitempty"`
	Permission      string   `json:"permission,omitempty"`
	Tag             string   `json:"tag,omitempty"`
	Owner           string   `json:"owner,omitempty"`
	SharedBy        string   `json:"shared_by,omitempty"`
	Visibility      string   `json:"visibility,omitempty"`
	OwnerID         string   `json:"owner_id,omitempty"`
	Topic           string   `json:"topic,omitempty"`
	Contact         string   `json:"contact,omitempty"`
	State           string   `json:"state,omitempty"`
	ListPermissions string   `json:"list_perms,omitempty"`
	InvitedBy       string   `json:"invited_by,omitempty"`
	UserID          string   `json:"user_id,omitempty"`
	DomainID        string   `json:"domain_id,omitempty"`
	Relation        string   `json:"relation,omitempty"`
	User            string   `json:"user,omitempty"`
	Channel         string   `json:"channel,omitempty"`
	Group           string   `json:"group,omitempty"`
	Thing           string   `json:"thing,omitempty"`
	Domain          string   `json:"domain,omitempty"`
	Operation       string   `json:"operation,omitempty"`
	From            int64    `json:"from,omitempty"`
	To              int64    `json:"to,omitempty"`
	WithMetadata    bool     `json:"with_metadata,omitempty"`
	WithAttributes  bool     `json:"with_attributes,omitempty"`
	ID              string   `json:"id,omitempty"`
}

// Credentials represent client credentials: it contains
// "identity" which can be a username, email, generated name;
// and "secret" which can be a password or access token.
type Credentials struct {
	Identity string `json:"identity,omitempty"` // username or generated login ID
	Secret   string `json:"secret,omitempty"`   // password or token
}

// SDK contains Magistrala API.
//
//go:generate mockery --name SDK --output=../mocks --filename sdk.go --quiet --note "Copyright (c) Abstract Machines"
type SDK interface {
	// CreateUser registers magistrala user.
	//
	// example:
	//  user := sdk.User{
	//    Name:	 "John Doe",
	//    Credentials: sdk.Credentials{
	//      Identity: "john.doe@example",
	//      Secret:   "12345678",
	//    },
	//  }
	//  user, _ := sdk.CreateUser(user)
	//  fmt.Println(user)
	CreateUser(user User, token string) (User, errors.SDKError)

	// User returns user object by id.
	//
	// example:
	//  user, _ := sdk.User("userID", "token")
	//  fmt.Println(user)
	User(id, token string) (User, errors.SDKError)

	// Users returns list of users.
	//
	// example:
	//	pm := sdk.PageMetadata{
	//		Offset: 0,
	//		Limit:  10,
	//		Name:   "John Doe",
	//	}
	//	users, _ := sdk.Users(pm, "token")
	//	fmt.Println(users)
	Users(pm PageMetadata, token string) (UsersPage, errors.SDKError)

	// Members returns list of users that are members of a group.
	//
	// example:
	//	pm := sdk.PageMetadata{
	//		Offset: 0,
	//		Limit:  10,
	//	}
	//	members, _ := sdk.Members("groupID", pm, "token")
	//	fmt.Println(members)
	Members(groupID string, meta PageMetadata, token string) (UsersPage, errors.SDKError)

	// UserProfile returns user logged in.
	//
	// example:
	//  user, _ := sdk.UserProfile("token")
	//  fmt.Println(user)
	UserProfile(token string) (User, errors.SDKError)

	// UpdateUser updates existing user.
	//
	// example:
	//  user := sdk.User{
	//    ID:   "userID",
	//    Name: "John Doe",
	//    Metadata: sdk.Metadata{
	//      "key": "value",
	//    },
	//  }
	//  user, _ := sdk.UpdateUser(user, "token")
	//  fmt.Println(user)
	UpdateUser(user User, token string) (User, errors.SDKError)

	// UpdateUserTags updates the user's tags.
	//
	// example:
	//  user := sdk.User{
	//    ID:   "userID",
	//    Tags: []string{"tag1", "tag2"},
	//  }
	//  user, _ := sdk.UpdateUserTags(user, "token")
	//  fmt.Println(user)
	UpdateUserTags(user User, token string) (User, errors.SDKError)

	// UpdateUserIdentity updates the user's identity
	//
	// example:
	//  user := sdk.User{
	//    ID:   "userID",
	//    Credentials: sdk.Credentials{
	//      Identity: "john.doe@example",
	//    },
	//  }
	//  user, _ := sdk.UpdateUserIdentity(user, "token")
	//  fmt.Println(user)
	UpdateUserIdentity(user User, token string) (User, errors.SDKError)

	// UpdateUserRole updates the user's role.
	//
	// example:
	//  user := sdk.User{
	//    ID:   "userID",
	//    Role: "role",
	//  }
	//  user, _ := sdk.UpdateUserRole(user, "token")
	//  fmt.Println(user)
	UpdateUserRole(user User, token string) (User, errors.SDKError)

	// ResetPasswordRequest sends a password request email to a user.
	//
	// example:
	//  err := sdk.ResetPasswordRequest("example@email.com")
	//  fmt.Println(err)
	ResetPasswordRequest(email string) errors.SDKError

	// ResetPassword changes a user's password to the one passed in the argument.
	//
	// example:
	//  err := sdk.ResetPassword("password","password","token")
	//  fmt.Println(err)
	ResetPassword(password, confPass, token string) errors.SDKError

	// UpdatePassword updates user password.
	//
	// example:
	//  user, _ := sdk.UpdatePassword("oldPass", "newPass", "token")
	//  fmt.Println(user)
	UpdatePassword(oldPass, newPass, token string) (User, errors.SDKError)

	// EnableUser changes the status of the user to enabled.
	//
	// example:
	//  user, _ := sdk.EnableUser("userID", "token")
	//  fmt.Println(user)
	EnableUser(id, token string) (User, errors.SDKError)

	// DisableUser changes the status of the user to disabled.
	//
	// example:
	//  user, _ := sdk.DisableUser("userID", "token")
	//  fmt.Println(user)
	DisableUser(id, token string) (User, errors.SDKError)

	// DeleteUser deletes a user with the given id.
	//
	// example:
	//  err := sdk.DeleteUser("userID", "token")
	//  fmt.Println(err)
	DeleteUser(id, token string) errors.SDKError

	// CreateToken receives credentials and returns user token.
	//
	// example:
	//  lt := sdk.Login{
	//      Identity: "john.doe@example",
	//      Secret:   "12345678",
	//  }
	//  token, _ := sdk.CreateToken(lt)
	//  fmt.Println(token)
	CreateToken(lt Login) (Token, errors.SDKError)

	// RefreshToken receives credentials and returns user token.
	//
	// example:
	//  lt := sdk.Login{
	//      DomainID:   "domain_id",
	//  }
	// example:
	//  token, _ := sdk.RefreshToken(lt,"refresh_token")
	//  fmt.Println(token)
	RefreshToken(lt Login, token string) (Token, errors.SDKError)

	// ListUserChannels list all channels belongs a particular user id.
	//
	// example:
	//	pm := sdk.PageMetadata{
	//		Offset: 0,
	//		Limit:  10,
	//		Permission: "edit", // available Options:  "administrator", "administrator", "delete", edit", "view", "share", "owner", "owner", "admin", "editor", "viewer", "guest", "editor", "contributor", "create"
	//	}
	//  channels, _ := sdk.ListUserChannels(pm, "token")
	//  fmt.Println(channels)
	ListUserChannels(pm PageMetadata, token string) (ChannelsPage, errors.SDKError)

	// ListUserGroups list all groups belongs a particular user id.
	//
	// example:
	//	pm := sdk.PageMetadata{
	//		Offset: 0,
	//		Limit:  10,
	//		Permission: "edit", // available Options:  "administrator", "administrator", "delete", edit", "view", "share", "owner", "owner", "admin", "editor", "contributor", "editor", "viewer", "guest", "create"
	//	}
	//  groups, _ := sdk.ListUserGroups(pm, "token")
	//  fmt.Println(channels)
	ListUserGroups(pm PageMetadata, token string) (GroupsPage, errors.SDKError)

	// ListUserThings list all things belongs a particular user id.
	//
	// example:
	//	pm := sdk.PageMetadata{
	//		Offset: 0,
	//		Limit:  10,
	//		Permission: "edit", // available Options:  "administrator", "administrator", "delete", edit", "view", "share", "owner", "owner", "admin", "editor", "contributor", "editor", "viewer", "guest", "create"
	//	}
	//  things, _ := sdk.ListUserThings(pm, "token")
	//  fmt.Println(things)
	ListUserThings(userID string, pm PageMetadata, token string) (ThingsPage, errors.SDKError)

	// SeachUsers filters users and returns a page result.
	//
	// example:
	//  pm := sdk.PageMetadata{
	//	Offset: 0,
	//	Limit:  10,
	//	Name:   "John Doe",
	//  }
	//  users, _ := sdk.SearchUsers(pm, "token")
	//  fmt.Println(users)
	SearchUsers(pm PageMetadata, token string) (UsersPage, errors.SDKError)

	// CreateThing registers new thing and returns its id.
	//
	// example:
	//  thing := sdk.Thing{
	//    Name: "My Thing",
	//    Metadata: sdk.Metadata{
	//      "key": "value",
	//    },
	//  }
	//  thing, _ := sdk.CreateThing(thing, "token")
	//  fmt.Println(thing)
	CreateThing(thing Thing, token string) (Thing, errors.SDKError)

	// CreateThings registers new things and returns their ids.
	//
	// example:
	//  things := []sdk.Thing{
	//    {
	//      Name: "My Thing 1",
	//      Metadata: sdk.Metadata{
	//        "key": "value",
	//      },
	//    },
	//    {
	//      Name: "My Thing 2",
	//      Metadata: sdk.Metadata{
	//        "key": "value",
	//      },
	//    },
	//  }
	//  things, _ := sdk.CreateThings(things, "token")
	//  fmt.Println(things)
	CreateThings(things []Thing, token string) ([]Thing, errors.SDKError)

	// Filters things and returns a page result.
	//
	// example:
	//  pm := sdk.PageMetadata{
	//    Offset: 0,
	//    Limit:  10,
	//    Name:   "My Thing",
	//  }
	//  things, _ := sdk.Things(pm, "token")
	//  fmt.Println(things)
	Things(pm PageMetadata, token string) (ThingsPage, errors.SDKError)

	// ThingsByChannel returns page of things that are connected to specified channel.
	//
	// example:
	//  pm := sdk.PageMetadata{
	//    Offset: 0,
	//    Limit:  10,
	//    Channel: "channelID",
	//    Name:   "My Thing",
	//  }
	//  things, _ := sdk.ThingsByChannel("channelID", pm, "token")
	//  fmt.Println(things)
	ThingsByChannel(pm PageMetadata, token string) (ThingsPage, errors.SDKError)

	// Thing returns thing object by id.
	//
	// example:
	//  thing, _ := sdk.Thing("thingID", "token")
	//  fmt.Println(thing)
	Thing(id, token string) (Thing, errors.SDKError)

	// ThingPermissions returns user permissions on the thing id.
	//
	// example:
	//  thing, _ := sdk.Thing("thingID", "token")
	//  fmt.Println(thing)
	ThingPermissions(id, token string) (Thing, errors.SDKError)

	// UpdateThing updates existing thing.
	//
	// example:
	//  thing := sdk.Thing{
	//    ID:   "thingID",
	//    Name: "My Thing",
	//    Metadata: sdk.Metadata{
	//      "key": "value",
	//    },
	//  }
	//  thing, _ := sdk.UpdateThing(thing, "token")
	//  fmt.Println(thing)
	UpdateThing(thing Thing, token string) (Thing, errors.SDKError)

	// UpdateThingTags updates the client's tags.
	//
	// example:
	//  thing := sdk.Thing{
	//    ID:   "thingID",
	//    Tags: []string{"tag1", "tag2"},
	//  }
	//  thing, _ := sdk.UpdateThingTags(thing, "token")
	//  fmt.Println(thing)
	UpdateThingTags(thing Thing, token string) (Thing, errors.SDKError)

	// UpdateThingSecret updates the client's secret
	//
	// example:
	//  thing, err := sdk.UpdateThingSecret("thingID", "newSecret", "token")
	//  fmt.Println(thing)
	UpdateThingSecret(id, secret, token string) (Thing, errors.SDKError)

	// EnableThing changes client status to enabled.
	//
	// example:
	//  thing, _ := sdk.EnableThing("thingID", "token")
	//  fmt.Println(thing)
	EnableThing(id, token string) (Thing, errors.SDKError)

	// DisableThing changes client status to disabled - soft delete.
	//
	// example:
	//  thing, _ := sdk.DisableThing("thingID", "token")
	//  fmt.Println(thing)
	DisableThing(id, token string) (Thing, errors.SDKError)

	// ShareThing shares thing with other users.
	//
	// example:
	// req := sdk.UsersRelationRequest{
	//		Relation: "contributor", // available options: "owner", "admin", "editor", "contributor", "guest"
	//  	UserIDs: ["user_id_1", "user_id_2", "user_id_3"]
	// }
	//  err := sdk.ShareThing("thing_id", req, "token")
	//  fmt.Println(err)
	ShareThing(thingID string, req UsersRelationRequest, token string) errors.SDKError

	// UnshareThing unshare a thing with other users.
	//
	// example:
	// req := sdk.UsersRelationRequest{
	//		Relation: "contributor", // available options: "owner", "admin", "editor", "contributor", "guest"
	//  	UserIDs: ["user_id_1", "user_id_2", "user_id_3"]
	// }
	//  err := sdk.UnshareThing("thing_id", req, "token")
	//  fmt.Println(err)
	UnshareThing(thingID string, req UsersRelationRequest, token string) errors.SDKError

	// ListThingUsers all users in a thing.
	//
	// example:
	//	pm := sdk.PageMetadata{
	//		Offset: 0,
	//		Limit:  10,
	//		Permission: "edit", // available Options:  "administrator", "administrator", "delete", edit", "view", "share", "owner", "owner", "admin", "editor", "contributor", "editor", "viewer", "guest", "create"
	//	}
	//  users, _ := sdk.ListThingUsers(pm, "token")
	//  fmt.Println(users)
	ListThingUsers(pm PageMetadata, token string) (UsersPage, errors.SDKError)

	// DeleteThing deletes a thing with the given id.
	//
	// example:
	//  err := sdk.DeleteThing("thingID", "token")
	//  fmt.Println(err)
	DeleteThing(id, token string) errors.SDKError

	// CreateGroup creates new group and returns its id.
	//
	// example:
	//  group := sdk.Group{
	//    Name: "My Group",
	//    Metadata: sdk.Metadata{
	//      "key": "value",
	//    },
	//  }
	//  group, _ := sdk.CreateGroup(group, "token")
	//  fmt.Println(group)
	CreateGroup(group Group, token string) (Group, errors.SDKError)

	// Groups returns page of groups.
	//
	// example:
	//  pm := sdk.PageMetadata{
	//    Offset: 0,
	//    Limit:  10,
	//    Name:   "My Group",
	//  }
	//  groups, _ := sdk.Groups(pm, "token")
	//  fmt.Println(groups)
	Groups(pm PageMetadata, token string) (GroupsPage, errors.SDKError)

	// Parents returns page of users groups.
	//
	// example:
	//  pm := sdk.PageMetadata{
	//    Offset: 0,
	//    Limit:  10,
	//    Name:   "My Group",
	//  }
	//  groups, _ := sdk.Parents("groupID", pm, "token")
	//  fmt.Println(groups)
	Parents(id string, pm PageMetadata, token string) (GroupsPage, errors.SDKError)

	// Children returns page of users groups.
	//
	// example:
	//  pm := sdk.PageMetadata{
	//    Offset: 0,
	//    Limit:  10,
	//    Name:   "My Group",
	//  }
	//  groups, _ := sdk.Children("groupID", pm, "token")
	//  fmt.Println(groups)
	Children(id string, pm PageMetadata, token string) (GroupsPage, errors.SDKError)

	// Group returns users group object by id.
	//
	// example:
	//  group, _ := sdk.Group("groupID", "token")
	//  fmt.Println(group)
	Group(id, token string) (Group, errors.SDKError)

	// GroupPermissions returns user permissions by group ID.
	//
	// example:
	//  group, _ := sdk.Group("groupID", "token")
	//  fmt.Println(group)
	GroupPermissions(id, token string) (Group, errors.SDKError)

	// UpdateGroup updates existing group.
	//
	// example:
	//  group := sdk.Group{
	//    ID:   "groupID",
	//    Name: "My Group",
	//    Metadata: sdk.Metadata{
	//      "key": "value",
	//    },
	//  }
	//  group, _ := sdk.UpdateGroup(group, "token")
	//  fmt.Println(group)
	UpdateGroup(group Group, token string) (Group, errors.SDKError)

	// EnableGroup changes group status to enabled.
	//
	// example:
	//  group, _ := sdk.EnableGroup("groupID", "token")
	//  fmt.Println(group)
	EnableGroup(id, token string) (Group, errors.SDKError)

	// DisableGroup changes group status to disabled - soft delete.
	//
	// example:
	//  group, _ := sdk.DisableGroup("groupID", "token")
	//  fmt.Println(group)
	DisableGroup(id, token string) (Group, errors.SDKError)

	// AddUserToGroup add user to a group.
	//
	// example:
	// req := sdk.UsersRelationRequest{
	//		Relation: "contributor", // available options: "owner", "admin", "editor", "contributor", "guest"
	//  	UserIDs: ["user_id_1", "user_id_2", "user_id_3"]
	// }
	// err := sdk.AddUserToGroup("groupID",req, "token")
	// fmt.Println(err)
	AddUserToGroup(groupID string, req UsersRelationRequest, token string) errors.SDKError

	// RemoveUserFromGroup remove user from a group.
	//
	// example:
	// req := sdk.UsersRelationRequest{
	//		Relation: "contributor", // available options: "owner", "admin", "editor", "contributor", "guest"
	//  	UserIDs: ["user_id_1", "user_id_2", "user_id_3"]
	// }
	// err := sdk.RemoveUserFromGroup("groupID",req, "token")
	// fmt.Println(err)
	RemoveUserFromGroup(groupID string, req UsersRelationRequest, token string) errors.SDKError

	// ListGroupUsers list all users in the group id .
	//
	// example:
	//	pm := sdk.PageMetadata{
	//		Offset: 0,
	//		Limit:  10,
	//		Permission: "edit", // available Options:  "administrator", "administrator", "delete", edit", "view", "share", "owner", "owner", "admin", "editor", "contributor", "editor", "viewer", "guest", "create"
	//	}
	//  groups, _ := sdk.ListGroupUsers(pm, "token")
	//  fmt.Println(groups)
	ListGroupUsers(pm PageMetadata, token string) (UsersPage, errors.SDKError)

	// ListGroupChannels list all channels in the group id .
	//
	// example:
	//	pm := sdk.PageMetadata{
	//		Offset: 0,
	//		Limit:  10,
	//		Permission: "edit", // available Options:  "administrator", "administrator", "delete", edit", "view", "share", "owner", "owner", "admin", "editor", "contributor", "editor", "viewer", "guest", "create"
	//	}
	//  groups, _ := sdk.ListGroupChannels(pm, "token")
	//  fmt.Println(groups)
	ListGroupChannels(groupID string, pm PageMetadata, token string) (ChannelsPage, errors.SDKError)

	// DeleteGroup delete given group id.
	//
	// example:
	//  err := sdk.DeleteGroup("groupID", "token")
	//  fmt.Println(err)
	DeleteGroup(id, token string) errors.SDKError

	// CreateChannel creates new channel and returns its id.
	//
	// example:
	//  channel := sdk.Channel{
	//    Name: "My Channel",
	//    Metadata: sdk.Metadata{
	//      "key": "value",
	//    },
	//  }
	//  channel, _ := sdk.CreateChannel(channel, "token")
	//  fmt.Println(channel)
	CreateChannel(channel Channel, token string) (Channel, errors.SDKError)

	// Channels returns page of channels.
	//
	// example:
	//  pm := sdk.PageMetadata{
	//    Offset: 0,
	//    Limit:  10,
	//    Name:   "My Channel",
	//  }
	//  channels, _ := sdk.Channels(pm, "token")
	//  fmt.Println(channels)
	Channels(pm PageMetadata, token string) (ChannelsPage, errors.SDKError)

	// ChannelsByThing returns page of channels that are connected to specified thing.
	//
	// example:
	//  pm := sdk.PageMetadata{
	//    Offset: 0,
	//    Limit:  10,
	//    Name:   "My Channel",
	//	  Thing:  "thingID",
	//  }
	//  channels, _ := sdk.ChannelsByThing(pm, "token")
	//  fmt.Println(channels)
	ChannelsByThing(pm PageMetadata, token string) (ChannelsPage, errors.SDKError)

	// Channel returns channel data by id.
	//
	// example:
	//  channel, _ := sdk.Channel("channelID", "token")
	//  fmt.Println(channel)
	Channel(id, token string) (Channel, errors.SDKError)

	// ChannelPermissions returns user permissions on the channel ID.
	//
	// example:
	//  channel, _ := sdk.Channel("channelID", "token")
	//  fmt.Println(channel)
	ChannelPermissions(id, token string) (Channel, errors.SDKError)

	// UpdateChannel updates existing channel.
	//
	// example:
	//  channel := sdk.Channel{
	//    ID:   "channelID",
	//    Name: "My Channel",
	//    Metadata: sdk.Metadata{
	//      "key": "value",
	//    },
	//  }
	//  channel, _ := sdk.UpdateChannel(channel, "token")
	//  fmt.Println(channel)
	UpdateChannel(channel Channel, token string) (Channel, errors.SDKError)

	// EnableChannel changes channel status to enabled.
	//
	// example:
	//  channel, _ := sdk.EnableChannel("channelID", "token")
	//  fmt.Println(channel)
	EnableChannel(id, token string) (Channel, errors.SDKError)

	// DisableChannel changes channel status to disabled - soft delete.
	//
	// example:
	//  channel, _ := sdk.DisableChannel("channelID", "token")
	//  fmt.Println(channel)
	DisableChannel(id, token string) (Channel, errors.SDKError)

	// AddUserToChannel add user to a channel.
	//
	// example:
	// req := sdk.UsersRelationRequest{
	//		Relation: "contributor", // available options: "owner", "admin", "editor", "contributor", "guest"
	// 		UserIDs: ["user_id_1", "user_id_2", "user_id_3"]
	// }
	// err := sdk.AddUserToChannel("channel_id", req, "token")
	// fmt.Println(err)
	AddUserToChannel(channelID string, req UsersRelationRequest, token string) errors.SDKError

	// RemoveUserFromChannel remove user from a group.
	//
	// example:
	// req := sdk.UsersRelationRequest{
	//		Relation: "contributor", // available options: "owner", "admin", "editor", "contributor", "guest"
	//  	UserIDs: ["user_id_1", "user_id_2", "user_id_3"]
	// }
	// err := sdk.RemoveUserFromChannel("channel_id", req, "token")
	// fmt.Println(err)
	RemoveUserFromChannel(channelID string, req UsersRelationRequest, token string) errors.SDKError

	// ListChannelUsers list all users in a channel .
	//
	// example:
	//	pm := sdk.PageMetadata{
	//		Offset: 0,
	//		Limit:  10,
	//		Permission: "edit",  // available Options:  "administrator", "administrator", "delete", edit", "view", "share", "owner", "owner", "admin", "editor", "contributor", "editor", "viewer", "guest", "create"
	//	}
	//  users, _ := sdk.ListChannelUsers(pm, "token")
	//  fmt.Println(users)
	ListChannelUsers(pm PageMetadata, token string) (UsersPage, errors.SDKError)

	// AddUserGroupToChannel add user group to a channel.
	//
	// example:
	// req := sdk.UserGroupsRequest{
	//  	GroupsIDs: ["group_id_1", "group_id_2", "group_id_3"]
	// }
	// err := sdk.AddUserGroupToChannel("channel_id",req, "token")
	// fmt.Println(err)
	AddUserGroupToChannel(channelID string, req UserGroupsRequest, token string) errors.SDKError

	// RemoveUserGroupFromChannel remove user group from a channel.
	//
	// example:
	// req := sdk.UserGroupsRequest{
	//  	GroupsIDs: ["group_id_1", "group_id_2", "group_id_3"]
	// }
	// err := sdk.RemoveUserGroupFromChannel("channel_id",req, "token")
	// fmt.Println(err)
	RemoveUserGroupFromChannel(channelID string, req UserGroupsRequest, token string) errors.SDKError

	// ListChannelUserGroups list all user groups in a channel.
	//
	// example:
	//	pm := sdk.PageMetadata{
	//		Offset: 0,
	//		Limit:  10,
	// 	    Channel: "channel_id",
	//		Permission: "view",
	//	}
	//  groups, _ := sdk.ListChannelUserGroups(pm, "token")
	//  fmt.Println(groups)
	ListChannelUserGroups(pm PageMetadata, token string) (GroupsPage, errors.SDKError)

	// DeleteChannel delete given group id.
	//
	// example:
	//  err := sdk.DeleteChannel("channelID", "token")
	//  fmt.Println(err)
	DeleteChannel(id, token string) errors.SDKError

	// Connect bulk connects things to channels specified by id.
	//
	// example:
	//  conns := sdk.Connection{
	//    ChannelID: "channel_id_1",
	//    ThingID:   "thing_id_1",
	//  }
	//  err := sdk.Connect(conns, "token")
	//  fmt.Println(err)
	Connect(conns Connection, token string) errors.SDKError

	// Disconnect
	//
	// example:
	//  conns := sdk.Connection{
	//    ChannelID: "channel_id_1",
	//    ThingID:   "thing_id_1",
	//  }
	//  err := sdk.Disconnect(conns, "token")
	//  fmt.Println(err)
	Disconnect(connIDs Connection, token string) errors.SDKError

	// ConnectThing connects thing to specified channel by id.
	//
	// The `ConnectThing` method calls the `CreateThingPolicy` method under the hood.
	//
	// example:
	//  err := sdk.ConnectThing("thingID", "channelID", "token")
	//  fmt.Println(err)
	ConnectThing(thingID, chanID, token string) errors.SDKError

	// DisconnectThing disconnect thing from specified channel by id.
	//
	// The `DisconnectThing` method calls the `DeleteThingPolicy` method under the hood.
	//
	// example:
	//  err := sdk.DisconnectThing("thingID", "channelID", "token")
	//  fmt.Println(err)
	DisconnectThing(thingID, chanID, token string) errors.SDKError

	// SendMessage send message to specified channel.
	//
	// example:
	//  msg := '[{"bn":"some-base-name:","bt":1.276020076001e+09, "bu":"A","bver":5, "n":"voltage","u":"V","v":120.1}, {"n":"current","t":-5,"v":1.2}, {"n":"current","t":-4,"v":1.3}]'
	//  err := sdk.SendMessage("channelID", msg, "thingSecret")
	//  fmt.Println(err)
	SendMessage(chanID, msg, key string) errors.SDKError

	// ReadMessages read messages of specified channel.
	//
	// example:
	//  pm := sdk.MessagePageMetadata{
	//    Offset: 0,
	//    Limit:  10,
	//  }
	//  msgs, _ := sdk.ReadMessages(pm,"channelID", "token")
	//  fmt.Println(msgs)
	ReadMessages(pm MessagePageMetadata, chanID, token string) (MessagesPage, errors.SDKError)

	// SetContentType sets message content type.
	//
	// example:
	//  err := sdk.SetContentType("application/json")
	//  fmt.Println(err)
	SetContentType(ct ContentType) errors.SDKError

	// Health returns service health check.
	//
	// example:
	//  health, _ := sdk.Health("service")
	//  fmt.Println(health)
	Health(service string) (HealthInfo, errors.SDKError)

	// AddBootstrap add bootstrap configuration
	//
	// example:
	//  cfg := sdk.BootstrapConfig{
	//    ThingID: "thingID",
	//    Name: "bootstrap",
	//    ExternalID: "externalID",
	//    ExternalKey: "externalKey",
	//    Channels: []string{"channel1", "channel2"},
	//  }
	//  id, _ := sdk.AddBootstrap(cfg, "token")
	//  fmt.Println(id)
	AddBootstrap(cfg BootstrapConfig, token string) (string, errors.SDKError)

	// View returns Thing Config with given ID belonging to the user identified by the given token.
	//
	// example:
	//  bootstrap, _ := sdk.ViewBootstrap("id", "token")
	//  fmt.Println(bootstrap)
	ViewBootstrap(id, token string) (BootstrapConfig, errors.SDKError)

	// Update updates editable fields of the provided Config.
	//
	// example:
	//  cfg := sdk.BootstrapConfig{
	//    ThingID: "thingID",
	//    Name: "bootstrap",
	//    ExternalID: "externalID",
	//    ExternalKey: "externalKey",
	//    Channels: []string{"channel1", "channel2"},
	//  }
	//  err := sdk.UpdateBootstrap(cfg, "token")
	//  fmt.Println(err)
	UpdateBootstrap(cfg BootstrapConfig, token string) errors.SDKError

	// Update bootstrap config certificates.
	//
	// example:
	//  err := sdk.UpdateBootstrapCerts("id", "clientCert", "clientKey", "ca", "token")
	//  fmt.Println(err)
	UpdateBootstrapCerts(id string, clientCert, clientKey, ca string, token string) (BootstrapConfig, errors.SDKError)

	// UpdateBootstrapConnection updates connections performs update of the channel list corresponding Thing is connected to.
	//
	// example:
	//  err := sdk.UpdateBootstrapConnection("id", []string{"channel1", "channel2"}, "token")
	//  fmt.Println(err)
	UpdateBootstrapConnection(id string, channels []string, token string) errors.SDKError

	// Remove removes Config with specified token that belongs to the user identified by the given token.
	//
	// example:
	//  err := sdk.RemoveBootstrap("id", "token")
	//  fmt.Println(err)
	RemoveBootstrap(id, token string) errors.SDKError

	// Bootstrap returns Config to the Thing with provided external ID using external key.
	//
	// example:
	//  bootstrap, _ := sdk.Bootstrap("externalID", "externalKey")
	//  fmt.Println(bootstrap)
	Bootstrap(externalID, externalKey string) (BootstrapConfig, errors.SDKError)

	// BootstrapSecure retrieves a configuration with given external ID and encrypted external key.
	//
	// example:
	//  bootstrap, _ := sdk.BootstrapSecure("externalID", "externalKey", "cryptoKey")
	//  fmt.Println(bootstrap)
	BootstrapSecure(externalID, externalKey, cryptoKey string) (BootstrapConfig, errors.SDKError)

	// Bootstraps retrieves a list of managed configs.
	//
	// example:
	//  pm := sdk.PageMetadata{
	//    Offset: 0,
	//    Limit:  10,
	//  }
	//  bootstraps, _ := sdk.Bootstraps(pm, "token")
	//  fmt.Println(bootstraps)
	Bootstraps(pm PageMetadata, token string) (BootstrapPage, errors.SDKError)

	// Whitelist updates Thing state Config with given ID belonging to the user identified by the given token.
	//
	// example:
	//  err := sdk.Whitelist("thingID", 1, "token")
	//  fmt.Println(err)
	Whitelist(thingID string, state int, token string) errors.SDKError

	// IssueCert issues a certificate for a thing required for mTLS.
	//
	// example:
	//  cert, _ := sdk.IssueCert("thingID", "24h", "token")
	//  fmt.Println(cert)
	IssueCert(thingID, validity, token string) (Cert, errors.SDKError)

	// ViewCert returns a certificate given certificate ID
	//
	// example:
	//  cert, _ := sdk.ViewCert("certID", "token")
	//  fmt.Println(cert)
	ViewCert(certID, token string) (Cert, errors.SDKError)

	// ViewCertByThing retrieves a list of certificates' serial IDs for a given thing ID.
	//
	// example:
	//  cserial, _ := sdk.ViewCertByThing("thingID", "token")
	//  fmt.Println(cserial)
	ViewCertByThing(thingID, token string) (CertSerials, errors.SDKError)

	// RevokeCert revokes certificate for thing with thingID
	//
	// example:
	//  tm, _ := sdk.RevokeCert("thingID", "token")
	//  fmt.Println(tm)
	RevokeCert(thingID, token string) (time.Time, errors.SDKError)

	// CreateSubscription creates a new subscription
	//
	// example:
	//  subscription, _ := sdk.CreateSubscription("topic", "contact", "token")
	//  fmt.Println(subscription)
	CreateSubscription(topic, contact, token string) (string, errors.SDKError)

	// ListSubscriptions list subscriptions given list parameters.
	//
	// example:
	//  pm := sdk.PageMetadata{
	//    Offset: 0,
	//    Limit:  10,
	//  }
	//  subscriptions, _ := sdk.ListSubscriptions(pm, "token")
	//  fmt.Println(subscriptions)
	ListSubscriptions(pm PageMetadata, token string) (SubscriptionPage, errors.SDKError)

	// ViewSubscription retrieves a subscription with the provided id.
	//
	// example:
	//  subscription, _ := sdk.ViewSubscription("id", "token")
	//  fmt.Println(subscription)
	ViewSubscription(id, token string) (Subscription, errors.SDKError)

	// DeleteSubscription removes a subscription with the provided id.
	//
	// example:
	//  err := sdk.DeleteSubscription("id", "token")
	//  fmt.Println(err)
	DeleteSubscription(id, token string) errors.SDKError

	// CreateDomain creates new domain and returns its details.
	//
	// example:
	//  domain := sdk.Domain{
	//    Name: "My Domain",
	//    Metadata: sdk.Metadata{
	//      "key": "value",
	//    },
	//  }
	//  domain, _ := sdk.CreateDomain(group, "token")
	//  fmt.Println(domain)
	CreateDomain(d Domain, token string) (Domain, errors.SDKError)

	// Domain retrieve domain information of given domain ID .
	//
	// example:
	//  domain, _ := sdk.Domain("domainID", "token")
	//  fmt.Println(domain)
	Domain(domainID, token string) (Domain, errors.SDKError)

	// DomainPermissions retrieve user permissions on the given domain ID .
	//
	// example:
	//  permissions, _ := sdk.DomainPermissions("domainID", "token")
	//  fmt.Println(permissions)
	DomainPermissions(domainID, token string) (Domain, errors.SDKError)

	// UpdateDomain updates details of the given domain ID.
	//
	// example:
	//  domain := sdk.Domain{
	//    ID : "domainID"
	//    Name: "New Domain Name",
	//    Metadata: sdk.Metadata{
	//      "key": "value",
	//    },
	//  }
	//  domain, _ := sdk.UpdateDomain(domain, "token")
	//  fmt.Println(domain)
	UpdateDomain(d Domain, token string) (Domain, errors.SDKError)

	// Domains returns list of domain for the given filters.
	//
	// example:
	//  pm := sdk.PageMetadata{
	//    Offset: 0,
	//    Limit:  10,
	//    Name:   "My Domain",
	//    Permission : "view"
	//  }
	//  domains, _ := sdk.Domains(pm, "token")
	//  fmt.Println(domains)
	Domains(pm PageMetadata, token string) (DomainsPage, errors.SDKError)

	// ListDomainUsers returns list of users for the given domain ID and filters.
	//
	// example:
	//  pm := sdk.PageMetadata{
	//    Offset: 0,
	//    Limit:  10,
	//	  Domain: "domainID",
	//    Permission : "view"
	//  }
	//  users, _ := sdk.ListDomainUsers(pm, "token")
	//  fmt.Println(users)
	ListDomainUsers(pm PageMetadata, token string) (UsersPage, errors.SDKError)

	// ListUserDomains returns list of domains for the given user ID and filters.
	//
	// example:
	//  pm := sdk.PageMetadata{
	//    Offset: 0,
	//    Limit:  10,
	//	  User: "userID",
	//    Permission : "view"
	//  }
	//  domains, _ := sdk.ListUserDomains(pm, "token")
	//  fmt.Println(domains)
	ListUserDomains(pm PageMetadata, token string) (DomainsPage, errors.SDKError)

	// EnableDomain changes the status of the domain to enabled.
	//
	// example:
	//  err := sdk.EnableDomain("domainID", "token")
	//  fmt.Println(err)
	EnableDomain(domainID, token string) errors.SDKError

	// DisableDomain changes the status of the domain to disabled.
	//
	// example:
	//  err := sdk.DisableDomain("domainID", "token")
	//  fmt.Println(err)
	DisableDomain(domainID, token string) errors.SDKError

	// AddUserToDomain adds a user to a domain.
	//
	// example:
	// req := sdk.UsersRelationRequest{
	//		Relation: "contributor", // available options: "owner", "admin", "editor", "contributor",  "member", "guest"
	//  	UserIDs: ["user_id_1", "user_id_2", "user_id_3"]
	// }
	// err := sdk.AddUserToDomain("domainID", req, "token")
	// fmt.Println(err)
	AddUserToDomain(domainID string, req UsersRelationRequest, token string) errors.SDKError

	// RemoveUserFromDomain removes a user from a domain.
	//
	// example:
	// err := sdk.RemoveUserFromDomain("domainID", "userID", "token")
	// fmt.Println(err)
	RemoveUserFromDomain(domainID, userID, token string) errors.SDKError

	// SendInvitation sends an invitation to the email address associated with the given user.
	//
	// For example:
	//  invitation := sdk.Invitation{
	//    DomainID: "domainID",
	//    UserID:   "userID",
	//    Relation: "contributor", // available options: "owner", "admin", "editor", "contributor", "guest"
	//  }
	//  err := sdk.SendInvitation(invitation, "token")
	//  fmt.Println(err)
	SendInvitation(invitation Invitation, token string) (err error)

	// Invitation returns an invitation.
	//
	// For example:
	//  invitation, _ := sdk.Invitation("userID", "domainID", "token")
	//  fmt.Println(invitation)
	Invitation(userID, domainID, token string) (invitation Invitation, err error)

	// Invitations returns a list of invitations.
	//
	// For example:
	//  invitations, _ := sdk.Invitations(PageMetadata{Offset: 0, Limit: 10, Domain: "domainID"}, "token")
	//  fmt.Println(invitations)
	Invitations(pm PageMetadata, token string) (invitations InvitationPage, err error)

	// AcceptInvitation accepts an invitation by adding the user to the domain that they were invited to.
	//
	// For example:
	//  err := sdk.AcceptInvitation("domainID", "token")
	//  fmt.Println(err)
	AcceptInvitation(domainID, token string) (err error)

	// DeleteInvitation deletes an invitation.
	//
	// For example:
	//  err := sdk.DeleteInvitation("userID", "domainID", "token")
	//  fmt.Println(err)
	DeleteInvitation(userID, domainID, token string) (err error)

	// Journal returns a list of journal logs.
	//
	// For example:
	//  journals, _ := sdk.Journal("thing", "thingID", PageMetadata{Offset: 0, Limit: 10, Operation: "users.create"}, "token")
	//  fmt.Println(journals)
	Journal(entityType, entityID string, pm PageMetadata, token string) (journal JournalsPage, err error)
}

type mgSDK struct {
	bootstrapURL   string
	certsURL       string
	httpAdapterURL string
	readerURL      string
	thingsURL      string
	usersURL       string
	domainsURL     string
	invitationsURL string
	journalURL     string
	HostURL        string

	msgContentType ContentType
	client         *http.Client
	curlFlag       bool
}

// Config contains sdk configuration parameters.
type Config struct {
	BootstrapURL   string
	CertsURL       string
	HTTPAdapterURL string
	ReaderURL      string
	ThingsURL      string
	UsersURL       string
	DomainsURL     string
	InvitationsURL string
	JournalURL     string
	HostURL        string

	MsgContentType  ContentType
	TLSVerification bool
	CurlFlag        bool
}

// NewSDK returns new magistrala SDK instance.
func NewSDK(conf Config) SDK {
	return &mgSDK{
		bootstrapURL:   conf.BootstrapURL,
		certsURL:       conf.CertsURL,
		httpAdapterURL: conf.HTTPAdapterURL,
		readerURL:      conf.ReaderURL,
		thingsURL:      conf.ThingsURL,
		usersURL:       conf.UsersURL,
		domainsURL:     conf.DomainsURL,
		invitationsURL: conf.InvitationsURL,
		journalURL:     conf.JournalURL,
		HostURL:        conf.HostURL,

		msgContentType: conf.MsgContentType,
		client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: !conf.TLSVerification,
				},
			},
		},
		curlFlag: conf.CurlFlag,
	}
}

// processRequest creates and send a new HTTP request, and checks for errors in the HTTP response.
// It then returns the response headers, the response body, and the associated error(s) (if any).
func (sdk mgSDK) processRequest(method, reqUrl, token string, data []byte, headers map[string]string, expectedRespCodes ...int) (http.Header, []byte, errors.SDKError) {
	req, err := http.NewRequest(method, reqUrl, bytes.NewReader(data))
	if err != nil {
		return make(http.Header), []byte{}, errors.NewSDKError(err)
	}

	// Sets a default value for the Content-Type.
	// Overridden if Content-Type is passed in the headers arguments.
	req.Header.Add("Content-Type", string(CTJSON))

	for key, value := range headers {
		req.Header.Add(key, value)
	}

	if token != "" {
		if !strings.Contains(token, ThingPrefix) {
			token = BearerPrefix + token
		}
		req.Header.Set("Authorization", token)
	}

	if sdk.curlFlag {
		curlCommand, err := http2curl.GetCurlCommand(req)
		if err != nil {
			return nil, nil, errors.NewSDKError(err)
		}
		log.Println(curlCommand.String())
	}

	resp, err := sdk.client.Do(req)
	if err != nil {
		return make(http.Header), []byte{}, errors.NewSDKError(err)
	}
	defer resp.Body.Close()

	sdkerr := errors.CheckError(resp, expectedRespCodes...)
	if sdkerr != nil {
		return make(http.Header), []byte{}, sdkerr
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return make(http.Header), []byte{}, errors.NewSDKError(err)
	}

	return resp.Header, body, nil
}

func (sdk mgSDK) withQueryParams(baseURL, endpoint string, pm PageMetadata) (string, error) {
	q, err := pm.query()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/%s?%s", baseURL, endpoint, q), nil
}

func (pm PageMetadata) query() (string, error) {
	q := url.Values{}
	if pm.Offset != 0 {
		q.Add("offset", strconv.FormatUint(pm.Offset, 10))
	}
	if pm.Limit != 0 {
		q.Add("limit", strconv.FormatUint(pm.Limit, 10))
	}
	if pm.Total != 0 {
		q.Add("total", strconv.FormatUint(pm.Total, 10))
	}
	if pm.Order != "" {
		q.Add("order", pm.Order)
	}
	if pm.Direction != "" {
		q.Add("dir", pm.Direction)
	}
	if pm.Level != 0 {
		q.Add("level", strconv.FormatUint(pm.Level, 10))
	}
	if pm.Identity != "" {
		q.Add("identity", pm.Identity)
	}
	if pm.Name != "" {
		q.Add("name", pm.Name)
	}
	if pm.ID != "" {
		q.Add("id", pm.ID)
	}
	if pm.Type != "" {
		q.Add("type", pm.Type)
	}
	if pm.Visibility != "" {
		q.Add("visibility", pm.Visibility)
	}
	if pm.Status != "" {
		q.Add("status", pm.Status)
	}
	if pm.Metadata != nil {
		md, err := json.Marshal(pm.Metadata)
		if err != nil {
			return "", errors.NewSDKError(err)
		}
		q.Add("metadata", string(md))
	}
	if pm.Action != "" {
		q.Add("action", pm.Action)
	}
	if pm.Subject != "" {
		q.Add("subject", pm.Subject)
	}
	if pm.Object != "" {
		q.Add("object", pm.Object)
	}
	if pm.Tag != "" {
		q.Add("tag", pm.Tag)
	}
	if pm.Owner != "" {
		q.Add("owner", pm.Owner)
	}
	if pm.SharedBy != "" {
		q.Add("shared_by", pm.SharedBy)
	}
	if pm.Topic != "" {
		q.Add("topic", pm.Topic)
	}
	if pm.Contact != "" {
		q.Add("contact", pm.Contact)
	}
	if pm.State != "" {
		q.Add("state", pm.State)
	}
	if pm.Permission != "" {
		q.Add("permission", pm.Permission)
	}
	if pm.ListPermissions != "" {
		q.Add("list_perms", pm.ListPermissions)
	}
	if pm.InvitedBy != "" {
		q.Add("invited_by", pm.InvitedBy)
	}
	if pm.UserID != "" {
		q.Add("user_id", pm.UserID)
	}
	if pm.DomainID != "" {
		q.Add("domain_id", pm.DomainID)
	}
	if pm.Relation != "" {
		q.Add("relation", pm.Relation)
	}
	if pm.User != "" {
		q.Add("user", pm.User)
	}
	if pm.Channel != "" {
		q.Add("channel", pm.Channel)
	}
	if pm.Group != "" {
		q.Add("group", pm.Group)
	}
	if pm.Thing != "" {
		q.Add("thing", pm.Thing)
	}
	if pm.Domain != "" {
		q.Add("domain", pm.Domain)
	}

	if pm.Operation != "" {
		q.Add("operation", pm.Operation)
	}
	if pm.From != 0 {
		q.Add("from", strconv.FormatInt(pm.From, 10))
	}
	if pm.To != 0 {
		q.Add("to", strconv.FormatInt(pm.To, 10))
	}
	q.Add("with_attributes", strconv.FormatBool(pm.WithAttributes))
	q.Add("with_metadata", strconv.FormatBool(pm.WithMetadata))

	return q.Encode(), nil
}
