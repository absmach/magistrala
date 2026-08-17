# MG-03 — Group membership, hierarchy and group kinds

| | |
|---|---|
| **Repo** | `absmach/magistrala` (Go) |
| **Priority** | P1 |
| **Depends on** | ATOM-04 |
| **Blocks** | MG-04, MG-09 |
| **Status** | Draft |

## Scope note — groups are for sharing only

Object groups in this model have **exactly one purpose: granting access**
([spec §3.5](../architecture.md#35-groups-and-sharing)). They are not used for
device↔gateway topology — that is a `gateways []` attribute on the device
([spec §8 A10](../architecture.md#8-decision-record)) — and not for any other
grouping.

An intermediate design put gateway fleets in this same namespace; it was reversed
because a group-scoped grant on a fleet group would silently hand over an entire
gateway's devices. Keep the namespace single-purpose.

## Problem

`pkg/atom` can create, read, update, delete and list groups — and nothing else.
It cannot add a member, remove a member, list members, or create a nested group.
The only code in the tree that writes group membership is the one-shot migration
tool, via raw SQL (`tools/atom-migration/migrator.go:598,619`).

Without membership there is no sharing model: the customer story in
[architecture.md §5.2](../architecture.md#35-groups-and-sharing) depends on a group
holding a device set.

## What Atom already provides

| Operation | Location |
|---|---|
| `addGroupMember`, `removeGroupMember` | `src/graphql/groups.rs:645,699` |
| `groupMembers(groupId)`, `entityGroups(entityId)` | `:90,104` |
| `createObjectGroup`, `createPrincipalGroup` | `:346,355` |
| `objectGroups`, `principalGroups`, `childGroups` | `:154,184,121` |
| `setGroupParent`, `removeGroupParent` | `:429,478` |
| Object-side variants `setObjectGroupParent`, `removeObjectGroupParent` | `:468,517` |

No Atom change is required.

## Two things the current client gets wrong

### Object groups vs principal groups

Atom has two distinct group namespaces backed by separate tables —
`object_groups` (`001_initial.sql:427-441`) and `principal_groups`
(`:386-400`), unioned only by a compatibility view (`:461-466`). `pkg/atom` uses
the generic `createGroup`, so which table a group lands in is implicit.

Magistrala's usage divides cleanly:

- **Object group** — a set of devices or channels. What customer sharing needs.
- **Principal group** — a set of users. A subject in a policy.

These must be explicit in the client. Creating the wrong kind produces a group
that silently cannot be used as intended.

### `parentId` is dropped

`groupCreateInput` (`client.go:751-760`) builds the create input without
`parentId` even though `Group` carries it (`types.go:39`). Nested groups are
therefore uncreatable from Go. The hierarchy is needed for
Customer → Site → meters roll-ups via `includeDescendants`.

## Scope

**In scope**

- Membership: `AddGroupMember`, `RemoveGroupMember`, `GroupMembers`,
  `EntityGroups`.
- Typed creation: `CreateObjectGroup`, `CreatePrincipalGroup`. Retain the generic
  `CreateGroup` only if a caller genuinely needs kind-agnostic behaviour;
  otherwise remove it so the choice is always explicit.
- Hierarchy: `SetGroupParent`, `RemoveGroupParent`, `ChildGroups`; fix
  `groupCreateInput` to send `parentId`.
- Listing: `ObjectGroups`, `PrincipalGroups`, and `includeDescendants` where Atom
  accepts it.

**Out of scope**

- Group-scoped permission blocks — MG-04.
- Resource membership (`object_group_resources`). Add when a channel-grouping
  requirement appears; there is none yet.

## Membership is many-to-many (after ATOM-04)

Originally Atom enforced `PRIMARY KEY (entity_id)` on `object_group_entities`,
meaning an entity belonged to at most one object group — and
`set_entity_parent_group_in_tx` (`src/identity/repo.rs:511-523`) **silently
moved** it between groups on re-add.

[ATOM-04](./ATOM-04-many-to-many-group-membership.md) removes that constraint per
[spec A1](../architecture.md#8-decision-record), so:

- `AddGroupMember` is **additive**. An entity in group A added to group B is in
  both.
- Re-adding to a group it already belongs to is idempotent.
- `RemoveGroupMember` takes a group ID and removes only that membership.
- `EntityGroups` returning a list (`groups.rs:104`) is now literally correct.

**Do not start this PRD before ATOM-04 lands.** Building against the old
move-on-conflict semantics produces a client whose documented behaviour inverts
under it.

### Consequence for grants

A device can now reach a permission block through more than one group. Nothing in
the client changes for that, but it means removing a device from one sharing
group does **not** necessarily revoke access — another group may still grant it.
Any UI showing "who can see this device" must ask
[ATOM-03](./ATOM-03-reverse-policy-lookup.md) rather than inferring from a single
group.

## Acceptance criteria

1. Create an object group, add three devices, list members — all three returned.
2. Remove one; the remaining two are returned.
3. `EntityGroups` for a member returns every group it belongs to.
4. A device added to two groups appears in both member listings, and
   `RemoveGroupMember` on one leaves the other intact.
5. Create a nested group by passing `parentId` at creation; `ChildGroups` on the
   parent returns it.
6. `SetGroupParent` / `RemoveGroupParent` reparent an existing group.
7. Object and principal groups are created in the correct namespace, verified by
   listing each kind separately.
8. Members from another tenant are rejected.

## Test plan

- Integration (Atom in Docker) for all of the above — membership semantics cannot
  be established against a mock, since they live in Atom's schema and repository
  layer.
- Explicit test for criterion 4; multi-group membership is the whole point of
  ATOM-04 and the behaviour most likely to regress.
- Hierarchy: 3-level tree, assert direct vs descendant listing at each level.
- Authorization: a caller without `manage` on the group is refused
  (`groups.rs:660-668`).

## Risks

- **Sequencing.** Built against pre-ATOM-04 Atom, `AddGroupMember` moves rather
  than adds. Every acceptance criterion around multi-group membership would pass
  vacuously or invert. Gate on ATOM-04.
- **Revocation is no longer "remove from the group".** With multi-group
  membership, a device may retain access through another group. Anything that
  presents removal as revocation will be wrong; this needs to be explicit in the
  method documentation and in whatever UI consumes it.
