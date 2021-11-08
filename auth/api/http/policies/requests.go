package policies

import (
	"github.com/mainflux/mainflux/auth"
	"github.com/mainflux/mainflux/things"
)

// Action represents an enum for the policies used in the Mainflux.
type Action int

const (
	Create Action = iota
	Read
	Write
	Delete
	Access
	Member
	Unknown
)

var actions = map[string]Action{
	"create": Create,
	"read":   Read,
	"write":  Write,
	"delete": Delete,
	"access": Access,
	"member": Member,
}

type policiesReq struct {
	token      string
	SubjectIDs []string `json:"subjects"`
	Policies   []string `json:"policies"`
	Object     string   `json:"object"`
}

func (req policiesReq) validate() error {
	if req.token == "" {
		return auth.ErrUnauthorizedAccess
	}

	if len(req.SubjectIDs) == 0 || len(req.Policies) == 0 || req.Object == "" {
		return auth.ErrMalformedEntity
	}

	for _, policy := range req.Policies {
		if _, ok := actions[policy]; !ok {
			return auth.ErrMalformedEntity
		}
	}

	for _, subject := range req.SubjectIDs {
		if subject == "" {
			return things.ErrMalformedEntity
		}
	}

	return nil
}
