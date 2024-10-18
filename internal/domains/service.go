package domains

import (
	"context"
	"time"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/authz"
	"github.com/absmach/magistrala/pkg/domains"
	"github.com/absmach/magistrala/pkg/entityroles"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/policies"
	"github.com/absmach/magistrala/pkg/roles"
	"github.com/absmach/magistrala/pkg/svcutil"
)

const defLimit = 100

var (
	errCreateDomainPolicy = errors.New("failed to create domain policy")
	errRollbackRepo       = errors.New("failed to rollback repo")
	errRemovePolicyEngine = errors.New("failed to remove from policy engine")
)

type identity struct {
	ID       string
	DomainID string
	UserID   string
}
type service struct {
	repo       domains.DomainsRepository
	authz      authz.Authorization
	policies   policies.Service
	idProvider magistrala.IDProvider
	opp        svcutil.OperationPerm
	entityroles.RolesSvc
}

var _ domains.Service = (*service)(nil)

func New(repo domains.DomainsRepository, policiessvc policies.Service, idProvider magistrala.IDProvider, sidProvider magistrala.IDProvider) (domains.Service, error) {

	rolesSvc, err := entityroles.NewRolesSvc(policies.DomainType, repo, sidProvider, policiessvc, domains.AvailableActions(), domains.BuiltInRoles())
	if err != nil {
		return nil, err
	}

	opp := domains.NewOperationPerm()
	if err := opp.AddOperationPermissionMap(domains.NewOperationPermissionMap()); err != nil {
		return &service{}, err
	}
	if err := opp.Validate(); err != nil {
		return &service{}, err
	}

	return &service{
		repo:       repo,
		policies:   policiessvc,
		idProvider: idProvider,
		opp:        opp,
		RolesSvc:   rolesSvc,
	}, nil
}

func (svc service) CreateDomain(ctx context.Context, session authn.Session, d domains.Domain) (do domains.Domain, err error) {

	d.CreatedBy = session.UserID

	domainID, err := svc.idProvider.ID()
	if err != nil {
		return domains.Domain{}, errors.Wrap(svcerr.ErrCreateEntity, err)
	}
	d.ID = domainID

	if d.Status != domains.DisabledStatus && d.Status != domains.EnabledStatus {
		return domains.Domain{}, svcerr.ErrInvalidStatus
	}

	d.CreatedAt = time.Now()

	// Domain is created in repo first, because Roles table have foreign key relation with Domain ID
	dom, err := svc.repo.Save(ctx, d)
	if err != nil {
		return domains.Domain{}, errors.Wrap(svcerr.ErrCreateEntity, err)
	}
	defer func() {
		if err != nil {
			if errRollBack := svc.repo.Delete(ctx, domainID); errRollBack != nil {
				err = errors.Wrap(err, errors.Wrap(errRollbackRepo, errRollBack))
			}
		}
	}()

	newBuiltInRoleMembers := map[roles.BuiltInRoleName][]roles.Member{
		domains.BuiltInRoleAdmin:      {roles.Member(session.UserID)},
		domains.BuiltInRoleMembership: {},
	}

	optionalPolicies := []roles.OptionalPolicy{
		{
			Subject:     policies.MagistralaObject,
			SubjectType: policies.PlatformType,
			Relation:    "organization",
			Object:      d.ID,
			ObjectType:  policies.DomainType,
		},
	}

	if _, err := svc.AddNewEntityRoles(ctx, session.UserID, domainID, domainID, newBuiltInRoleMembers, optionalPolicies); err != nil {
		return domains.Domain{}, errors.Wrap(errCreateDomainPolicy, err)
	}

	return dom, nil
}

func (svc service) RetrieveDomain(ctx context.Context, session authn.Session, id string) (domains.Domain, error) {
	domain, err := svc.repo.RetrieveByID(ctx, id)
	if err != nil {
		return domains.Domain{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return domain, nil
}

func (svc service) UpdateDomain(ctx context.Context, session authn.Session, id string, d domains.DomainReq) (domains.Domain, error) {
	dom, err := svc.repo.Update(ctx, id, session.UserID, d)
	if err != nil {
		return domains.Domain{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return dom, nil
}

func (svc service) EnableDomain(ctx context.Context, session authn.Session, id string) (domains.Domain, error) {
	status := domains.EnabledStatus
	dom, err := svc.repo.Update(ctx, id, session.UserID, domains.DomainReq{Status: &status})
	if err != nil {
		return domains.Domain{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return dom, nil
}

func (svc service) DisableDomain(ctx context.Context, session authn.Session, id string) (domains.Domain, error) {
	status := domains.DisabledStatus
	dom, err := svc.repo.Update(ctx, id, session.UserID, domains.DomainReq{Status: &status})
	if err != nil {
		return domains.Domain{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return dom, nil
}

// Only SuperAdmin can freeze the domain
func (svc service) FreezeDomain(ctx context.Context, session authn.Session, id string) (domains.Domain, error) {
	status := domains.FreezeStatus
	dom, err := svc.repo.Update(ctx, id, session.UserID, domains.DomainReq{Status: &status})
	if err != nil {
		return domains.Domain{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return dom, nil
}

func (svc service) ListDomains(ctx context.Context, session authn.Session, p domains.Page) (domains.DomainsPage, error) {
	p.SubjectID = session.UserID
	//ToDo : Check list without below function and confirm and decide to remove or not
	if session.SuperAdmin {
		p.SubjectID = ""
	}

	dp, err := svc.repo.ListDomains(ctx, p)
	if err != nil {
		return domains.DomainsPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return dp, nil
}

func (svc service) DeleteUserFromDomains(ctx context.Context, id string) (err error) {
	domainsPage, err := svc.repo.ListDomains(ctx, domains.Page{SubjectID: id, Limit: defLimit})
	if err != nil {
		return err
	}

	if domainsPage.Total > defLimit {
		for i := defLimit; i < int(domainsPage.Total); i += defLimit {
			page := domains.Page{SubjectID: id, Offset: uint64(i), Limit: defLimit}
			dp, err := svc.repo.ListDomains(ctx, page)
			if err != nil {
				return err
			}
			domainsPage.Domains = append(domainsPage.Domains, dp.Domains...)
		}
	}

	if err := svc.RemoveMembersFromAllRoles(ctx, authn.Session{}, []string{id}); err != nil {
		return err
	}
	////////////ToDo//////////////
	// Remove user from all roles in all domains
	//////////////////////////

	// for _, domain := range domainsPage.Domains {
	// 	req := policies.Policy{
	// 		Subject:     policies.EncodeDomainUserID(domain.ID, id),
	// 		SubjectType: policies.UserType,
	// 	}
	// 	if err := svc.policies.DeletePolicyFilter(ctx, req); err != nil {
	// 		return err
	// 	}
	// }

	// if err := svc.repo.DeleteUserPolicies(ctx, id); err != nil {
	// 	return err
	// }

	return nil
}
