// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package keto

import (
	"context"
	"regexp"
	"strings"

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
	reader  acl.ReadServiceClient
}

// NewPolicyAgent returns a gRPC communication functionalities
// to communicate with ORY Keto.
func NewPolicyAgent(checker acl.CheckServiceClient, writer acl.WriteServiceClient, reader acl.ReadServiceClient) auth.PolicyAgent {
	return policyAgent{checker: checker, writer: writer, reader: reader}
}

func (pa policyAgent) CheckPolicy(ctx context.Context, pr auth.PolicyReq) error {
	res, err := pa.checker.Check(context.Background(), &acl.CheckRequest{
		Namespace: ketoNamespace,
		Object:    pr.Object,
		Relation:  pr.Relation,
		Subject:   getSubject(pr),
	})
	if err != nil {
		return errors.Wrap(err, errors.ErrAuthorization)
	}
	if !res.GetAllowed() {
		return errors.ErrAuthorization
	}
	return nil
}

func (pa policyAgent) AddPolicy(ctx context.Context, pr auth.PolicyReq) error {
	var ss *acl.Subject
	switch isSubjectSet(pr.Subject) {
	case true:
		namespace, object, relation := parseSubjectSet(pr.Subject)
		ss = &acl.Subject{
			Ref: &acl.Subject_Set{Set: &acl.SubjectSet{Namespace: namespace, Object: object, Relation: relation}},
		}
	default:
		ss = &acl.Subject{Ref: &acl.Subject_Id{Id: pr.Subject}}
	}

	trt := pa.writer.TransactRelationTuples
	_, err := trt(context.Background(), &acl.TransactRelationTuplesRequest{
		RelationTupleDeltas: []*acl.RelationTupleDelta{
			{
				Action: acl.RelationTupleDelta_INSERT,
				RelationTuple: &acl.RelationTuple{
					Namespace: ketoNamespace,
					Object:    pr.Object,
					Relation:  pr.Relation,
					Subject:   ss,
				},
			},
		},
	})
	return err
}

func (pa policyAgent) DeletePolicy(ctx context.Context, pr auth.PolicyReq) error {
	trt := pa.writer.TransactRelationTuples
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

func (pa policyAgent) RetrievePolicies(ctx context.Context, pr auth.PolicyReq) ([]*acl.RelationTuple, error) {
	var ss *acl.Subject
	switch isSubjectSet(pr.Subject) {
	case true:
		namespace, object, relation := parseSubjectSet(pr.Subject)
		ss = &acl.Subject{
			Ref: &acl.Subject_Set{Set: &acl.SubjectSet{Namespace: namespace, Object: object, Relation: relation}},
		}
	default:
		ss = &acl.Subject{Ref: &acl.Subject_Id{Id: pr.Subject}}
	}

	res, err := pa.reader.ListRelationTuples(ctx, &acl.ListRelationTuplesRequest{
		Query: &acl.ListRelationTuplesRequest_Query{
			Namespace: ketoNamespace,
			Relation:  pr.Relation,
			Subject:   ss,
		},
	})
	if err != nil {
		return []*acl.RelationTuple{}, err
	}

	tuple := res.GetRelationTuples()
	for res.NextPageToken != "" {
		tuple = append(tuple, res.GetRelationTuples()...)
	}

	return tuple, nil
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

func parseSubjectSet(subjectSet string) (namespace, object, relation string) {
	r := strings.Split(subjectSet, ":")
	if len(r) != 2 {
		return
	}
	namespace = r[0]

	r = strings.Split(r[1], "#")
	if len(r) != 2 {
		return
	}

	object = r[0]
	relation = r[1]

	return
}
