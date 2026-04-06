// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package spicedb

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/absmach/magistrala/pkg/roles"
	corev1 "github.com/authzed/spicedb/pkg/proto/core/v1"
	"github.com/authzed/spicedb/pkg/schemadsl/compiler"
	"github.com/authzed/spicedb/pkg/schemadsl/input"
)

func GetActionsFromSchema(schemaPath string, objectType string) ([]roles.Action, error) {
	objectType = strings.TrimSpace(objectType)
	if objectType == "" {
		return []roles.Action{}, fmt.Errorf("object type is empty string")
	}

	file, err := os.Open(schemaPath)
	if err != nil {
		return []roles.Action{}, err
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return []roles.Action{}, err
	}

	compiledSchema, err := compiler.Compile(compiler.InputSchema{
		Source:       input.Source("schema"),
		SchemaString: string(data),
	}, compiler.AllowUnprefixedObjectType())
	if err != nil {
		return []roles.Action{}, err
	}

	actions := []roles.Action{}
	for _, od := range compiledSchema.ObjectDefinitions {
		if objectType == od.Name {
			for _, relation := range od.Relation {
				if relation.UsersetRewrite == nil && relation.TypeInformation != nil && isAction(relation.TypeInformation) {
					relName := strings.TrimSpace(relation.GetName())
					if relName == "" {
						return []roles.Action{}, fmt.Errorf("got empty relation name")
					}
					actions = append(actions, roles.Action(relName))
				}
			}
		}
	}

	if len(actions) == 0 {
		return []roles.Action{}, fmt.Errorf("no actions found for type %s", objectType)
	}
	return actions, nil
}

func isAction(ti *corev1.TypeInformation) bool {
	for _, ar := range ti.AllowedDirectRelations {
		if ar.GetNamespace() == "role" && ar.GetRelation() == "member" {
			return true
		}
	}
	return false
}
