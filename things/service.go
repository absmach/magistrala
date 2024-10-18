// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0
package things

import (
	"context"
	"time"

	"github.com/absmach/magistrala"
	mgauth "github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/pkg/authn"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/entityroles"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/policies"
	"github.com/absmach/magistrala/pkg/roles"
	"github.com/absmach/magistrala/pkg/svcutil"
	"github.com/absmach/magistrala/things/postgres"
	"golang.org/x/sync/errgroup"
)

var (
	errCreateThingsPolicies = errors.New("failed to create things policies")
	errRollbackRepo         = errors.New("failed to rollback repo")
)

type identity struct {
	ID       string
	DomainID string
	UserID   string
}
type service struct {
	evaluator   policies.Evaluator
	policysvc   policies.Service
	clients     postgres.Repository
	clientCache Cache
	idProvider  magistrala.IDProvider
	opp         svcutil.OperationPerm
	entityroles.RolesSvc
}

// NewService returns a new Clients service implementation.
func NewService(policysvc policies.Service, policyEvaluator policies.Evaluator, repo postgres.Repository, tcache Cache, idp magistrala.IDProvider, sidProvider magistrala.IDProvider) (Service, error) {

	rolesSvc, err := entityroles.NewRolesSvc(policies.DomainType, repo, sidProvider, policysvc, ThingAvailableActions(), ThingBuiltInRoles())
	if err != nil {
		return nil, err
	}

	opp := NewOperationPerm()
	if err := opp.AddOperationPermissionMap(NewOperationPermissionMap()); err != nil {
		return service{}, err
	}
	if err := opp.Validate(); err != nil {
		return service{}, err
	}

	return service{
		evaluator:   policyEvaluator,
		policysvc:   policysvc,
		clients:     repo,
		clientCache: tcache,
		idProvider:  idp,
		opp:         opp,
		RolesSvc:    rolesSvc,
	}, nil
}

func (svc service) Authorize(ctx context.Context, req AuthzReq) (string, error) {
	thingID, err := svc.Identify(ctx, req.ThingKey)
	if err != nil {
		return "", err
	}

	r := policies.Policy{
		SubjectType: policies.GroupType,
		Subject:     req.ChannelID,
		ObjectType:  policies.ThingType,
		Object:      thingID,
		Permission:  req.Permission,
	}
	err = svc.evaluator.CheckPolicy(ctx, r)
	if err != nil {
		return "", errors.Wrap(svcerr.ErrAuthorization, err)
	}

	return thingID, nil
}

func (svc service) CreateThings(ctx context.Context, session authn.Session, cls ...mgclients.Client) ([]mgclients.Client, error) {
	var clients []mgclients.Client
	for _, c := range cls {
		if c.ID == "" {
			clientID, err := svc.idProvider.ID()
			if err != nil {
				return []mgclients.Client{}, err
			}
			c.ID = clientID
		}
		if c.Credentials.Secret == "" {
			key, err := svc.idProvider.ID()
			if err != nil {
				return []mgclients.Client{}, err
			}
			c.Credentials.Secret = key
		}
		if c.Status != mgclients.DisabledStatus && c.Status != mgclients.EnabledStatus {
			return []mgclients.Client{}, svcerr.ErrInvalidStatus
		}
		c.Domain = session.DomainID
		c.CreatedAt = time.Now()
		clients = append(clients, c)
	}

	saved, err := svc.clients.Save(ctx, clients...)
	if err != nil {
		return nil, errors.Wrap(svcerr.ErrCreateEntity, err)
	}
	clientIDs := []string{}
	for _, c := range saved {
		clientIDs = append(clientIDs, c.ID)
	}

	defer func() {
		if err != nil {
			if errRollBack := svc.clients.RemoveThings(ctx, clientIDs); errRollBack != nil {
				err = errors.Wrap(err, errors.Wrap(errRollbackRepo, errRollBack))
			}
		}
	}()

	newBuiltInRoleMembers := map[roles.BuiltInRoleName][]roles.Member{
		ThingBuiltInRoleAdmin: {roles.Member(session.UserID)},
	}

	optionalPolicies := []roles.OptionalPolicy{}

	for _, clientID := range clientIDs {
		optionalPolicies = append(optionalPolicies,
			roles.OptionalPolicy{
				Namespace:   session.DomainID,
				SubjectType: policies.UserType,
				Subject:     session.DomainUserID,
				Relation:    policies.AdministratorRelation,
				ObjectKind:  policies.NewThingKind,
				ObjectType:  policies.ThingType,
				Object:      clientID,
			},
			roles.OptionalPolicy{

				Namespace:   session.DomainID,
				SubjectType: policies.UserType,
				Subject:     session.DomainUserID,
				Relation:    policies.DomainRelation,
				ObjectType:  policies.ThingType,
				Object:      clientID,
			},
		)
	}

	if _, err := svc.AddNewEntityRoles(ctx, session.UserID, session.DomainID, session.DomainID, newBuiltInRoleMembers, optionalPolicies); err != nil {
		return []mgclients.Client{}, errors.Wrap(errCreateThingsPolicies, err)
	}

	return saved, nil
}

func (svc service) ViewClient(ctx context.Context, session authn.Session, id string) (mgclients.Client, error) {
	client, err := svc.clients.RetrieveByID(ctx, id)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return client, nil
}

func (svc service) ViewClientPerms(ctx context.Context, session authn.Session, id string) ([]string, error) {
	permissions, err := svc.listUserThingPermission(ctx, session.DomainUserID, id)
	if err != nil {
		return nil, err
	}
	if len(permissions) == 0 {
		return nil, svcerr.ErrAuthorization
	}
	return permissions, nil
}

func (svc service) ListClients(ctx context.Context, session authn.Session, reqUserID string, pm mgclients.Page) (mgclients.ClientsPage, error) {
	var ids []string
	var err error
	switch {
	case (reqUserID != "" && reqUserID != session.UserID):
		rtids, err := svc.listClientIDs(ctx, mgauth.EncodeDomainUserID(session.DomainID, reqUserID), pm.Permission)
		if err != nil {
			return mgclients.ClientsPage{}, errors.Wrap(svcerr.ErrNotFound, err)
		}
		ids, err = svc.filterAllowedThingIDs(ctx, session.DomainUserID, pm.Permission, rtids)
		if err != nil {
			return mgclients.ClientsPage{}, errors.Wrap(svcerr.ErrNotFound, err)
		}
	default:
		switch session.SuperAdmin {
		case true:
			pm.Domain = session.DomainID
		default:
			ids, err = svc.listClientIDs(ctx, session.DomainUserID, pm.Permission)
			if err != nil {
				return mgclients.ClientsPage{}, errors.Wrap(svcerr.ErrNotFound, err)
			}
		}
	}

	if len(ids) == 0 && pm.Domain == "" {
		return mgclients.ClientsPage{}, nil
	}
	pm.IDs = ids
	tp, err := svc.clients.SearchClients(ctx, pm)
	if err != nil {
		return mgclients.ClientsPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}

	if pm.ListPerms && len(tp.Clients) > 0 {
		g, ctx := errgroup.WithContext(ctx)

		for i := range tp.Clients {
			// Copying loop variable "i" to avoid "loop variable captured by func literal"
			iter := i
			g.Go(func() error {
				return svc.retrievePermissions(ctx, session.DomainUserID, &tp.Clients[iter])
			})
		}

		if err := g.Wait(); err != nil {
			return mgclients.ClientsPage{}, err
		}
	}
	return tp, nil
}

// Experimental functions used for async calling of svc.listUserThingPermission. This might be helpful during listing of large number of entities.
func (svc service) retrievePermissions(ctx context.Context, userID string, client *mgclients.Client) error {
	permissions, err := svc.listUserThingPermission(ctx, userID, client.ID)
	if err != nil {
		return err
	}
	client.Permissions = permissions
	return nil
}

func (svc service) listUserThingPermission(ctx context.Context, userID, thingID string) ([]string, error) {
	permissions, err := svc.policysvc.ListPermissions(ctx, policies.Policy{
		SubjectType: policies.UserType,
		Subject:     userID,
		Object:      thingID,
		ObjectType:  policies.ThingType,
	}, []string{})
	if err != nil {
		return []string{}, errors.Wrap(svcerr.ErrAuthorization, err)
	}
	return permissions, nil
}

func (svc service) listClientIDs(ctx context.Context, userID, permission string) ([]string, error) {
	tids, err := svc.policysvc.ListAllObjects(ctx, policies.Policy{
		SubjectType: policies.UserType,
		Subject:     userID,
		Permission:  permission,
		ObjectType:  policies.ThingType,
	})
	if err != nil {
		return nil, errors.Wrap(svcerr.ErrNotFound, err)
	}
	return tids.Policies, nil
}

func (svc service) filterAllowedThingIDs(ctx context.Context, userID, permission string, thingIDs []string) ([]string, error) {
	var ids []string
	tids, err := svc.policysvc.ListAllObjects(ctx, policies.Policy{
		SubjectType: policies.UserType,
		Subject:     userID,
		Permission:  permission,
		ObjectType:  policies.ThingType,
	})
	if err != nil {
		return nil, errors.Wrap(svcerr.ErrNotFound, err)
	}
	for _, thingID := range thingIDs {
		for _, tid := range tids.Policies {
			if thingID == tid {
				ids = append(ids, thingID)
			}
		}
	}
	return ids, nil
}

func (svc service) UpdateClient(ctx context.Context, session authn.Session, cli mgclients.Client) (mgclients.Client, error) {
	client := mgclients.Client{
		ID:        cli.ID,
		Name:      cli.Name,
		Metadata:  cli.Metadata,
		UpdatedAt: time.Now(),
		UpdatedBy: session.UserID,
	}
	client, err := svc.clients.Update(ctx, client)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return client, nil
}

func (svc service) UpdateClientTags(ctx context.Context, session authn.Session, cli mgclients.Client) (mgclients.Client, error) {
	client := mgclients.Client{
		ID:        cli.ID,
		Tags:      cli.Tags,
		UpdatedAt: time.Now(),
		UpdatedBy: session.UserID,
	}
	client, err := svc.clients.UpdateTags(ctx, client)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return client, nil
}

func (svc service) UpdateClientSecret(ctx context.Context, session authn.Session, id, key string) (mgclients.Client, error) {
	client := mgclients.Client{
		ID: id,
		Credentials: mgclients.Credentials{
			Secret: key,
		},
		UpdatedAt: time.Now(),
		UpdatedBy: session.UserID,
		Status:    mgclients.EnabledStatus,
	}
	client, err := svc.clients.UpdateSecret(ctx, client)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return client, nil
}

func (svc service) EnableClient(ctx context.Context, session authn.Session, id string) (mgclients.Client, error) {
	client := mgclients.Client{
		ID:        id,
		Status:    mgclients.EnabledStatus,
		UpdatedAt: time.Now(),
	}
	client, err := svc.changeClientStatus(ctx, session, client)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(mgclients.ErrEnableClient, err)
	}

	return client, nil
}

func (svc service) DisableClient(ctx context.Context, session authn.Session, id string) (mgclients.Client, error) {
	client := mgclients.Client{
		ID:        id,
		Status:    mgclients.DisabledStatus,
		UpdatedAt: time.Now(),
	}
	client, err := svc.changeClientStatus(ctx, session, client)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(mgclients.ErrDisableClient, err)
	}

	if err := svc.clientCache.Remove(ctx, client.ID); err != nil {
		return client, errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	return client, nil
}

func (svc service) Share(ctx context.Context, session authn.Session, id, relation string, userids ...string) error {
	policyList := []policies.Policy{}
	for _, userid := range userids {
		policyList = append(policyList, policies.Policy{
			SubjectType: policies.UserType,
			Subject:     mgauth.EncodeDomainUserID(session.DomainID, userid),
			Relation:    relation,
			ObjectType:  policies.ThingType,
			Object:      id,
		})
	}
	if err := svc.policysvc.AddPolicies(ctx, policyList); err != nil {
		return errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	return nil
}

func (svc service) Unshare(ctx context.Context, session authn.Session, id, relation string, userids ...string) error {
	policyList := []policies.Policy{}
	for _, userid := range userids {
		policyList = append(policyList, policies.Policy{
			SubjectType: policies.UserType,
			Subject:     mgauth.EncodeDomainUserID(session.DomainID, userid),
			Relation:    relation,
			ObjectType:  policies.ThingType,
			Object:      id,
		})
	}
	if err := svc.policysvc.DeletePolicies(ctx, policyList); err != nil {
		return errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	return nil
}

func (svc service) DeleteClient(ctx context.Context, session authn.Session, id string) error {
	if err := svc.clientCache.Remove(ctx, id); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	req := policies.Policy{
		Object:     id,
		ObjectType: policies.ThingType,
	}

	if err := svc.policysvc.DeletePolicyFilter(ctx, req); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	if err := svc.clients.Delete(ctx, id); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	return nil
}

func (svc service) changeClientStatus(ctx context.Context, session authn.Session, client mgclients.Client) (mgclients.Client, error) {
	dbClient, err := svc.clients.RetrieveByID(ctx, client.ID)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	if dbClient.Status == client.Status {
		return mgclients.Client{}, errors.ErrStatusAlreadyAssigned
	}

	client.UpdatedBy = session.UserID

	client, err = svc.clients.ChangeStatus(ctx, client)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return client, nil
}

func (svc service) Identify(ctx context.Context, key string) (string, error) {
	id, err := svc.clientCache.ID(ctx, key)
	if err == nil {
		return id, nil
	}

	client, err := svc.clients.RetrieveBySecret(ctx, key)
	if err != nil {
		return "", errors.Wrap(svcerr.ErrAuthorization, err)
	}
	if err := svc.clientCache.Save(ctx, key, client.ID); err != nil {
		return "", errors.Wrap(svcerr.ErrAuthorization, err)
	}

	return client.ID, nil
}

func (svc service) addThingPolicies(ctx context.Context, userID, domainID string, things []mgclients.Client) error {
	policyList := []policies.Policy{}
	for _, thing := range things {
		policyList = append(policyList, policies.Policy{
			Domain:      domainID,
			SubjectType: policies.UserType,
			Subject:     userID,
			Relation:    policies.AdministratorRelation,
			ObjectKind:  policies.NewThingKind,
			ObjectType:  policies.ThingType,
			Object:      thing.ID,
		})
		policyList = append(policyList, policies.Policy{
			Domain:      domainID,
			SubjectType: policies.DomainType,
			Subject:     domainID,
			Relation:    policies.DomainRelation,
			ObjectType:  policies.ThingType,
			Object:      thing.ID,
		})
	}
	if err := svc.policysvc.AddPolicies(ctx, policyList); err != nil {
		return errors.Wrap(svcerr.ErrCreateEntity, err)
	}

	return nil
}

func (svc service) addThingPoliciesRollback(ctx context.Context, userID, domainID string, things []mgclients.Client) error {
	policyList := []policies.Policy{}
	for _, thing := range things {
		policyList = append(policyList, policies.Policy{
			Domain:      domainID,
			SubjectType: policies.UserType,
			Subject:     userID,
			Relation:    policies.AdministratorRelation,
			ObjectKind:  policies.NewThingKind,
			ObjectType:  policies.ThingType,
			Object:      thing.ID,
		})
		policyList = append(policyList, policies.Policy{
			Domain:      domainID,
			SubjectType: policies.DomainType,
			Subject:     domainID,
			Relation:    policies.DomainRelation,
			ObjectType:  policies.ThingType,
			Object:      thing.ID,
		})
	}
	if err := svc.policysvc.DeletePolicies(ctx, policyList); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	return nil
}
