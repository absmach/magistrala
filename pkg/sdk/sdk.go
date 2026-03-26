// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"bytes"
	"context"
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

	"github.com/absmach/supermq/certs"
	"github.com/absmach/supermq/pkg/errors"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
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

type Operator uint8

const (
	OrOp Operator = iota
	AndOp
)

type TagsQuery struct {
	Elements []string
	Operator Operator
}

func ToTagsQuery(s string) TagsQuery {
	switch {
	case strings.Contains(s, "+"):
		elements := strings.Split(s, "+")
		for i := range elements {
			elements[i] = strings.TrimSpace(elements[i])
		}
		return TagsQuery{Elements: elements, Operator: AndOp}
	case strings.Contains(s, ","):
		elements := strings.Split(s, ",")
		for i := range elements {
			elements[i] = strings.TrimSpace(elements[i])
		}
		return TagsQuery{Elements: elements, Operator: OrOp}
	default:
		return TagsQuery{Elements: []string{s}, Operator: OrOp}
	}
}

type PageMetadata struct {
	Total           uint64    `json:"total"`
	Offset          uint64    `json:"offset"`
	Limit           uint64    `json:"limit"`
	Order           string    `json:"order,omitempty"`
	Direction       string    `json:"direction,omitempty"`
	Level           uint64    `json:"level,omitempty"`
	Identity        string    `json:"identity,omitempty"`
	Email           string    `json:"email,omitempty"`
	Username        string    `json:"username,omitempty"`
	LastName        string    `json:"last_name,omitempty"`
	FirstName       string    `json:"first_name,omitempty"`
	Name            string    `json:"name,omitempty"`
	Type            string    `json:"type,omitempty"`
	Metadata        Metadata  `json:"metadata,omitempty"`
	Status          string    `json:"status,omitempty"`
	Action          string    `json:"action,omitempty"`
	Subject         string    `json:"subject,omitempty"`
	Object          string    `json:"object,omitempty"`
	Permission      string    `json:"permission,omitempty"`
	Tags            TagsQuery `json:"tags,omitempty"`
	Owner           string    `json:"owner,omitempty"`
	SharedBy        string    `json:"shared_by,omitempty"`
	Visibility      string    `json:"visibility,omitempty"`
	OwnerID         string    `json:"owner_id,omitempty"`
	Topic           string    `json:"topic,omitempty"`
	Contact         string    `json:"contact,omitempty"`
	State           string    `json:"state,omitempty"`
	ListPermissions string    `json:"list_perms,omitempty"`
	InvitedBy       string    `json:"invited_by,omitempty"`
	UserID          string    `json:"user_id,omitempty"`
	DomainID        string    `json:"domain_id,omitempty"`
	Relation        string    `json:"relation,omitempty"`
	Operation       string    `json:"operation,omitempty"`
	From            int64     `json:"from,omitempty"`
	To              int64     `json:"to,omitempty"`
	WithMetadata    bool      `json:"with_metadata,omitempty"`
	WithAttributes  bool      `json:"with_attributes,omitempty"`
	ID              string    `json:"id,omitempty"`
	Tree            bool      `json:"tree,omitempty"`
	StartLevel      int64     `json:"start_level,omitempty"`
	EndLevel        int64     `json:"end_level,omitempty"`
	CreatedFrom     time.Time `json:"created_from,omitempty"`
	CreatedTo       time.Time `json:"created_to,omitempty"`
	Dir             string    `json:"dir,omitempty"`
	Tag             string    `json:"tag,omitempty"`
	InputChannel    string    `json:"input_channel,omitempty"`
	RuleID          string    `json:"rule_id,omitempty"`
	ChannelID       string    `json:"channel_id,omitempty"`
	ClientID        string    `json:"client_id,omitempty"`
	Subtopic        string    `json:"subtopic,omitempty"`
	AssigneeID      string    `json:"assignee_id,omitempty"`
	Severity        uint8     `json:"severity,omitempty"`
	UpdatedBy       string    `json:"updated_by,omitempty"`
	AssignedBy      string    `json:"assigned_by,omitempty"`
	AcknowledgedBy  string    `json:"acknowledged_by,omitempty"`
	ResolvedBy      string    `json:"resolved_by,omitempty"`
	EntityID        string    `json:"entity_id,omitempty"`
	CommonName      string    `json:"common_name,omitempty"`
	TTL             string    `json:"ttl,omitempty"`
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

// CertStatus represents the status of a certificate.
type CertStatus int

const (
	CertValid   CertStatus = iota
	CertRevoked CertStatus = iota
	CertUnknown CertStatus = iota
)

func (c CertStatus) String() string {
	switch c {
	case CertValid:
		return "Valid"
	case CertRevoked:
		return "Revoked"
	default:
		return "Unknown"
	}
}

func (c CertStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.String())
}

// Certificate holds certificate data returned by the certs service SDK.
type Certificate struct {
	SerialNumber string    `json:"serial_number,omitempty"`
	Certificate  string    `json:"certificate,omitempty"`
	Key          string    `json:"key,omitempty"`
	Revoked      bool      `json:"revoked,omitempty"`
	ExpiryTime   time.Time `json:"expiry_time,omitempty"`
	EntityID     string    `json:"entity_id,omitempty"`
	DownloadUrl  string    `json:"-"`
}

// CertificatePage holds a page of certificates.
type CertificatePage struct {
	Total        uint64        `json:"total"`
	Offset       uint64        `json:"offset"`
	Limit        uint64        `json:"limit"`
	Certificates []Certificate `json:"certificates,omitempty"`
}

// CertificateBundle holds CA and certificate data for download.
type CertificateBundle struct {
	CA          []byte `json:"ca"`
	Certificate []byte `json:"certificate"`
	PrivateKey  []byte `json:"private_key"`
}

// OCSPResponse holds the OCSP status response for a certificate.
type OCSPResponse struct {
	Status           CertStatus `json:"status"`
	SerialNumber     string     `json:"serial_number"`
	RevokedAt        *time.Time `json:"revoked_at,omitempty"`
	ProducedAt       *time.Time `json:"produced_at,omitempty"`
	ThisUpdate       *time.Time `json:"this_update,omitempty"`
	NextUpdate       *time.Time `json:"next_update,omitempty"`
	Certificate      []byte     `json:"certificate,omitempty"`
	IssuerHash       string     `json:"issuer_hash,omitempty"`
	RevocationReason int        `json:"revocation_reason,omitempty"`
}

// Options holds certificate subject options for issuance.
type Options struct {
	CommonName         string   `json:"common_name"`
	Organization       []string `json:"organization"`
	OrganizationalUnit []string `json:"organizational_unit"`
	Country            []string `json:"country"`
	Province           []string `json:"province"`
	Locality           []string `json:"locality"`
	StreetAddress      []string `json:"street_address"`
	PostalCode         []string `json:"postal_code"`
	DnsNames           []string `json:"dns_names"`
}

// SDK contains SuperMQ API.
type SDK interface {
	// CreateUser registers supermq user.
	//
	// example:
	//  ctx := context.Background()
	//  user := sdk.User{
	//    Name:	 "John Doe",
	// 	  Email: "john.doe@example",
	//    Credentials: sdk.Credentials{
	//      Username: "john.doe",
	//      Secret:   "12345678",
	//    },
	//  }
	//  user, _ := sdk.CreateUser(ctx, user)
	//  fmt.Println(user)
	CreateUser(ctx context.Context, user User, token string) (User, errors.SDKError)

	// SendVerification sends a verification email to the user.
	//
	// example:
	//  err := sdk.SendVerification("token")
	//  fmt.Println(err)
	SendVerification(ctx context.Context, token string) errors.SDKError

	// VerifyEmail verifies the user's email address using the provided token.
	//
	// example:
	//  err := sdk.VerifyEmail("verificationToken")
	//  fmt.Println(user)
	VerifyEmail(ctx context.Context, verificationToken string) errors.SDKError

	// User returns user object by id.
	//
	// example:
	//  ctx := context.Background()
	//  user, _ := sdk.User(ctx, "userID", "token")
	//  fmt.Println(user)
	User(ctx context.Context, id, token string) (User, errors.SDKError)

	// Users returns list of users.
	//
	// example:
	//  ctx := context.Background()
	//	pm := sdk.PageMetadata{
	//		Offset: 0,
	//		Limit:  10,
	//		Name:   "John Doe",
	//	}
	//	users, _ := sdk.Users(ctx, pm, "token")
	//	fmt.Println(users)
	Users(ctx context.Context, pm PageMetadata, token string) (UsersPage, errors.SDKError)

	// UserProfile returns user logged in.
	//
	// example:
	//  ctx := context.Background()
	//  user, _ := sdk.UserProfile(ctx, "token")
	//  fmt.Println(user)
	UserProfile(ctx context.Context, token string) (User, errors.SDKError)

	// UpdateUser updates existing user.
	//
	// example:
	//  ctx := context.Background()
	//  user := sdk.User{
	//    ID:   "userID",
	//    Name: "John Doe",
	//    Metadata: sdk.Metadata{
	//      "key": "value",
	//    },
	//  }
	//  user, _ := sdk.UpdateUser(ctx, user, "token")
	//  fmt.Println(user)
	UpdateUser(ctx context.Context, user User, token string) (User, errors.SDKError)

	// UpdateUserEmail updates the user's email
	//
	// example:
	//  ctx := context.Background()
	//  user := sdk.User{
	//    ID:   "userID",
	//    Credentials: sdk.Credentials{
	//      Email: "john.doe@example",
	//    },
	//  }
	//  user, _ := sdk.UpdateUserEmail(ctx, user, "token")
	//  fmt.Println(user)
	UpdateUserEmail(ctx context.Context, user User, token string) (User, errors.SDKError)

	// UpdateUserTags updates the user's tags.
	//
	// example:
	//  ctx := context.Background()
	//  user := sdk.User{
	//    ID:   "userID",
	//    Tags: []string{"tag1", "tag2"},
	//  }
	//  user, _ := sdk.UpdateUserTags(ctx, user, "token")
	//  fmt.Println(user)
	UpdateUserTags(ctx context.Context, user User, token string) (User, errors.SDKError)

	// UpdateUsername updates the user's Username.
	//
	// example:
	//  ctx := context.Background()
	//  user := sdk.User{
	//    ID:   "userID",
	//    Credentials: sdk.Credentials{
	//	  	Username: "john.doe",
	//		},
	//  }
	//  user, _ := sdk.UpdateUsername(ctx, user, "token")
	//  fmt.Println(user)
	UpdateUsername(ctx context.Context, user User, token string) (User, errors.SDKError)

	// UpdateProfilePicture updates the user's profile picture.
	//
	// example:
	//  ctx := context.Background()
	//  user := sdk.User{
	//    ID:            "userID",
	//    ProfilePicture: "https://cloudstorage.example.com/bucket-name/user-images/profile-picture.jpg",
	//  }
	//  user, _ := sdk.UpdateProfilePicture(ctx, user, "token")
	//  fmt.Println(user)
	UpdateProfilePicture(ctx context.Context, user User, token string) (User, errors.SDKError)

	// UpdateUserRole updates the user's role.
	//
	// example:
	//  ctx := context.Background()
	//  user := sdk.User{
	//    ID:   "userID",
	//    Role: "role",
	//  }
	//  user, _ := sdk.UpdateUserRole(ctx, user, "token")
	//  fmt.Println(user)
	UpdateUserRole(ctx context.Context, user User, token string) (User, errors.SDKError)

	// ResetPasswordRequest sends a password request email to a user.
	//
	// example:
	//  ctx := context.Background()
	//  err := sdk.ResetPasswordRequest(ctx, "example@email.com")
	//  fmt.Println(err)
	ResetPasswordRequest(ctx context.Context, email string) errors.SDKError

	// ResetPassword changes a user's password to the one passed in the argument.
	//
	// example:
	//  ctx := context.Background()
	//  err := sdk.ResetPassword(ctx, "password","password","token")
	//  fmt.Println(err)
	ResetPassword(ctx context.Context, password, confPass, token string) errors.SDKError

	// UpdatePassword updates user password.
	//
	// example:
	//  ctx := context.Background()
	//  user, _ := sdk.UpdatePassword(ctx, "oldPass", "newPass", "token")
	//  fmt.Println(user)
	UpdatePassword(ctx context.Context, oldPass, newPass, token string) (User, errors.SDKError)

	// EnableUser changes the status of the user to enabled.
	//
	// example:
	//  ctx := context.Background()
	//  user, _ := sdk.EnableUser(ctx, "userID", "token")
	//  fmt.Println(user)
	EnableUser(ctx context.Context, id, token string) (User, errors.SDKError)

	// DisableUser changes the status of the user to disabled.
	//
	// example:
	//  ctx := context.Background()
	//  user, _ := sdk.DisableUser(ctx, "userID", "token")
	//  fmt.Println(user)
	DisableUser(ctx context.Context, id, token string) (User, errors.SDKError)

	// DeleteUser deletes a user with the given id.
	//
	// example:
	//  ctx := context.Background()
	//  err := sdk.DeleteUser(ctx, "userID", "token")
	//  fmt.Println(err)
	DeleteUser(ctx context.Context, id, token string) errors.SDKError

	// CreateToken receives credentials and returns user token.
	//
	// example:
	//  ctx := context.Background()
	//  lt := sdk.Login{
	//      Identity: "email"/"username",
	//      Secret:   "12345678",
	//  }
	//  token, _ := sdk.CreateToken(ctx, lt)
	//  fmt.Println(token)
	CreateToken(ctx context.Context, lt Login) (Token, errors.SDKError)

	// RefreshToken receives credentials and returns user token.
	//
	// example:
	//  ctx := context.Background()
	//  token, _ := sdk.RefreshToken(ctx, "refresh_token")
	//  fmt.Println(token)
	RefreshToken(ctx context.Context, token string) (Token, errors.SDKError)

	// SeachUsers filters users and returns a page result.
	//
	// example:
	//  ctx := context.Background()
	//  pm := sdk.PageMetadata{
	//	Offset: 0,
	//	Limit:  10,
	//	Name:   "John Doe",
	//  }
	//  users, _ := sdk.SearchUsers(ctx, pm, "token")
	//  fmt.Println(users)
	SearchUsers(ctx context.Context, pm PageMetadata, token string) (UsersPage, errors.SDKError)

	// CreateClient registers new client and returns its id.
	//
	// example:
	//  ctx := context.Background()
	//  client := sdk.Client{
	//    Name: "My Client",
	//    Metadata: sdk.Metadata{"domain_1"
	//      "key": "value",
	//    },
	//  }
	//  client, _ := sdk.CreateClient(ctx, client, "domainID", "token")
	//  fmt.Println(client)
	CreateClient(ctx context.Context, client Client, domainID, token string) (Client, errors.SDKError)

	// CreateClients registers new clients and returns their ids.
	//
	// example:
	//  ctx := context.Background()
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
	//  clients, _ := sdk.CreateClients(ctx, clients, "domainID", "token")
	//  fmt.Println(clients)
	CreateClients(ctx context.Context, client []Client, domainID, token string) ([]Client, errors.SDKError)

	// Filters clients and returns a page result.
	//
	// example:
	//  ctx := context.Background()
	//  pm := sdk.PageMetadata{
	//    Offset: 0,
	//    Limit:  10,
	//    Name:   "My Client",
	//  }
	//  clients, _ := sdk.Clients(ctx, pm, "domainID", "token")
	//  fmt.Println(clients)
	Clients(ctx context.Context, pm PageMetadata, domainID, token string) (ClientsPage, errors.SDKError)

	// Client returns client object by id.
	//
	// example:
	//  ctx := context.Background()
	//  client, _ := sdk.Client(ctx, "clientID", "domainID", "token")
	//  fmt.Println(client)
	Client(ctx context.Context, id, domainID, token string) (Client, errors.SDKError)

	// UpdateClient updates existing client.
	//
	// example:
	//  ctx := context.Background()
	//  client := sdk.Client{
	//    ID:   "clientID",
	//    Name: "My Client",
	//    Metadata: sdk.Metadata{
	//      "key": "value",
	//    },
	//  }
	//  client, _ := sdk.UpdateClient(ctx, client, "domainID", "token")
	//  fmt.Println(client)
	UpdateClient(ctx context.Context, client Client, domainID, token string) (Client, errors.SDKError)

	// UpdateClientTags updates the client's tags.
	//
	// example:
	//  ctx := context.Background()
	//  client := sdk.Client{
	//    ID:   "clientID",
	//    Tags: []string{"tag1", "tag2"},
	//  }
	//  client, _ := sdk.UpdateClientTags(ctx, client, "domainID", "token")
	//  fmt.Println(client)
	UpdateClientTags(ctx context.Context, client Client, domainID, token string) (Client, errors.SDKError)

	// UpdateClientSecret updates the client's secret
	//
	// example:
	//  ctx := context.Background()
	//  client, err := sdk.UpdateClientSecret(ctx, "clientID", "newSecret", "domainID," "token")
	//  fmt.Println(client)
	UpdateClientSecret(ctx context.Context, id, secret, domainID, token string) (Client, errors.SDKError)

	// EnableClient changes client status to enabled.
	//
	// example:
	//  ctx := context.Background()
	//  client, _ := sdk.EnableClient(ctx, "clientID", "domainID", "token")
	//  fmt.Println(client)
	EnableClient(ctx context.Context, id, domainID, token string) (Client, errors.SDKError)

	// DisableClient changes client status to disabled - soft delete.
	//
	// example:
	//  ctx := context.Background()
	//  client, _ := sdk.DisableClient(ctx, "clientID", "domainID", "token")
	//  fmt.Println(client)
	DisableClient(ctx context.Context, id, domainID, token string) (Client, errors.SDKError)

	// DeleteClient deletes a client with the given id.
	//
	// example:
	//  ctx := context.Background()
	//  err := sdk.DeleteClient(ctx, "clientID", "domainID", "token")
	//  fmt.Println(err)
	DeleteClient(ctx context.Context, id, domainID, token string) errors.SDKError

	// SetClientParent sets the parent group of a client.
	//
	// example:
	//  ctx := context.Background()
	//  err := sdk.SetClientParent(ctx, "clientID", "domainID", "groupID", "token")
	//  fmt.Println(err)
	SetClientParent(ctx context.Context, id, domainID, groupID, token string) errors.SDKError

	// RemoveClientParent removes the parent group of a client.
	//
	// example:
	//  ctx := context.Background()
	//  err := sdk.RemoveClientParent(ctx, "clientID", "domainID", "groupID", "token")
	//  fmt.Println(err)
	RemoveClientParent(ctx context.Context, id, domainID, groupID, token string) errors.SDKError

	// CreateClientRole creates new client role and returns its id.
	//
	// example:
	//  ctx := context.Background()
	//  rq := sdk.RoleReq{
	//    RoleName: "My Role",
	//    OptionalActions: []string{"read", "update"},
	//    OptionalMembers: []string{"member_id_1", "member_id_2"},
	//  }
	//  role, _ := sdk.CreateClientRole(ctx, "clientID", "domainID", rq, "token")
	//  fmt.Println(role)
	CreateClientRole(ctx context.Context, id, domainID string, rq RoleReq, token string) (Role, errors.SDKError)

	// ClientRoles returns client roles.
	//
	// example:
	//  ctx := context.Background()
	// pm := sdk.PageMetadata{
	//   Offset: 0,
	//   Limit:  10,
	// }
	//  roles, _ := sdk.ClientRoles(ctx, "clientID", "domainID", pm, "token")
	//  fmt.Println(roles)
	ClientRoles(ctx context.Context, id, domainID string, pm PageMetadata, token string) (RolesPage, errors.SDKError)

	// ClientRole returns client role object by roleID.
	//
	// example:
	//  ctx := context.Background()
	//  role, _ := sdk.ClientRole(ctx, "clientID", "roleID", "domainID", "token")
	//  fmt.Println(role)
	ClientRole(ctx context.Context, id, roleID, domainID, token string) (Role, errors.SDKError)

	// UpdateClientRole updates existing client role name.
	//
	// example:
	//  ctx := context.Background()
	//  role, _ := sdk.UpdateClientRole(ctx, "clientID", "roleID", "newName", "domainID", "token")
	//  fmt.Println(role)
	UpdateClientRole(ctx context.Context, id, roleID, newName, domainID string, token string) (Role, errors.SDKError)

	// DeleteClientRole deletes a client role with the given clientID and  roleID.
	//
	// example:
	//  ctx := context.Background()
	//  err := sdk.DeleteClientRole(ctx, "clientID", "roleID", "domainID", "token")
	//  fmt.Println(err)
	DeleteClientRole(ctx context.Context, id, roleID, domainID, token string) errors.SDKError

	// AddClientRoleActions adds actions to a client role.
	//
	// example:
	//  ctx := context.Background()
	//  actions := []string{"read", "update"}
	//  actions, _ := sdk.AddClientRoleActions(ctx, "clientID", "roleID", "domainID", actions, "token")
	//  fmt.Println(actions)
	AddClientRoleActions(ctx context.Context, id, roleID, domainID string, actions []string, token string) ([]string, errors.SDKError)

	// ClientRoleActions returns client role actions by roleID.
	//
	// example:
	//  ctx := context.Background()
	//  actions, _ := sdk.ClientRoleActions(ctx, "clientID", "roleID", "domainID", "token")
	//  fmt.Println(actions)
	ClientRoleActions(ctx context.Context, id, roleID, domainID string, token string) ([]string, errors.SDKError)

	// RemoveClientRoleActions removes actions from a client role.
	//
	// example:
	//  ctx := context.Background()
	//  actions := []string{"read", "update"}
	//  err := sdk.RemoveClientRoleActions(ctx, "clientID", "roleID", "domainID", actions, "token")
	//  fmt.Println(err)
	RemoveClientRoleActions(ctx context.Context, id, roleID, domainID string, actions []string, token string) errors.SDKError

	// RemoveAllClientRoleActions removes all actions from a client role.
	//
	// example:
	//  ctx := context.Background()
	//  err := sdk.RemoveAllClientRoleActions(ctx, "clientID", "roleID", "domainID", "token")
	//  fmt.Println(err)
	RemoveAllClientRoleActions(ctx context.Context, id, roleID, domainID, token string) errors.SDKError

	// AddClientRoleMembers adds members to a client role.
	//
	// example:
	//  ctx := context.Background()
	//  members := []string{"member_id_1", "member_id_2"}
	//  members, _ := sdk.AddClientRoleMembers(ctx, "clientID", "roleID", "domainID", members, "token")
	//  fmt.Println(members)
	AddClientRoleMembers(ctx context.Context, id, roleID, domainID string, members []string, token string) ([]string, errors.SDKError)

	// ClientRoleMembers returns client role members by roleID.
	//
	// example:
	//  ctx := context.Background()
	// pm := sdk.PageMetadata{
	//   Offset: 0,
	//  Limit:  10,
	// }
	//  members, _ := sdk.ClientRoleMembers(ctx, "clientID", "roleID", "domainID", pm,"token")
	//  fmt.Println(members)
	ClientRoleMembers(ctx context.Context, id, roleID, domainID string, pm PageMetadata, token string) (RoleMembersPage, errors.SDKError)

	// RemoveClientRoleMembers removes members from a client role.
	//
	// example:
	//  ctx := context.Background()
	//  members := []string{"member_id_1", "member_id_2"}
	//  err := sdk.RemoveClientRoleMembers(ctx, "clientID", "roleID", "domainID", members, "token")
	//  fmt.Println(err)
	RemoveClientRoleMembers(ctx context.Context, id, roleID, domainID string, members []string, token string) errors.SDKError

	// RemoveAllClientRoleMembers removes all members from a client role.
	//
	// example:
	//  ctx := context.Background()
	//  err := sdk.RemoveAllClientRoleMembers(ctx, "clientID", "roleID", "domainID", "token")
	//  fmt.Println(err)
	RemoveAllClientRoleMembers(ctx context.Context, id, roleID, domainID, token string) errors.SDKError

	// AvailableClientRoleActions returns available actions for a client role.
	//
	// example:
	//  ctx := context.Background()
	//  actions, _ := sdk.AvailableClientRoleActions(ctx, "domainID", "token")
	//  fmt.Println(actions)
	AvailableClientRoleActions(ctx context.Context, domainID, token string) ([]string, errors.SDKError)

	// ListClientMembers list all members from all roles in a client .
	//
	// example:
	//  ctx := context.Background()
	//	pm := sdk.PageMetadata{
	//		Offset: 0,
	//		Limit:  10,
	//	}
	//  members, _ := sdk.ListClientMembers(ctx, "client_id","domainID", pm, "token")
	//  fmt.Println(members)
	ListClientMembers(ctx context.Context, clientID, domainID string, pm PageMetadata, token string) (EntityMembersPage, errors.SDKError)

	// CreateGroup creates new group and returns its id.
	//
	// example:
	//  ctx := context.Background()
	//  group := sdk.Group{
	//    Name: "My Group",
	//    Metadata: sdk.Metadata{
	//      "key": "value",
	//    },
	//  }
	//  group, _ := sdk.CreateGroup(ctx, group, "domainID", "token")
	//  fmt.Println(group)
	CreateGroup(ctx context.Context, group Group, domainID, token string) (Group, errors.SDKError)

	// Groups returns page of groups.
	//
	// example:
	//  ctx := context.Background()
	//  pm := sdk.PageMetadata{
	//    Offset: 0,
	//    Limit:  10,
	//    Name:   "My Group",
	//  }
	//  groups, _ := sdk.Groups(ctx, pm, "domainID", "token")
	//  fmt.Println(groups)
	Groups(ctx context.Context, pm PageMetadata, domainID, token string) (GroupsPage, errors.SDKError)

	// Group returns users group object by id.
	//
	// example:
	//  ctx := context.Background()
	//  group, _ := sdk.Group(ctx, "groupID", "domainID", "token")
	//  fmt.Println(group)
	Group(ctx context.Context, id, domainID, token string) (Group, errors.SDKError)

	// UpdateGroup updates existing group.
	//
	// example:
	//  ctx := context.Background()
	//  group := sdk.Group{
	//    ID:   "groupID",
	//    Name: "My Group",
	//    Metadata: sdk.Metadata{
	//      "key": "value",
	//    },
	//  }
	//  group, _ := sdk.UpdateGroup(ctx, group, "domainID", "token")
	//  fmt.Println(group)
	UpdateGroup(ctx context.Context, group Group, domainID, token string) (Group, errors.SDKError)

	// UpdateGroupTags updates tags for existing group.
	//
	// example:
	//  ctx := context.Background()
	//  group := sdk.Group{
	//    ID:   "groupID",
	//    Tags: []string{"tag1", "tag2"}
	//  }
	//  group, _ := sdk.UpdateGroupTags(ctx, group, "domainID", "token")
	//  fmt.Println(group)
	UpdateGroupTags(ctx context.Context, group Group, domainID, token string) (Group, errors.SDKError)

	// SetGroupParent sets the parent group of a group.
	//
	// example:
	//  ctx := context.Background()
	//  err := sdk.SetGroupParent(ctx, "groupID", "domainID", "groupID", "token")
	//  fmt.Println(err)
	SetGroupParent(ctx context.Context, id, domainID, groupID, token string) errors.SDKError

	// RemoveGroupParent removes the parent group of a group.
	//
	// example:
	//  ctx := context.Background()
	//  err := sdk.RemoveGroupParent(ctx, "groupID", "domainID", "groupID", "token")
	//  fmt.Println(err)
	RemoveGroupParent(ctx context.Context, id, domainID, groupID, token string) errors.SDKError

	// AddChildren adds children groups to a group.
	//
	// example:
	//  ctx := context.Background()
	//  groupIDs := []string{"groupID1", "groupID2"}
	//  err := sdk.AddChildren(ctx, "groupID", "domainID", groupIDs, "token")
	//  fmt.Println(err)
	AddChildren(ctx context.Context, id, domainID string, groupIDs []string, token string) errors.SDKError

	// RemoveChildren removes children groups from a group.
	//
	// example:
	//  ctx := context.Background()
	//  groupIDs := []string{"groupID1", "groupID2"}
	//  err := sdk.RemoveChildren(ctx, "groupID", "domainID", groupIDs, "token")
	//  fmt.Println(err)
	RemoveChildren(ctx context.Context, id, domainID string, groupIDs []string, token string) errors.SDKError

	// RemoveAllChildren removes all children groups from a group.
	//
	// example:
	//  ctx := context.Background()
	//  err := sdk.RemoveAllChildren(ctx, "groupID", "domainID", "token")
	//  fmt.Println(err)
	RemoveAllChildren(ctx context.Context, id, domainID, token string) errors.SDKError

	// Children returns page of children groups.
	//
	// example:
	//  ctx := context.Background()
	//  pm := sdk.PageMetadata{
	//    Offset: 0,
	//    Limit:  10,
	//  }
	//  groups, _ := sdk.Children(ctx, "groupID", "domainID", pm, "token")
	//  fmt.Println(groups)
	Children(ctx context.Context, id, domainID string, pm PageMetadata, token string) (GroupsPage, errors.SDKError)

	// EnableGroup changes group status to enabled.
	//
	// example:
	//  ctx := context.Background()
	//  group, _ := sdk.EnableGroup(ctx, "groupID", "domainID", "token")
	//  fmt.Println(group)
	EnableGroup(ctx context.Context, id, domainID, token string) (Group, errors.SDKError)

	// DisableGroup changes group status to disabled - soft delete.
	//
	// example:
	//  ctx := context.Background()
	//  group, _ := sdk.DisableGroup(ctx, "groupID", "domainID", "token")
	//  fmt.Println(group)
	DisableGroup(ctx context.Context, id, domainID, token string) (Group, errors.SDKError)

	// DeleteGroup delete given group id.
	//
	// example:
	//  ctx := context.Background()
	//  err := sdk.DeleteGroup(ctx, "groupID", "domainID", "token")
	//  fmt.Println(err)
	DeleteGroup(ctx context.Context, id, domainID, token string) errors.SDKError

	// Hierarchy returns page of groups hierarchy.
	//
	// example:
	//  ctx := context.Background()
	//  pm := sdk.PageMetadata{
	//    Level: 2,
	//    Direction : -1,
	//	  Tree: true,
	//  }
	// groups, _ := sdk.Hierarchy(ctx, "groupID", "domainID", pm, "token")
	// fmt.Println(groups)
	Hierarchy(ctx context.Context, id, domainID string, pm PageMetadata, token string) (GroupsHierarchyPage, errors.SDKError)

	// CreateGroupRole creates new group role and returns its id.
	//
	// example:
	//  ctx := context.Background()
	//  rq := sdk.RoleReq{
	//    RoleName: "My Role",
	//    OptionalActions: []string{"read", "update"},
	//    OptionalMembers: []string{"member_id_1", "member_id_2"},
	//  }
	//  role, _ := sdk.CreateGroupRole(ctx, "groupID", "domainID", rq, "token")
	//  fmt.Println(role)
	CreateGroupRole(ctx context.Context, id, domainID string, rq RoleReq, token string) (Role, errors.SDKError)

	// GroupRoles returns group roles.
	//
	// example:
	//  ctx := context.Background()
	//  pm := sdk.PageMetadata{
	//   Offset: 0,
	//   Limit:  10,
	// }
	//  roles, _ := sdk.GroupRoles(ctx, "groupID", "domainID",pm, "token")
	//  fmt.Println(roles)
	GroupRoles(ctx context.Context, id, domainID string, pm PageMetadata, token string) (RolesPage, errors.SDKError)

	// GroupRole returns group role object by roleID.
	//
	// example:
	//  ctx := context.Background()
	//  role, _ := sdk.GroupRole(ctx, "groupID", "roleID", "domainID", "token")
	//  fmt.Println(role)
	GroupRole(ctx context.Context, id, roleID, domainID, token string) (Role, errors.SDKError)

	// UpdateGroupRole updates existing group role name.
	//
	// example:
	//  ctx := context.Background()
	//  role, _ := sdk.UpdateGroupRole(ctx, "groupID", "roleID", "newName", "domainID", "token")
	//  fmt.Println(role)
	UpdateGroupRole(ctx context.Context, id, roleID, newName, domainID string, token string) (Role, errors.SDKError)

	// DeleteGroupRole deletes a group role with the given groupID and  roleID.
	//
	// example:
	//  ctx := context.Background()
	//  err := sdk.DeleteGroupRole(ctx, "groupID", "roleID", "domainID", "token")
	//  fmt.Println(err)
	DeleteGroupRole(ctx context.Context, id, roleID, domainID, token string) errors.SDKError

	// AddGroupRoleActions adds actions to a group role.
	//
	// example:
	//  ctx := context.Background()
	//  actions := []string{"read", "update"}
	//  actions, _ := sdk.AddGroupRoleActions(ctx, "groupID", "roleID", "domainID", actions, "token")
	//  fmt.Println(actions)
	AddGroupRoleActions(ctx context.Context, id, roleID, domainID string, actions []string, token string) ([]string, errors.SDKError)

	// GroupRoleActions returns group role actions by roleID.
	//
	// example:
	//  ctx := context.Background()
	//  actions, _ := sdk.GroupRoleActions(ctx, "groupID", "roleID", "domainID", "token")
	//  fmt.Println(actions)
	GroupRoleActions(ctx context.Context, id, roleID, domainID string, token string) ([]string, errors.SDKError)

	// RemoveGroupRoleActions removes actions from a group role.
	//
	// example:
	//  ctx := context.Background()
	//  actions := []string{"read", "update"}
	//  err := sdk.RemoveGroupRoleActions(ctx, "groupID", "roleID", "domainID", actions, "token")
	//  fmt.Println(err)
	RemoveGroupRoleActions(ctx context.Context, id, roleID, domainID string, actions []string, token string) errors.SDKError

	// RemoveAllGroupRoleActions removes all actions from a group role.
	//
	// example:
	//  ctx := context.Background()
	//  err := sdk.RemoveAllGroupRoleActions(ctx, "groupID", "roleID", "domainID", "token")
	//  fmt.Println(err)
	RemoveAllGroupRoleActions(ctx context.Context, id, roleID, domainID, token string) errors.SDKError

	// AddGroupRoleMembers adds members to a group role.
	//
	// example:
	//  ctx := context.Background()
	//  members := []string{"member_id_1", "member_id_2"}
	//  members, _ := sdk.AddGroupRoleMembers(ctx, "groupID", "roleID", "domainID", members, "token")
	//  fmt.Println(members)
	AddGroupRoleMembers(ctx context.Context, id, roleID, domainID string, members []string, token string) ([]string, errors.SDKError)

	// GroupRoleMembers returns group role members by roleID.
	//
	// example:
	// ctx := context.Background()
	// pm := sdk.PageMetadata{
	//   Offset: 0,
	//  Limit:  10,
	// }
	//  members, _ := sdk.GroupRoleMembers(ctx, "groupID", "roleID", "domainID", "token")
	//  fmt.Println(members)
	GroupRoleMembers(ctx context.Context, id, roleID, domainID string, pm PageMetadata, token string) (RoleMembersPage, errors.SDKError)

	// RemoveGroupRoleMembers removes members from a group role.
	//
	// example:
	//  ctx := context.Background()
	//  members := []string{"member_id_1", "member_id_2"}
	//  err := sdk.RemoveGroupRoleMembers(ctx, "groupID", "roleID", "domainID", members, "token")
	//  fmt.Println(err)
	RemoveGroupRoleMembers(ctx context.Context, id, roleID, domainID string, members []string, token string) errors.SDKError

	// RemoveAllGroupRoleMembers removes all members from a group role.
	//
	// example:
	//  ctx := context.Background()
	//  err := sdk.RemoveAllGroupRoleMembers(ctx, "groupID", "roleID", "domainID", "token")
	//  fmt.Println(err)
	RemoveAllGroupRoleMembers(ctx context.Context, id, roleID, domainID, token string) errors.SDKError

	// AvailableGroupRoleActions returns available actions for a group role.
	//
	// example:
	//  ctx := context.Background()
	//  actions, _ := sdk.AvailableGroupRoleActions(ctx, "groupID", "token")
	//  fmt.Println(actions)
	AvailableGroupRoleActions(ctx context.Context, id, token string) ([]string, errors.SDKError)

	// ListGroupMembers list all members from all roles in a group .
	//
	// example:
	//	ctx := context.Background()
	//	pm := sdk.PageMetadata{
	//		Offset: 0,
	//		Limit:  10,
	//	}
	//  members, _ := sdk.ListGroupMembers(ctx, "group_id","domainID", pm, "token")
	//  fmt.Println(members)
	ListGroupMembers(ctx context.Context, groupID, domainID string, pm PageMetadata, token string) (EntityMembersPage, errors.SDKError)

	// CreateChannel creates new channel and returns its id.
	//
	// example:
	//  ctx := context.Background()
	//  channel := sdk.Channel{
	//    Name: "My Channel",
	//    Metadata: sdk.Metadata{
	//      "key": "value",
	//    },
	//  }
	//  channel, _ := sdk.CreateChannel(ctx, channel, "domainID", "token")
	//  fmt.Println(channel)
	CreateChannel(ctx context.Context, channel Channel, domainID, token string) (Channel, errors.SDKError)

	// CreateChannels creates new channels and returns their ids.
	//
	// example:
	//  ctx := context.Background()
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
	//  channels, _ := sdk.CreateChannels(ctx, channels, "domainID", "token")
	//  fmt.Println(channels)
	CreateChannels(ctx context.Context, channels []Channel, domainID, token string) ([]Channel, errors.SDKError)

	// Channels returns page of channels.
	//
	// example:
	//  ctx := context.Background()
	//  pm := sdk.PageMetadata{
	//    Offset: 0,
	//    Limit:  10,
	//    Name:   "My Channel",
	//  }
	//  channels, _ := sdk.Channels(ctx, pm, "domainID", "token")
	//  fmt.Println(channels)
	Channels(ctx context.Context, pm PageMetadata, domainID, token string) (ChannelsPage, errors.SDKError)

	// Channel returns channel data by id.
	//
	// example:
	//  ctx := context.Background()
	//  channel, _ := sdk.Channel(ctx, "channelID", "domainID", "token")
	//  fmt.Println(channel)
	Channel(ctx context.Context, id, domainID, token string) (Channel, errors.SDKError)

	// UpdateChannel updates existing channel.
	//
	// example:
	//  ctx := context.Background()
	//  channel := sdk.Channel{
	//    ID:   "channelID",
	//    Name: "My Channel",
	//    Metadata: sdk.Metadata{
	//      "key": "value",
	//    },
	//  }
	//  channel, _ := sdk.UpdateChannel(ctx, channel, "domainID", "token")
	//  fmt.Println(channel)
	UpdateChannel(ctx context.Context, channel Channel, domainID, token string) (Channel, errors.SDKError)

	// UpdateChannelTags updates the channel's tags.
	//
	// example:
	//  ctx := context.Background()
	//  channel := sdk.Channel{
	//    ID:   "channelID",
	//    Tags: []string{"tag1", "tag2"},
	//  }
	//  channel, _ := sdk.UpdateChannelTags(ctx, channel, "domainID", "token")
	//  fmt.Println(channel)
	UpdateChannelTags(ctx context.Context, c Channel, domainID, token string) (Channel, errors.SDKError)

	// EnableChannel changes channel status to enabled.
	//
	// example:
	//  ctx := context.Background()
	//  channel, _ := sdk.EnableChannel(ctx, "channelID", "domainID", "token")
	//  fmt.Println(channel)
	EnableChannel(ctx context.Context, id, domainID, token string) (Channel, errors.SDKError)

	// DisableChannel changes channel status to disabled - soft delete.
	//
	// example:
	//  ctx := context.Background()
	//  channel, _ := sdk.DisableChannel(ctx, "channelID", "domainID", "token")
	//  fmt.Println(channel)
	DisableChannel(ctx context.Context, id, domainID, token string) (Channel, errors.SDKError)

	// DeleteChannel delete given group id.
	//
	// example:
	//  ctx := context.Background()
	//  err := sdk.DeleteChannel(ctx, "channelID", "domainID", "token")
	//  fmt.Println(err)
	DeleteChannel(ctx context.Context, id, domainID, token string) errors.SDKError

	// SetChannelParent sets the parent group of a channel.
	//
	// example:
	//  ctx := context.Background()
	//  err := sdk.SetChannelParent(ctx, "channelID", "domainID", "groupID", "token")
	//  fmt.Println(err)
	SetChannelParent(ctx context.Context, id, domainID, groupID, token string) errors.SDKError

	// RemoveChannelParent removes the parent group of a channel.
	//
	// example:
	//  ctx := context.Background()
	//  err := sdk.RemoveChannelParent(ctx, "channelID", "domainID", "groupID", "token")
	//  fmt.Println(err)
	RemoveChannelParent(ctx context.Context, id, domainID, groupID, token string) errors.SDKError

	// Connect bulk connects clients to channels specified by id.
	//
	// example:
	//  ctx := context.Background()
	//  conns := sdk.Connection{
	//    ChannelIDs: []string{"channel_id_1"},
	//    ClientIDs:  []string{"client_id_1"},
	//    Types:   	  []string{"Publish", "Subscribe"},
	//  }
	//  err := sdk.Connect(ctx, conns, "domainID", "token")
	//  fmt.Println(err)
	Connect(ctx context.Context, conn Connection, domainID, token string) errors.SDKError

	// Disconnect
	//
	// example:
	//  ctx := context.Background()
	//  conns := sdk.Connection{
	//    ChannelIDs: []string{"channel_id_1"},
	//    ClientIDs:  []string{"client_id_1"},
	//    Types:   	  []string{"Publish", "Subscribe"},
	//  }
	//  err := sdk.Disconnect(ctx, conns, "domainID", "token")
	//  fmt.Println(err)
	Disconnect(ctx context.Context, conn Connection, domainID, token string) errors.SDKError

	// ConnectClient connects client to specified channel by id.
	//
	// example:
	//  ctx := context.Background()
	//  clientIDs := []string{"client_id_1", "client_id_2"}
	//  err := sdk.ConnectClients(ctx, "channelID", clientIDs, []string{"Publish", "Subscribe"}, "domainID", "token")
	//  fmt.Println(err)
	ConnectClients(ctx context.Context, channelID string, clientIDs, connTypes []string, domainID, token string) errors.SDKError

	// DisconnectClient disconnect client from specified channel by id.
	//
	// example:
	//  ctx := context.Background()
	//  clientIDs := []string{"client_id_1", "client_id_2"}
	//  err := sdk.DisconnectClients(ctx, "channelID", clientIDs, []string{"Publish", "Subscribe"}, "domainID", "token")
	//  fmt.Println(err)
	DisconnectClients(ctx context.Context, channelID string, clientIDs, connTypes []string, domainID, token string) errors.SDKError

	// ListChannelMembers list all members from all roles in a channel .
	//
	// example:
	//	ctx := context.Background()
	//	pm := sdk.PageMetadata{
	//		Offset: 0,
	//		Limit:  10,
	//	}
	//  members, _ := sdk.ListChannelMembers(ctx, "channel_id","domainID", pm, "token")
	//  fmt.Println(members)
	ListChannelMembers(ctx context.Context, channelID, domainID string, pm PageMetadata, token string) (EntityMembersPage, errors.SDKError)

	// SendMessage send message to specified channel.
	//
	// example:
	//  ctx := context.Background()
	//  msg := '[{"bn":"some-base-name:","bt":1.276020076001e+09, "bu":"A","bver":5, "n":"voltage","u":"V","v":120.1}, {"n":"current","t":-5,"v":1.2}, {"n":"current","t":-4,"v":1.3}]'
	//  err := sdk.SendMessage(ctx, "domainID", "topic", msg, "clientSecret")
	//  fmt.Println(err)
	SendMessage(ctx context.Context, domainID, topic, msg, secret string) errors.SDKError

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

	// CreateDomain creates new domain and returns its details.
	//
	// example:
	//  ctx := context.Background()
	//  domain := sdk.Domain{
	//    Name: "My Domain",
	//    Metadata: sdk.Metadata{
	//      "key": "value",
	//    },
	//  }
	//  domain, _ := sdk.CreateDomain(ctx, group, "token")
	//  fmt.Println(domain)
	CreateDomain(ctx context.Context, d Domain, token string) (Domain, errors.SDKError)

	// Domain retrieve domain information of given domain ID .
	//
	// example:
	//  ctx := context.Background()
	//  domain, _ := sdk.Domain(ctx, "domainID", "token")
	//  fmt.Println(domain)
	Domain(ctx context.Context, domainID, token string) (Domain, errors.SDKError)

	// UpdateDomain updates details of the given domain ID.
	//
	// example:
	//  ctx := context.Background()
	//  domain := sdk.Domain{
	//    ID : "domainID"
	//    Name: "New Domain Name",
	//    Metadata: sdk.Metadata{
	//      "key": "value",
	//    },
	//  }
	//  domain, _ := sdk.UpdateDomain(ctx, domain, "token")
	//  fmt.Println(domain)
	UpdateDomain(ctx context.Context, d Domain, token string) (Domain, errors.SDKError)

	// Domains returns list of domain for the given filters.
	//
	// example:
	//  ctx := context.Background()
	//  pm := sdk.PageMetadata{
	//    Offset: 0,
	//    Limit:  10,
	//    Name:   "My Domain",
	//    Permission : "view"
	//  }
	//  domains, _ := sdk.Domains(ctx, pm, "token")
	//  fmt.Println(domains)
	Domains(ctx context.Context, pm PageMetadata, token string) (DomainsPage, errors.SDKError)

	// EnableDomain changes the status of the domain to enabled.
	//
	// example:
	//  ctx := context.Background()
	//  err := sdk.EnableDomain(ctx, "domainID", "token")
	//  fmt.Println(err)
	EnableDomain(ctx context.Context, domainID, token string) errors.SDKError

	// DisableDomain changes the status of the domain to disabled.
	//
	// example:
	//  ctx := context.Background()
	//  err := sdk.DisableDomain(ctx, "domainID", "token")
	//  fmt.Println(err)
	DisableDomain(ctx context.Context, domainID, token string) errors.SDKError

	// FreezeDomain changes the status of the domain to frozen.
	//
	// example:
	//  ctx := context.Background()
	//  err := sdk.FreezeDomain(ctx, "domainID", "token")
	//  fmt.Println(err)
	FreezeDomain(ctx context.Context, domainID, token string) errors.SDKError

	// CreateDomainRole creates new domain role and returns its id.
	//
	// example:
	//  ctx := context.Background()
	//  rq := sdk.RoleReq{
	//    RoleName: "My Role",
	//    OptionalActions: []string{"read", "update"},
	//    OptionalMembers: []string{"member_id_1", "member_id_2"},
	//  }
	//  role, _ := sdk.CreateDomainRole(ctx, "domainID", rq, "token")
	//  fmt.Println(role)
	CreateDomainRole(ctx context.Context, id string, rq RoleReq, token string) (Role, errors.SDKError)

	// DomainRoles returns domain roles.
	//
	// example:
	//  ctx := context.Background()
	//  pm := sdk.PageMetadata{
	//   Offset: 0,
	//   Limit:  10,
	// }
	//  roles, _ := sdk.DomainRoles(ctx, "domainID", pm, "token")
	//  fmt.Println(roles)
	DomainRoles(ctx context.Context, id string, pm PageMetadata, token string) (RolesPage, errors.SDKError)

	// DomainRole returns domain role object by roleID.
	//
	// example:
	//  ctx := context.Background()
	//  role, _ := sdk.DomainRole(ctx, "domainID", "roleID", "token")
	//  fmt.Println(role)
	DomainRole(ctx context.Context, id, roleID, token string) (Role, errors.SDKError)

	// UpdateDomainRole updates existing domain role name.
	//
	// example:
	//  ctx := context.Background()
	//  role, _ := sdk.UpdateDomainRole(ctx, "domainID", "roleID", "newName", "token")
	//  fmt.Println(role)
	UpdateDomainRole(ctx context.Context, id, roleID, newName string, token string) (Role, errors.SDKError)

	// DeleteDomainRole deletes a domain role with the given domainID and roleID.
	//
	// example:
	//  ctx := context.Background()
	//  err := sdk.DeleteDomainRole(ctx, "domainID", "roleID", "token")
	//  fmt.Println(err)
	DeleteDomainRole(ctx context.Context, id, roleID, token string) errors.SDKError

	// AddDomainRoleActions adds actions to a domain role.
	//
	// example:
	//  ctx := context.Background()
	//  actions := []string{"read", "update"}
	//  actions, _ := sdk.AddDomainRoleActions(ctx, "domainID", "roleID", actions, "token")
	//  fmt.Println(actions)
	AddDomainRoleActions(ctx context.Context, id, roleID string, actions []string, token string) ([]string, errors.SDKError)

	// DomainRoleActions returns domain role actions by roleID.
	//
	// example:
	//  ctx := context.Background()
	//  actions, _ := sdk.DomainRoleActions(ctx, "domainID", "roleID", "token")
	//  fmt.Println(actions)
	DomainRoleActions(ctx context.Context, id, roleID string, token string) ([]string, errors.SDKError)

	// RemoveDomainRoleActions removes actions from a domain role.
	//
	// example:
	//  ctx := context.Background()
	//  actions := []string{"read", "update"}
	//  err := sdk.RemoveDomainRoleActions(ctx, "domainID", "roleID", actions, "token")
	//  fmt.Println(err)
	RemoveDomainRoleActions(ctx context.Context, id, roleID string, actions []string, token string) errors.SDKError

	// RemoveAllDomainRoleActions removes all actions from a domain role.
	//
	// example:
	//  ctx := context.Background()
	//  err := sdk.RemoveAllDomainRoleActions(ctx, "domainID", "roleID", "token")
	//  fmt.Println(err)
	RemoveAllDomainRoleActions(ctx context.Context, id, roleID, token string) errors.SDKError

	// AddDomainRoleMembers adds members to a domain role.
	//
	// example:
	//  ctx := context.Background()
	//  members := []string{"member_id_1", "member_id_2"}
	//  members, _ := sdk.AddDomainRoleMembers(ctx, "domainID", "roleID", members, "token")
	//  fmt.Println(members)
	AddDomainRoleMembers(ctx context.Context, id, roleID string, members []string, token string) ([]string, errors.SDKError)

	// DomainRoleMembers returns domain role members by roleID.
	//
	// example:
	//  ctx := context.Background()
	//  pm := sdk.PageMetadata{
	//    Offset: 0,
	//    Limit:  10,
	//  }
	//  members, _ := sdk.DomainRoleMembers(ctx, "domainID", "roleID", "token")
	//  fmt.Println(members)
	DomainRoleMembers(ctx context.Context, id, roleID string, pm PageMetadata, token string) (RoleMembersPage, errors.SDKError)

	// RemoveDomainRoleMembers removes members from a domain role.
	//
	// example:
	//  ctx := context.Background()
	//  members := []string{"member_id_1", "member_id_2"}
	//  err := sdk.RemoveDomainRoleMembers(ctx, "domainID", "roleID", members, "token")
	//  fmt.Println(err)
	RemoveDomainRoleMembers(ctx context.Context, id, roleID string, members []string, token string) errors.SDKError

	// RemoveAllDomainRoleMembers removes all members from a domain role.
	//
	// example:
	//  ctx := context.Background()
	//  err := sdk.RemoveAllDomainRoleMembers(ctx, "domainID", "roleID", "token")
	//  fmt.Println(err)
	RemoveAllDomainRoleMembers(ctx context.Context, id, roleID, token string) errors.SDKError

	// AvailableDomainRoleActions returns available actions for a domain role.
	//
	// example:
	//  ctx := context.Background()
	//  actions, _ := sdk.AvailableDomainRoleActions(ctx, "token")
	//  fmt.Println(actions)
	AvailableDomainRoleActions(ctx context.Context, token string) ([]string, errors.SDKError)

	// ListDomainUsers returns list of users for the given domain ID and filters.
	//
	// example:
	//  ctx := context.Background()
	//  pm := sdk.PageMetadata{
	//    Offset: 0,
	//    Limit:  10,
	//  }
	//  members, _ := sdk.ListDomainMembers(ctx, "domain_id", pm, "token")
	//  fmt.Println(members)
	ListDomainMembers(ctx context.Context, domainID string, pm PageMetadata, token string) (EntityMembersPage, errors.SDKError)

	// SendInvitation sends an invitation to the email address associated with the given user.
	//
	// For example:
	//  ctx := context.Background()
	//  invitation := sdk.Invitation{
	//    DomainID: "domainID",
	//    UserID:   "userID",
	//    Relation: "contributor", // available options: "owner", "admin", "editor", "contributor", "guest"
	//  }
	//  err := sdk.SendInvitation(ctx, invitation, "token")
	//  fmt.Println(err)
	SendInvitation(ctx context.Context, invitation Invitation, token string) (err error)

	// Invitations returns a list of invitations.
	//
	// For example:
	//  ctx := context.Background()
	//  invitations, _ := sdk.Invitations(ctx, PageMetadata{Offset: 0, Limit: 10}, "token")
	//  fmt.Println(invitations)
	Invitations(ctx context.Context, pm PageMetadata, token string) (invitations InvitationPage, err error)

	// AcceptInvitation accepts an invitation by adding the user to the domain that they were invited to.
	//
	// For example:
	//  ctx := context.Background()
	//  err := sdk.AcceptInvitation(ctx, "domainID", "token")
	//  fmt.Println(err)
	AcceptInvitation(ctx context.Context, domainID, token string) (err error)

	// RejectInvitation rejects an invitation.
	//
	// For example:
	//  ctx := context.Background()
	//  err := sdk.RejectInvitation(ctx, "domainID", "token")
	//  fmt.Println(err)
	RejectInvitation(ctx context.Context, domainID, token string) (err error)

	// DeleteInvitation deletes an invitation.
	//
	// For example:
	//  ctx := context.Background()
	//  err := sdk.DeleteInvitation(ctx, "userID", "domainID", "token")
	//  fmt.Println(err)
	DeleteInvitation(ctx context.Context, userID, domainID, token string) (err error)

	// Journal returns a list of journal logs.
	//
	// For example:
	//  ctx := context.Background()
	//  journals, _ := sdk.Journal(ctx, "client", "clientID","domainID", PageMetadata{Offset: 0, Limit: 10, Operation: "client.create"}, "token")
	//  fmt.Println(journals)
	Journal(ctx context.Context, entityType, entityID, domainID string, pm PageMetadata, token string) (journal JournalsPage, err error)

	// DomainInvitations returns a list of invitations for a specific domain.
	// For example:
	//  ctx := context.Background()
	//  pm := sdk.PageMetadata{
	//    Offset: 0,
	//    Limit:  10,
	//  }
	//  invitations, _ := sdk.DomainInvitations(ctx, "domainID", pm, "token")
	//  fmt.Println(invitations)
	DomainInvitations(ctx context.Context, pm PageMetadata, token, domainID string) (invitations InvitationPage, err error)

	// AddBootstrap add bootstrap configuration
	AddBootstrap(ctx context.Context, cfg BootstrapConfig, domainID, token string) (string, errors.SDKError)

	// ViewBootstrap returns Client Config with given ID belonging to the user identified by the given token.
	ViewBootstrap(ctx context.Context, id, domainID, token string) (BootstrapConfig, errors.SDKError)

	// UpdateBootstrap updates editable fields of the provided Config.
	UpdateBootstrap(ctx context.Context, cfg BootstrapConfig, domainID, token string) errors.SDKError

	// UpdateBootstrapCerts updates bootstrap config certificates.
	UpdateBootstrapCerts(ctx context.Context, id string, clientCert, clientKey, ca string, domainID, token string) (BootstrapConfig, errors.SDKError)

	// UpdateBootstrapConnection updates connections performs update of the channel list corresponding Client is connected to.
	UpdateBootstrapConnection(ctx context.Context, id string, channels []string, domainID, token string) errors.SDKError

	// RemoveBootstrap removes Config with specified token that belongs to the user identified by the given token.
	RemoveBootstrap(ctx context.Context, id, domainID, token string) errors.SDKError

	// Bootstrap returns Config to the Client with provided external ID using external key.
	Bootstrap(ctx context.Context, externalID, externalKey string) (BootstrapConfig, errors.SDKError)

	// BootstrapSecure retrieves a configuration with given external ID and encrypted external key.
	BootstrapSecure(ctx context.Context, externalID, externalKey, cryptoKey string) (BootstrapConfig, errors.SDKError)

	// Bootstraps retrieves a list of managed configs.
	Bootstraps(ctx context.Context, pm PageMetadata, domainID, token string) (BootstrapPage, errors.SDKError)

	// Whitelist updates Client state Config with given ID belonging to the user identified by the given token.
	Whitelist(ctx context.Context, clientID string, state int, domainID, token string) errors.SDKError

	// ReadMessages reads messages of specified channel.
	ReadMessages(ctx context.Context, pm MessagePageMetadata, chanID, domainID, token string) (MessagesPage, errors.SDKError)

	// CreateSubscription creates a new subscription.
	CreateSubscription(ctx context.Context, topic, contact, token string) (string, errors.SDKError)

	// ListSubscriptions list subscriptions given list parameters.
	ListSubscriptions(ctx context.Context, pm PageMetadata, token string) (SubscriptionPage, errors.SDKError)

	// ViewSubscription retrieves a subscription with the provided id.
	ViewSubscription(ctx context.Context, id, token string) (Subscription, errors.SDKError)

	// DeleteSubscription removes a subscription with the provided id.
	DeleteSubscription(ctx context.Context, id, token string) errors.SDKError

	// UpdateAlarm updates an existing alarm.
	UpdateAlarm(ctx context.Context, alarm Alarm, domainID, token string) (Alarm, errors.SDKError)

	// ViewAlarm retrieves an alarm by its ID.
	ViewAlarm(ctx context.Context, id, domainID, token string) (Alarm, errors.SDKError)

	// ListAlarms retrieves a page of alarms.
	ListAlarms(ctx context.Context, pm PageMetadata, domainID, token string) (AlarmsPage, errors.SDKError)

	// DeleteAlarm deletes an alarm.
	DeleteAlarm(ctx context.Context, id, domainID, token string) errors.SDKError

	// AddReportConfig creates a new report configuration.
	AddReportConfig(ctx context.Context, cfg ReportConfig, domainID, token string) (ReportConfig, errors.SDKError)

	// ViewReportConfig retrieves a report config by its ID.
	ViewReportConfig(ctx context.Context, id, domainID, token string) (ReportConfig, errors.SDKError)

	// UpdateReportConfig updates an existing report configuration.
	UpdateReportConfig(ctx context.Context, cfg ReportConfig, domainID, token string) (ReportConfig, errors.SDKError)

	// UpdateReportSchedule updates an existing report configuration's schedule.
	UpdateReportSchedule(ctx context.Context, cfg ReportConfig, domainID, token string) (ReportConfig, errors.SDKError)

	// RemoveReportConfig deletes a report config.
	RemoveReportConfig(ctx context.Context, id, domainID, token string) errors.SDKError

	// ListReportsConfig retrieves a page of report configs.
	ListReportsConfig(ctx context.Context, pm PageMetadata, domainID, token string) (ReportConfigPage, errors.SDKError)

	// EnableReportConfig enables a report config.
	EnableReportConfig(ctx context.Context, id, domainID, token string) (ReportConfig, errors.SDKError)

	// DisableReportConfig disables a report config.
	DisableReportConfig(ctx context.Context, id, domainID, token string) (ReportConfig, errors.SDKError)

	// UpdateReportTemplate updates a report template.
	UpdateReportTemplate(ctx context.Context, cfg ReportConfig, domainID, token string) errors.SDKError

	// ViewReportTemplate retrieves a report template.
	ViewReportTemplate(ctx context.Context, id, domainID, token string) (ReportTemplate, errors.SDKError)

	// DeleteReportTemplate deletes a report template.
	DeleteReportTemplate(ctx context.Context, id, domainID, token string) errors.SDKError

	// GenerateReport generates a report from a configuration.
	GenerateReport(ctx context.Context, config ReportConfig, action ReportAction, domainID, token string) (ReportPage, *ReportFile, errors.SDKError)

	// AddRule creates a new rule.
	AddRule(ctx context.Context, r Rule, domainID, token string) (Rule, errors.SDKError)

	// ViewRule retrieves a rule by its ID.
	ViewRule(ctx context.Context, id, domainID, token string) (Rule, errors.SDKError)

	// UpdateRule updates an existing rule.
	UpdateRule(ctx context.Context, r Rule, domainID, token string) (Rule, errors.SDKError)

	// UpdateRuleTags updates an existing rule's tags.
	UpdateRuleTags(ctx context.Context, r Rule, domainID, token string) (Rule, errors.SDKError)

	// UpdateRuleSchedule updates an existing rule's schedule.
	UpdateRuleSchedule(ctx context.Context, r Rule, domainID, token string) (Rule, errors.SDKError)

	// ListRules retrieves a page of rules.
	ListRules(ctx context.Context, pm PageMetadata, domainID, token string) (Page, errors.SDKError)

	// RemoveRule deletes a rule.
	RemoveRule(ctx context.Context, id, domainID, token string) errors.SDKError

	// EnableRule enables a rule.
	EnableRule(ctx context.Context, id, domainID, token string) (Rule, errors.SDKError)

	// DisableRule disables a rule.
	DisableRule(ctx context.Context, id, domainID, token string) (Rule, errors.SDKError)

	// IssueCert issues a certificate for an entity.
	//
	// example:
	//  cert, _ := sdk.IssueCert(context.Background(), "entityID", "8760h", []string{"127.0.0.1"}, sdk.Options{CommonName: "cn"}, "domainID", "token")
	IssueCert(ctx context.Context, entityID, ttl string, ipAddrs []string, opts Options, domainID, token string) (Certificate, errors.SDKError)

	// RevokeCert revokes a certificate by serial number.
	//
	// example:
	//  err := sdk.RevokeCert(context.Background(), "serialNumber", "domainID", "token")
	RevokeCert(ctx context.Context, serialNumber, domainID, token string) errors.SDKError

	// RenewCert renews a certificate by serial number.
	//
	// example:
	//  cert, _ := sdk.RenewCert(context.Background(), "serialNumber", "domainID", "token")
	RenewCert(ctx context.Context, serialNumber, domainID, token string) (Certificate, errors.SDKError)

	// ListCerts lists certificates matching the given metadata filter.
	//
	// example:
	//  page, _ := sdk.ListCerts(context.Background(), sdk.PageMetadata{Limit: 10}, "domainID", "token")
	ListCerts(ctx context.Context, pm PageMetadata, domainID, token string) (CertificatePage, errors.SDKError)

	// DeleteCert deletes all certificates for the given entity ID.
	//
	// example:
	//  err := sdk.DeleteCert(context.Background(), "entityID", "domainID", "token")
	DeleteCert(ctx context.Context, entityID, domainID, token string) errors.SDKError

	// ViewCert retrieves a certificate by serial number.
	//
	// example:
	//  cert, _ := sdk.ViewCert(context.Background(), "serialNumber", "domainID", "token")
	ViewCert(ctx context.Context, serialNumber, domainID, token string) (Certificate, errors.SDKError)

	// OCSP checks the revocation status of a certificate.
	//
	// example:
	//  resp, _ := sdk.OCSP(context.Background(), "serialNumber", "")
	OCSP(ctx context.Context, serialNumber, cert string) (OCSPResponse, errors.SDKError)

	// ViewCA views the signing CA certificate.
	//
	// example:
	//  cert, _ := sdk.ViewCA(context.Background())
	ViewCA(ctx context.Context) (Certificate, errors.SDKError)

	// DownloadCA downloads the signing CA certificate bundle.
	//
	// example:
	//  bundle, _ := sdk.DownloadCA(context.Background())
	DownloadCA(ctx context.Context) (CertificateBundle, errors.SDKError)

	// IssueFromCSR issues a certificate from a provided CSR.
	//
	// example:
	//  cert, _ := sdk.IssueFromCSR(context.Background(), "entityID", "8760h", csrPEM, "domainID", "token")
	IssueFromCSR(ctx context.Context, entityID, ttl, csr, domainID, token string) (Certificate, errors.SDKError)

	// IssueFromCSRInternal issues a certificate from a CSR using agent authentication.
	//
	// example:
	//  cert, _ := sdk.IssueFromCSRInternal(context.Background(), "entityID", "8760h", csrPEM, "agentToken")
	IssueFromCSRInternal(ctx context.Context, entityID, ttl, csr, token string) (Certificate, errors.SDKError)

	// GenerateCRL generates a Certificate Revocation List.
	//
	// example:
	//  crl, _ := sdk.GenerateCRL(context.Background())
	GenerateCRL(ctx context.Context) ([]byte, errors.SDKError)

	// RevokeAll revokes all certificates for an entity ID.
	//
	// example:
	//  err := sdk.RevokeAll(context.Background(), "entityID", "domainID", "token")
	RevokeAll(ctx context.Context, entityID, domainID, token string) errors.SDKError

	// EntityID gets the entity ID for a certificate by serial number.
	//
	// example:
	//  id, _ := sdk.EntityID(context.Background(), "serialNumber", "domainID", "token")
	EntityID(ctx context.Context, serialNumber, domainID, token string) (string, errors.SDKError)

	// CreateCSR creates a Certificate Signing Request from metadata and a private key.
	//
	// example:
	//  csr, _ := sdk.CreateCSR(context.Background(), metadata, privateKeyBytes)
	CreateCSR(ctx context.Context, metadata certs.CSRMetadata, privKey any) (certs.CSR, errors.SDKError)
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
	bootstrapURL   string
	readersURL     string
	alarmsURL      string
	reportsURL     string
	rulesEngineURL string

	msgContentType ContentType
	client         *http.Client
	curlFlag       bool
	roles          bool
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
	BootstrapURL   string
	ReaderURL      string
	AlarmsURL      string
	ReportsURL     string
	RulesEngineURL string

	MsgContentType  ContentType
	TLSVerification bool
	CurlFlag        bool
	Roles           bool
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
		bootstrapURL:   conf.BootstrapURL,
		readersURL:     conf.ReaderURL,
		alarmsURL:      conf.AlarmsURL,
		reportsURL:     conf.ReportsURL,
		rulesEngineURL: conf.RulesEngineURL,

		msgContentType: conf.MsgContentType,
		client: &http.Client{Transport: otelhttp.NewTransport(&http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: !conf.TLSVerification,
			},
			DisableKeepAlives: true,
		})},
		curlFlag: conf.CurlFlag,
		roles:    conf.Roles,
	}
}

// processRequest creates and send a new HTTP request, and checks for errors in the HTTP response.
// It then returns the response headers, the response body, and the associated error(s) (if any).
func (sdk mgSDK) processRequest(ctx context.Context, method, reqUrl, token string, data []byte, headers map[string]string, expectedRespCodes ...int) (http.Header, []byte, errors.SDKError) {
	if sdk.roles {
		reqUrl = fmt.Sprintf("%s?roles=%v", reqUrl, true)
	}
	req, err := http.NewRequestWithContext(ctx, method, reqUrl, bytes.NewReader(data))
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
			token = fmt.Sprintf("%s%s", BearerPrefix, token)
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

	sdkErr := errors.CheckError(resp, expectedRespCodes...)
	if sdkErr != nil {
		return make(http.Header), []byte{}, sdkErr
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
	if len(pm.Tags.Elements) > 0 {
		switch pm.Tags.Operator {
		case AndOp:
			str := strings.Join(pm.Tags.Elements, "-")
			q.Add("tags", str)
		default:
			str := strings.Join(pm.Tags.Elements, ",")
			q.Add("tags", str)
		}
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
	if !pm.CreatedFrom.IsZero() {
		q.Add("created_from", pm.CreatedFrom.Format(time.RFC3339))
	}
	if !pm.CreatedTo.IsZero() {
		q.Add("created_to", pm.CreatedTo.Format(time.RFC3339))
	}
	q.Add("with_attributes", strconv.FormatBool(pm.WithAttributes))
	q.Add("with_metadata", strconv.FormatBool(pm.WithMetadata))
	if pm.EntityID != "" {
		q.Add("entity_id", pm.EntityID)
	}
	if pm.CommonName != "" {
		q.Add("common_name", pm.CommonName)
	}
	if pm.TTL != "" {
		q.Add("ttl", pm.TTL)
	}

	return q.Encode(), nil
}
