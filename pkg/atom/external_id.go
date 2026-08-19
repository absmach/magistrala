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

// entityBatchSize bounds how many entities one GraphQL document asks for.
// Atom exposes no list-of-ids filter on `entities`, so a batch is a document of
// aliased `entity(id:)` selections — one round trip, one indexed lookup per id.
const entityBatchSize = 100

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
	raw, err := c.batchEntities(ctx, ids, "EntityExternalIDs", "id external_id: externalId")
	if err != nil {
		return nil, err
	}

	out := make(map[string]string, len(raw))
	for id, data := range raw {
		var entity Entity
		if err := json.Unmarshal(data, &entity); err != nil {
			return nil, err
		}
		if entity.ExternalID != "" {
			out[id] = entity.ExternalID
		}
	}
	return out, nil
}

// entitiesExist reports which of the given entity ids currently resolve —
// exist and are readable by this client — without fetching anything beyond
// their id. Used to narrow a declared set (e.g. a device's `gateways`
// attribute) down to the ones that still exist, in batches of at most
// entityBatchSize round trips instead of one GetEntity call per id.
func (c *Client) entitiesExist(ctx context.Context, ids []string) (map[string]struct{}, error) {
	raw, err := c.batchEntities(ctx, ids, "EntityExistence", "id")
	if err != nil {
		return nil, err
	}

	out := make(map[string]struct{}, len(raw))
	for id := range raw {
		out[id] = struct{}{}
	}
	return out, nil
}

// batchEntities resolves ids to whatever fields the caller selects, deduping
// and chunking them into batches of at most entityBatchSize before handing
// each batch to entityBatch. The result only ever contains ids that were
// actually asked for and actually resolved.
func (c *Client) batchEntities(ctx context.Context, ids []string, opName, fields string) (map[string]json.RawMessage, error) {
	out := make(map[string]json.RawMessage, len(ids))
	seen := make(map[string]struct{}, len(ids))

	batch := make([]string, 0, entityBatchSize)
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		resolved, err := c.entityBatch(ctx, batch, opName, fields)
		if err != nil {
			return err
		}
		for id, data := range resolved {
			out[id] = data
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
		if len(batch) == entityBatchSize {
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

// entityBatch asks for one aliased `fields` selection per id, in a single
// GraphQL request named opName. Per-entity failures — a deleted entity, one
// the caller may not read — come back as a null alias alongside an errors
// array; those ids are simply absent from the result. Failing the whole
// batch instead would let a single stale id deny a caller every device they
// hold (or every gateway they declared), so the tolerant decode is the
// safer direction: an unresolved id contributes nothing to whatever it
// feeds.
func (c *Client) entityBatch(ctx context.Context, ids []string, opName, fields string) (map[string]json.RawMessage, error) {
	var (
		selection strings.Builder
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
		fmt.Fprintf(&selection, "\n\t\t%s: entity(id: $%s) { %s }", alias, variable, fields)
	}

	query := fmt.Sprintf("query %s(%s) {%s\n\t}", opName, strings.Join(params, ", "), selection.String())

	var response graphQLResponse
	if err := c.do(ctx, http.MethodPost, "/graphql", graphQLRequest{Query: query, Variables: variables}, &response); err != nil {
		return nil, err
	}

	// A per-entity failure (see the doc comment above) leaves response.Data a
	// real object with just that alias null, accompanied by an errors entry
	// for it — that case must fall through to the tolerant decode below, not
	// bail out here. Only the absence of a data document at all — a genuine
	// top-level failure — is fatal, and it has to be checked for both the
	// way encoding/json actually represents that ("data" omitted, so
	// response.Data is empty) and the way a GraphQL server commonly spells it
	// on the wire (`"data": null`, which is 4 bytes, not zero, and would
	// otherwise unmarshal into a nil map and silently return an empty batch
	// instead of surfacing the failure).
	if len(response.Data) == 0 || string(response.Data) == "null" {
		if len(response.Errors) > 0 {
			return nil, graphQLErr(response.Errors)
		}
		return nil, Error{StatusCode: http.StatusInternalServerError, Message: "atom GraphQL response did not include data"}
	}

	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(response.Data, &decoded); err != nil {
		return nil, err
	}

	out := make(map[string]json.RawMessage, len(decoded))
	for alias, raw := range decoded {
		id, ok := aliases[alias]
		if !ok || raw == nil || string(raw) == "null" {
			continue
		}
		out[id] = raw
	}
	return out, nil
}
