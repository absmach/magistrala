// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"encoding/json"
)

// PolicyReq represents an argument struct for making a policy related
// function calls.
type PolicyReq struct {
	Namespace       string `json:",omitempty"`
	Subject         string `json:"subject"`
	SubjectType     string `json:"subject_type"`
	SubjectKind     string `json:"subject_kind"`
	SubjectRelation string `json:",omitempty"`
	Object          string `json:"object"`
	ObjectType      string `json:"object_type"`
	Relation        string `json:"relation"`
	Permission      string `json:",omitempty"`
}

func (pr PolicyReq) String() string {
	data, _ := json.Marshal(pr)
	return string(data)
}

type PolicyRes struct {
	Namespace       string
	Subject         string
	SubjectType     string
	SubjectRelation string
	Object          string
	ObjectType      string
	Relation        string
	Permission      string
}

type PolicyPage struct {
	Policies      []string
	NextPageToken string
}

// Authz represents a authorization service. It exposes
// functionalities through `auth` to perform authorization.
type Authz interface {
	// Authorize checks authorization of the given `subject`. Basically,
	// Authorize verifies that Is `subject` allowed to `relation` on
	// `object`. Authorize returns a non-nil error if the subject has
	// no relation on the object (which simply means the operation is
	// denied).
	Authorize(ctx context.Context, pr PolicyReq) error

	// AddPolicy creates a policy for the given subject, so that, after
	// AddPolicy, `subject` has a `relation` on `object`. Returns a non-nil
	// error in case of failures.
	AddPolicy(ctx context.Context, pr PolicyReq) error

	// AddPolicies adds new policies for given subjects. This method is
	// only allowed to use as an admin.
	AddPolicies(ctx context.Context, token, object string, subjectIDs, relations []string) error

	// DeletePolicy removes a policy.
	DeletePolicy(ctx context.Context, pr PolicyReq) error

	// DeletePolicies deletes policies for given subjects. This method is
	// only allowed to use as an admin.
	DeletePolicies(ctx context.Context, token, object string, subjectIDs, relations []string) error

	// ListObjects lists policies based on the given PolicyReq structure.
	ListObjects(ctx context.Context, pr PolicyReq, nextPageToken string, limit int32) (PolicyPage, error)

	// ListAllObjects lists all policies based on the given PolicyReq structure.
	ListAllObjects(ctx context.Context, pr PolicyReq) (PolicyPage, error)

	// CountPolicies count policies based on the given PolicyReq structure.
	CountObjects(ctx context.Context, pr PolicyReq) (int, error)

	// ListSubjects lists subjects based on the given PolicyReq structure.
	ListSubjects(ctx context.Context, pr PolicyReq, nextPageToken string, limit int32) (PolicyPage, error)

	// ListAllSubjects lists all subjects based on the given PolicyReq structure.
	ListAllSubjects(ctx context.Context, pr PolicyReq) (PolicyPage, error)

	// CountSubjects count policies based on the given PolicyReq structure.
	CountSubjects(ctx context.Context, pr PolicyReq) (int, error)
}

// PolicyAgent facilitates the communication to authorization
// services and implements Authz functionalities for certain
// authorization services (e.g. ORY Keto).
type PolicyAgent interface {
	// CheckPolicy checks if the subject has a relation on the object.
	// It returns a non-nil error if the subject has no relation on
	// the object (which simply means the operation is denied).
	CheckPolicy(ctx context.Context, pr PolicyReq) error

	// AddPolicy creates a policy for the given subject, so that, after
	// AddPolicy, `subject` has a `relation` on `object`. Returns a non-nil
	// error in case of failures.
	AddPolicy(ctx context.Context, pr PolicyReq) error

	// AddPolicies creates a Bulk Policies  for the given request
	AddPolicies(ctx context.Context, prs []PolicyReq) error

	// DeletePolicy removes a policy.
	DeletePolicy(ctx context.Context, pr PolicyReq) error

	// DeletePolicy removes a policy.
	DeletePolicies(ctx context.Context, pr []PolicyReq) error

	// RetrieveObjects
	RetrieveObjects(ctx context.Context, pr PolicyReq, nextPageToken string, limit int32) ([]PolicyRes, string, error)

	// RetrieveAllObjects
	RetrieveAllObjects(ctx context.Context, pr PolicyReq) ([]PolicyRes, error)

	// RetrieveAllObjectsCount
	RetrieveAllObjectsCount(ctx context.Context, pr PolicyReq) (int, error)

	// RetrieveSubjects
	RetrieveSubjects(ctx context.Context, pr PolicyReq, nextPageToken string, limit int32) ([]PolicyRes, string, error)

	// RetrieveAllSubjects
	RetrieveAllSubjects(ctx context.Context, pr PolicyReq) ([]PolicyRes, error)

	// RetrieveAllSubjectsCount
	RetrieveAllSubjectsCount(ctx context.Context, pr PolicyReq) (int, error)
}
