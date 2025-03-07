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

	"github.com/absmach/supermq/pkg/errors"
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

	ClientPrefix = "Client "
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
	Email           string   `json:"email,omitempty"`
	Username        string   `json:"username,omitempty"`
	LastName        string   `json:"last_name,omitempty"`
	FirstName       string   `json:"first_name,omitempty"`
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
	Operation       string   `json:"operation,omitempty"`
	From            int64    `json:"from,omitempty"`
	To              int64    `json:"to,omitempty"`
	WithMetadata    bool     `json:"with_metadata,omitempty"`
	WithAttributes  bool     `json:"with_attributes,omitempty"`
	ID              string   `json:"id,omitempty"`
	Tree            bool     `json:"tree,omitempty"`
	StartLevel      int64    `json:"start_level,omitempty"`
	EndLevel        int64    `json:"end_level,omitempty"`
}

type Role struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	EntityID        string    `json:"entity_id"`
	CreatedBy       string    `json:"created_by"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedBy       string    `json:"updated_by"`
	UpdatedAt       time.Time `json:"updated_at"`
	OptionalActions []string  `json:"optional_actions,omitempty"`
	OptionalMembers []string  `json:"optional_members,omitempty"`
}

type RolesPage struct {
	Total  uint64 `json:"total"`
	Offset uint64 `json:"offset"`
	Limit  uint64 `json:"limit"`
	Roles  []Role `json:"roles"`
}

// Credentials represent client credentials: it contains
// "username" which can be a username, generated name;
// and "secret" which can be a password or access token.
type Credentials struct {
	Username string `json:"username,omitempty"` // username or generated login ID
	Secret   string `json:"secret,omitempty"`   // password or token
}

// SDK contains SuperMQ API.
//
//go:generate mockery --name SDK --output=./mocks --filename sdk.go --quiet --note "Copyright (c) Abstract Machines"
type SDK interface {
	// CreateUser registers supermq user.
	//
	// example:
	//  user := sdk.User{
	//    Name:	 "John Doe",
	// 	  Email: "john.doe@example",
	//    Credentials: sdk.Credentials{
	//      Username: "john.doe",
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

	// UpdateUserEmail updates the user's email
	//
	// example:
	//  user := sdk.User{
	//    ID:   "userID",
	//    Credentials: sdk.Credentials{
	//      Email: "john.doe@example",
	//    },
	//  }
	//  user, _ := sdk.UpdateUserEmail(user, "token")
	//  fmt.Println(user)
	UpdateUserEmail(user User, token string) (User, errors.SDKError)

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

	// UpdateUsername updates the user's Username.
	//
	// example:
	//  user := sdk.User{
	//    ID:   "userID",
	//    Credentials: sdk.Credentials{
	//	  	Username: "john.doe",
	//		},
	//  }
	//  user, _ := sdk.UpdateUsername(user, "token")
	//  fmt.Println(user)
	UpdateUsername(user User, token string) (User, errors.SDKError)

	// UpdateProfilePicture updates the user's profile picture.
	//
	// example:
	//  user := sdk.User{
	//    ID:            "userID",
	//    ProfilePicture: "https://cloudstorage.example.com/bucket-name/user-images/profile-picture.jpg",
	//  }
	//  user, _ := sdk.UpdateProfilePicture(user, "token")
	//  fmt.Println(user)
	UpdateProfilePicture(user User, token string) (User, errors.SDKError)

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
	//      Identity: "email"/"username",
	//      Secret:   "12345678",
	//  }
	//  token, _ := sdk.CreateToken(lt)
	//  fmt.Println(token)
	CreateToken(lt Login) (Token, errors.SDKError)

	// RefreshToken receives credentials and returns user token.
	//
	// example:
	//  token, _ := sdk.RefreshToken("refresh_token")
	//  fmt.Println(token)
	RefreshToken(token string) (Token, errors.SDKError)

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

	// CreateClient registers new client and returns its id.
	//
	// example:
	//  client := sdk.Client{
	//    Name: "My Client",
	//    Metadata: sdk.Metadata{"domain_1"
	//      "key": "value",
	//    },
	//  }
	//  client, _ := sdk.CreateClient(client, "domainID", "token")
	//  fmt.Println(client)
	CreateClient(client Client, domainID, token string) (Client, errors.SDKError)

	// CreateClients registers new clients and returns their ids.
	//
	// example:
	//  clients := []sdk.Client{
	//    {
	//      Name: "My Client 1",
	//      Metadata: sdk.Metadata{
	//        "key": "value",
	//      },
	//    },
	//    {
	//      Name: "My Client 2",
	//      Metadata: sdk.Metadata{
	//        "key": "value",
	//      },
	//    },
	//  }
	//  clients, _ := sdk.CreateClients(clients, "domainID", "token")
	//  fmt.Println(clients)
	CreateClients(client []Client, domainID, token string) ([]Client, errors.SDKError)

	// Filters clients and returns a page result.
	//
	// example:
	//  pm := sdk.PageMetadata{
	//    Offset: 0,
	//    Limit:  10,
	//    Name:   "My Client",
	//  }
	//  clients, _ := sdk.Clients(pm, "domainID", "token")
	//  fmt.Println(clients)
	Clients(pm PageMetadata, domainID, token string) (ClientsPage, errors.SDKError)

	// Client returns client object by id.
	//
	// example:
	//  client, _ := sdk.Client("clientID", "domainID", "token")
	//  fmt.Println(client)
	Client(id, domainID, token string) (Client, errors.SDKError)

	// UpdateClient updates existing client.
	//
	// example:
	//  client := sdk.Client{
	//    ID:   "clientID",
	//    Name: "My Client",
	//    Metadata: sdk.Metadata{
	//      "key": "value",
	//    },
	//  }
	//  client, _ := sdk.UpdateClient(client, "domainID", "token")
	//  fmt.Println(client)
	UpdateClient(client Client, domainID, token string) (Client, errors.SDKError)

	// UpdateClientTags updates the client's tags.
	//
	// example:
	//  client := sdk.Client{
	//    ID:   "clientID",
	//    Tags: []string{"tag1", "tag2"},
	//  }
	//  client, _ := sdk.UpdateClientTags(client, "domainID", "token")
	//  fmt.Println(client)
	UpdateClientTags(client Client, domainID, token string) (Client, errors.SDKError)

	// UpdateClientSecret updates the client's secret
	//
	// example:
	//  client, err := sdk.UpdateClientSecret("clientID", "newSecret", "domainID," "token")
	//  fmt.Println(client)
	UpdateClientSecret(id, secret, domainID, token string) (Client, errors.SDKError)

	// EnableClient changes client status to enabled.
	//
	// example:
	//  client, _ := sdk.EnableClient("clientID", "domainID", "token")
	//  fmt.Println(client)
	EnableClient(id, domainID, token string) (Client, errors.SDKError)

	// DisableClient changes client status to disabled - soft delete.
	//
	// example:
	//  client, _ := sdk.DisableClient("clientID", "domainID", "token")
	//  fmt.Println(client)
	DisableClient(id, domainID, token string) (Client, errors.SDKError)

	// DeleteClient deletes a client with the given id.
	//
	// example:
	//  err := sdk.DeleteClient("clientID", "domainID", "token")
	//  fmt.Println(err)
	DeleteClient(id, domainID, token string) errors.SDKError

	// SetClientParent sets the parent group of a client.
	//
	// example:
	//  err := sdk.SetClientParent("clientID", "domainID", "groupID", "token")
	//  fmt.Println(err)
	SetClientParent(id, domainID, groupID, token string) errors.SDKError

	// RemoveClientParent removes the parent group of a client.
	//
	// example:
	//  err := sdk.RemoveClientParent("clientID", "domainID", "groupID", "token")
	//  fmt.Println(err)
	RemoveClientParent(id, domainID, groupID, token string) errors.SDKError

	// CreateClientRole creates new client role and returns its id.
	//
	// example:
	//  rq := sdk.RoleReq{
	//    RoleName: "My Role",
	//    OptionalActions: []string{"read", "update"},
	//    OptionalMembers: []string{"member_id_1", "member_id_2"},
	//  }
	//  role, _ := sdk.CreateClientRole("clientID", "domainID", rq, "token")
	//  fmt.Println(role)
	CreateClientRole(id, domainID string, rq RoleReq, token string) (Role, errors.SDKError)

	// ClientRoles returns client roles.
	//
	// example:
	// pm := sdk.PageMetadata{
	//   Offset: 0,
	//   Limit:  10,
	// }
	//  roles, _ := sdk.ClientRoles("clientID", "domainID", pm, "token")
	//  fmt.Println(roles)
	ClientRoles(id, domainID string, pm PageMetadata, token string) (RolesPage, errors.SDKError)

	// ClientRole returns client role object by roleID.
	//
	// example:
	//  role, _ := sdk.ClientRole("clientID", "roleID", "domainID", "token")
	//  fmt.Println(role)
	ClientRole(id, roleID, domainID, token string) (Role, errors.SDKError)

	// UpdateClientRole updates existing client role name.
	//
	// example:
	//  role, _ := sdk.UpdateClientRole{"clientID", "roleID", "newName", "domainID", "token"}
	//  fmr.Println(role)
	UpdateClientRole(id, roleID, newName, domainID string, token string) (Role, errors.SDKError)

	// DeleteClientRole deletes a client role with the given clientID and  roleID.
	//
	// example:
	//  err := sdk.DeleteClientRole("clientID", "roleID", "domainID", "token")
	//  fmt.Println(err)
	DeleteClientRole(id, roleID, domainID, token string) errors.SDKError

	// AddClientRoleActions adds actions to a client role.
	//
	// example:
	//  actions := []string{"read", "update"}
	//  actions, _ := sdk.AddClientRoleActions("clientID", "roleID", "domainID", actions, "token")
	//  fmt.Println(actions)
	AddClientRoleActions(id, roleID, domainID string, actions []string, token string) ([]string, errors.SDKError)

	// ClientRoleActions returns client role actions by roleID.
	//
	// example:
	//  actions, _ := sdk.ClientRoleActions("clientID", "roleID", "domainID", "token")
	//  fmt.Println(actions)
	ClientRoleActions(id, roleID, domainID string, token string) ([]string, errors.SDKError)

	// RemoveClientRoleActions removes actions from a client role.
	//
	// example:
	//  actions := []string{"read", "update"}
	//  err := sdk.RemoveClientRoleActions("clientID", "roleID", "domainID", actions, "token")
	//  fmt.Println(err)
	RemoveClientRoleActions(id, roleID, domainID string, actions []string, token string) errors.SDKError

	// RemoveAllClientRoleActions removes all actions from a client role.
	//
	// example:
	//  err := sdk.RemoveAllClientRoleActions("clientID", "roleID", "domainID", "token")
	//  fmt.Println(err)
	RemoveAllClientRoleActions(id, roleID, domainID, token string) errors.SDKError

	// AddClientRoleMembers adds members to a client role.
	//
	// example:
	//  members := []string{"member_id_1", "member_id_2"}
	//  members, _ := sdk.AddClientRoleMembers("clientID", "roleID", "domainID", members, "token")
	//  fmt.Println(members)
	AddClientRoleMembers(id, roleID, domainID string, members []string, token string) ([]string, errors.SDKError)

	// ClientRoleMembers returns client role members by roleID.
	//
	// example:
	// pm := sdk.PageMetadata{
	//   Offset: 0,
	//  Limit:  10,
	// }
	//  members, _ := sdk.ClientRoleMembers("clientID", "roleID", "domainID", pm,"token")
	//  fmt.Println(members)
	ClientRoleMembers(id, roleID, domainID string, pm PageMetadata, token string) (RoleMembersPage, errors.SDKError)

	// RemoveClientRoleMembers removes members from a client role.
	//
	// example:
	//  members := []string{"member_id_1", "member_id_2"}
	//  err := sdk.RemoveClientRoleMembers("clientID", "roleID", "domainID", members, "token")
	//  fmt.Println(err)
	RemoveClientRoleMembers(id, roleID, domainID string, members []string, token string) errors.SDKError

	// RemoveAllClientRoleMembers removes all members from a client role.
	//
	// example:
	//  err := sdk.RemoveAllClientRoleMembers("clientID", "roleID", "domainID", "token")
	//  fmt.Println(err)
	RemoveAllClientRoleMembers(id, roleID, domainID, token string) errors.SDKError

	// AvailableClientRoleActions returns available actions for a client role.
	//
	// example:
	//  actions, _ := sdk.AvailableClientRoleActions("domainID", "token")
	//  fmt.Println(actions)
	AvailableClientRoleActions(domainID, token string) ([]string, errors.SDKError)

	// ListClientMembers list all members from all roles in a client .
	//
	// example:
	//	pm := sdk.PageMetadata{
	//		Offset: 0,
	//		Limit:  10,
	//	}
	//  members, _ := sdk.ListClientMembers("client_id","domainID", pm, "token")
	//  fmt.Println(members)
	ListClientMembers(clientID, domainID string, pm PageMetadata, token string) (EntityMembersPage, errors.SDKError)

	// CreateGroup creates new group and returns its id.
	//
	// example:
	//  group := sdk.Group{
	//    Name: "My Group",
	//    Metadata: sdk.Metadata{
	//      "key": "value",
	//    },
	//  }
	//  group, _ := sdk.CreateGroup(group, "domainID", "token")
	//  fmt.Println(group)
	CreateGroup(group Group, domainID, token string) (Group, errors.SDKError)

	// Groups returns page of groups.
	//
	// example:
	//  pm := sdk.PageMetadata{
	//    Offset: 0,
	//    Limit:  10,
	//    Name:   "My Group",
	//  }
	//  groups, _ := sdk.Groups(pm, "domainID", "token")
	//  fmt.Println(groups)
	Groups(pm PageMetadata, domainID, token string) (GroupsPage, errors.SDKError)

	// Group returns users group object by id.
	//
	// example:
	//  group, _ := sdk.Group("groupID", "domainID", "token")
	//  fmt.Println(group)
	Group(id, domainID, token string) (Group, errors.SDKError)

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
	//  group, _ := sdk.UpdateGroup(group, "domainID", "token")
	//  fmt.Println(group)
	UpdateGroup(group Group, domainID, token string) (Group, errors.SDKError)

	// SetGroupParent sets the parent group of a group.
	//
	// example:
	//  err := sdk.SetGroupParent("groupID", "domainID", "groupID", "token")
	//  fmt.Println(err)
	SetGroupParent(id, domainID, groupID, token string) errors.SDKError

	// RemoveGroupParent removes the parent group of a group.
	//
	// example:
	//  err := sdk.RemoveGroupParent("groupID", "domainID", "groupID", "token")
	//  fmt.Println(err)
	RemoveGroupParent(id, domainID, groupID, token string) errors.SDKError

	// AddChildren adds children groups to a group.
	//
	// example:
	//  groupIDs := []string{"groupID1", "groupID2"}
	//  err := sdk.AddChildren("groupID", "domainID", groupIDs, "token")
	//  fmt.Println(err)
	AddChildren(id, domainID string, groupIDs []string, token string) errors.SDKError

	// RemoveChildren removes children groups from a group.
	//
	// example:
	//  groupIDs := []string{"groupID1", "groupID2"}
	//  err := sdk.RemoveChildren("groupID", "domainID", groupIDs, "token")
	//  fmt.Println(err)
	RemoveChildren(id, domainID string, groupIDs []string, token string) errors.SDKError

	// RemoveAllChildren removes all children groups from a group.
	//
	// example:
	//  err := sdk.RemoveAllChildren("groupID", "domainID", "token")
	//  fmt.Println(err)
	RemoveAllChildren(id, domainID, token string) errors.SDKError

	// Children returns page of children groups.
	//
	// example:
	//  pm := sdk.PageMetadata{
	//    Offset: 0,
	//    Limit:  10,
	//  }
	//  groups, _ := sdk.Children("groupID", "domainID", pm, "token")
	//  fmt.Println(groups)
	Children(id, domainID string, pm PageMetadata, token string) (GroupsPage, errors.SDKError)

	// EnableGroup changes group status to enabled.
	//
	// example:
	//  group, _ := sdk.EnableGroup("groupID", "domainID", "token")
	//  fmt.Println(group)
	EnableGroup(id, domainID, token string) (Group, errors.SDKError)

	// DisableGroup changes group status to disabled - soft delete.
	//
	// example:
	//  group, _ := sdk.DisableGroup("groupID", "domainID", "token")
	//  fmt.Println(group)
	DisableGroup(id, domainID, token string) (Group, errors.SDKError)

	// DeleteGroup delete given group id.
	//
	// example:
	//  err := sdk.DeleteGroup("groupID", "domainID", "token")
	//  fmt.Println(err)
	DeleteGroup(id, domainID, token string) errors.SDKError

	// Hierarchy returns page of groups hierarchy.
	//
	// example:
	//  pm := sdk.PageMetadata{
	//    Level: 2,
	//    Direction : -1,
	//	  Tree: true,
	//  }
	// groups, _ := sdk.Hierarchy("groupID", "domainID", pm, "token")
	// fmt.Println(groups)
	Hierarchy(id, domainID string, pm PageMetadata, token string) (GroupsHierarchyPage, errors.SDKError)

	// CreateGroupRole creates new group role and returns its id.
	//
	// example:
	//  rq := sdk.RoleReq{
	//    RoleName: "My Role",
	//    OptionalActions: []string{"read", "update"},
	//    OptionalMembers: []string{"member_id_1", "member_id_2"},
	//  }
	//  role, _ := sdk.CreateGroupRole("groupID", "domainID", rq, "token")
	//  fmt.Println(role)
	CreateGroupRole(id, domainID string, rq RoleReq, token string) (Role, errors.SDKError)

	// GroupRoles returns group roles.
	//
	// example:
	//  pm := sdk.PageMetadata{
	//   Offset: 0,
	//   Limit:  10,
	// }
	//  roles, _ := sdk.GroupRoles("groupID", "domainID",pm, "token")
	//  fmt.Println(roles)
	GroupRoles(id, domainID string, pm PageMetadata, token string) (RolesPage, errors.SDKError)

	// GroupRole returns group role object by roleID.
	//
	// example:
	//  role, _ := sdk.GroupRole("groupID", "roleID", "domainID", "token")
	//  fmt.Println(role)
	GroupRole(id, roleID, domainID, token string) (Role, errors.SDKError)

	// UpdateGroupRole updates existing group role name.
	//
	// example:
	//  role, _ := sdk.UpdateGroupRole{"groupID", "roleID", "newName", "domainID", "token"}
	//  fmr.Println(role)
	UpdateGroupRole(id, roleID, newName, domainID string, token string) (Role, errors.SDKError)

	// DeleteGroupRole deletes a group role with the given groupID and  roleID.
	//
	// example:
	//  err := sdk.DeleteGroupRole("groupID", "roleID", "domainID", "token")
	//  fmt.Println(err)
	DeleteGroupRole(id, roleID, domainID, token string) errors.SDKError

	// AddGroupRoleActions adds actions to a group role.
	//
	// example:
	//  actions := []string{"read", "update"}
	//  actions, _ := sdk.AddGroupRoleActions("groupID", "roleID", "domainID", actions, "token")
	//  fmt.Println(actions)
	AddGroupRoleActions(id, roleID, domainID string, actions []string, token string) ([]string, errors.SDKError)

	// GroupRoleActions returns group role actions by roleID.
	//
	// example:
	//  actions, _ := sdk.GroupRoleActions("groupID", "roleID", "domainID", "token")
	//  fmt.Println(actions)
	GroupRoleActions(id, roleID, domainID string, token string) ([]string, errors.SDKError)

	// RemoveGroupRoleActions removes actions from a group role.
	//
	// example:
	//  actions := []string{"read", "update"}
	//  err := sdk.RemoveGroupRoleActions("groupID", "roleID", "domainID", actions, "token")
	//  fmt.Println(err)
	RemoveGroupRoleActions(id, roleID, domainID string, actions []string, token string) errors.SDKError

	// RemoveAllGroupRoleActions removes all actions from a group role.
	//
	// example:
	//  err := sdk.RemoveAllGroupRoleActions("groupID", "roleID", "domainID", "token")
	//  fmt.Println(err)
	RemoveAllGroupRoleActions(id, roleID, domainID, token string) errors.SDKError

	// AddGroupRoleMembers adds members to a group role.
	//
	// example:
	//  members := []string{"member_id_1", "member_id_2"}
	//  members, _ := sdk.AddGroupRoleMembers("groupID", "roleID", "domainID", members, "token")
	//  fmt.Println(members)
	AddGroupRoleMembers(id, roleID, domainID string, members []string, token string) ([]string, errors.SDKError)

	// GroupRoleMembers returns group role members by roleID.
	//
	// example:
	// pm := sdk.PageMetadata{
	//   Offset: 0,
	//  Limit:  10,
	// }
	//  members, _ := sdk.GroupRoleMembers("groupID", "roleID", "domainID", "token")
	//  fmt.Println(members)
	GroupRoleMembers(id, roleID, domainID string, pm PageMetadata, token string) (RoleMembersPage, errors.SDKError)

	// RemoveGroupRoleMembers removes members from a group role.
	//
	// example:
	//  members := []string{"member_id_1", "member_id_2"}
	//  err := sdk.RemoveGroupRoleMembers("groupID", "roleID", "domainID", members, "token")
	//  fmt.Println(err)
	RemoveGroupRoleMembers(id, roleID, domainID string, members []string, token string) errors.SDKError

	// RemoveAllGroupRoleMembers removes all members from a group role.
	//
	// example:
	//  err := sdk.RemoveAllGroupRoleMembers("groupID", "roleID", "domainID", "token")
	//  fmt.Println(err)
	RemoveAllGroupRoleMembers(id, roleID, domainID, token string) errors.SDKError

	// AvailableGroupRoleActions returns available actions for a group role.
	//
	// example:
	//  actions, _ := sdk.AvailableGroupRoleActions("groupID", "token")
	//  fmt.Println(actions)
	AvailableGroupRoleActions(id, token string) ([]string, errors.SDKError)

	// ListGroupMembers list all members from all roles in a group .
	//
	// example:
	//	pm := sdk.PageMetadata{
	//		Offset: 0,
	//		Limit:  10,
	//	}
	//  members, _ := sdk.ListGroupMembers("group_id","domainID", pm, "token")
	//  fmt.Println(members)
	ListGroupMembers(groupID, domainID string, pm PageMetadata, token string) (EntityMembersPage, errors.SDKError)

	// CreateChannel creates new channel and returns its id.
	//
	// example:
	//  channel := sdk.Channel{
	//    Name: "My Channel",
	//    Metadata: sdk.Metadata{
	//      "key": "value",
	//    },
	//  }
	//  channel, _ := sdk.CreateChannel(channel, "domainID", "token")
	//  fmt.Println(channel)
	CreateChannel(channel Channel, domainID, token string) (Channel, errors.SDKError)

	// CreateChannels creates new channels and returns their ids.
	//
	// example:
	//  channels := []sdk.Channel{
	//    {
	//      Name: "My Channel 1",
	//      Metadata: sdk.Metadata{
	//        "key": "value",
	//      },
	//    },
	//    {
	//      Name: "My Channel 2",
	//      Metadata: sdk.Metadata{
	//        "key": "value",
	//      },
	//    },
	//  }
	//  channels, _ := sdk.CreateChannels(channels, "domainID", "token")
	//  fmt.Println(channels)
	CreateChannels(channels []Channel, domainID, token string) ([]Channel, errors.SDKError)

	// Channels returns page of channels.
	//
	// example:
	//  pm := sdk.PageMetadata{
	//    Offset: 0,
	//    Limit:  10,
	//    Name:   "My Channel",
	//  }
	//  channels, _ := sdk.Channels(pm, "domainID", "token")
	//  fmt.Println(channels)
	Channels(pm PageMetadata, domainID, token string) (ChannelsPage, errors.SDKError)

	// Channel returns channel data by id.
	//
	// example:
	//  channel, _ := sdk.Channel("channelID", "domainID", "token")
	//  fmt.Println(channel)
	Channel(id, domainID, token string) (Channel, errors.SDKError)

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
	//  channel, _ := sdk.UpdateChannel(channel, "domainID", "token")
	//  fmt.Println(channel)
	UpdateChannel(channel Channel, domainID, token string) (Channel, errors.SDKError)

	// UpdateChannelTags updates the channel's tags.
	//
	// example:
	//  channel := sdk.Channel{
	//    ID:   "channelID",
	//    Tags: []string{"tag1", "tag2"},
	//  }
	//  channel, _ := sdk.UpdateChannelTags(channel, "domainID", "token")
	//  fmt.Println(channel)
	UpdateChannelTags(c Channel, domainID, token string) (Channel, errors.SDKError)

	// EnableChannel changes channel status to enabled.
	//
	// example:
	//  channel, _ := sdk.EnableChannel("channelID", "domainID", "token")
	//  fmt.Println(channel)
	EnableChannel(id, domainID, token string) (Channel, errors.SDKError)

	// DisableChannel changes channel status to disabled - soft delete.
	//
	// example:
	//  channel, _ := sdk.DisableChannel("channelID", "domainID", "token")
	//  fmt.Println(channel)
	DisableChannel(id, domainID, token string) (Channel, errors.SDKError)

	// DeleteChannel delete given group id.
	//
	// example:
	//  err := sdk.DeleteChannel("channelID", "domainID", "token")
	//  fmt.Println(err)
	DeleteChannel(id, domainID, token string) errors.SDKError

	// SetChannelParent sets the parent group of a channel.
	//
	// example:
	//  err := sdk.SetChannelParent("channelID", "domainID", "groupID", "token")
	//  fmt.Println(err)
	SetChannelParent(id, domainID, groupID, token string) errors.SDKError

	// RemoveChannelParent removes the parent group of a channel.
	//
	// example:
	//  err := sdk.RemoveChannelParent("channelID", "domainID", "groupID", "token")
	//  fmt.Println(err)
	RemoveChannelParent(id, domainID, groupID, token string) errors.SDKError

	// Connect bulk connects clients to channels specified by id.
	//
	// example:
	//  conns := sdk.Connection{
	//    ChannelIDs: []string{"channel_id_1"},
	//    ClientIDs:  []string{"client_id_1"},
	//    Types:   	  []string{"Publish", "Subscribe"},
	//  }
	//  err := sdk.Connect(conns, "domainID", "token")
	//  fmt.Println(err)
	Connect(conn Connection, domainID, token string) errors.SDKError

	// Disconnect
	//
	// example:
	//  conns := sdk.Connection{
	//    ChannelIDs: []string{"channel_id_1"},
	//    ClientIDs:  []string{"client_id_1"},
	//    Types:   	  []string{"Publish", "Subscribe"},
	//  }
	//  err := sdk.Disconnect(conns, "domainID", "token")
	//  fmt.Println(err)
	Disconnect(conn Connection, domainID, token string) errors.SDKError

	// ConnectClient connects client to specified channel by id.
	//
	// example:
	//  clientIDs := []string{"client_id_1", "client_id_2"}
	//  err := sdk.ConnectClient("channelID", clientIDs, []string{"Publish", "Subscribe"}, "token")
	//  fmt.Println(err)
	ConnectClients(channelID string, clientIDs, connTypes []string, domainID, token string) errors.SDKError

	// DisconnectClient disconnect client from specified channel by id.
	//
	// example:
	//  clientIDs := []string{"client_id_1", "client_id_2"}
	//  err := sdk.DisconnectClient("channelID", clientIDs, []string{"Publish", "Subscribe"}, "token")
	//  fmt.Println(err)
	DisconnectClients(channelID string, clientIDs, connTypes []string, domainID, token string) errors.SDKError

	// ListChannelMembers list all members from all roles in a channel .
	//
	// example:
	//	pm := sdk.PageMetadata{
	//		Offset: 0,
	//		Limit:  10,
	//	}
	//  members, _ := sdk.ListChannelMembers("channel_id","domainID", pm, "token")
	//  fmt.Println(members)
	ListChannelMembers(channelID, domainID string, pm PageMetadata, token string) (EntityMembersPage, errors.SDKError)

	// SendMessage send message to specified channel.
	//
	// example:
	//  msg := '[{"bn":"some-base-name:","bt":1.276020076001e+09, "bu":"A","bver":5, "n":"voltage","u":"V","v":120.1}, {"n":"current","t":-5,"v":1.2}, {"n":"current","t":-4,"v":1.3}]'
	//  err := sdk.SendMessage("channelID", msg, "clientSecret")
	//  fmt.Println(err)
	SendMessage(chanID, msg, key string) errors.SDKError

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

	// IssueCert issues a certificate for a client required for mTLS.
	//
	// example:
	//  cert, _ := sdk.IssueCert("clientID", "24h", "domainID", "token")
	//  fmt.Println(cert)
	IssueCert(clientID, validity, domainID, token string) (Cert, errors.SDKError)

	// ViewCert returns a certificate given certificate ID
	//
	// example:
	//  cert, _ := sdk.ViewCert("certID", "domainID", "token")
	//  fmt.Println(cert)
	ViewCert(certID, domainID, token string) (Cert, errors.SDKError)

	// ViewCertByClient retrieves a list of certificates' serial IDs for a given client ID.
	//
	// example:
	//  cserial, _ := sdk.ViewCertByClient("clientID", "domainID", "token")
	//  fmt.Println(cserial)
	ViewCertByClient(clientID, domainID, token string) (CertSerials, errors.SDKError)

	// RevokeCert revokes certificate for client with clientID
	//
	// example:
	//  tm, _ := sdk.RevokeCert("clientID", "domainID", "token")
	//  fmt.Println(tm)
	RevokeCert(clientID, domainID, token string) (time.Time, errors.SDKError)

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

	// FreezeDomain changes the status of the domain to frozen.
	//
	// example:
	//  err := sdk.FreezeDomain("domainID", "token")
	//  fmt.Println(err)
	FreezeDomain(domainID, token string) errors.SDKError

	// CreateDomainRole creates new domain role and returns its id.
	//
	// example:
	//  rq := sdk.RoleReq{
	//    RoleName: "My Role",
	//    OptionalActions: []string{"read", "update"},
	//    OptionalMembers: []string{"member_id_1", "member_id_2"},
	//  }
	//  role, _ := sdk.CreateDomainRole("domainID", rq, "token")
	//  fmt.Println(role)
	CreateDomainRole(id string, rq RoleReq, token string) (Role, errors.SDKError)

	// DomainRoles returns domain roles.
	//
	// example:
	//  pm := sdk.PageMetadata{
	//   Offset: 0,
	//   Limit:  10,
	// }
	//  roles, _ := sdk.DomainRoles("domainID", pm, "token")
	//  fmt.Println(roles)
	DomainRoles(id string, pm PageMetadata, token string) (RolesPage, errors.SDKError)

	// DomainRole returns domain role object by roleID.
	//
	// example:
	//  role, _ := sdk.DomainRole("domainID", "roleID", "token")
	//  fmt.Println(role)
	DomainRole(id, roleID, token string) (Role, errors.SDKError)

	// UpdateDomainRole updates existing domain role name.
	//
	// example:
	//  role, _ := sdk.UpdateDomainRole("domainID", "roleID", "newName", "token")
	//  fmt.Println(role)
	UpdateDomainRole(id, roleID, newName string, token string) (Role, errors.SDKError)

	// DeleteDomainRole deletes a domain role with the given domainID and roleID.
	//
	// example:
	//  err := sdk.DeleteDomainRole("domainID", "roleID", "token")
	//  fmt.Println(err)
	DeleteDomainRole(id, roleID, token string) errors.SDKError

	// AddDomainRoleActions adds actions to a domain role.
	//
	// example:
	//  actions := []string{"read", "update"}
	//  actions, _ := sdk.AddDomainRoleActions("domainID", "roleID", actions, "token")
	//  fmt.Println(actions)
	AddDomainRoleActions(id, roleID string, actions []string, token string) ([]string, errors.SDKError)

	// DomainRoleActions returns domain role actions by roleID.
	//
	// example:
	//  actions, _ := sdk.DomainRoleActions("domainID", "roleID", "token")
	//  fmt.Println(actions)
	DomainRoleActions(id, roleID string, token string) ([]string, errors.SDKError)

	// RemoveDomainRoleActions removes actions from a domain role.
	//
	// example:
	//  actions := []string{"read", "update"}
	//  err := sdk.RemoveDomainRoleActions("domainID", "roleID", actions, "token")
	//  fmt.Println(err)
	RemoveDomainRoleActions(id, roleID string, actions []string, token string) errors.SDKError

	// RemoveAllDomainRoleActions removes all actions from a domain role.
	//
	// example:
	//  err := sdk.RemoveAllDomainRoleActions("domainID", "roleID", "token")
	//  fmt.Println(err)
	RemoveAllDomainRoleActions(id, roleID, token string) errors.SDKError

	// AddDomainRoleMembers adds members to a domain role.
	//
	// example:
	//  members := []string{"member_id_1", "member_id_2"}
	//  members, _ := sdk.AddDomainRoleMembers("domainID", "roleID", members, "token")
	//  fmt.Println(members)
	AddDomainRoleMembers(id, roleID string, members []string, token string) ([]string, errors.SDKError)

	// DomainRoleMembers returns domain role members by roleID.
	//
	// example:
	//  pm := sdk.PageMetadata{
	//    Offset: 0,
	//    Limit:  10,
	//  }
	//  members, _ := sdk.DomainRoleMembers("domainID", "roleID", "token")
	//  fmt.Println(members)
	DomainRoleMembers(id, roleID string, pm PageMetadata, token string) (RoleMembersPage, errors.SDKError)

	// RemoveDomainRoleMembers removes members from a domain role.
	//
	// example:
	//  members := []string{"member_id_1", "member_id_2"}
	//  err := sdk.RemoveDomainRoleMembers("domainID", "roleID", members, "token")
	//  fmt.Println(err)
	RemoveDomainRoleMembers(id, roleID string, members []string, token string) errors.SDKError

	// RemoveAllDomainRoleMembers removes all members from a domain role.
	//
	// example:
	//  err := sdk.RemoveAllDomainRoleMembers("domainID", "roleID", "token")
	//  fmt.Println(err)
	RemoveAllDomainRoleMembers(id, roleID, token string) errors.SDKError

	// AvailableDomainRoleActions returns available actions for a domain role.
	//
	// example:
	//  actions, _ := sdk.AvailableDomainRoleActions("token")
	//  fmt.Println(actions)
	AvailableDomainRoleActions(token string) ([]string, errors.SDKError)

	// ListDomainUsers returns list of users for the given domain ID and filters.
	//
	// example:
	//  pm := sdk.PageMetadata{
	//    Offset: 0,
	//    Limit:  10,
	//  }
	//  members, _ := sdk.ListDomainMembers("domain_id", pm, "token")
	//  fmt.Println(members)
	ListDomainMembers(domainID string, pm PageMetadata, token string) (EntityMembersPage, errors.SDKError)

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
	//  invitations, _ := sdk.Invitations(PageMetadata{Offset: 0, Limit: 10}, "token")
	//  fmt.Println(invitations)
	Invitations(pm PageMetadata, token string) (invitations InvitationPage, err error)

	// AcceptInvitation accepts an invitation by adding the user to the domain that they were invited to.
	//
	// For example:
	//  err := sdk.AcceptInvitation("domainID", "token")
	//  fmt.Println(err)
	AcceptInvitation(domainID, token string) (err error)

	// RejectInvitation rejects an invitation.
	//
	// For example:
	//  err := sdk.RejectInvitation("domainID", "token")
	//  fmt.Println(err)
	RejectInvitation(domainID, token string) (err error)

	// DeleteInvitation deletes an invitation.
	//
	// For example:
	//  err := sdk.DeleteInvitation("userID", "domainID", "token")
	//  fmt.Println(err)
	DeleteInvitation(userID, domainID, token string) (err error)

	// Journal returns a list of journal logs.
	//
	// For example:
	//  journals, _ := sdk.Journal("client", "clientID","domainID", PageMetadata{Offset: 0, Limit: 10, Operation: "client.create"}, "token")
	//  fmt.Println(journals)
	Journal(entityType, entityID, domainID string, pm PageMetadata, token string) (journal JournalsPage, err error)
}

type mgSDK struct {
	certsURL       string
	httpAdapterURL string
	clientsURL     string
	usersURL       string
	groupsURL      string
	channelsURL    string
	domainsURL     string
	journalURL     string
	HostURL        string

	msgContentType ContentType
	client         *http.Client
	curlFlag       bool
}

// Config contains sdk configuration parameters.
type Config struct {
	CertsURL       string
	HTTPAdapterURL string
	ClientsURL     string
	UsersURL       string
	GroupsURL      string
	ChannelsURL    string
	DomainsURL     string
	JournalURL     string
	HostURL        string

	MsgContentType  ContentType
	TLSVerification bool
	CurlFlag        bool
}

// NewSDK returns new supermq SDK instance.
func NewSDK(conf Config) SDK {
	return &mgSDK{
		certsURL:       conf.CertsURL,
		httpAdapterURL: conf.HTTPAdapterURL,
		clientsURL:     conf.ClientsURL,
		usersURL:       conf.UsersURL,
		groupsURL:      conf.GroupsURL,
		channelsURL:    conf.ChannelsURL,
		domainsURL:     conf.DomainsURL,
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
		if !strings.Contains(token, ClientPrefix) {
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
	if pm.Email != "" {
		q.Add("email", pm.Email)
	}
	if pm.Identity != "" {
		q.Add("identity", pm.Identity)
	}
	if pm.Username != "" {
		q.Add("username", pm.Username)
	}
	if pm.FirstName != "" {
		q.Add("first_name", pm.FirstName)
	}
	if pm.LastName != "" {
		q.Add("last_name", pm.LastName)
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
