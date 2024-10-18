package channels

import (
	"context"
	"time"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/auth"
	grpcclient "github.com/absmach/magistrala/auth/api/grpc"
	"github.com/absmach/magistrala/pkg/channels"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/entityroles"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/roles"
	"github.com/absmach/magistrala/pkg/svcutil"
	"github.com/absmach/magistrala/things"
	"golang.org/x/sync/errgroup"
)

var (
	errCreateChannelsPolicies = errors.New("failed to create channels policies")
	errRollbackRepo           = errors.New("failed to rollback repo")
)

type channelsService struct {
	auth        grpcclient.AuthServiceClient
	repo        channels.Repository
	policy      magistrala.PolicyServiceClient
	idProvider  magistrala.IDProvider
	sidProvider magistrala.IDProvider
	opp         svcutil.OperationPerm
	entityroles.RolesSvc
}

var _ channels.Service = (*channelsService)(nil)

func New(repo channels.Repository, authClient grpcclient.AuthServiceClient, policyClient magistrala.PolicyServiceClient, idProvider magistrala.IDProvider, sidProvider magistrala.IDProvider) (channels.Service, error) {
	rolesSvc, err := entityroles.NewRolesSvc(auth.DomainType, repo, sidProvider, authClient, policyClient, channels.AvailableActions(), channels.BuiltInRoles(), channels.NewRolesOperationPermissionMap())
	if err != nil {
		return nil, err
	}

	opp := channels.NewOperationPerm()
	if err := opp.AddOperationPermissionMap(channels.NewOperationPermissionMap()); err != nil {
		return channelsService{}, err
	}
	if err := opp.Validate(); err != nil {
		return channelsService{}, err
	}
	return channelsService{
		auth:        authClient,
		repo:        repo,
		policy:      policyClient,
		idProvider:  idProvider,
		sidProvider: sidProvider,
		opp:         opp,
		RolesSvc:    rolesSvc,
	}, nil
}

func (cs channelsService) CreateChannels(ctx context.Context, token string, chs ...channels.Channel) ([]channels.Channel, error) {
	userInfo, err := cs.identify(ctx, token)
	if err != nil {
		return []channels.Channel{}, err
	}
	// If domain is disabled , then this authorization will fail for all non-admin domain users
	if _, err := cs.authorize(ctx, "", auth.UserType, auth.UsersKind, userInfo.ID, channels.OpCreateChannel, auth.DomainType, userInfo.DomainID); err != nil {
		return []channels.Channel{}, err
	}

	var clients []channels.Channel
	for _, c := range chs {
		if c.ID == "" {
			clientID, err := cs.idProvider.ID()
			if err != nil {
				return []channels.Channel{}, err
			}
			c.ID = clientID
		}

		if c.Status != mgclients.DisabledStatus && c.Status != mgclients.EnabledStatus {
			return []channels.Channel{}, svcerr.ErrInvalidStatus
		}
		c.Domain = userInfo.DomainID
		c.CreatedAt = time.Now()
		clients = append(clients, c)
	}

	saved, err := cs.repo.Save(ctx, clients...)
	if err != nil {
		return nil, errors.Wrap(svcerr.ErrCreateEntity, err)
	}
	chIDs := []string{}
	for _, c := range saved {
		chIDs = append(chIDs, c.ID)
	}

	defer func() {
		if err != nil {
			if errRollBack := cs.repo.Remove(ctx, chIDs...); errRollBack != nil {
				err = errors.Wrap(err, errors.Wrap(errRollbackRepo, errRollBack))
			}
		}
	}()

	newBuiltInRoleMembers := map[roles.BuiltInRoleName][]roles.Member{
		channels.BuiltInRoleAdmin: {roles.Member(userInfo.UserID)},
	}

	optionalPolicies := []roles.OptionalPolicy{}

	for _, chID := range chIDs {
		optionalPolicies = append(optionalPolicies,
			roles.OptionalPolicy{
				Namespace:   userInfo.DomainID,
				SubjectType: auth.UserType,
				Subject:     userInfo.ID,
				Relation:    auth.AdministratorRelation,
				ObjectType:  auth.ChannelType,
				Object:      chID,
			},
			roles.OptionalPolicy{

				Namespace:   userInfo.DomainID,
				SubjectType: auth.UserType,
				Subject:     userInfo.ID,
				Relation:    auth.DomainRelation,
				ObjectType:  auth.ChannelType,
				Object:      chID,
			},
		)
	}
	if _, err := cs.AddNewEntityRoles(ctx, userInfo.UserID, userInfo.DomainID, userInfo.DomainID, newBuiltInRoleMembers, optionalPolicies); err != nil {
		return []channels.Channel{}, errors.Wrap(errCreateChannelsPolicies, err)
	}
	return saved, nil
}

func (cs channelsService) UpdateChannel(ctx context.Context, token string, ch channels.Channel) (channels.Channel, error) {
	userID, err := cs.authorize(ctx, "", auth.UserType, auth.TokenKind, token, channels.OpUpdateChannel, auth.ChannelType, ch.ID)
	if err != nil {
		return channels.Channel{}, errors.Wrap(svcerr.ErrAuthorization, err)
	}

	channel := channels.Channel{
		ID:        ch.ID,
		Name:      ch.Name,
		Metadata:  ch.Metadata,
		UpdatedAt: time.Now(),
		UpdatedBy: userID,
	}
	channel, err = cs.repo.Update(ctx, channel)
	if err != nil {
		return channels.Channel{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return channel, nil
}

func (cs channelsService) UpdateChannelTags(ctx context.Context, token string, ch channels.Channel) (channels.Channel, error) {
	userID, err := cs.authorize(ctx, "", auth.UserType, auth.TokenKind, token, channels.OpUpdateChannelTags, auth.ChannelType, ch.ID)
	if err != nil {
		return channels.Channel{}, errors.Wrap(svcerr.ErrAuthorization, err)
	}

	channel := channels.Channel{
		ID:        ch.ID,
		Tags:      ch.Tags,
		UpdatedAt: time.Now(),
		UpdatedBy: userID,
	}
	channel, err = cs.repo.UpdateTags(ctx, channel)
	if err != nil {
		return channels.Channel{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return channel, nil
}

func (cs channelsService) EnableChannel(ctx context.Context, token, id string) (channels.Channel, error) {
	channel := channels.Channel{
		ID:        id,
		Status:    mgclients.EnabledStatus,
		UpdatedAt: time.Now(),
	}
	ch, err := cs.changeChannelStatus(ctx, token, channel)
	if err != nil {
		return channels.Channel{}, errors.Wrap(mgclients.ErrEnableClient, err)
	}

	return ch, nil
}

func (cs channelsService) DisableChannel(ctx context.Context, token, id string) (channels.Channel, error) {
	channel := channels.Channel{
		ID:        id,
		Status:    mgclients.DisabledStatus,
		UpdatedAt: time.Now(),
	}
	ch, err := cs.changeChannelStatus(ctx, token, channel)
	if err != nil {
		return channels.Channel{}, errors.Wrap(mgclients.ErrDisableClient, err)
	}

	return ch, nil
}

func (cs channelsService) ViewChannel(ctx context.Context, token, id string) (channels.Channel, error) {
	_, err := cs.authorize(ctx, "", auth.UserType, auth.TokenKind, token, channels.OpViewChannel, auth.ChannelType, id)
	if err != nil {
		return channels.Channel{}, err
	}
	channel, err := cs.repo.RetrieveByID(ctx, id)
	if err != nil {
		return channels.Channel{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return channel, nil
}

func (cs channelsService) ListChannels(ctx context.Context, token string, pm channels.PageMetadata) (channels.Page, error) {
	var ids []string

	userInfo, err := cs.identify(ctx, token)
	if err != nil {
		return channels.Page{}, err
	}

	if err := cs.checkSuperAdmin(ctx, userInfo.UserID); err != nil {
		if _, err := cs.authorize(ctx, "", auth.UserType, auth.UsersKind, userInfo.ID, channels.OpListChannel, auth.DomainType, userInfo.DomainID); err != nil {
			return channels.Page{}, err
		}
		ids, err = cs.listChannelIDs(ctx, userInfo.ID, pm.Permission)
		if err != nil {
			return channels.Page{}, errors.Wrap(svcerr.ErrNotFound, err)
		}
	}
	if len(ids) == 0 && pm.Domain == "" {
		return channels.Page{}, nil
	}
	pm.IDs = ids

	cp, err := cs.repo.RetrieveAll(ctx, pm)
	if err != nil {
		return channels.Page{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}

	if pm.ListPerms && len(cp.Channels) > 0 {
		g, ctx := errgroup.WithContext(ctx)

		for i := range cp.Channels {
			// Copying loop variable "i" to avoid "loop variable captured by func literal"
			iter := i
			g.Go(func() error {
				return cs.retrievePermissions(ctx, userInfo.ID, &cp.Channels[iter])
			})
		}

		if err := g.Wait(); err != nil {
			return channels.Page{}, err
		}
	}
	return cp, nil
}

func (cs channelsService) ListChannelsByThing(ctx context.Context, token, thID string, pm channels.PageMetadata) (channels.Page, error) {

	return channels.Page{}, nil
}

func (cs channelsService) RemoveChannel(ctx context.Context, token, id string) error {
	userInfo, err := cs.identify(ctx, token)
	if err != nil {
		return err
	}
	if _, err := cs.authorize(ctx, userInfo.DomainID, auth.UserType, auth.UsersKind, userInfo.ID, channels.OpDeleteChannel, auth.ChannelType, id); err != nil {
		return err
	}

	deleteRes, err := cs.policy.DeleteEntityPolicies(ctx, &magistrala.DeleteEntityPoliciesReq{
		EntityType: auth.ThingType,
		Id:         id,
	})
	if err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}
	if !deleteRes.Deleted {
		return svcerr.ErrAuthorization
	}

	if err := cs.repo.Remove(ctx, id); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	return nil
}

func (cs channelsService) Connect(ctx context.Context, token string, chIDs, thIDs []string) (retErr error) {
	userInfo, err := cs.identify(ctx, token)
	if err != nil {
		return err
	}
	//ToDo: This authorization will be changed with Bulk Authorization. For this we need to add bulk authorization API in auth.
	for _, chID := range chIDs {
		if _, err := cs.authorize(ctx, userInfo.DomainID, auth.UserType, auth.UsersKind, userInfo.ID, channels.OpConnectThingChannel, auth.ChannelType, chID); err != nil {
			return err
		}
	}

	for _, thID := range thIDs {
		if _, err := cs.authorize(ctx, userInfo.DomainID, auth.UserType, auth.UsersKind, userInfo.ID, things.OpConnectChannelThing, auth.ChannelType, thID); err != nil {
			return err
		}
	}

	prs := []*magistrala.AddPolicyReq{}
	rbPrs := []*magistrala.DeletePolicyReq{}
	for _, chID := range chIDs {
		for _, thID := range thIDs {
			prs = append(prs, &magistrala.AddPolicyReq{
				SubjectType: auth.ThingType,
				Subject:     thID,
				Relation:    "connect",
				Object:      chID,
				ObjectType:  auth.ChannelType,
			})
			rbPrs = append(rbPrs, &magistrala.DeletePolicyReq{
				SubjectType: auth.ThingType,
				Subject:     thID,
				Relation:    "connect",
				Object:      chID,
				ObjectType:  auth.ChannelType,
			})
		}
	}
	if _, err := cs.policy.AddPolicies(ctx, &magistrala.AddPoliciesReq{AddPoliciesReq: prs}); err != nil {
		return errors.Wrap(svcerr.ErrAddPolicies, err)
	}
	defer func() {
		if retErr != nil {
			if _, errRollback := cs.policy.DeletePolicies(ctx, &magistrala.DeletePoliciesReq{DeletePoliciesReq: rbPrs}); errRollback != nil {
				retErr = errors.Wrap(retErr, errRollback)
			}
		}
	}()

	if err := cs.repo.Connect(ctx, chIDs, thIDs); err != nil {
		return errors.Wrap(svcerr.ErrCreateEntity, err)
	}

	return nil
}

func (cs channelsService) Disconnect(ctx context.Context, token string, chIDs, thIDs []string) error {
	userInfo, err := cs.identify(ctx, token)
	if err != nil {
		return err
	}
	//ToDo: This authorization will be changed with Bulk Authorization. For this we need to add bulk authorization API in auth.
	for _, chID := range chIDs {
		if _, err := cs.authorize(ctx, userInfo.DomainID, auth.UserType, auth.UsersKind, userInfo.ID, channels.OpDisconnectThingChannel, auth.ChannelType, chID); err != nil {
			return err
		}
	}

	for _, thID := range thIDs {
		if _, err := cs.authorize(ctx, userInfo.DomainID, auth.UserType, auth.UsersKind, userInfo.ID, things.OpDisconnectChannelThing, auth.ChannelType, thID); err != nil {
			return err
		}
	}

	prs := []*magistrala.DeletePolicyReq{}
	for _, chID := range chIDs {
		for _, thID := range thIDs {

			prs = append(prs, &magistrala.DeletePolicyReq{
				SubjectType: auth.ThingType,
				Subject:     thID,
				Relation:    "connect",
				Object:      chID,
				ObjectType:  auth.ChannelType,
			})
		}
	}
	if _, err := cs.policy.DeletePolicies(ctx, &magistrala.DeletePoliciesReq{DeletePoliciesReq: prs}); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	if err := cs.repo.Disconnect(ctx, chIDs, thIDs); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	return nil
}

type identity struct {
	ID       string
	DomainID string
	UserID   string
}

func (cs channelsService) identify(ctx context.Context, token string) (identity, error) {
	resp, err := cs.auth.Identify(ctx, &magistrala.IdentityReq{Token: token})
	if err != nil {
		return identity{}, errors.Wrap(svcerr.ErrAuthentication, err)
	}
	return identity{ID: resp.GetId(), DomainID: resp.GetDomainId(), UserID: resp.GetUserId()}, nil
}

func (cs channelsService) authorize(ctx context.Context, domainID, subjType, subjKind, subj string, op svcutil.Operation, objType, obj string) (string, error) {
	perm, err := cs.opp.GetPermission(op)
	if err != nil {
		return "", err
	}

	req := &magistrala.AuthorizeReq{
		Domain:      domainID,
		SubjectType: subjType,
		SubjectKind: subjKind,
		Subject:     subj,
		Permission:  perm.String(),
		ObjectType:  objType,
		Object:      obj,
	}
	res, err := cs.auth.Authorize(ctx, req)
	if err != nil {
		return "", errors.Wrap(svcerr.ErrAuthorization, err)
	}
	if !res.GetAuthorized() {
		return "", errors.Wrap(svcerr.ErrAuthorization, err)
	}

	return res.GetId(), nil
}

func (cs channelsService) checkSuperAdmin(ctx context.Context, userID string) error {
	res, err := cs.auth.Authorize(ctx, &magistrala.AuthorizeReq{
		SubjectType: auth.UserType,
		Subject:     userID,
		Permission:  auth.AdminPermission,
		ObjectType:  auth.PlatformType,
		Object:      auth.MagistralaObject,
	})
	if err != nil {
		return err
	}
	if !res.Authorized {
		return svcerr.ErrAuthorization
	}
	return nil
}

func (cs channelsService) listChannelIDs(ctx context.Context, userID, permission string) ([]string, error) {
	tids, err := cs.policy.ListAllObjects(ctx, &magistrala.ListObjectsReq{
		SubjectType: auth.UserType,
		Subject:     userID,
		Permission:  permission,
		ObjectType:  auth.ChannelType,
	})
	if err != nil {
		return nil, errors.Wrap(svcerr.ErrNotFound, err)
	}
	return tids.Policies, nil
}

func (cs channelsService) retrievePermissions(ctx context.Context, userID string, channel *channels.Channel) error {
	permissions, err := cs.listUserThingPermission(ctx, userID, channel.ID)
	if err != nil {
		return err
	}
	channel.Permissions = permissions
	return nil
}

func (cs channelsService) listUserThingPermission(ctx context.Context, userID, thingID string) ([]string, error) {
	lp, err := cs.policy.ListPermissions(ctx, &magistrala.ListPermissionsReq{
		SubjectType: auth.UserType,
		Subject:     userID,
		Object:      thingID,
		ObjectType:  auth.ChannelType,
	})
	if err != nil {
		return []string{}, errors.Wrap(svcerr.ErrAuthorization, err)
	}
	return lp.GetPermissions(), nil
}

func (cs channelsService) changeChannelStatus(ctx context.Context, userID string, channel channels.Channel) (channels.Channel, error) {

	dbchannel, err := cs.repo.RetrieveByID(ctx, channel.ID)
	if err != nil {
		return channels.Channel{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	if dbchannel.Status == channel.Status {
		return channels.Channel{}, errors.ErrStatusAlreadyAssigned
	}

	channel.UpdatedBy = userID

	channel, err = cs.repo.ChangeStatus(ctx, channel)
	if err != nil {
		return channels.Channel{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return channel, nil
}
