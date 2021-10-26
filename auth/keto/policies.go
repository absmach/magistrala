// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package keto

import (
	"context"
	"regexp"

	"github.com/mainflux/mainflux/auth"
	"github.com/mainflux/mainflux/pkg/errors"
	acl "github.com/ory/keto/proto/ory/keto/acl/v1alpha1"
)

const (
	subjectSetRegex = "^.{1,}:.{1,}#.{1,}$" // expected subject set structure is <namespace>:<object>#<relation>
	ketoNamespace   = "members"
)

type policyAgent struct {
	writer  acl.WriteServiceClient
	checker acl.CheckServiceClient
}

// NewPolicyAgent returns a gRPC communication functionalities
// to communicate with ORY Keto.
func NewPolicyAgent(checker acl.CheckServiceClient, writer acl.WriteServiceClient) auth.PolicyAgent {
	return policyAgent{checker: checker, writer: writer}
}

func (c policyAgent) CheckPolicy(ctx context.Context, pr auth.PolicyReq) error {
	res, err := c.checker.Check(context.Background(), &acl.CheckRequest{
		Namespace: ketoNamespace,
		Object:    pr.Object,
		Relation:  pr.Relation,
		Subject:   getSubject(pr),
	})
	if err != nil {
		return errors.Wrap(err, auth.ErrAuthorization)
	}
	if !res.GetAllowed() {
		return auth.ErrAuthorization
	}
	return nil
}

func (c policyAgent) AddPolicy(ctx context.Context, pr auth.PolicyReq) error {
	trt := c.writer.TransactRelationTuples
	_, err := trt(context.Background(), &acl.TransactRelationTuplesRequest{
		RelationTupleDeltas: []*acl.RelationTupleDelta{
			{
				Action: acl.RelationTupleDelta_INSERT,
				RelationTuple: &acl.RelationTuple{
					Namespace: ketoNamespace,
					Object:    pr.Object,
					Relation:  pr.Relation,
					Subject: &acl.Subject{Ref: &acl.Subject_Id{
						Id: pr.Subject,
					}},
				},
			},
		},
	})
	return err
}

func (c policyAgent) DeletePolicy(ctx context.Context, pr auth.PolicyReq) error {
	trt := c.writer.TransactRelationTuples
	_, err := trt(context.Background(), &acl.TransactRelationTuplesRequest{
		RelationTupleDeltas: []*acl.RelationTupleDelta{
			{
				Action: acl.RelationTupleDelta_DELETE,
				RelationTuple: &acl.RelationTuple{
					Namespace: ketoNamespace,
					Object:    pr.Object,
					Relation:  pr.Relation,
					Subject: &acl.Subject{Ref: &acl.Subject_Id{
						Id: pr.Subject,
					}},
				},
			},
		},
	})
	return err
}

// getSubject returns a 'subject' field for ACL(access control lists).
// If the given PolicyReq argument contains a subject as subject set,
// it returns subject set; otherwise, it returns a subject.
func getSubject(pr auth.PolicyReq) *acl.Subject {
	if isSubjectSet(pr.Subject) {
		return &acl.Subject{
			Ref: &acl.Subject_Set{Set: &acl.SubjectSet{
				Namespace: ketoNamespace,
				Object:    pr.Object,
				Relation:  pr.Relation,
			}},
		}
	}

	return &acl.Subject{Ref: &acl.Subject_Id{Id: pr.Subject}}
}

// isSubjectSet returns true when given subject is subject set.
// Otherwise, it returns false.
func isSubjectSet(subject string) bool {
	r, err := regexp.Compile(subjectSetRegex)
	if err != nil {
		return false
	}
	return r.MatchString(subject)
}
