// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"encoding/json"
	"fmt"
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
