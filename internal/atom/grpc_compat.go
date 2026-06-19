// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package atom

import (
	"context"
	"net/http"
	"strings"

	channelsv1 "github.com/absmach/magistrala/api/grpc/channels/v1"
	clientsv1 "github.com/absmach/magistrala/api/grpc/clients/v1"
	commonv1 "github.com/absmach/magistrala/api/grpc/common/v1"
	domainsv1 "github.com/absmach/magistrala/api/grpc/domains/v1"
	smqauthn "github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/connections"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AtomClientsCompat struct {
	Authn  smqauthn.Authentication
	Client *Client
}

func NewClientsCompat(authn smqauthn.Authentication, client ...*Client) clientsv1.ClientsServiceClient {
	atomClient := NewClient(LoadConfig())
	if len(client) > 0 && client[0] != nil {
		atomClient = client[0]
	}
	return AtomClientsCompat{Authn: authn, Client: atomClient}
}

func (c AtomClientsCompat) Authenticate(ctx context.Context, in *clientsv1.AuthnReq, _ ...grpc.CallOption) (*clientsv1.AuthnRes, error) {
	token := in.GetToken()
	if prefix, id, key, err := smqauthn.AuthUnpack(token); err == nil {
		switch prefix {
		case smqauthn.BasicAuth:
			res, loginErr := c.Client.LoginPassword(ctx, id, key)
			if loginErr == nil {
				return &clientsv1.AuthnRes{Authenticated: true, Id: res.EntityID}, nil
			}
			if !isAtomUnauthorized(loginErr) {
				return nil, loginErr
			}
			token = key
		case smqauthn.DomainAuth:
			token = key
		case smqauthn.Unknown:
			token = key
		}
	}
	session, err := c.Authn.Authenticate(ctx, token)
	if err != nil {
		return nil, err
	}
	return &clientsv1.AuthnRes{Authenticated: true, Id: session.UserID}, nil
}

func isAtomUnauthorized(err error) bool {
	atomErr, ok := err.(Error)
	return ok && atomErr.StatusCode == http.StatusUnauthorized
}

func (c AtomClientsCompat) RetrieveEntity(context.Context, *commonv1.RetrieveEntityReq, ...grpc.CallOption) (*commonv1.RetrieveEntityRes, error) {
	return nil, status.Error(codes.Unimplemented, "atom clients compatibility only supports Authenticate")
}

func (c AtomClientsCompat) RetrieveEntities(context.Context, *commonv1.RetrieveEntitiesReq, ...grpc.CallOption) (*commonv1.RetrieveEntitiesRes, error) {
	return nil, status.Error(codes.Unimplemented, "atom clients compatibility only supports Authenticate")
}

func (c AtomClientsCompat) AddConnections(context.Context, *commonv1.AddConnectionsReq, ...grpc.CallOption) (*commonv1.AddConnectionsRes, error) {
	return nil, status.Error(codes.Unimplemented, "atom clients compatibility only supports Authenticate")
}

func (c AtomClientsCompat) RemoveConnections(context.Context, *commonv1.RemoveConnectionsReq, ...grpc.CallOption) (*commonv1.RemoveConnectionsRes, error) {
	return nil, status.Error(codes.Unimplemented, "atom clients compatibility only supports Authenticate")
}

func (c AtomClientsCompat) RemoveChannelConnections(context.Context, *clientsv1.RemoveChannelConnectionsReq, ...grpc.CallOption) (*clientsv1.RemoveChannelConnectionsRes, error) {
	return nil, status.Error(codes.Unimplemented, "atom clients compatibility only supports Authenticate")
}

func (c AtomClientsCompat) UnsetParentGroupFromClient(context.Context, *clientsv1.UnsetParentGroupFromClientReq, ...grpc.CallOption) (*clientsv1.UnsetParentGroupFromClientRes, error) {
	return nil, status.Error(codes.Unimplemented, "atom clients compatibility only supports Authenticate")
}

type AtomDomainsCompat struct {
	Client *Client
}

func NewDomainsCompat(client *Client) domainsv1.DomainsServiceClient {
	return AtomDomainsCompat{Client: client}
}

func (c AtomDomainsCompat) DeleteUserFromDomains(context.Context, *domainsv1.DeleteUserReq, ...grpc.CallOption) (*domainsv1.DeleteUserRes, error) {
	return nil, status.Error(codes.Unimplemented, "atom domains compatibility does not delete user memberships")
}

func (c AtomDomainsCompat) RetrieveStatus(ctx context.Context, in *commonv1.RetrieveEntityReq, _ ...grpc.CallOption) (*commonv1.RetrieveEntityRes, error) {
	tenant, err := c.Client.GetTenant(ctx, in.GetId())
	if err != nil {
		return nil, err
	}
	return &commonv1.RetrieveEntityRes{Entity: &commonv1.EntityBasic{
		Id:     tenant.ID,
		Status: atomStatusCode(tenant.Status),
	}}, nil
}

func (c AtomDomainsCompat) RetrieveIDByRoute(ctx context.Context, in *commonv1.RetrieveIDByRouteReq, _ ...grpc.CallOption) (*commonv1.RetrieveEntityRes, error) {
	tenants, err := c.Client.ListTenants(ctx, Query{Route: in.GetRoute(), Limit: 1})
	if err != nil {
		return nil, err
	}
	if len(tenants.Items) == 0 {
		return nil, status.Errorf(codes.NotFound, "tenant route %q not found", in.GetRoute())
	}
	tenant := tenants.Items[0]
	return &commonv1.RetrieveEntityRes{Entity: &commonv1.EntityBasic{
		Id:     tenant.ID,
		Status: atomStatusCode(tenant.Status),
	}}, nil
}

type AtomChannelsCompat struct {
	Client Authorizer
	Atom   *Client
}

func NewChannelsCompat(client Authorizer) channelsv1.ChannelsServiceClient {
	atomClient, _ := client.(*Client)
	return AtomChannelsCompat{Client: client, Atom: atomClient}
}

func (c AtomChannelsCompat) Authorize(ctx context.Context, in *channelsv1.AuthzReq, _ ...grpc.CallOption) (*channelsv1.AuthzRes, error) {
	action := "subscribe"
	if connections.ConnType(in.GetType()) == connections.Publish {
		action = "publish"
	}
	subjectID := strings.TrimPrefix(in.GetClientId(), in.GetDomainId()+"_")
	resp, err := c.Client.CheckAuthz(ctx, AuthzRequest{
		SubjectID:  subjectID,
		Action:     action,
		ResourceID: in.GetChannelId(),
		ObjectKind: atomObjectKindResource,
		ObjectID:   in.GetChannelId(),
		Context: map[string]any{
			"domain_id": in.GetDomainId(),
		},
	})
	if err != nil {
		return nil, err
	}
	return &channelsv1.AuthzRes{Authorized: resp.Allowed}, nil
}

func (c AtomChannelsCompat) RemoveClientConnections(context.Context, *channelsv1.RemoveClientConnectionsReq, ...grpc.CallOption) (*channelsv1.RemoveClientConnectionsRes, error) {
	return nil, status.Error(codes.Unimplemented, "atom channels compatibility only supports Authorize")
}

func (c AtomChannelsCompat) UnsetParentGroupFromChannels(context.Context, *channelsv1.UnsetParentGroupFromChannelsReq, ...grpc.CallOption) (*channelsv1.UnsetParentGroupFromChannelsRes, error) {
	return nil, status.Error(codes.Unimplemented, "atom channels compatibility only supports Authorize")
}

func (c AtomChannelsCompat) RetrieveEntity(context.Context, *commonv1.RetrieveEntityReq, ...grpc.CallOption) (*commonv1.RetrieveEntityRes, error) {
	return nil, status.Error(codes.Unimplemented, "atom channels compatibility requires a concrete Atom client")
}

func (c AtomChannelsCompat) RetrieveIDByRoute(ctx context.Context, in *commonv1.RetrieveIDByRouteReq, _ ...grpc.CallOption) (*commonv1.RetrieveEntityRes, error) {
	if c.Atom == nil {
		return nil, status.Error(codes.Unimplemented, "atom channels compatibility requires a concrete Atom client")
	}
	resources, err := c.Atom.ListResources(ctx, Query{
		Kind:     KindChannel,
		TenantID: in.GetDomainId(),
		Q:        in.GetRoute(),
		Limit:    20,
	})
	if err != nil {
		return nil, err
	}
	for _, resource := range resources.Items {
		if resource.Name == in.GetRoute() || attrString(resource.Attributes, "route") == in.GetRoute() {
			return &commonv1.RetrieveEntityRes{Entity: &commonv1.EntityBasic{
				Id:       resource.ID,
				DomainId: resource.TenantID,
				Status:   atomStatusCode(attrString(resource.Attributes, "status")),
			}}, nil
		}
	}
	return nil, status.Errorf(codes.NotFound, "channel route %q not found", in.GetRoute())
}

func atomStatusCode(value string) uint32 {
	switch strings.ToLower(value) {
	case "", atomStatusActive, atomStatusEnabled:
		return 0
	case atomStatusInactive, atomStatusDisabled, atomStatusFrozen, atomStatusSuspended:
		return 1
	case atomStatusDeleted:
		return 2
	default:
		return 0
	}
}

func attrString(attrs Attributes, key string) string {
	if attrs == nil {
		return ""
	}
	value, ok := attrs[key]
	if !ok || value == nil {
		return ""
	}
	str, _ := value.(string)
	return str
}
