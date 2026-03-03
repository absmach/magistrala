// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package channels

import (
	"context"
	"strings"
	"time"

	"github.com/absmach/supermq/internal/nullable"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/connections"
	"github.com/absmach/supermq/pkg/roles"
)

// Metadata represents arbitrary JSON.
type Metadata map[string]any

// Channel represents a SuperMQ "communication topic". This topic
// contains the clients that can exchange messages between each other.
type Channel struct {
	ID          string    `json:"id"`
	Name        string    `json:"name,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	ParentGroup string    `json:"parent_group_id,omitempty"`
	Domain      string    `json:"domain_id,omitempty"`
	Route       string    `json:"route,omitempty"`
	Metadata    Metadata  `json:"metadata,omitempty"`
	CreatedBy   string    `json:"created_by,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
	UpdatedBy   string    `json:"updated_by,omitempty"`
	Status      Status    `json:"status,omitempty"` // 1 for enabled, 0 for disabled
	// Extended
	ParentGroupPath           string                    `json:"parent_group_path,omitempty"`
	RoleID                    string                    `json:"role_id,omitempty"`
	RoleName                  string                    `json:"role_name,omitempty"`
	Actions                   []string                  `json:"actions,omitempty"`
	AccessType                string                    `json:"access_type,omitempty"`
	AccessProviderId          string                    `json:"access_provider_id,omitempty"`
	AccessProviderRoleId      string                    `json:"access_provider_role_id,omitempty"`
	AccessProviderRoleName    string                    `json:"access_provider_role_name,omitempty"`
	AccessProviderRoleActions []string                  `json:"access_provider_role_actions,omitempty"`
	ConnectionTypes           []connections.ConnType    `json:"connection_types,omitempty"`
	MemberId                  string                    `json:"member_id,omitempty"`
	Roles                     []roles.MemberRoleActions `json:"roles,omitempty"`
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

type Page struct {
	Total          uint64                 `json:"total"`
	Offset         uint64                 `json:"offset"`
	Limit          uint64                 `json:"limit"`
	OnlyTotal      bool                   `json:"only_total"`
	Order          string                 `json:"order,omitempty"`
	Dir            string                 `json:"dir,omitempty"`
	ID             string                 `json:"id,omitempty"`
	Name           string                 `json:"name,omitempty"`
	Metadata       Metadata               `json:"metadata,omitempty"`
	Domain         string                 `json:"domain,omitempty"`
	Tags           TagsQuery              `json:"tags,omitempty"`
	Status         Status                 `json:"status,omitempty"`
	Group          nullable.Value[string] `json:"group,omitempty"`
	Client         string                 `json:"client,omitempty"`
	ConnectionType string                 `json:"connection_type,omitempty"`
	RoleName       string                 `json:"role_name,omitempty"`
	RoleID         string                 `json:"role_id,omitempty"`
	Actions        []string               `json:"actions,omitempty"`
	AccessType     string                 `json:"access_type,omitempty"`
	IDs            []string               `json:"-"`
}

// ChannelsPage contains page related metadata as well as list of channels that
// belong to this page.
type ChannelsPage struct {
	Page
	Channels []Channel
}

type Connection struct {
	ClientID  string
	ChannelID string
	DomainID  string
	Type      connections.ConnType
}

type AuthzReq struct {
	DomainID   string
	ChannelID  string
	ClientID   string
	ClientType string
	Type       connections.ConnType
}

type Service interface {
	// CreateChannels adds channels to the user.
	CreateChannels(ctx context.Context, session authn.Session, channels ...Channel) ([]Channel, []roles.RoleProvision, error)

	// ViewChannel retrieves data about the channel identified by the provided
	// ID, that belongs to the user.
	ViewChannel(ctx context.Context, session authn.Session, id string, withRoles bool) (Channel, error)

	// UpdateChannel updates the channel identified by the provided ID, that
	// belongs to the user.
	UpdateChannel(ctx context.Context, session authn.Session, channel Channel) (Channel, error)

	// UpdateChannelTags updates the channel's tags.
	UpdateChannelTags(ctx context.Context, session authn.Session, channel Channel) (Channel, error)

	EnableChannel(ctx context.Context, session authn.Session, id string) (Channel, error)

	DisableChannel(ctx context.Context, session authn.Session, id string) (Channel, error)

	// ListChannels retrieves data about subset of channels that belongs to the user.
	ListChannels(ctx context.Context, session authn.Session, pm Page) (ChannelsPage, error)

	// ListUserChannels retrieves data about subset of channels that belong to the specified user.
	ListUserChannels(ctx context.Context, session authn.Session, userID string, pm Page) (ChannelsPage, error)

	// RemoveChannel removes the client identified by the provided ID, that
	// belongs to the user.
	RemoveChannel(ctx context.Context, session authn.Session, id string) error

	// Connect adds clients to the channels list of connected clients.
	Connect(ctx context.Context, session authn.Session, chIDs, clIDs []string, connType []connections.ConnType) error

	// Disconnect removes clients from the channels list of connected clients.
	Disconnect(ctx context.Context, session authn.Session, chIDs, clIDs []string, connType []connections.ConnType) error

	SetParentGroup(ctx context.Context, session authn.Session, parentGroupID string, id string) error

	RemoveParentGroup(ctx context.Context, session authn.Session, id string) error

	roles.RoleManager
}

// ChannelRepository specifies a channel persistence API.
type Repository interface {
	// Save persists multiple channels. Channels are saved using a transaction. If one channel
	// fails then none will be saved. Successful operation is indicated by non-nil error response.
	Save(ctx context.Context, chs ...Channel) ([]Channel, error)

	// Update performs an update to the existing channel.
	Update(ctx context.Context, c Channel) (Channel, error)

	UpdateTags(ctx context.Context, ch Channel) (Channel, error)

	ChangeStatus(ctx context.Context, channel Channel) (Channel, error)

	// RetrieveUserChannels retrieves the channel of given domainID and userID.
	RetrieveUserChannels(ctx context.Context, domainID, userID string, pm Page) (ChannelsPage, error)

	// RetrieveByID retrieves the channel having the provided identifier
	RetrieveByID(ctx context.Context, id string) (Channel, error)

	// RetrieveByRoute retrieves the channel having the provided route
	RetrieveByRoute(ctx context.Context, route, domainID string) (Channel, error)

	// RetrieveByIDWithRoles retrieves channel by its unique ID along with member roles.
	RetrieveByIDWithRoles(ctx context.Context, id, memberID string) (Channel, error)

	// RetrieveAll retrieves the subset of channels.
	RetrieveAll(ctx context.Context, pm Page) (ChannelsPage, error)

	// Remove removes the channel having the provided identifier
	Remove(ctx context.Context, ids ...string) error

	// SetParentGroup set parent group id to a given channel id
	SetParentGroup(ctx context.Context, ch Channel) error

	// RemoveParentGroup remove parent group id fr given chanel id
	RemoveParentGroup(ctx context.Context, ch Channel) error

	AddConnections(ctx context.Context, conns []Connection) error

	RemoveConnections(ctx context.Context, conns []Connection) error

	CheckConnection(ctx context.Context, conn Connection) error

	ClientAuthorize(ctx context.Context, conn Connection) error

	ChannelConnectionsCount(ctx context.Context, id string) (uint64, error)

	DoesChannelHaveConnections(ctx context.Context, id string) (bool, error)

	RemoveClientConnections(ctx context.Context, clientID string) error

	RemoveChannelConnections(ctx context.Context, channelID string) error

	RetrieveParentGroupChannels(ctx context.Context, parentGroupID string) ([]Channel, error)

	UnsetParentGroupFromChannels(ctx context.Context, parentGroupID string) error

	roles.Repository
}

// Cache contains channels caching interface.
type Cache interface {
	// Save stores the channelID for the given domain ID and channel route.
	Save(ctx context.Context, channelRoute, domainID, channelID string) error

	// ID retrieves the channelID for the given domain ID and channel route.
	ID(ctx context.Context, channelRoute, domainID string) (string, error)

	// Remove removes the channel ID for the given domain ID and channel route.
	Remove(ctx context.Context, channelRoute, domainID string) error
}
