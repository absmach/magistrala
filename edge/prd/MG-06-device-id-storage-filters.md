# MG-06 — Persist and filter `device_id`

| | |
|---|---|
| **Repo** | `absmach/magistrala` (Go) |
| **Priority** | P2 |
| **Depends on** | MG-05 |
| **Blocks** | MG-08 |
| **Status** | **⏸ Phase 2 — deferred** |

> **⏸ Deferred to phase 2** ([spec §8 A14](../architecture.md#8-decision-record)).
> Nothing to store: `device_id` arrives only with MG-05.
>
> The design below is unchanged and remains the target — this is a scope call,
> not a reversal.

## Problem

MG-05 puts `device_id` on the message. Nothing persists it and nothing can query
by it, so per-meter history — the actual user-visible requirement — still does
not exist.

## Scope

**In scope**

- `device_id` column on the SenML tables in both backends:
  `consumers/writers/postgres/init.go:15-43`,
  `consumers/writers/timescale/init.go:15-69`.
- Same for the on-demand JSON tables
  (`consumers/writers/postgres/consumer.go:158-173`).
- Carry `device_id` through both transformers into their message structs
  (`pkg/transformers/senml/transformer.go:59-91`,
  `pkg/transformers/json/transformer.go:48-58`) and the INSERTs
  (`postgres/consumer.go:53-58`, `timescale/consumer.go:59-64`).
- `DeviceIDs []string` on `readers.PageMetadata` (`readers/messages.go:43-63`),
  with the `= ANY(:device_ids)` condition in both backends.
- Expose on all three transports: gRPC proto
  (`internal/proto/readers/v1/readers.proto`), HTTP query key
  (`readers/api/http/transport.go:27-49`), SDK (`pkg/sdk/sdk.go:83`).

**Out of scope**

- Authorization. `DeviceIDs` here is a **convenience filter**, exactly like
  `publishers` today. MG-08 makes it a boundary. Do not half-enforce it here —
  partial enforcement is worse than none, because it reads as a guarantee.
- Backfill of historical rows.

## Design

### Follow the merged template

Commit `14d6db968` ("Add multi-publisher filter to readers PageMetadata", #3550)
is a complete worked example of this exact change: struct field with `omitempty`,
param bind, `= ANY(...)` condition in both backends, proto field, gRPC wiring.
Mirror it.

Note that commit added `publishers` to **gRPC only** — no HTTP query key, not in
the SDK. Close that gap for `publishers` while adding `device_ids`, so the two
filters are not asymmetric across transports.

### Filter mechanics

The WHERE builders JSON-marshal `PageMetadata` and iterate the resulting map, so
`omitempty` is what makes a filter "unset". Consequences to respect:

- `DeviceIDs: []string{}` is omitted entirely — an empty slice cannot mean "match
  nothing". **MG-08 depends on this distinction**, because an authorized set that
  is legitimately empty must return zero rows, not all rows. Either represent it
  as `*[]string`, or have MG-08 short-circuit before reaching the query. Decide
  here and write it down; discovering it in MG-08 means reworking both.

### What `device_id` contains — state this before writing any SQL

**The device's serial, verbatim.** Not a platform UUID, not a resolved entity
reference — the exact string that appeared in the publish topic
([spec §2.5](../architecture.md#25--phase-2--message-attribution),
[§8 A8](../architecture.md#8-decision-record)).

Consequences that shape this PRD:

- The column is `TEXT`. There is no foreign key and can be none — the publish path
  performs no lookup, by design.
- **Rows may reference devices that do not exist.** Late binding means data
  arrives for serials with no device record, and is stored anyway. Registering the
  device later makes its history queryable retroactively. Nothing here may reject
  or quarantine such rows.
- **MG-08 must translate.** `authorizedObjectIds` returns Atom UUIDs; this column
  holds serials. The authorized set is mapped UUID → `external_id` before it can
  filter. Getting this wrong yields a filter matching nothing, which presents as a
  permissions bug.
- Serials never contain `/` (rejected at device creation, MG-09), so no escaping
  or encoding is involved at any layer.

### Schema

Primary keys currently include `publisher` — Postgres `(time, publisher,
subtopic, name)` (`postgres/init.go:37-43`), Timescale `(time, channel, subtopic,
protocol, publisher, name)` (`timescale/init.go`). Add `device_id` **alongside**;
do not replace `publisher`, which remains the audit identity.

Timescale indexes lead with `channel` and end with `name, time DESC`. Add
`(channel, device_id, name, time DESC)` to match that convention.

### Migration

Existing rows have no device. `device_id` must be nullable (or empty-string
default) and every query must treat "no device" as a first-class case — direct
publishers legitimately have none.

## Acceptance criteria

1. A message published with a device segment is stored with `device_id` set to
   the **exact serial string** from the topic — byte-identical, including case
   and `.`/`-`/`:` characters.
1a. A message for a serial with **no device record** is stored and queryable, and
   becomes attributable once that device is created.
2. A message published without one is stored with `device_id` empty/null, and
   every existing query returns identical results to before.
3. `DeviceIDs` filtering returns only matching rows, in both backends.
4. It composes with `publisher`, `publishers`, `subtopic`, `name`, time range and
   aggregation.
5. Available over gRPC, HTTP and SDK, with consistent semantics.
6. `publishers` is reachable over HTTP and SDK, closing the #3550 gap.
7. A SenML pack fanned out to N rows carries the same `device_id` on every row.
8. Migration against a populated table succeeds and leaves existing rows queryable.

## Test plan

- Unit: WHERE-builder output for `DeviceIDs` set, unset, single and multiple;
  and explicitly for the empty-slice case, asserting the decision above.
- Integration (`ory/dockertest`, as the reader/writer suites already use):
  write messages with distinct `device_id`s to one channel, assert filtering in
  both backends, assert `total` correctness under pagination.
- Migration: run against a table seeded with pre-change rows.
- Regression: full existing reader suite.

## Risks

- **Primary-key change on a hypertable.** Timescale PK changes on a populated
  hypertable can be expensive or need a rewrite. Confirm the migration strategy
  against a realistically sized table before merging — this is the highest-risk
  operational item in the PRD.
- **The empty-slice semantics** are a genuine trap. Called out above precisely so
  MG-08 does not inherit it as a surprise.
