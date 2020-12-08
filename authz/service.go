package authz

import (
	context "context"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/pkg/errors"

	casbin "github.com/casbin/casbin/v2"
)

var (
	// ErrUnauthorizedAccess represents unauthorized access.
	ErrUnauthorizedAccess = errors.New("unauthorized access")

	// ErrMalformedEntity malformed entity
	ErrMalformedEntity = errors.New("malformed request")

	// ErrNotFound indicates entity not found
	ErrNotFound = errors.New("entity not found")

	// ErrInvalidReq  invalid request
	ErrInvalidReq = errors.New("invalid request")
)

type Policy struct {
	Subject string `json:"subject"`
	Object  string `json:"object"`
	Action  string `json:"action"`
}

type Service interface {
	// AddPolicy creates new policy
	AddPolicy(context.Context, string, Policy) (bool, error)
	// RemovePolicy removes existing policy
	RemovePolicy(context.Context, string, Policy) (bool, error)
	// Authorize - checks if request is authorized
	// against saved policies in database.
	Authorize(context.Context, Policy) (bool, error)
}

var _ Service = (*service)(nil)

type service struct {
	enforcer *casbin.SyncedEnforcer
	auth     mainflux.AuthNServiceClient
}

// New instantiates the auth service implementation.
func New(e *casbin.SyncedEnforcer, auth mainflux.AuthNServiceClient) Service {
	return &service{
		enforcer: e,
		auth:     auth,
	}
}

func (svc service) AddPolicy(ctx context.Context, token string, p Policy) (bool, error) {
	if _, err := svc.auth.Identify(ctx, &mainflux.Token{Value: token}); err != nil {
		return false, errors.Wrap(ErrUnauthorizedAccess, err)
	}
	return svc.enforcer.AddPolicy(p.Subject, p.Object, p.Action)
}

func (svc service) RemovePolicy(ctx context.Context, token string, p Policy) (bool, error) {
	if _, err := svc.auth.Identify(ctx, &mainflux.Token{Value: token}); err != nil {
		return false, errors.Wrap(ErrUnauthorizedAccess, err)
	}
	return svc.enforcer.RemovePolicy(p.Subject, p.Object, p.Action)
}

func (svc service) Authorize(ctx context.Context, p Policy) (bool, error) {
	return svc.enforcer.Enforce(p.Subject, p.Object, p.Action)
}
