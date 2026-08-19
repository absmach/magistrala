# MG-05 — Topic grammar: device segment and `device_id`

| | |
|---|---|
| **Repo** | `absmach/magistrala` (Go) |
| **Priority** | P2 |
| **Depends on** | — |
| **Blocks** | MG-06 |
| **Status** | **⏸ Phase 2 — deferred** |

> **⏸ Deferred to phase 2** ([spec §8 A14](../architecture.md#8-decision-record)).
> Topics are application-level in phase 1 — no `d` segment is added.
>
> The design below is unchanged and remains the target — this is a scope call,
> not a reversal.

## Problem

A gateway publishing for 200 meters produces 200 streams of data attributed to
one publisher. `Message.publisher` is stamped by the broker from the
authenticated connection and cannot carry per-meter identity, so per-meter views
are impossible.

Two identities are needed, answering different questions:

| Field | Question | Set by | Spoofable |
|---|---|---|---|
| `publisher` | Who sent it | Broker, from the authenticated connection | No |
| `device_id` | Whose data it is | Taken verbatim from the topic | Yes, within the publisher's channels |

`publisher` semantics stay **exactly** as they are. This PRD only adds.

## Scope

**In scope**

- Topic grammar:
  ```
  m/<domain>/c/<channel>[/d/<device>][/<subtopic>]
  hc/<domain>
  ```
  `<device>` is the device's **serial** — an arbitrary string, carried verbatim.
  Serial *is* the device id in this model
  ([spec §8 A8](../architecture.md#8-decision-record)); there is no separate platform-assigned
  identifier in the topic.

  **Slash is the one constraint**, since it is the topic separator. Uppercase,
  `.`, `-`, `:` and unicode pass through untouched. A serial containing `/` is
  rejected at device creation (MG-09), never mangled or encoded here —
  percent-encoding was rejected because it puts encode/decode on the publish path
  and breaks "verbatim".
- `DeviceTopicPrefix = 'd'` alongside `MsgTopicPrefix` / `ChannelTopicPrefix`
  (`pkg/messaging/topics.go:20-23`).
- Extend `ParseTopic` (`topics.go:373-452`) and every caller of the changed
  signature: `ParsePublishTopic` (`:249`), `ParseSubscribeTopic` (`:283`),
  encoders (`:333-364`), FluxMQ variant (`pkg/messaging/fluxmq/topic.go:82-97`).
- `string device_id = 10` on `pkg/messaging/message.proto:10-21`; regenerate.
- Populate in the inbound constructor (`pkg/messaging/fluxmq/pubsub.go:233-252`)
  and mirror in `messageProperties` (`pkg/messaging/fluxmq/publisher.go:146-162`)
  so republishes preserve it.

  **Republish preservation is what makes aggregation attribute correctly.** Per
  [spec §2.2](../architecture.md#22-what-follows-from-that--normative)
  consequence 5, a value computed *about* meter-7 — a daily total, a rolling
  average — belongs to meter-7 and must carry its `device_id`, not the identity of
  whatever computed it. Dropping `device_id` on republish would silently
  re-attribute every derived value.

**Out of scope**

- **Resolution.** The device segment is carried **verbatim** — it is the device's
  external identifier (serial), not an Atom UUID, and is never looked up on the
  publish path ([spec §8 A8](../architecture.md#8-decision-record)). Earlier drafts of this PRD
  specified route→ID resolution through the ristretto cache; that is removed. A
  per-message entity lookup is exactly what channel-as-boundary exists to avoid.
- **Attachment validation.** There is none — the channel is the boundary
  ([spec §8 A7](../architecture.md#8-decision-record)). [MG-07](./MG-07-gateway-attachment-enforcement.md)
  is withdrawn.
- Persistence and query — MG-06.
- Command/downlink topics.

## Design

### Why a marked segment

Subtopic is already load-bearing: `pkg/transformers/json/transformer.go:69` uses
the **last** subtopic segment as the destination table name. Device IDs in the
subtopic would create a table per device. The `d` marker leaves subtopic
semantics untouched and makes the device position unambiguous.

### Parsing

After the channel ID, if the next segment is exactly `d`, the following segment
is the device and the remainder is subtopic. `ParseTopic` is a hand-rolled byte
scanner with no regex — keep it that way; it is on the hot path.

### The reserved segment

`m/dom/c/chan/d/x` is ambiguous: `x` could be a device, or `d/x` a subtopic.
Resolve by **reserving `d` as a first subtopic segment**. Publishing to a
subtopic beginning with a `d` segment is rejected at validation
(`ParsePublishSubtopic`, `topics.go:262-281`).

Breaking, deliberate, and cheap before 1.0. It must be documented in the topic
grammar comment (`topics.go:366-372`) and the messaging README, not left as a
parser quirk.

### Per-message, not per-record

Attribution comes from the topic, so it is per message — like `publisher`. One
publish carries one device's data. This is simpler than the SenML `bn` route
(no base-name tracking, no split after `Normalize`, no per-record resolution) at
the cost of no multi-sensor batching in a single pack.

## Acceptance criteria

1. `m/dom/c/chan/d/dev123/sub/topic` parses to domain, channel, device `dev123`,
   subtopic `sub/topic`.
2. `m/dom/c/chan/d/dev123` parses with an empty subtopic.
3. `m/dom/c/chan/sub/topic` parses with an empty device — existing behaviour is
   byte-identical.
4. Publishing to a subtopic whose first segment is `d` is rejected with a clear
   error.
5. The device segment is carried **verbatim**, including uppercase, `.` and `-`.
   No lookup occurs and no error is raised for an unknown identifier — the
   publish path performs zero entity resolution, asserted by counting Atom calls.
6. `Message.DeviceId` is populated end to end from an MQTT publish.
7. Republishing through the writer path preserves `device_id`.
8. Existing topics without a device segment are unaffected everywhere —
   subscribe, health, wildcards.

## Test plan

- Table-driven `ParseTopic` tests extending `pkg/messaging/topics_test.go`: with
  and without device, with and without subtopic, leading slash, the reserved-`d`
  case, malformed forms (`d` with no device, trailing `d/`), and wildcards in
  subscribe topics.
- Zero-resolution: assert the publish path makes no Atom call, including for an
  identifier no device holds.
- Integration: publish over MQTT with a device segment; assert `device_id` on the
  consumed message.
- Regression: the full existing topic suite must pass untouched — this is the
  primary guard for criterion 8.

## Risks

- **Signature change to `ParseTopic` fans out.** It is called from the broker hot
  path and from FluxMQ's own topic handling. Missing a caller is a compile error
  in-repo, but check for callers outside it.
- **Hot-path cost.** The scanner runs per publish. The device branch adds one
  segment comparison; keep allocations at zero and benchmark before merging.
- **Reserved `d`** breaks any existing deployment publishing to such a subtopic.
  Low likelihood, but it is a real break and belongs in release notes.
