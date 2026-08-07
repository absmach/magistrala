# MG-04 — Group-scoped permission blocks

| | |
|---|---|
| **Repo** | `absmach/magistrala` (Go) |
| **Priority** | P1 |
| **Depends on** | MG-01, MG-03 |
| **Blocks** | MG-08 |
| **Status** | Draft |

## Problem

`pkg/atom` writes only three of Atom's ten scope modes — `object`, `tenant` and
`platform` (`policy_service.go:196-205`). Everything that is not a domain or the
platform collapses to a single-instance `object` block (`:203`).

Granting a customer read access to 500 meters therefore means 500 permission
blocks and 500 direct policies, because `AddPolicy` (`:38-67`) creates a fresh
block per call with no reuse. That is slow to write, expensive to evaluate, and
walks straight into the truncating revocation path fixed in MG-01.

Atom supports exactly what is needed and it is unreachable from Go.

## What Atom supports

`scope_mode` accepts (`migrations/001_initial.sql:607`):

```
platform, tenant, object_kind, object_type, object,
group, group_direct_objects, group_descendant_objects,
group_child_groups, group_descendant_groups
```

Constraints per mode at `:619-627`. For `group_direct_objects` /
`group_descendant_objects`: `tenant_id` and `group_id` required, `object_kind ∈
{entity, resource}`, `object_id` must be null (`:625`).

Atom computes the scope reference as `{group_id}:{object_type}` —
`src/authz/engine.rs:1334` produces `"{group_id}:entity:device"`, matching
`001_initial.sql:729`.

## Scope

**In scope**

- Extend `policyGrantScopeMode` (`policy_service.go:196-205`) and the companion
  `policyGrantObjectKind` / `ObjectType` / `ObjectID` functions (`:207-248`) to
  emit group-scoped blocks.
- A direct API for the customer-sharing case rather than squeezing it through the
  legacy `policies.Policy` shape:
  ```go
  GrantGroupAccess(ctx, GroupGrant{
      TenantID, GroupID, SubjectKind, SubjectID,
      ObjectKind, ObjectType, Actions []string,
      IncludeDescendants bool,
  }) error
  RevokeGroupAccess(ctx, GroupGrant) error
  ListGroupGrants(ctx, groupID) ([]GroupGrant, error)
  ```
- `directPolicyMatches` (`:250-266`) must compare `GroupID` too, or revocation
  will not match group-scoped blocks.

**Out of scope**

- `object_kind` / `object_type` scope modes ("every device in the tenant"). No
  requirement yet; adding unused grant shapes is how permission models rot.
- `group_child_groups` / `group_descendant_groups` (grants over *groups*, not
  their contents). Add when group-management delegation is actually needed.
- Block reuse for `object`-scoped grants. Group scoping removes the case that
  made it urgent.

## Design

### Grant shape

One block, one policy, per customer:

```
PermissionBlock {
  scope_mode:  "group_direct_objects",
  tenant_id:   <domain>,
  group_id:    <customer group>,
  object_kind: "entity",
  object_type: "entity:device",
  effect:      "allow",
  actions:     [read]
}
DirectPolicy { subject_kind: "entity", subject_id: <customer user>, … }
```

Adding a meter to the customer's group grants access. Removing it revokes.
**Membership becomes the sharing operation** — no policy write per device.

### Direct vs descendant

`group_direct_objects` covers immediate members only; `group_descendant_objects`
walks the tree. Customer → Site → meters requires descendant scoping if the grant
is made at the customer level. Expose it as `IncludeDescendants` and make the
default explicit rather than implied.

### Interaction with MG-01

Uses the namespaced object-type helper from MG-01. Do not reconstruct
`"entity:device"` here — that duplication is the exact cause of the MG-01 bug.

## Acceptance criteria

1. Granting a subject `read` over a group produces **one** block and **one**
   direct policy, regardless of member count.
2. `authzCheck(subject, "read", entity, <member device>)` returns allowed.
3. A device **not** in the group is denied.
4. Adding a device to the group grants access with no policy write.
5. Removing it revokes access.
6. `IncludeDescendants: true` reaches devices in child groups;
   `false` does not.
7. `RevokeGroupAccess` removes the block and policy; access is denied afterwards
   for all members.
8. A grant over a 500-member group is a constant number of writes — asserted, not
   assumed.
9. Existing object- and tenant-scoped grants are unchanged.

## Test plan

- Unit: scope-mode selection across every input; `directPolicyMatches` including
  the `GroupID` comparison.
- Integration (Atom in Docker): the full customer scenario from
  [architecture.md §5.2](../architecture.md#35-groups-and-sharing) — two customers,
  three meters, one gateway; assert each customer sees only their own.
- Descendant scoping over a 3-level tree.
- Write-count assertion for criterion 8.
- Regression: existing connection grants (`grpc_compat.go:112-129`) still work.

## Risks

- **Revocation must match on `group_id`.** If `directPolicyMatches` is not
  extended, `RevokeGroupAccess` silently no-ops and access persists. This is the
  single highest-risk line in the PRD and needs a dedicated test.
- **Multi-group membership changes what revocation means.** Following
  [spec A1](../architecture.md#8-decision-record) and ATOM-04, a device can reach grants
  through several groups. `RevokeGroupAccess` on one group is therefore *not*
  "this subject can no longer see this device" — another group may still grant
  it. Acceptance criterion 7 must be read as "denied **for members reachable only
  through this group**", and any UI phrasing revocation as absolute will be
  wrong. Use [ATOM-03](./ATOM-03-reverse-policy-lookup.md) to answer "who can
  still see this?"
