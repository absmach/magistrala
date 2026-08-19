// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package atom

import (
	"context"
	"encoding/json"
	"errors"
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
	// A per-entity failure is deliberately tolerated here: an id that could
	// not be read simply contributes nothing to the resulting filter, which
	// is the safe direction for this caller (see batchEntities).
	raw, _, err := c.batchEntities(ctx, ids, "EntityExternalIDs", "id external_id: externalId")
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
//
// Unlike EntityExternalIDs, absence here is load-bearing: DeviceGateways and
// SetDeviceGateways treat a dropped id as "this gateway link was deleted" and
// write that conclusion back. An id batchEntities could not read — a
// permission failure, a transient Atom error — must not be folded into that
// same "does not exist" bucket, or a caller doing read-modify-write can
// permanently erase a link that was never actually gone.
func (c *Client) entitiesExist(ctx context.Context, ids []string) (map[string]struct{}, error) {
	raw, unreadable, err := c.batchEntities(ctx, ids, "EntityExistence", "id")
	if err != nil {
		return nil, err
	}
	if len(unreadable) > 0 {
		return nil, fmt.Errorf("atom: could not determine existence of %d of %d entities (e.g. %s): %w",
			len(unreadable), len(ids), firstKey(unreadable), errEntitiesUnreadable)
	}

	out := make(map[string]struct{}, len(raw))
	for id := range raw {
		out[id] = struct{}{}
	}
	return out, nil
}

func firstKey(m map[string]string) string {
	for k := range m {
		return k
	}
	return ""
}

// errEntitiesUnreadable marks the entitiesExist error above so callers can
// tell it apart from an ordinary transport failure if they ever need to.
var errEntitiesUnreadable = errors.New("one or more entities could not be read")

// batchEntities resolves ids to whatever fields the caller selects, deduping
// and chunking them into batches of at most entityBatchSize before handing
// each batch to entityBatch. The resolved map only ever contains ids that
// were actually asked for and actually resolved; the second map holds ids
// that came back with a per-entity error attached, distinct from ids simply
// absent because they do not exist (see entityBatch).
func (c *Client) batchEntities(ctx context.Context, ids []string, opName, fields string) (map[string]json.RawMessage, map[string]string, error) {
	out := make(map[string]json.RawMessage, len(ids))
	unreadable := make(map[string]string)
	seen := make(map[string]struct{}, len(ids))

	batch := make([]string, 0, entityBatchSize)
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		resolved, failed, err := c.entityBatch(ctx, batch, opName, fields)
		if err != nil {
			return err
		}
		for id, data := range resolved {
			out[id] = data
		}
		for id, msg := range failed {
			unreadable[id] = msg
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
				return nil, nil, err
			}
		}
	}
	if err := flush(); err != nil {
		return nil, nil, err
	}

	return out, unreadable, nil
}

// entityBatch asks for one aliased `fields` selection per id, in a single
// GraphQL request named opName. Per-entity failures — a deleted entity, one
// the caller may not read — come back as a null alias, sometimes alongside an
// errors array entry naming that alias in its path. Failing the whole batch
// instead would let a single stale id deny a caller every device they hold
// (or every gateway they declared), so entityBatch never bails out on a
// per-entity failure.
//
// It does, however, tell the two cases apart in its return: a null alias
// with no error attached is a genuine "does not exist" and lands only in the
// absent set (dropped silently, same as before); a null alias with an error
// attached — permission denied, a transient failure on that one field — lands
// in the second map instead, so a caller for whom absence is load-bearing
// (entitiesExist) can refuse to treat "could not tell" as "does not exist".
func (c *Client) entityBatch(ctx context.Context, ids []string, opName, fields string) (resolved map[string]json.RawMessage, unreadable map[string]string, err error) {
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
		return nil, nil, err
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
			return nil, nil, graphQLErr(response.Errors)
		}
		return nil, nil, Error{StatusCode: http.StatusInternalServerError, Message: "atom GraphQL response did not include data"}
	}

	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(response.Data, &decoded); err != nil {
		return nil, nil, err
	}

	// Map each per-entity error back to the alias it was reported against, so
	// a null caused by "could not read" can be told apart below from a null
	// caused by "does not exist".
	erroredAliases := make(map[string]string, len(response.Errors))
	for _, item := range response.Errors {
		if len(item.Path) == 0 {
			continue
		}
		if alias, ok := item.Path[0].(string); ok {
			erroredAliases[alias] = item.Message
		}
	}

	resolved = make(map[string]json.RawMessage, len(decoded))
	unreadable = make(map[string]string, len(erroredAliases))
	for alias, raw := range decoded {
		id, ok := aliases[alias]
		if !ok {
			continue
		}
		if msg, failed := erroredAliases[alias]; failed {
			unreadable[id] = msg
			continue
		}
		if raw == nil || string(raw) == "null" {
			continue
		}
		resolved[id] = raw
	}
	return resolved, unreadable, nil
}
