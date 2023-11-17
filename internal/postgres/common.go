// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

var (
	// ErrAssignToGroup indicates failure to assign member to a group.
	ErrAssignToGroup = errors.New("failed to assign member to a group")

	// ErrUnassignFromGroup indicates failure to unassign member from a group.
	ErrUnassignFromGroup = errors.New("failed to unassign member from a group")

	// ErrMissingParent indicates that parent can't be found.
	ErrMissingParent = errors.New("failed to retrieve parent")

	// ErrGroupNotEmpty indicates group is not empty, can't be deleted.
	ErrGroupNotEmpty = errors.New("group is not empty")

	// ErrMemberAlreadyAssigned indicates that members is already assigned.
	ErrMemberAlreadyAssigned = errors.New("member is already assigned")

	// ErrFailedToRetrieveMembers failed to retrieve group members.
	ErrFailedToRetrieveMembers = errors.New("failed to retrieve group members")

	// ErrFailedToRetrieveMembership failed to retrieve memberships.
	ErrFailedToRetrieveMembership = errors.New("failed to retrieve memberships")

	// ErrFailedToRetrieveAll failed to retrieve groups.
	ErrFailedToRetrieveAll = errors.New("failed to retrieve all groups")

	// ErrFailedToRetrieveParents failed to retrieve groups.
	ErrFailedToRetrieveParents = errors.New("failed to retrieve all groups")

	// ErrFailedToRetrieveChildren failed to retrieve groups.
	ErrFailedToRetrieveChildren = errors.New("failed to retrieve all groups")
)

func CreateMetadataQuery(entity string, um map[string]interface{}) (string, []byte, error) {
	if len(um) == 0 {
		return "", nil, nil
	}
	param, err := json.Marshal(um)
	if err != nil {
		return "", nil, err
	}
	query := fmt.Sprintf("%smetadata @> :metadata", entity)

	return query, param, nil
}

func Total(ctx context.Context, db Database, query string, params interface{}) (uint64, error) {
	rows, err := db.NamedQueryContext(ctx, query, params)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	total := uint64(0)
	if rows.Next() {
		if err := rows.Scan(&total); err != nil {
			return 0, err
		}
	}
	return total, nil
}
