// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package atom

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// externalIDBatchSize bounds how many entities one GraphQL document asks for.
// Atom exposes no list-of-ids filter on `entities`, so a batch is a document of
// aliased `entity(id:)` selections — one round trip, one indexed lookup per id.
const externalIDBatchSize = 100

// EntityExternalIDs resolves entity UUIDs to the external identifiers they were
// registered with (ATOM-06). Only entities that exist and carry an external id
// appear in the result, so the map is a subset of ids and callers must treat a
// missing key as "no external identity" rather than an error.
//
// This is the UUID → external_id direction that authorization needs: policy
// lookups deal in Atom UUIDs, while message rows carry the device's serial
// verbatim (MG-05, MG-06). Without the translation a filter built from policy
// results matches no rows at all, which reads as a permissions bug rather than
// a missing conversion.
func (c *Client) EntityExternalIDs(ctx context.Context, ids []string) (map[string]string, error) {
	out := make(map[string]string, len(ids))
	seen := make(map[string]struct{}, len(ids))

	batch := make([]string, 0, externalIDBatchSize)
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		resolved, err := c.entityExternalIDBatch(ctx, batch)
		if err != nil {
			return err
		}
		for id, externalID := range resolved {
			out[id] = externalID
		}
		batch = batch[:0]
		return nil
	}

	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}

		batch = append(batch, id)
		if len(batch) == externalIDBatchSize {
			if err := flush(); err != nil {
				return nil, err
			}
		}
	}
	if err := flush(); err != nil {
		return nil, err
	}

	return out, nil
}

// entityExternalIDBatch asks for one aliased selection per id. Per-entity
// failures — a deleted entity, one the caller may not read — come back as a null
// alias alongside an errors array; those ids are simply absent from the result.
// Failing the whole batch instead would let a single stale id deny a caller
// every device they hold, so the tolerant decode is the safer direction: an
// unresolved id contributes nothing to the filter it feeds.
func (c *Client) entityExternalIDBatch(ctx context.Context, ids []string) (map[string]string, error) {
	var (
		fields    strings.Builder
		params    = make([]string, 0, len(ids))
		variables = make(map[string]any, len(ids))
		aliases   = make(map[string]string, len(ids))
	)
	for i, id := range ids {
		alias := fmt.Sprintf("e%d", i)
		variable := fmt.Sprintf("id%d", i)
		aliases[alias] = id
		params = append(params, "$"+variable+": ID!")
		variables[variable] = id
		fmt.Fprintf(&fields, "\n\t\t%s: entity(id: $%s) { id external_id: externalId }", alias, variable)
	}

	query := fmt.Sprintf("query EntityExternalIDs(%s) {%s\n\t}", strings.Join(params, ", "), fields.String())

	var response graphQLResponse
	if err := c.do(ctx, http.MethodPost, "/graphql", graphQLRequest{Query: query, Variables: variables}, &response); err != nil {
		return nil, err
	}

	if len(response.Data) == 0 {
		if len(response.Errors) > 0 {
			return nil, graphQLErr(response.Errors)
		}
		return nil, Error{StatusCode: http.StatusInternalServerError, Message: "atom GraphQL response did not include data"}
	}

	var decoded map[string]*Entity
	if err := json.Unmarshal(response.Data, &decoded); err != nil {
		return nil, err
	}

	out := make(map[string]string, len(decoded))
	for alias, entity := range decoded {
		id, ok := aliases[alias]
		if !ok || entity == nil || entity.ExternalID == "" {
			continue
		}
		out[id] = entity.ExternalID
	}
	return out, nil
}
