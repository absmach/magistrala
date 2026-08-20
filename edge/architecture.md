# Magistrala Edge Model

**Status:** draft for review. **This is the single source of truth.**
**Companion:** [prd/](./prd/) — work breakdown · [plan.md](./plan.md) — superseded.

### How to read this

| Part | What it is |
|---|---|
| **1–2** | The problem, and the model — **stated in Magistrala's own terms, with no reference to Atom.** If this part does not stand on its own, the model is wrong. |
| **3** | The model in detail — identity, relations, attribution, access, views. |
| **4** | How it maps onto Atom, and what Atom must change. |
| **5** | Where Bootstrap and the edge agent stand against it. |
| **6–8** | Summary, open questions, and the **decision record** — every ruling, the options weighed, and the reversals. |

Parts 1–5 are normative. Part 8 is rationale: why the model is this and not
something else.

### Scope

The deliverable is the **Magistrala model**. Bootstrap and the edge agent are
*sources of edge cases*, not work items — the agent keeps publishing over MQTT
exactly as it does today, to a topic with one extra segment.

---

## 0. Phasing — read this first

The model below is complete. **What is being built now is narrower.**

| | Phase 1 — now | Phase 2 — deferred |
|---|---|---|
| **Entity model** | Device, `is_gateway`, `gateways[]`, groups, sharing of device *records* | Device types |
| **Identity** | `external_id` on Atom entities | — |
| **Messaging** | *unchanged* — topics stay exactly as they are today | `d` topic segment, `device_id` on messages, per-device storage and filters |
| **Access** | Grants on device *entities* | Per-device filtering of *message data* |
| **UI** | Device/gateway management, gateway view (declared only), sharing | Observed devices, fault correlation, type editor |

**Topics are application-level in phase 1.** Magistrala adds no device segment,
imposes no serial format, and reserves nothing. What an application puts in a
subtopic is its own business.

Two consequences to hold on to:

- **Per-device data sharing does not work in phase 1.** A customer can be granted
  a device *record*, but message data cannot be filtered by device — that needs
  `device_id` on messages, which is phase 2. The driving watermeter requirement is
  therefore only partly met until then. This is an accepted trade, not an
  oversight.
- **Local bus addressing is out again.** Without config delivery there is nothing
  to render, so `address` on the edge is deferred with it ([§8 A13](#8-decision-record),
  reversed by [A14](#8-decision-record)). The relation carries gateway IDs only.

**UI is part of phase 1.** The model deliberately leaves gateway-ness to the UI —
`is_gateway` and `gateways[]` are attributes and a query filter, nothing more — so
UI-01/02/03 in [prd/](./prd/) are what make "gateway" a thing an operator can use
rather than a shape in a database.

Sections marked **⏸ phase 2** describe the finished model, not current scope.

---

## 1. The problem

Magistrala has one concept for "a thing in the field": **Client**. It fuses two
concepts that are independent:

| | Connectivity identity | Data-producing identity |
|---|---|---|
| What it is | Authenticates, holds credentials, opens a session, publishes | What measurements belong to; what a user looks at; what access is granted on |
| Watermeter deployment | The gateway | Each individual meter |

For a directly-connected device these coincide, so the fusion was invisible. A
gateway fronting non-IP sensors breaks it: every message is attributed to the
gateway, and access is granted per channel — so *"customer A sees meter 1 and 3
but not meter 2"* has no representation at all.

### The driving deployment

> Watermeters speak BLE/Modbus/wM-Bus to a gateway running the Magistrala agent.
> Meters identify by serial number, have no IP and no credentials, and emit
> consumption and battery readings. The operator must see which meters sit behind
> which gateway and view data per meter. Customers must see only *their* meters —
> a subset of one gateway's, possibly spanning several. Different meter models
> with different capabilities hang off the same gateway.

**This is a modelling problem, not a naming problem.** Renaming Client to Device
without splitting the concept would buy nothing.

---

## 2. The model

Five concepts. Nothing else.

| Concept | What it is |
|---|---|
| **Device** | A thing that produces or consumes data. Identified by a **serial** — an arbitrary string, unique within a domain. Has a **device type**. Credentials optional. |
| **Gateway** | **Not a separate concept — a capability.** A Device with `is_gateway` set: it holds credentials, publishes its own telemetry like any device, and additionally serves as a **reachability path** for devices that cannot reach the platform themselves. |
| **Device Type** | A named, versioned declaration of what a class of devices measures and accepts as commands. |
| **Channel** | The messaging path. Unchanged from today. |
| **Group** | A named set of devices, for **sharing**. One purpose, no others. |

And one relation:

| Relation | Shape |
|---|---|
| **Device → Gateways** | Which gateways a device is **reachable through**. **0, 1 or many.** Stored as a `gateways []` attribute on the device — §3.3 |

```
Device  meter-7 { serial: "WM-2024-ABC.123", type: watermeter-v2,
                  gateways: ["gw-a", "gw-b"] }
```

### 2.1 A gateway is a path, not a container

Every framing that feels wrong about gateways shares one assumption: that a
gateway **contains** devices. It does not, and dropping that resolves the
awkwardness.

A gateway does not contain a meter. The meter exists whether or not the gateway
does — bolted to a pipe, keeping its own register, its serial stamped on the
case. Remove the gateway and the meter does not cease to exist; **it goes dark.**
That is a different relationship from membership.

What a gateway provides is **reachability**: it is *how* a device's data gets
here.

The nearest well-understood analogue is a Kubernetes Node. A Node has its own
identity, metrics and configuration, and Pods run on it — but a Node is not a
group of Pods and does not own them. Nobody struggles to classify a Node, because
"host" is understood as a role rather than a container. The same is true of a
router and its hosts, or a BLE central and its peripherals.

So a Gateway is **a Device with the `is_gateway` capability**:

- **Not a container** — devices are not inside it.
- **Not a group** — groups are for sharing; this is transport.
- **Not a separate type** — a device that has taken on a second job.

A capability rather than a type, because **the two compose**. A smart electricity
meter that also concentrates wM-Bus water meters produces its own readings *and*
relays others. Under exclusive types it must be one or the other, and either
answer is wrong.

Every comparable platform reaches the same conclusion: ThingsBoard uses an
`Is gateway` boolean on an ordinary device; AWS Greengrass makes core devices and
client devices both `AWS IoT thing`s; Azure makes both device identities. The only
exception is ChirpStack, where gateways are dumb radio infrastructure that produce
no data — which ours are not.

The property that makes the whole model work is that **Device is
credential-optional**. That single choice lets one model serve a BLE meter behind
a gateway and the same meter model connecting directly over NB-IoT, with no
remodelling.

**Sensor** is not a concept here. It is an informal word for a device with no
credentials that reaches the platform through a gateway.

### 2.2 What follows from that — normative

Six consequences. Each settles a question that would otherwise be decided ad hoc
by whoever implements it first.

**1. Devices are primary.** A device is created, identified and typed
independently of any gateway. It is never "created inside" one. This is why the
relation is a property of the device, not of the gateway.

**2. Deleting a gateway never deletes devices.** Under containment a cascade
would be expected. Under reachability, deleting a gateway means *this path is
gone* — the devices remain, now unreachable through it. Anything that cascades is
a bug.

**3. Gateway access and device access are orthogonal.** Managing a gateway —
reboot, reconfigure, read its metrics — is a different grant from reading a
meter's consumption. A field technician needs the first; a customer needs the
second. Neither implies the other.

**4. A gateway's own telemetry is ordinary device telemetry.** CPU, uptime, link
quality: it publishes as itself, with `publisher == device_id`. No special case
in ingest, storage or query.

**5. Aggregation attributes to the subject, not the computer.** If a gateway
computes a daily total for meter-7, that is meter-7's data and carries meter-7's
`device_id`. If it reports how many devices it is serving, that is the gateway's
own. The two-identity attribution (§2.5) already carries this; the rule is what
was missing.

**6. Downlink routes along the reachability edge.** A command for a non-IP device
goes to one of its gateways, addressed by the device's serial. The relation *is*
the routing table. (The command path itself is not designed here — §5.3.)

### 2.3 The relation is a relation — not a group

Cardinality follows the physical link:

| Link | Gateways | Example |
|---|---|---|
| Device connects for itself | **0** | NB-IoT meter |
| Wired, exclusive | **1** | Modbus meter on a bus |
| Broadcast, promiscuous | **many** | wM-Bus meter heard by every gateway in range |

> **An earlier draft modelled this as a group owned by each gateway. That was
> wrong.** A group is a user-facing organisational object — named, created
> deliberately, shared. A gateway's device list is a fact about reachability.
> Conflating them produced two objects per gateway, lifecycle coupling, and a
> namespace where granting on the wrong group silently hands over an entire
> gateway's fleet.
>
> A relation is a relation. Groups now mean exactly one thing: sharing.

### 2.4 Declared and observed

The relation above is **declared** — what commissioning says. There is a second,
free source of truth: **observed**, from message traffic. A gateway publishes
under its own identity, carrying the device's serial, so "which devices has this
gateway actually relayed for" is a query, not a record.

Both are needed, because **the difference between them is the operational
signal**:

| Declared | Observed | Meaning |
|---|---|---|
| ✓ | ✓ | **Healthy** |
| ✓ | ✗ | **Silent** — commissioned, never heard. For a wired link this is the fault condition |
| ✗ | ✓ | **Undeclared** — undocumented device, or a neighbour's broadcast |

Declared alone cannot see a device nobody registered. Observed alone cannot see a
device that has gone quiet — and for a Modbus meter, silence *is* the alarm.

### 2.5 ⏸ phase 2 — Message attribution

> **Deferred.** Phase 1 leaves topics and messages untouched. The design below is
> what phase 2 builds; nothing in phase 1 depends on it.

Every message carries two identities answering different questions:

| Field | Question | Set by |
|---|---|---|
| `publisher` | **Who sent it** — the authenticated gateway | The broker, from the connection |
| `device_id` | **Whose data it is** — the meter's serial | Taken verbatim from the topic |

```
m/<domain>/c/<channel>[/d/<serial>][/<subtopic>]
```

`device_id` is the serial, carried as an **opaque string**. It is not resolved to
a platform identifier on the publish path.

That gives **late binding**: data may arrive for a serial no device record
exists for, and is stored and attributed anyway. Registering the device later
makes its *history* viewable and shareable retroactively. Registration is not a
precondition for attribution.

### 2.6 Access

| Question | Answer |
|---|---|
| May this gateway publish for that meter? | Is it authorised on the channel? Nothing finer |
| Which meters may this customer read? | The devices in the groups granted to them |
| Which devices did this gateway relay for? | Declared relation ∪ observed traffic |

Publish authority is **channel-scoped**, deliberately. Per-device publish checks
would put an entity lookup on every message; the channel is the trust boundary
instead. The cost is that a compromised gateway can fabricate data for any device
on channels it holds — mitigated by segregating channels per site or customer.

---


## 3. The model in detail

### 3.1 Device

| Field | Notes |
|---|---|
| `id` | Platform-assigned, stable |
| `serial` | **The identifier that matters.** Arbitrary string, unique per domain, appears verbatim in publish topics. Must not contain `/` — the topic separator |
| `name`, `tags`, `metadata` | As today |
| `device_type_id`, `device_type_version_id` | What it measures and accepts |
| `gateways []` | Declared relation, 0..N |
| `credentials` | **Optional** |
| `status` | Enabled / disabled |
| `provisioning_state` | `pending` / `provisioned` / `rejected`. Lifecycle only — nothing on the publish path reads it |

### 3.2 Gateway

A Device with `is_gateway` set. Everything above, plus: credentials are
mandatory, it connects, and other devices may name it as a reachability path.

**The capability composes with everything else.** A device may be a gateway *and*
report its own measurements — a concentrator-meter is one device, one type, both
jobs. Nothing in the model treats gateways as a disjoint population.

It is *not* a container (§2.1). Its "fleet" is the reverse of the relation —
devices that name it — and is derived on demand, never stored on the gateway.

The distinct kind earns its place in three ways: gateway *types* namespace
separately from device types (so a watermeter type cannot be bound to a gateway);
"every gateway in this domain" becomes expressible as a single grant; and
listing gateways is a native filter rather than an attribute scan.

A device may be promoted to a gateway. The kind changes; identity, history and
grants do not.

### 3.3 Linking a device to gateways

The link is **an attribute on the device**. Nothing else — no join table, no
group, no Atom primitive beyond what already exists.

```jsonc
// Atom entity — a wM-Bus meter reachable through two gateways
{
  "id":          "018f...",              // Atom UUID
  "kind":        "device",
  "external_id": "WM-2024-ABC.123",      // the serial; what appears in topics
  "attributes": {
    "is_gateway": false,
    "gateways":   ["018a-gw-a", "018b-gw-b"]     // phase 1: IDs only
  }
}

// The gateway itself — same kind, one flag different
{
  "id":          "018a-gw-a",
  "kind":        "device",
  "external_id": "GW-SITE-5",
  "attributes":  { "is_gateway": true }
}
```

Empty or absent `gateways` means a standalone device. One entry means an
exclusive link. Several means broadcast.

#### ⏸ phase 2 — `address` on the edge

*Deferred. Phase 1 stores gateway IDs only; the block below is the finished
design, retained because the reasoning holds whenever config delivery is built.*

#### `address` — the local bus address

Present only for **bus-addressed protocols**. A Modbus unit ID is an address, not
a serial, and the gateway cannot derive one from the other — so something must
hold `unit 7 → WM-2024-ABC.123`. wM-Bus, BLE and anything self-identifying omit
it entirely.

It sits **on the edge**, not on the device or the gateway, because that is whose
property it is: the same meter on two buses can have two different unit IDs.

**Magistrala never parses it.** It is stored, rendered into gateway config, and
compared for equality — nothing more. No Modbus, BLE, M-Bus or LoRa knowledge
enters the platform. That constraint is what makes holding this data safe.

> **This reverses A3**, which put local addressing entirely agent-side. Bootstrap
> config is generated by the cloud, so if the cloud does not hold the map,
> bootstrap cannot deliver it — and a replaced gateway must then be reconfigured
> by hand. See [§8 A13](#8-decision-record).

> **No format constraint.** The serial is an arbitrary string — uppercase, `/`,
> `.`, spaces, unicode all pass through. Atom constrains nothing (A6) and
> Magistrala constrains nothing either.
>
> If an application puts a serial in a topic, encoding it is that application's
> problem. An earlier draft rejected `/` at device creation to protect the topic
> grammar; with topics application-level (A14) there is no grammar to protect.

#### Both directions

| Question | How |
|---|---|
| Which gateways does this device use? | Read the device — `attributes.gateways` |
| Which devices are declared on this gateway? | `entities(attributesContains: { gateways: [{ "id": "018a-gw-a" }] })` — JSONB containment, **needs A1** |
| Is this bus address already taken on this gateway? | `attributesContains: { gateways: [{ "id": "018a-gw-a", "address": { "modbus_unit": 7 } }] }` |
| Which devices has this gateway *actually* relayed for? | `DISTINCT device_id WHERE publisher = G` over stored messages |
| Which gateways have relayed for this device? | `DISTINCT publisher WHERE device_id = serial` |
| List all gateways | `entities(attributesContains: { is_gateway: true })` |

The first two are the **declared** relation, from Atom. The next two are
**observed**, from the message store. §3.7 merges them.

#### Query semantics — verified, not assumed

Atom implements `attributesContains` as `attributes @> $9::jsonb`
(`src/authz/repo.rs:176`). Tested against Postgres:

| Property | Result |
|---|---|
| Match an edge by `id` inside an object array | ✅ |
| No false match on a different `id` | ✅ |
| **Containment is per-element** — `{id: gw-a, address: {unit: 9}}` does *not* match a device where gw-a is unit 7 and gw-b is unit 9 | ✅ |
| GIN-indexed — `idx_entities_attrs` (`001_initial.sql:119`) exists already; 60k-row test used a Bitmap Heap Scan | ✅ |

Per-element containment is what makes **address-conflict detection sound**: two
devices claiming byte-identical address blobs on one gateway is detectable by
containment alone, with no knowledge of what `modbus_unit` means. The
"never parse" rule survives.

Two things are **not** expressible, because `{}` is contained in everything:
"devices with *any* address" and "devices missing an address". Both fall to
application logic; neither is needed today.

#### Writing it

```
PUT /{domain}/devices/{id}/gateways   { "gateways": ["gw-a", "gw-b"] }
```

Replace-the-list rather than attach/detach: the list is short, and replacement
avoids the read-modify-write races that two-sided attach APIs invite.

For the operator ergonomics of "add these 40 meters to this gateway", the API may
offer a gateway-side bulk convenience that fans out to N device writes — but the
**storage stays on the device**, because that is what makes 0/1/N natural and
gives the reverse lookup a single indexed query.

#### Why the declared relation exists at all

Beyond the UI, it is **the authoritative source for generating gateway config.**

For self-identifying protocols there is no mapping to generate — the serial flows
through untouched:

```
wM-Bus telegram carries serial → gateway puts it in the topic
  → m/dom/c/chan/d/WM-2024-ABC.123 → device_id → Atom external_id
```

Same string end to end, which is the payoff of `serial == device_id ==
external_id`. Late binding means the device record may be created before or after
the data arrives.

**Bus-addressed protocols are the exception.** A Modbus unit ID is an address,
not a serial, and the gateway cannot derive one from the other. Something must
hold `unit 7 → WM-2024-ABC.123`, and per A13 that is the `address` blob on the
edge — held by the cloud, rendered into gateway config, never parsed.

Generating that config needs the serials *and their addresses* for this gateway,
which is exactly the declared relation:

```
entities(attributesContains: { gateways: ["gw-a"] })
  → serials for gateway gw-a
  → operator supplies bus addresses against them
```

Without it there is nothing to generate from. This is a stronger justification
than the gateway view, and it holds even if the view is never built.

#### What this costs

**Read-modify-write on the whole array.** Atom's `update_entity`
(`src/identity/repo.rs:335-341`) is `COALESCE($n, col)` — last write wins, with no
version check. Two operators editing one device's gateways concurrently silently
lose an edit. The replace-list API makes the full-list semantics explicit, but
safety needs **optimistic concurrency**, which Atom does not currently offer
(§7).

**Config generation is a two-step, and the second step is client-side.** The
query returns whole devices with *all* their edges; picking the entry for gateway
G is done by the caller. Trivial at fleet scale, but it does transfer edges that
are not wanted.

**No referential integrity.** Deleting a gateway leaves its ID in every device
that named it. Two mitigations, both cheap:

- **Resolve-and-drop on read** — the gateway list is resolved to entities when
  displayed; misses are omitted. Correct without any write.
- **Sweep on gateway deletion** — optional tidiness, not correctness.

This is the one thing a join table or a group would have given for free. It is
worth less than the complexity either would have cost — see A10.

### 3.4 Device Type

A named, versioned capability declaration:

```
DeviceType "watermeter-v2" v1
  measurements: volume (m³, read), battery (%, read)
  commands:     set_interval(seconds)
```

Versioned, with lifecycle — a device binds to a *specific version*, and adding a
new version never invalidates deployed devices. Deprecating a version blocks new
bindings and leaves existing ones working.

The type drives UI rendering and validates the device's own attributes.

> **It does not validate telemetry.** Message payloads never pass through the
> entity store. A device type constrains device *metadata*, not readings, unless
> ingest-time validation is built separately. This is the most likely
> misunderstanding of the feature and belongs in the API documentation.

### 3.5 Groups and sharing

A group is a named set of devices, used for one thing: granting access.

Groups nest (Customer → Site → meters) and a device may belong to **several**
groups, so overlapping sets work — a meter can be both "Customer A's" and
"Building 5's" where neither contains the other.

A customer is a user holding a **read grant over a group**. One grant per
customer, not one per device: adding a meter to the group grants access, removing
it revokes. **Membership is the sharing operation.**

> With multi-group membership, removing a device from one group does not
> necessarily revoke access — another group may still grant it. Anything
> presenting removal as revocation is wrong; "who can still see this device"
> needs a reverse policy lookup.

### 3.6 ⏸ phase 2 — Fault correlation

Reachability being a *function* means a device's silence has more than one cause,
and the platform holds enough to tell them apart:

| Gateway | Its declared devices | Diagnosis |
|---|---|---|
| Publishing | one silent | That device is faulty |
| Publishing | all silent | Bus or wiring fault |
| Silent | all silent | **The gateway is down** — one incident, not N |

Without this, a gateway losing power raises an alert per device.

Two notes. It is **derived**, not stored — computed from facts the model already
has. And it needs the **declared** relation to work at all: if the gateway is
down nothing is observed, so there is nothing to correlate. That is the strongest
argument for keeping declared as a first-class fact rather than deriving
everything from traffic.

Specifying it needs a notion of gateway liveness (last-seen) that the model does
not yet carry. Deliberately deferred; recorded here so it is designed rather than
discovered.

### 3.7 The gateway view

The concrete test of the model. Click a gateway, get three panels:

**Gateway information** — one entity read. Name, serial, status, type,
credentials, certificate metadata.

**Bootstrap configuration** — the enrollment for this gateway, found by an
explicit reference from the config to the gateway.

**Devices** — in phase 1, the **declared** list only:
`entities(attributesContains: { gateways: ["gw-a"] })`.

⏸ *Phase 2* adds the observed half from message traffic and merges the two into
the healthy / silent / undeclared status — which is what makes the view
diagnostic rather than just a list.

Both device lists are authorisation-filtered: a customer viewing a gateway sees
only their own devices on it, never the full fleet.

### 3.8 Scenarios

Each of these is a test of the model, drawn from real deployments:

| Scenario | Model answer |
|---|---|
| Meter heard by three gateways | Three publishers, one serial. Declared list holds all three, or none |
| Meter installed before registration | Late binding — data stored, visible once registered |
| Meter moved to another gateway | Update its `gateways`. History intact under both publishers |
| Gateway replaced | New publisher, same serials. Declared list re-pointed |
| Wired meter goes silent | Declared ✓, observed ✗ → **fault**, visible in the gateway view |
| Two domains, same serial | Distinct devices — serial is unique *per domain* |
| Serial with `/`, `.`, uppercase | `.` and uppercase fine; `/` rejected at creation |
| Gateway publishes an unknown serial | Stored as orphan data; visible to operators, grantable to nobody |
| Meter replaced, same serial | Same device, continuous history. Correct for a swap; an operational choice, not a platform one |

---


## 4. Mapping to Atom

Atom is the entity store and authorization engine. It is deliberately
domain-agnostic: it knows about principals, objects, groups and policies, and
nothing about meters.

### 4.1 Atom's vocabulary

| Primitive | Role |
|---|---|
| `Tenant` | Isolation root |
| `Entity` (`kind ∈ human, device, service, workload, application`) | Principals — the only things that hold credentials and act as authz subjects |
| `Resource` (`kind` = open string) | Non-actor objects |
| `Object Group` / `Principal Group` | Two group namespaces — objects vs subjects |
| `Profile` + `ProfileVersion` | Versioned JSON Schema + UI Schema bound to an object |
| `PermissionBlock` + `DirectPolicy` | Scope + actions + effect, bound to a subject |

### 4.2 The mapping

| Magistrala | Atom | Notes |
|---|---|---|
| Domain | `Tenant` | Unchanged |
| **Device** | `Entity`, `kind = device` | |
| **Gateway** | `Entity`, `kind = device`, attribute `is_gateway: true` | No new Atom kind — §4.4 |
| **Serial** | `entities.external_id` | Arbitrary string, unique per tenant — §4.4 |
| **Device Type** | `Profile` (`object_kind=entity`, `kind=device`) + `ProfileVersion` | **Already exists** — §4.3 |
| **Gateway Type** | `Profile` (`kind=device`) | Same namespace as device types — a device that both measures and relays needs one type, not two |
| **Device → Gateways** | `gateways: [...]` entity attribute | Opaque to Atom; reverse lookup needs §4.4 A1 |
| **Group** | `Object Group` | Sharing only |
| Channel | `Resource`, `kind = channel` | Unchanged |
| User | `Entity`, `kind = human` | Unchanged |

Atom stores the relation without interpreting it — an attribute is an opaque
JSON blob to Atom, which is exactly right. **Atom is never asked what a gateway
is**, and the relation is never consulted for authorization.

### 4.3 What Atom already provides

The Go client uses a small fraction of Atom's surface. Most of what this model
needs exists server-side and is simply not wired up:

| Capability | Atom | `pkg/atom` |
|---|---|---|
| Device types, versioned, with **JSON Schema enforcement on write** | `profiles` + `profile_versions`; `src/identity/repo.rs:641` | No profile API at all |
| Group membership | `addGroupMember`, `removeGroupMember`, `groupMembers`, `entityGroups` (`graphql/groups.rs:645,699,90,104`) | **None** — only the migration tool, via raw SQL |
| Object vs principal groups | `createObjectGroup`, `createPrincipalGroup` (`:346,355`) | Generic `createGroup` only |
| Group hierarchy | `setGroupParent`, `childGroups`, `includeDescendants` (`:429,121`) | `groupCreateInput` **drops** `parentId` (`api.go:751-760`) |
| Grant scope modes | 10, incl. `object_type`, `group_direct_objects`, `group_descendant_objects` (`001_initial.sql:607`) | Writes only `object`, `tenant`, `platform` (`policy_service.go:196-205`) |
| Entity list filters | `profileId`, `parentGroupId`, `includeDescendants`, `status` (`entities.rs:74-87`) | Sends none |
| Authorization-filtered listing | `entities()` routes through `authorized_object_ids` (`entities.rs:123`) | Unused |
| Domain events (~40 kinds, transactional outbox) | `migrations/004_event_outbox.sql`, `src/events/publisher.rs` | Not consumed; not even enabled |

### 4.4 Required Atom changes

Six, in dependency order. The test applied to each: **would a non-IoT product
want this?** A change adding a generic capability passes; one that teaches Atom
what a gateway is fails.

| # | Change | Why generic | Size |
|---|---|---|---|
| **A1** | `attributesContains` on the `entities` query — **backs `gateways []` reverse lookup and the gateway list** | Already implemented in the repo layer and **already exposed on `resources()`** (`graphql/resources.rs:51`); hardcoded `None` for entities (`entities.rs:134`). A symmetry fix | Plumbing |
| **A2** ⏸ | Scoping filters on `authorizedObjectIds` — *phase 2* | The query struct and repository support `attributesContains`, `profileId`, `parentGroupId`, `includeDescendants`; the resolver pins them to `None` (`graphql/authz.rs:47-61`) | Plumbing |
| **A3** | `directPolicies(objectId:)` | "Who can access this object" — the inverse of a query Atom already answers. Any sharing UI needs both directions | Small |
| **A4** | Many-to-many object group membership | `PRIMARY KEY (entity_id)` (`001_initial.sql:525-532`) allows one group per entity. M:N is the ordinary case for a grouping primitive — Atom's own API already reads that way (`entityGroups` returns a *list*) | Moderate — touches the authz evaluation path |
| **A6** | `external_id` on entities, unique per tenant | Every system mirroring things it did not create carries foreign identifiers. `alias` cannot serve — slug-constrained (`001_initial.sql:105-113`) | Small migration + index |

**A1 is required, not optional.** With the device→gateway relation stored as an
attribute, "which devices are declared on this gateway" is exactly an
`attributesContains` query. An earlier draft demoted A1 after the group-based
design removed its justification; the relation restores it.

> **An earlier revision added a sixth change — a `gateway` entity kind.** It was
> withdrawn: gateway is a capability, not a type (§2.1), so gateways stay
> `entity_kind: device`. That also removes a trap it carried — the existing
> `{entity_kind: device, publish, resource:channel}` guardrail
> (`pkg/atom/bootstrap.go:72-87`) keeps applying, where a new kind would have
> silently stripped every gateway's right to publish.

### 4.5 Not asked of Atom

| Rejected | Why |
|---|---|
| `sensor` as an entity kind | Profile uniqueness is `(tenant, object_kind, kind, key)`, so one hardware model deployed both behind a gateway and directly would need two profiles that then drift. It also encodes topology into identity |
| Device↔gateway as an Atom relation | Topology is Magistrala's domain semantics. Atom stores the attribute and never interprets it |
| Capability schemas as an Atom concept | Already generic — a JSON Schema on a profile version. Atom needs no IoT awareness |
| Message or telemetry concepts | Atom is identity and authorization |

---

## 5. Bootstrap and the agent — where we stand

Both are **scenario sources**, not deliverables. The question here is only
whether the model fits them, and what the minimum is to make the gateway view
work.

### 5.1 The agent

Unchanged. It connects over MQTT and publishes, exactly as today, to a topic with
one extra segment:

```
m/<domain>/c/<channel>/d/<serial>
```

Nothing in the model requires the agent to hold a roster, resolve identifiers, or
maintain state the platform depends on. Late binding (§2.4) and channel-scoped
publish authority (§2.5) were both chosen partly to keep it that way.

Local addressing is **delivered to** the gateway rather than configured on it
(A13): the cloud holds `unit 7 → serial` as an opaque blob on the edge and renders
it into bootstrap config. The gateway still owns the bus and still resolves serial
→ local address at runtime; what changed is that the map now survives a gateway
replacement instead of being rebuilt by hand.

No protocol-specific knowledge enters Magistrala — the address is stored, rendered
and compared, never parsed.

### 5.2 Bootstrap

[PR #3555](https://github.com/absmach/magistrala/pull/3555) reintroduces the
Bootstrap service (74 files, ~16k additions) with enrollments, profile templates,
binding slots and an authenticated device-facing protocol. It predates this model
and must be rebased — it branches from `c09020a29`, where the Atom package was
`internal/atom`, and reports `mergeable: false`.

Bootstrap is **in scope** as of A13: it is the mechanism that delivers the
serial → bus-address map to the agent, generated from the declared relation.
MG-12 returns from deferred.

**One gap blocks the gateway view.** `configProjection` (`bootstrap/atom.go`)
projects an enrollment into Atom carrying only `profile_id` — there is **no
reference to the gateway entity**. A gateway therefore cannot find its own
bootstrap config, by reference or by value.

One attribute fixes it:

```go
res.Attributes["gateway_id"] = <gateway entity id>
```

after which `resources(kind: "bootstrap-config", attributesContains: {gateway_id: …})`
works with **no Atom change** — `attributesContains` is already exposed for
resources.

That is the only thing the model asks of Bootstrap. Everything else — enrollment
crypto, templates, binding slots — is untouched and out of scope.

### 5.3 What is deliberately not designed

- **Gateway-announced discovery.** The scenario is real and the model accommodates
  it (an announce would create devices and set their `gateways`), but the flow is
  not specified here.
- **Downlink commands.** The topic grammar accommodates them and device types
  declare the command set, but routing, addressing and ack semantics need their
  own pass.

---


## 6. Summary of required changes

### Phase 1

| Layer | Change | PRD |
|---|---|---|
| **Atom** | A1 `attributesContains` on entities — backs the gateway→devices lookup and the gateway list | ATOM-01 |
| **Atom** | A3 `directPolicies(objectId:)` reverse lookup | ATOM-03 |
| **Atom** | A4 many-to-many object group membership | ATOM-04 |
| **Atom** | A6 `external_id` on entities, unique per tenant | ATOM-06 |
| **`pkg/atom`** | Fix `objectType` → `entity:device`; policy pagination; entity applicability | MG-01 |
| **`pkg/atom`** | Group membership, hierarchy, object/principal groups | MG-03 |
| **`pkg/atom`** | Group-scoped permission blocks | MG-04 |
| **Readers** | Authorize the existing `publishers` filter — **live security defect** | MG-08 (A) |
| **API/SDK** | Device, Gateway (`is_gateway`), the `gateways[]` relation; retire Client | MG-09 |
| **CLI/PAT/perms/OpenAPI** | Surface for the above | MG-11 |
| **UI** | Device and gateway management; setting `gateways[]` | UI-01 |
| **UI** | Gateway view — information and declared devices | UI-02 |
| **UI** | Sharing — groups, grants, "who can see this device" | UI-03 |

### ⏸ Phase 2

| Layer | Change | PRD |
|---|---|---|
| **Atom** | A2 scoping filters on `authorizedObjectIds` | ATOM-02 |
| **`pkg/atom`** | Device Type API | MG-02 |
| **`pkg/atom`** | Consume Atom domain events for cache invalidation | MG-14 |
| **Messaging** | `d` topic segment; `device_id` on `Message`, carried verbatim | MG-05 |
| **Storage** | `device_id` column + index; `DeviceIDs` reader filter | MG-06 |
| **Readers** | Intersect the device filter with the authorized set | MG-08 (B) |
| **Readers** | Gateway view — declared ∪ observed, merged with status | MG-15 |
| **API/SDK/CLI** | Device Type surface; `address` on the relation | MG-10 |
| **Bootstrap** | `gateway_id` on the projection; render the serial → address map | MG-12 |

### The security fix is not optional

Reader authorization is **not a boundary today**.
`readers/api/http/transport.go:251-266` authorizes `subscribe` on the *channel*,
then applies the caller-supplied `publisher` / `publishers` filters without
validating them (`readers/messages.go:49-50`). Any user who can read a channel
can read every publisher on it by changing a query parameter.

The customer model in §3.5 depends on fixing this. It is also a live issue today,
independent of everything else here.

Three further defects sit directly underneath:

| Defect | Location |
|---|---|
| `objectType` sent as `"device"`; Atom requires `"entity:device"` — the filter can never match | `pkg/atom/policy_service.go:139` vs `:225` |
| `DeletePolicyFilter` pages at 100; policies past #100 silently never deleted | `pkg/atom/policy_service.go:78-105` |
| No applicability registered for `objectKind: entity`, yet `read` on an entity is already checked | `pkg/atom/bootstrap.go:29-70` |

---

## 7. Open questions

Split by phase, because most of what follows does **not** block starting.

### Blocks phase 1 — decide first

Two one-way doors on `external_id` (ATOM-06). Both are cheap now and unrecoverable
later, because a wrong choice silently merges two devices into one:

- **Case sensitivity.** Are `ABC123` and `abc123` one device or two? Choosing
  case-insensitive merges devices that no migration can separate. **Recommend
  case-sensitive** — it can be tightened later; the reverse cannot.
- **Whitespace normalisation.** `"ABC123 "` and `"ABC123"` are two devices unless
  trimmed. Trimming is probably right, but it must be a decision rather than an
  accident of whichever client writes first.

### Shapes phase 1 — decide during

- **Serial mutability.** Cheap in phase 1 — nothing references the serial yet.
  **Expensive in phase 2**, when it is denormalised onto every message row and
  changing it orphans that device's history. Decide now while it costs nothing:
  probably immutable in Magistrala even though Atom permits the update.
- **Atom has no optimistic concurrency.** `update_entity`
  (`src/identity/repo.rs:335-341`) is `COALESCE($n, col)` — last write wins. Setting
  a device's `gateways` is read-modify-write, so two operators commissioning
  concurrently silently lose an edit. Options: ask Atom for an `If-Match` on
  `updated_at` (generic, small); serialise Magistrala-side; or accept it. Low
  likelihood, *silent* failure — which is why it is recorded rather than left to
  be discovered.
- **Stale gateway references.** Deleting a gateway leaves its ID in every device
  that named it. Resolve-and-drop on read is enough for correctness; a sweep on
  deletion keeps the data honest. Decide which, and whether operators see stale
  entries.
- **B2 — reader cache TTL**, if MG-08 part A caches its authorization results.
  The TTL is the revocation SLA and therefore customer-visible.

### ⏸ Phase 2 — not blocking

| # | Question | Waits on |
|---|---|---|
| **A4** | Global vs tenant-scoped device types | MG-02 |
| **C5** | Reserving `d` as a first subtopic segment — **moot in phase 1**, which adds no topic grammar | MG-05 |
| **C6** | Does the `Connection` proto need a device field | MG-05/06 |
| **E1** | Telemetry validation against the device type — documentation at minimum | MG-02/10 |
| **E2** | Is the capability model expressive enough? Model two or three more real device types before freezing | MG-02 |
| **E4** | Downlink command path | — |
| — | **Address conflicts** — two devices on one gateway at the same bus address. Detectable by containment without parsing (§3.3); reject at write or warn in the view | MG-12 |

---

## 8. Decision record

Every question this design was blocked on, the options weighed, the ruling and
its consequences. The sections above are **normative** — they state what the
model *is*. This section is the **rationale** — why it is that and not
something else, including the reversals.

**Legend** 🔴 blocked code · 🟡 shaped design · ⚪ deferrable

### Status at a glance

| # | Question | Urgency | Blocks | Decision |
|---|---|---|---|---|
| A1 | Overlapping device sets? | 🔴 | MG-03, MG-04 | ✅ **M:N now** — ATOM-04 |
| A2 | `provisioning_state` placement | 🟡 | MG-09, MG-13 | ✅ **Device attribute** |
| A3 | Model local transport addressing? | ⚠️ | MG-12 | ⚠️ **Reversed by A13** — the cloud holds it, opaquely |
| A4 | Global device types? | 🟡 | MG-02, MG-10 | open |
| A5 | Lifecycle events in 1.0? | 🟡 | MG-07, MG-08 | mostly answered — Atom already emits them; consume via MG-14 |
| A6 | `gateway` as a native Atom kind? | 🔴 | MG-09, MG-10 | ⚠️ Superseded by A12 — **no**, it is a capability |
| A12 | Gateway: type or capability? | 🔴 | MG-09 | ✅ **Capability** `is_gateway`; ATOM-05 withdrawn |
| A13 | Where does the local bus address live? | ⏸ | MG-12 | ✅ **On the edge, opaque** — deferred to phase 2 by A14 |
| A14 | What is in scope now? | 🔴 | all | ✅ **Phase 1 = entity model**; messaging deferred |
| A7 | Gateway → device publish authority | 🔴 | MG-05, MG-09 | ✅ **Channel is the boundary** |
| A8 | Arbitrary-string serial resolution | 🔴 | MG-06, MG-08, MG-09 | ✅ **Atom `external_id`** (ATOM-06) |
| A9 | Declared device↔gateway association | 🔴 | MG-15 | ⚠️ Superseded by A10 |
| A10 | Group or relation? | 🔴 | MG-09, MG-15 | ✅ **Relation** — `gateways []` on the device |
| A11 | What kind of thing is a Gateway? | 🟡 | model-wide | ✅ **Device with a proxy role** — reachability, not containment |
| A15 | Can a gateway declare gateways of its own? | 🔴 | MG-09, UI-01 | ✅ **No — gateways do not chain**; enforced at write time |
| B1 | Pending device data | ⚪ | — | ⚠️ **Superseded by A7+A8** — accept, late-bind |
| B2 | Revocation window | 🟡 | MG-08 | open |
| B3 | Admin bypass mechanism | 🟡 | MG-08 | ✅ **Explicit capability check** |
| B4 | Attachment enforced MG-side | ⚪ | — | ⚠️ **Moot** — no attachment exists (A7) |
| B5 | Announce rate limits | 🟡 | MG-13 | open |
| C1 | Deployed unnamespaced objectType? | 🔴 | MG-01 | ✅ **No back-compat** |
| C2 | PAT enum strategy | ⚪ | MG-11 | ✅ **Remove outright** (premise was wrong) |
| C3 | Serial ↔ ExternalID | ⚪ | MG-09, MG-12 | ✅ **Independent, no enforced link** |
| C4 | Serial uniqueness | ⚪ | MG-09 | ✅ **Per tenant**; mechanism settled by A8 (ATOM-06) |
| C5 | Reserve `d` segment | 🟡 | MG-05 | open |
| C6 | Device field on Connection | 🟡 | MG-09 | open |
| C7 | Restore CLI binary | ⚪ | MG-11 | open |
| C8 | Channel roles | ⚪ | — | open |
| D1 | Fleet delivery mechanism | 🟡 | MG-12 | open |
| D2 | Fleet snapshot vs live | 🟡 | MG-12 | open |
| D3 | Announce transport | 🟡 | MG-13 | open |
| E1 | Telemetry validation | ⚪ | MG-10 | open |
| E2 | Capability model coverage | ⚪ | MG-10 | open |
| E3 | Group re-add semantics | ⚪ | MG-03 | ✅ **Dissolved by A1** |
| E4 | Command path | ⚪ | — | open |
| E5 | Agent-side device roster | 🟡 | MG-12, MG-13 end-to-end | open |

**All 🔴 are resolved.** Code can start on ATOM-04, MG-01 and the P2 attribution
track.

**MG-09 is unblocked.** With C3 and C4 settled, nothing gates freezing the device
API except the serial-uniqueness *mechanism*, which is a research task.

#### Highest-value remaining

| # | Why it matters now |
|---|---|
| **B2** | The cache TTL *is* the revocation SLA — a customer-visible number |
| **A4** | Global vs tenant-scoped device types shapes the MG-02/MG-10 API; hard to widen after 1.0 |
| **B5** | Announce rate limits are the only bound on entity creation from one gateway credential |
| **D1** | Fleet delivery mechanism — smaller now that transport addressing is out (A3) |

#### Research tasks, not decisions

| # | Task |
|---|---|
| C4 | ~~Serial formats vs slug pattern~~ — moot; `external_id` is unconstrained (A8) |
| D1 | Measure a realistic fleet's rendered config size (serial + type only, post-A3) |
| E2 | Model two or three more real device types against the capability shape |

---

### Section A — Model shape

These change the model itself. Everything else assumes answers here.

---

#### A1 🔴 Can a device be shared with two independent parties at once?

**Blocks:** MG-03, MG-04 · **Ref:** [architecture.md §5.3](./architecture.md#53-the-one-real-atom-constraint-single-group-membership)

Atom's `object_group_entities` has `PRIMARY KEY (entity_id)`
(`migrations/001_initial.sql:525-532`) — **an entity belongs to at most one object
group.** The whole customer-sharing design rests on group-scoped grants, so this
is the load-bearing constraint.

Concretely: a rented flat's watermeter visible to *both* the landlord and the
tenant, as independent parties, neither being a domain admin.

**Correction to the original framing.** This question conflated two things.
*Multiple users accessing one device* already works with no Atom change: a
`DirectPolicy` binds one subject to one `PermissionBlock`, but many policies can
point at the same block, and a subject may be a **principal group** of users. N
users — or whole teams — reading the same meters is N policies against one block.

The single-membership constraint bites only when a device must sit in **two
different object groups**, i.e. overlapping device *sets* that no hierarchy
contains:

```
"Customer A meters"   → granted to Customer A
"Building 5 meters"   → granted to a maintenance contractor
Meter 7 ∈ both, and neither set contains the other
```

**DECIDED: make Atom membership M:N now.**

```sql
ALTER TABLE object_group_entities
  DROP CONSTRAINT object_group_entities_pkey,
  ADD PRIMARY KEY (group_id, entity_id);
```

Rationale: data-preserving and near-trivial while tables are small; a risky
migration on a core table once deployments carry real data. It also removes a
constraint that would otherwise shape every downstream modelling decision.

**Consequences**

- New PRD **[ATOM-04](./prd/ATOM-04-many-to-many-group-membership.md)**.
- MG-03 drops its single-membership caveat; `entityGroups` returning a list
  becomes literally correct.
- MG-04's group-scoped grants are unaffected in shape — a device simply reaches
  them through more than one group.
- E3 (re-add semantics) dissolves: adding to a second group is now additive.
- Applies to `object_group_resources` too (`PRIMARY KEY (resource_id)`,
  `001_initial.sql:537-544`) — decide in ATOM-04 whether to change both for
  symmetry.

**Decision:** M:N, done now (ATOM-04)

**Notes:** Multiple-users-per-device was already supported; the change buys
overlapping device sets.

---

#### A2 🟡 Is `provisioning_state` a device attribute or a Bootstrap enrollment state?

**Blocks:** MG-09, MG-13 · **Ref:** [architecture.md §10](./architecture.md#10-open-questions)

Atom's `entities.status` is constrained to `active/inactive/suspended`
(`001_initial.sql:96`), so `pending` cannot live there.

- **A. Device attribute** (`provisioning_state` in `attributes`).
- **B. Bootstrap enrollment state** (`bootstrap/status.go`).
- **C. Both**, mirrored.

**DECIDED: A — device attribute.**

**Sensors have no enrollments.** A Bootstrap enrollment is a record used by
something that fetches its config with an external ID and key. A BLE watermeter
never does that — only the *gateway* bootstraps. So the devices most in need of a
provisioning state have no enrollment record to hold it. B would require
inventing enrollments for things that never enroll.

*(The original second reason — that MG-07 read this on the publish hot path — no
longer applies: A7 and A8 removed the per-device check entirely. The conclusion
is unchanged.)*

**Consequences**

- `provisioning_state` is a device attribute; Atom's `entities.status`
  (`active/inactive/suspended`, `001_initial.sql:96`) is untouched and means
  something different.
- It is a **lifecycle field only**. Nothing on the publish path consults it —
  data is accepted and attributed regardless (A8, late binding).
- Bootstrap enrollment status stays a separate, gateway-only concern. The two are
  not synonyms and should not be presented as such.

**Decision:** A — device attribute

**Notes:** More forced than the original framing implied; B is not really viable
for credential-less sensors.

---

#### A3 ⚠️ Does Magistrala model local transport addressing? — REVERSED by A13

**Blocks:** MG-12 · **Ref:** [architecture.md §10](./architecture.md#10-open-questions)

The agent needs to reach a meter over BLE or Modbus. Does the platform know the
BLE address / Modbus unit ID?

- **A. Yes — device attributes**, shape declared per device type via JSON Schema,
  rendered into the agent config.
- **B. No — agent-side only.** The gateway maps serial → local address itself.
- **C. Opaque passthrough** — stored and rendered, never interpreted.

**DECIDED: B — agent-side only.**

**The separation this establishes:** the cloud addresses devices by **serial**;
the gateway owns its local bus and resolves serial → local address itself.
Magistrala never learns what a Modbus unit ID is.

This is cleaner than the original recommendation credited. It keeps
protocol-specific knowledge — BLE, Modbus, M-Bus, LoRa each with different
address shapes — entirely out of the platform, and commands still work: the cloud
sends "`set_interval` on serial ABC123" and the gateway does the mapping.

**Consequences**

- MG-12: `DeviceContext.Transport` is **removed**. The rendered fleet carries
  serial, device type and identity — enough for the agent to know *which* meters
  are its own and match them against what it already found on the bus.
- MG-13: the announce payload drops `transport`. A gateway reports serials it
  discovered, not how it reaches them.
- MG-02/MG-10: device type schemas describe measurements and commands only. No
  transport shape, which keeps the capability model smaller.
- **Gateway replacement requires rediscovery.** A replacement gateway has no
  inherited address map and must rescan the bus. Acceptable when discovery is
  automatic; it is a real gap if any device needs manual local configuration.
- **No central record of physical wiring.** "Which Modbus unit is meter ABC123
  on?" is answerable only on the gateway. Worth knowing before it is needed for
  field diagnostics.

**Decision:** B — agent-side only

**Notes:** Reversible later — adding an optional typed attribute is additive.
Revisit if manual local configuration or remote diagnostics become requirements.

---

#### A4 🟡 Are global (tenant-less) device types exposed?

**Blocks:** MG-02, MG-10

Atom profiles can be global or tenant-scoped, with different uniqueness
constraints (`001_initial.sql:66-72`). A global "watermeter-v2" shared across all
domains, or per-domain types only?

- **A. Tenant-scoped only.** Always send `tenant_id`. Simple, isolated.
- **B. Both.** Platform-defined catalogue plus domain-specific types. Needs
  precedence rules when keys collide and an answer to who may edit global types.
- **C. Global read-only**, seeded by the platform operator; domains may bind but
  not modify.

**Recommendation:** A for 1.0. B is a genuine feature (a vendor catalogue) but
needs governance, and it can be added without breaking A.

**Decision:**

**Notes:**

---

#### A6 ⚠️ Should `gateway` be a native Atom entity kind? — SUPERSEDED by A12

**Blocks:** MG-09, MG-10 · **Ref:** [architecture.md §4.3](./architecture.md#43-gateway-is-its-own-atom-entity-kind-sensor-is-not)

Originally rejected on the grounds that `entities.kind` is a closed CHECK
constraint and "to Atom a gateway is a device". Two checks overturned that:

- **Atom anticipates new machine kinds.** `001_initial.sql:191-192`: *"The stable
  invariant enforced here is `shared_key => entity is non-human`, which holds as
  new machine kinds are added."* `is_machine()` is `!matches!(self, Human)`
  (`src/models/enums.rs:17-19`), so a new kind inherits machine handling free.
- **`kind` is mutable.** `updateEntity` accepts it
  (`src/graphql/entities.rs:307`), so device → gateway promotion is an update.

**DECIDED: yes for `gateway`, no for `sensor`.**

Kinds become `human, device, gateway, service, workload, application`. Sensors
and directly-connected devices are both `device`.

**What gateway-as-kind buys**

| Mechanism | Attribute | Native kind |
|---|---|---|
| `entity_object_type()` | `entity:device` | `entity:gateway` |
| `scope_mode: 'object_type'` grants | not expressible | "every gateway in this tenant" as one block |
| `authorizedObjectIds(objectType:)` | not expressible | native |
| `profiles (tenant, object_kind, kind, key)` | shared namespace — a watermeter type can be bound to a gateway | separate namespaces |
| `ActionAssignmentRule.entity_kind` | gateways inherit all device guardrails | gateway-specific rules |

`magistrala_kind` disappears for this distinction.

**Why not `sensor`.** `profiles` uniqueness is `(tenant, object_kind, kind, key)`,
and the same watermeter model deploys both behind a gateway and directly over
NB-IoT. A `sensor` kind needs **two profiles for one physical device type**, which
will drift. It also encodes topology into identity — "sensor" is where a thing
sits, not what it is, and attachment already says that. The apparent prize
("sensors can never be granted publish" enforced structurally) is unreachable
anyway, since direct devices need `{device, publish, …}` and sensors share that
kind regardless.

**Consequences**

- ~~New PRD ATOM-05~~ — **withdrawn by A12**; gateway is a capability.
- **Silent trap:** `pkg/atom/bootstrap.go:72-87` installs
  `{entity_kind: device, publish, resource:channel}` as the only publish
  guardrail. Once gateways stop being `device`, **they cannot publish** until
  gateway rules are added. Must ship in the same release as MG-09.
- ATOM-01 is still needed for `gateway_id` filtering, but listing gateways no
  longer depends on it.
- Kind mutation orphans object-type-scoped grants written against the old kind —
  correct, but needs pinning by test.

**Decision:** `gateway` yes, `sensor` no

**Notes:** Reverses the original §4.3 recommendation.

---

#### A7 🔴 How is gateway → device publish authority established?

**Blocks:** MG-05, MG-07, MG-09 · **Supersedes:** B4

The original model used a scalar `gateway_id` attribute and checked
`device.gateway_id == publisher` on every publish. **That model is dead:** one
device can broadcast to several gateways — a BLE meter heard by three gateways in
range is a normal deployment, not an error.

**DECIDED: the channel is the boundary.** A gateway connected to a channel may
publish any `device_id` on it. No per-device authorization, no device lookup on
the publish path.

**Consequences**

- **The `gateway_id` *attribute* is removed.** No attachment cache, no
  invalidation-on-re-homing. **A declared association still exists** — as a
  gateway fleet group, see A9. A7 governs the authorization path only; it does
  not mean the platform forgets which devices were commissioned where.
- **[MG-07](./prd/MG-07-gateway-attachment-enforcement.md) dissolves.** Its whole
  subject was enforcing an attachment that no longer exists.
- **GW↔device is derived, not stored.** "Which devices does this gateway serve?"
  is a query over stored messages — `DISTINCT device_id WHERE publisher = G`
  — which MG-06 already makes possible, and which handles many-gateways-per-device
  for free.
- **Accepted risk:** a compromised gateway can fabricate readings for any device
  on channels it holds. In particular it can impersonate *another customer's*
  meter if they share a channel. **Deployment guidance: segregate channels per
  site or customer** where that matters. This is the explicit trade for a
  lookup-free hot path.
- B4 (attachment enforced Magistrala-side vs in Atom) is moot — there is no
  attachment to enforce.

**Decision:** Channel is the boundary

**Notes:** Reverses the `gateway_id` design in §3.2 and §5.4.

---

#### A8 🔴 How does an arbitrary-string serial resolve to a device?

**Blocks:** MG-05, MG-06, MG-08, MG-09 · **Supersedes:** C4's mechanism question

`serial == device_id`, an arbitrary string, carried in every publish topic.
Atom's `alias` cannot hold it — slug-constrained to
`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$` (`001_initial.sql:105-113`) — and Atom has
no external-identifier field today.

**DECIDED: add a generic `external_id` to Atom entities**, unique per tenant,
format-unconstrained, indexed. New PRD
**[ATOM-06](./prd/ATOM-06-entity-external-id.md)**.

```sql
ALTER TABLE entities ADD COLUMN external_id TEXT;
CREATE UNIQUE INDEX ON entities (tenant_id, external_id)
  WHERE external_id IS NOT NULL AND deleted_at IS NULL;
```

Generic framing: every system integrating with the physical world carries
identifiers assigned elsewhere — serials, MACs, employee numbers, SKUs. This is
not IoT-specific.

**Consequences — and this is the part that matters**

- **Messages store the external string, not the Atom UUID.** Resolving on publish
  would reintroduce exactly the hot-path lookup A7 removed. `device_id` is an
  opaque `TEXT` column.
- **Late binding.** Data may arrive for a `device_id` with no device entity. It
  is stored and attributed regardless. Creating a device with that `external_id`
  later makes its *historical* data viewable and shareable retroactively.
- **This supersedes B1.** There is no need to reject data from unregistered or
  pending devices, because registration is no longer a precondition for
  attribution — see the revised B1 below.
- **MG-08 gains a translation step:** authorized device *entities* (UUIDs) must
  be mapped to their `external_id`s before filtering messages. Small, cacheable,
  once per query.
- **Orphan data** — a `device_id` with no entity — is readable only via
  channel-level access. It cannot be granted to a customer, because there is no
  object to grant.
- C4 (serial uniqueness per tenant) is now *enforced by the index* rather than by
  convention.

**Decision:** Atom `external_id`, unique per tenant

**Notes:** ATOM-01's two original justifications (list-by-gateway, serial lookup)
are both gone — see its revised status.

---

#### A9 ⚠️ How is the declared device↔gateway association stored? — SUPERSEDED by A10

**Refines:** A7 · **Enables:** the gateway UI view (§5.5)

A7 removed the gateway↔device link from the authorization path — correctly, the
channel is the publish boundary. An earlier revision then over-generalised that
into **no stored link at all**, deriving everything from traffic.

That breaks the wired case, and breaks it where it matters. A device may be
associated with **0, 1 or many** gateways depending on the physical link:

| Link | Cardinality |
|---|---|
| Standalone — connects for itself | 0 |
| Modbus, wired, exclusive | 1 |
| wM-Bus, broadcast | many |

Derived-from-traffic handles the broadcast case and fails the wired one: a
commissioned meter that has not yet published is invisible, there is nowhere to
record "meter 7 is on gateway G", and **"should be here but silent" cannot be
distinguished from "was never here"** — which for a wired link is precisely the
alarm condition.

**DECIDED: keep both** — that part stands. The *storage* choice below was
reversed by A10: it is a relation, not a group. Originally: an **Atom object
group per gateway** tagged
`attributes: {role: "gateway_fleet", gateway_id: …}`, with devices as members.

Options weighed:

| | Verdict |
|---|---|
| **Object group per gateway** | **Chosen.** M:N membership (A1/ATOM-04) gives 0/1/N directly; `groupMembers` and `entityGroups` give both directions natively; FK cleans up on device delete; needs no Atom work beyond ATOM-04 |
| `gateway_ids: []` attribute on the device | No referential integrity; reverse lookup needs ATOM-01 array containment; stale IDs after gateway deletion |
| `device_ids: []` on the gateway | A device cannot cheaply answer "which gateways serve me"; two gateways claiming one device means writing to both |

**Consequences**

- **A7 stands, narrowed.** No per-device check on the publish path; the channel
  remains the boundary. The declared association is for commissioning and
  operations, never for authorization.
- The difference between declared and observed becomes the operational signal —
  healthy / silent / undeclared (§3.2).
- Fleet groups and customer groups share the object-group namespace, discriminated
  by `attributes.role`. Safe under M:N membership, but **a group-scoped grant over
  a fleet group hands over every device on that gateway.** Sometimes wanted, never
  by accident — any UI offering group grants must show which kind it is.
- MG-15 grows: it must merge declared membership with observed traffic, not just
  aggregate messages.
- Bootstrap can populate fleet membership at commissioning time, which is the
  natural place for it — but the model does not depend on Bootstrap to do so.

**Decision:** Object group per gateway, `role: gateway_fleet`

**Notes:** Corrects an over-generalisation of A7, not A7 itself.

---

#### A10 🔴 Is the device↔gateway association a group or a relation?

**Supersedes:** A9

A9 stored the declared association as an Atom object group per gateway. That was
wrong, and the error is worth naming because it is a recurring one: **reaching
for an existing primitive instead of asking what the thing actually is.**

A group is a *user-facing organisational object* — named, created deliberately,
shared. A gateway's device list is *a fact about wiring*. Modelling the second as
the first produced:

- two Atom objects per gateway, with lifecycle coupling between them;
- fleet groups sharing a namespace with sharing groups, so a group-scoped grant
  on the wrong one silently hands over an entire gateway's fleet;
- a gateway that is simultaneously an entity that publishes and, indirectly, a
  group — two things for one concept.

**DECIDED: it is a relation.** A device declares which gateways it is reachable
through:

```
Device meter-7 { gateways: ["gw-a", "gw-b"] }
```

Stored as an entity attribute; opaque to Atom, which never interprets it.

| | Verdict |
|---|---|
| **Attribute on the device** | **Chosen.** 0/1/N natural; forward lookup free; reverse lookup is one `attributesContains` query; no second object; no lifecycle coupling; groups regain a single meaning |
| Object group per gateway (A9) | Rejected — the complexity above, for referential integrity that resolve-on-read supplies well enough |
| `device_ids: []` on the gateway | Rejected — a device cannot cheaply answer "which gateways serve me", and two gateways claiming one device means writing to both |

**Consequences**

- **Groups mean exactly one thing again: sharing.** The namespace hazard in A9
  disappears entirely.
- **A1 returns to required.** "Which devices are declared on this gateway" is an
  `attributesContains` query over entities. A9 had removed A1's justification and
  demoted it to P2; this restores it.
- **No referential integrity.** Deleting a gateway leaves its ID in every device
  that named it. Resolve-and-drop on read is sufficient for correctness; whether
  to sweep on deletion, and whether operators see stale entries, is open (§7).
- A7 is unaffected — the relation is never consulted for authorization.
- Declared and observed remain distinct, and their difference remains the
  operational signal (§2.4). Only the storage of *declared* changed.

**Decision:** A relation — `gateways []` attribute on the device

**Notes:** Second time the same failure mode has appeared. The first was treating
a constraint in Atom's schema as a deliberate boundary; this was treating an
available primitive as the right shape. Both were fixed by asking what the thing
is before asking what is at hand.

---

#### A11 🟡 What kind of thing is a Gateway?

**Refines:** A10 · **Shapes:** §2.1, §2.2

A gateway resisted classification: like a group in that devices belong to it, but
it has configuration and publishes messages; like a device in that it has
identity and telemetry, but it aggregates and fronts things that never speak for
themselves.

**The framings all shared one assumption: that a gateway *contains* devices.**
Dropping that resolves it.

A gateway does not contain a meter. The meter exists regardless — bolted to a
pipe, its serial stamped on the case. Remove the gateway and the meter does not
cease to exist; it goes dark. What a gateway provides is **reachability**: it is
*how* a device's data arrives.

**DECIDED: a Gateway is a Device with a proxy role.** Not a container, not a
group, not a separate class. The nearest well-understood analogue is a Kubernetes
Node — its own identity, metrics and config, with Pods running *on* it, owned by
nobody. "Host" is a role, not a container.

**Six normative consequences** (§2.2), each settling a question that would
otherwise be decided ad hoc:

1. Devices are primary — created and identified independently; the relation is a
   property of the device.
2. Deleting a gateway never deletes devices. Anything that cascades is a bug.
3. Gateway access and device access are orthogonal — technician vs customer.
4. A gateway's own telemetry is ordinary device telemetry; no special case.
5. Aggregation attributes to the subject, not the computer.
6. Downlink routes along the reachability edge; the relation is the routing table.

**Deferred but named:** fault correlation (§3.6). Reachability being a function
means a device's silence has several causes, and the platform can distinguish
them — one gateway down should be one incident, not forty alerts. It needs a
gateway-liveness notion the model does not yet carry, and it only works because
*declared* exists: with the gateway down, nothing is observed to correlate.

**Decision:** Device with a proxy role; reachability, not containment

**Notes:** Third occurrence of the same failure mode — reasoning from available
shapes rather than from what the thing is. The first two were treating an Atom
schema constraint as a boundary, and treating groups as the right primitive for
wiring.

---

#### A12 🔴 Is Gateway a device *type* or a device *capability*?

**Supersedes:** A6 · **Withdraws:** ATOM-05

A6 made `gateway` a native Atom entity kind. That contradicted the model's own
definition — §2.1 calls a gateway *a Device with a proxy role*, and **types are
exclusive while roles compose.**

The case it breaks is real and common: a smart electricity meter that also
concentrates wM-Bus water meters. It produces its own readings *and* relays
others. Under exclusive kinds it must be one or the other, and either answer is
wrong.

**Survey of comparable platforms — the evidence is one-directional:**

| Platform | Gateway is | Relation | Cardinality | Purpose of the relation |
|---|---|---|---|---|
| ThingsBoard | Device + **`Is gateway` boolean** | Generic `Manages` entity relation, re-wired at runtime | M:N | Downlink routing (attributes, RPC) |
| AWS Greengrass v2 | Core device — **an IoT `thing`, same type as clients** | `BatchAssociateClientDeviceWithCoreDevice` | **M:N** | Discovery — find the core, fetch certs |
| Azure IoT Edge | Device identity with edge capability | Stored **parent/child** | **1 parent**, ≤100 children | Auth delegation |
| ChirpStack | Separate infrastructure, shared across tenants | **none** | — | — (radio, per packet) |
| Akenza | Not first-class; upstream in the network server | — | — | — |

**Nobody makes gateway a distinct entity type.** The single exception is
ChirpStack, where gateways are dumb radio infrastructure producing no data —
which ours are not.

The survey also settles cardinality: **it follows purpose.** Azure needs exactly
one parent because the parent is a trust anchor. Greengrass allows M:N because
discovery does. ThingsBoard stores no fixed link because routing only cares who
is connected now. None of our purposes — commissioning record, fault correlation,
downlink hint, UI — is singular, so M:N is right for a reason rather than by
preference.

**DECIDED: a capability.** `is_gateway` on the Device. Atom entity kind stays
`device`.

The three benefits claimed for a distinct kind all collapse:

| Claimed | With a capability |
|---|---|
| Separate profile namespace | A concentrator-meter needs *one* type declaring both measurement and relay — more correct, not less |
| `entity:gateway` object-type grants | Attribute filter — needs A1, already required |
| Native list filter | Same |

Each reduces to "needs A1", which the reachability relation requires anyway. The
kind bought nothing the capability does not, and forbade a legitimate device.

**Consequences**

- **ATOM-05 withdrawn.** One fewer Atom migration.
- **A silent trap disappears with it.** `pkg/atom/bootstrap.go:72-87` installs
  `{entity_kind: device, publish, resource:channel}` as the only publish
  guardrail; a new kind would have stripped every gateway's right to publish
  until matching rules were added.
- Gateway and device types share one namespace — correct, given composition.
- Listing gateways is `attributesContains: {is_gateway: true}`.
- ChirpStack contributes one more confirmation: a device's data belongs to *its
  owner* regardless of which gateway relayed it. Our grants attach to devices,
  never to gateways, so this already holds.

**Decision:** Capability `is_gateway`, not an entity kind

**Notes:** Reached by checking how ThingsBoard, Greengrass, Azure, ChirpStack and
Akenza model it rather than by further reasoning from first principles.

---

#### A13 ⏸ Where does the local bus address live? — DEFERRED to phase 2 by A14

**Reverses:** A3 · **Shapes:** §3.3, §5.2 · **Returns MG-12 to scope**

For self-identifying protocols there is nothing to decide — a wM-Bus telegram
carries its serial, which flows through to the topic and to `external_id`
untouched. **Bus-addressed protocols are the exception:** a Modbus unit ID is an
address, not a serial, and the gateway cannot derive one from the other. Something
must hold `unit 7 → WM-2024-ABC.123`.

A3 put it agent-side only. That cannot stand alongside bootstrap delivering it,
because **bootstrap config is generated by the cloud** — if the cloud does not
hold the map, there is nothing to generate. The two positions were in direct
conflict and A3 loses, for reasons that were not on the table when it was decided:

- **Gateway replacement.** Agent-side-only means rebuilding the map by hand. Cloud-held
  means the replacement bootstraps and gets it.
- **Fleet scale.** 500 gateways × manual config is not viable.
- **It is the one thing the agent cannot derive.** Everything else about a Modbus
  meter it can discover; the serial behind unit 7 it cannot.

**DECIDED: on the edge, opaque.**

```jsonc
"gateways": [
  { "id": "gw-a", "address": { "modbus_unit": 7 } },
  { "id": "gw-b" }
]
```

Options weighed:

| | Verdict |
|---|---|
| **On the edge** | **Chosen.** The address is a property of the (device, gateway) pair — the same meter on two buses can have two different unit IDs. Single source of truth; every cardinality case falls out |
| On the device, flat | Breaks when one device sits on two buses at different addresses |
| On the gateway, as a map | Duplicates the relation; two places to update |
| Both device and gateway | Two sources of truth for one fact — they diverge |
| Agent-side only (A3) | Cannot be bootstrapped; lost on gateway replacement |
| Bus address *becomes* the serial (`gw-a:modbus:7`) | Rejected — identity becomes location-dependent, and re-addressing a meter would lose its history |

**What preserves A3's actual value: Magistrala never parses the address.** It is
stored, rendered into config, and compared for equality. No Modbus, BLE, M-Bus or
LoRa knowledge enters the platform. This is the "opaque passthrough" option A3
dismissed as "typed-minus-validation" — it is in fact the distinct and correct
middle.

**Consequences**

- **MG-12 returns to scope.** Bootstrap becomes the delivery mechanism rather than
  a scenario source, and gateway config is generated from the declared relation.
- **Conflict detection works without interpretation** — two devices with
  byte-identical address blobs on one gateway is a containment query
  (§3.3, verified).
- **Read-modify-write** on the attributes array, with no optimistic concurrency in
  Atom (§7).
- **A1 is required**, and its acceptance criterion changes from string-array to
  object-array containment.
- **Preferred population path is discovery, not typing.** Many Modbus meters
  expose their serial in a holding register and the device type says which one, so
  the agent can scan, read it, and announce `unit 7 = serial X`. Manual entry
  becomes the fallback. That turns "operator types 500 mappings" into "operator
  confirms 500 discovered mappings" — the same inversion gateway-announced
  discovery makes for device creation.
- This is the point where the model stops being purely UI-level: an address on the
  edge is platform data.

**No new coupling.** Atom stores an opaque blob in a field designed for opaque
blobs, and filters it with a mechanism that already exists for resources. Nothing
in A1–A6 mentions a device, gateway or protocol.

**Decision:** On the edge, opaque, rendered into bootstrap config

**Notes:** Reverses A3. Every other reversal in this design reduced what Atom must
know; this one adds data but not knowledge.

---

#### A14 🔴 Phase 1 is the entity model; messaging is deferred

**Defers:** A13 · **Relaxes:** the serial format constraint

Scope call: build the **entity model** now and leave messaging untouched.

| | Phase 1 | Phase 2 |
|---|---|---|
| Device, `is_gateway`, `gateways[]` | ✅ | |
| `external_id` on Atom entities | ✅ | |
| Device types, groups, sharing of device *records* | ✅ | |
| `d` topic segment, `device_id` on messages | | ⏸ |
| Per-device storage, filters, reader authorization | | ⏸ |
| Bus `address` on the edge, bootstrap delivery | | ⏸ |

**Topics become application-level.** Magistrala adds no device segment, imposes no
serial format, reserves no subtopic prefix. Two earlier rulings fall out of scope
with it:

- **A13 (address on the edge) is deferred.** Without config delivery there is
  nothing to render, so the relation carries gateway IDs only. The reasoning in
  A13 stands for whenever phase 2 arrives.
- **The `/` restriction on serials is dropped.** It existed to protect a topic
  grammar that phase 1 does not introduce. `external_id` is an arbitrary string
  and neither Atom nor Magistrala polices it; an application putting one in a
  topic owns its own encoding.

**No special treatment of gateways anywhere in the platform.** `is_gateway` is an
attribute, `gateways[]` is an attribute, and the gateway view is UI composition
over a query filter. No gateway-specific code path exists in ingest, storage,
authorization or routing.

**The consequence to be clear-eyed about**

Per-device *data* sharing does not work in phase 1. A customer can be granted a
device **record**, but message data cannot be filtered by device — that needs
`device_id` on messages. The watermeter requirement that started this design is
therefore only partly met until phase 2. Accepted knowingly.

**One thing that should not be deferred with it**

[MG-08](./prd/MG-08-reader-authorization.md) does two separable things. The second
— device-level filtering — belongs to phase 2. The first does not: today the
`publishers` query filter is applied **without any authorization check**
(`readers/api/http/transport.go:251-266`), so any user who can read a channel can
read every publisher on it by changing a parameter. That is a live defect on
`main`, it needs none of the device work to fix, and deferring it wholesale
leaves it open.

**Decision:** Phase 1 = entity model; messaging deferred; topics application-level

**Notes:** A staging decision, not a reversal of the model. The design above stays
the target.

---

#### A15 🔴 Can a gateway declare gateways of its own?

**Narrows:** A12's composability — a gateway may still report its own
telemetry directly, per §2.2 consequence 4; what this rules out is a gateway
being *reachable through* another gateway.

Nothing in phase 1 forbade it. `gateways []` is 0..N on **any** device (§2.3),
and a Gateway is just a Device (§2.1) — so by the letter of the model, gateway
A could declare `gateways: ["B"]` where B is also `is_gateway: true`, and the
implementation agreed: `pkg/atom/devices.go`'s `SetDeviceGateways` had no check
against it, and explicitly special-cased the degenerate instance of it — a
device naming *itself* — as supported, working behavior (the pre-A15 `P2` path,
re-reading the device after `MarkGateways` specifically so a self-referential
`is_gateway` write would not be clobbered).

**The product decision: reject it.** A gateway may not carry a non-empty
`gateways []` of its own — including naming itself. Two reasons:

- **The relation is a reachability path (§2.1), and a gateway's own reachability
  is not staged.** Every purpose the relation serves — commissioning record,
  fault correlation, downlink hint, UI — assumes the *far* end is a leaf device
  that cannot represent itself otherwise. A gateway already can: it holds
  credentials and connects directly (§2.2 consequence 4). Chaining adds a
  topology (and a downlink-routing question, §2.2 consequence 6) the model
  never designed for, for a case the capability model does not need — a
  concentrator-meter's own telemetry is already covered by A12 without chaining.
- **It was never exercised deliberately.** The self-reference path existed
  because nothing rejected it, not because a deployment needed it. Closing it
  now, before MG-09 freezes the public API (its own words: "widest blast radius
  in the programme"), is cheaper than closing it after the SDK and every client
  ship against the wider contract.

**DECIDED: gateways do not chain.** A device may not simultaneously carry
`is_gateway: true` and a non-empty `gateways []`. Enforced at write time by
`pkg/atom.ValidateGatewayChain`, called from both existing writers of these
attributes — `SetDeviceGateways` and the CLI's device create/update commands —
per the invariant contract on `AttributeGateways`. Self-reference is rejected
by the same check, since it is the case where the two facts land in one call.

**Consequences**

- **Does not touch A12.** A gateway that also reports its own measurements —
  the composite device UI-01's criterion 4 exists to guard — is unaffected: it
  connects directly and never populates its own `gateways []`. Only *upstream*
  gateways of a gateway are rejected.
- **Magistrala-only.** Atom has no notion of "gateway" (ATOM-05) — `is_gateway`
  and `gateways` are Magistrala-side attribute conventions on a generic
  `device` entity, so nothing in Atom changes.
- **Two write paths today, both updated.** `pkg/atom/devices.go`
  (`SetDeviceGateways`) and `cli/devices.go` (create/update, which bypass
  `SetDeviceGateways` entirely per the existing `P3` note). A future writer —
  MG-09's HTTP API/SDK, not yet built — must call `ValidateGatewayChain` too,
  the same obligation `AttributeGateways`'s doc comment already states for
  `MarkGateways`.
- **No migration performed here.** Existing data was only ever reachable
  through the CLI (no UI or API enforced anything before this). Nothing in
  this change sweeps or corrects devices already in that state; it only stops
  new writes from creating it.

**Decision:** Gateways do not chain — `is_gateway` and non-empty `gateways []`
are mutually exclusive on one device, self-reference included

**Notes:** A restriction relative to what phase 1 specified, not an
implementation bug fix — the pre-A15 code was behaving as designed. Reached
after `magistrala-ui`'s UI-side gateway-promotion logic diverged from the Go
model and needed one of the two changed to match.

---

#### A5 ⚪ Does 1.0 need device lifecycle events?

**Ref:** [architecture.md §10](./architecture.md#10-open-questions)

**Largely answered by a later finding.** The question assumed Magistrala would
have to *produce* these events. It does not — **Atom already publishes ~40 domain
events** through a transactional outbox with an AMQP publisher
(`atom/migrations/004_event_outbox.sql`, `atom/src/events/publisher.rs`),
including `entity.create/update/delete`, `group_member.add/remove`,
`direct_policy.create/delete` and `entity.parent_group.set/clear`.

It is dark by default — publishing is a no-op unless `ATOM_EVENTS_AMQP_URL` is
set (`atom/docker-compose.yml:51`), and Magistrala's compose sets no `ATOM_EVENTS*`
variable at all.

So the work is a **consumer**, not a producer: [MG-14](./prd/MG-14-atom-event-consumer.md).
Much smaller than B, and it de-risks both MG-07 and MG-08.

- **A. No consumer.** TTL-only caches; revisit post-1.0.
- **B. ~~Magistrala emits its own entity events.~~** Unnecessary — Atom's cover it.
- **C. Consume Atom's events** for cache invalidation (MG-14).

**Recommendation:** C. The producers deleted in `16ba29cf4` do not need
reintroducing.

**Decision:**

**Notes:** Remaining question is whether MG-14 lands before or after MG-07/MG-08;
both ship TTL-only without it.

---

### Section B — Security and enforcement

---

#### B1 🔴 What happens to data from a pending device?

**Blocks:** MG-07, MG-13 — **these must be decided together**

**What "unapproved" means.** The `pending` state exists *only* because of
gateway-announced discovery (MG-13). In that flow a gateway reports serials found
on its local bus, and each unknown serial **creates a device record** — so a
device can come into existence because a gateway asserted it, with no human
involved.

`pending` marks exactly that: **exists, but no operator has confirmed it is real,
assigned its device type, or decided who owns it.** Approval is where a human
supplies what a gateway cannot know.

Devices created cloud-first are never pending — an operator created them, so they
are `provisioned` on creation. If gateway-announced discovery were dropped, this
state would not exist at all.

- **A. Reject.** MG-07 additionally denies `pending`. Readings between install and
  approval are lost.
- **B. Quarantine.** Store, marked pending; becomes visible on approval.
- **C. Accept.** Data flows immediately; approval is metadata only.

~~**DECIDED: A — reject.**~~

**SUPERSEDED by A7 + A8 — the question no longer arises in this form.**

Rejecting pending-device data required a device lookup on the publish path, which
A7 removed and A8 made unnecessary. Under late binding:

- `device_id` is an opaque string on the message. Data is stored and attributed
  whether or not a device entity exists.
- "Approval" becomes **creating the device entity** with that `external_id`,
  which retroactively makes its history viewable and shareable.
- **No data is lost between installation and approval** — the original decision's
  main cost simply disappears.
- Orphan data (no matching entity) is readable only through channel-level access.
  It cannot be shared with a customer, because there is no object to grant.

Strictly better than the original ruling: approval still gates *access*, and no
longer gates *ingestion* — which it only did to compensate for a check we no
longer perform.

**Decision:** ~~Reject~~ → **Accept, late-bind** (per A8)

**Notes:** `provisioning_state` survives as a device-lifecycle field (A2) but is
no longer consulted on the publish path.

---

#### B2 🟡 What is the acceptable revocation window?

**Blocks:** MG-08

MG-08 caches each subject's authorized device set. The TTL *is* the revocation
SLA: after removing a customer's access, they keep reading until it expires.

- **A. Short (30–60s).** Fast revocation, more load on Atom.
- **B. Medium (5 min).** Standard cache behaviour.
- **C. Long (15 min+) with explicit invalidation** on grant changes.

**Updated by a later finding.** Atom already publishes `direct_policy.create`,
`direct_policy.delete`, `group_member.add` and `group_member.remove` through its
outbox (`atom/src/events/publisher.rs`), so C's "explicit invalidation" is
available — see [MG-14](./prd/MG-14-atom-event-consumer.md). It is not enabled in
Magistrala's deployment today.

That does **not** make the TTL a free parameter. Event delivery is at-least-once
over a broker that can be down, so the TTL remains the correctness floor: with
events flowing, revocation is near-immediate; without them, the TTL *is* the SLA
and must be defensible on its own.

**Recommendation:** A (30–60s) **plus** MG-14. Short enough to defend unaided,
with events making the common case near-instant. Do not stretch the TTL because
events exist.

**Decision:** ______ seconds

**Notes:**

---

#### B3 🟡 How is the reader admin bypass determined?

**Blocks:** MG-08

Domain admins legitimately read all devices on a channel. How is that recognised?

**Why this needs care.** After MG-08 the reader asks Atom "which devices may this
caller read?" and intersects that with the requested filter. A domain admin must
still see everything — so how does the code recognise one?

The tempting shortcut is *"if the authorized list is empty, skip filtering"*,
reading empty as unrestricted. It breaks, because two opposite situations produce
an identical empty list:

| Caller | Per-device grants | Should see |
|---|---|---|
| Domain admin | none — holds a *tenant-wide* grant | everything |
| User with no access at all | none | nothing |

Under that shortcut both get everything, so the caller with the fewest
permissions receives the most data.

- **A. Explicit tenant-scoped capability check** — "may read all devices in this
  domain".
- **B. Role-name matching.**
- **C. Treat an empty authorized set as unrestricted.**

**DECIDED: A.** Ask the question directly, so an empty list unambiguously means
"no access" and admin status comes from an actual grant. C is recorded as a
rejected option precisely because it is the shape a reasonable implementation
falls into.

**Decision:** A — explicit capability check

**Notes:** MG-08 acceptance criteria 4 and 5 are the guards; both need dedicated
tests rather than incidental coverage.

---

#### B4 🟡 Confirm: attachment enforced Magistrala-side, not in Atom?

**Blocks:** MG-07 · **Ref:** [architecture.md §6.3](./architecture.md#63-explicitly-rejected)

- **A. Magistrala-side** check with a local attachment cache. Atom stays generic.
- **B. Pass `device_id` in the authz context** and let Atom enforce. No new cache,
  but Atom must understand `gateway_id`.

**Recommendation:** A — B breaks Atom's domain-agnosticism, which is a stated
constraint. Recorded so it is rejected deliberately rather than rediscovered.

**Decision:**

**Notes:**

---

#### B5 🟡 Announce rate limits and pending caps

**Blocks:** MG-13

A gateway credential can otherwise create unbounded entities.

- Max announcements per gateway per hour: ______
- Max pending devices per gateway: ______
- Behaviour on breach: reject / throttle / alert: ______

**Recommendation:** Start restrictive — 10 announce calls/hour, 500 pending per
gateway, reject with a clear error. Loosening later is safe; tightening after
deployments depend on it is not.

**Decision:**

**Notes:**

---

### Section C — API shape and compatibility

---

#### C1 🔴 Does deployed data use the unnamespaced `objectType`?

**Blocks:** MG-01 — the only non-additive part of that PRD

`policy_service.go:139` reads `"device"` while `:225` writes `"entity:device"`.
Fixing the read path orphans anything written with the old form.

- **A. No deployed data** — straight fix, no migration.
- **B. Deployed data exists** — needs a migration, or a read-path fallback
  accepting both forms for one release.

**DECIDED: ignore backwards compatibility. No fallback, no dual-form read path.**

Fix `policy_service.go:139` to emit the namespaced form via the shared helper and
leave it there. Any block written with the unnamespaced value is simply wrong and
should be corrected or discarded, not accommodated.

**Consequences**

- MG-01's only non-additive risk is removed; the PRD becomes purely additive
  plus fixes.
- No compatibility shim to carry, and no dual-form matching in
  `directPolicyMatches`.
- If a deployment *does* hold old-form blocks, those grants stop matching and
  must be rewritten. Acceptable under this decision, but it belongs in release
  notes rather than being discovered in the field.

**Decision:** No backwards compatibility — straight fix

**Notes:** This ruling ("ignore back-compat, avoid technical debt") also settles
the residual half of C2.

---

#### C2 ⚪ PAT enum — RESOLVED, premise was wrong

**Blocks:** MG-11 (downgraded from 🔴 to a migration note)

**What a PAT scope is.** A personal access token carries *scopes* — tuples of
(domain, entity type, operation, entity id). `EntityType` is the "which kind of
thing" part, declared positionally (`auth/pat.go:75-86`):

```go
const (
    GroupsType EntityType = iota  // 0
    ChannelsType                  // 1
    ClientsType                   // 2
    BootstrapType                 // 3
    ...
)
```

**The original concern was that removing `ClientsType` would shift every later
value, silently changing what issued tokens mean. That concern was unfounded —
`EntityType` is never persisted or transmitted as a number.**

| Path | Representation | Evidence |
|---|---|---|
| Database | `entity_type VARCHAR(50)`, stores the name | `auth/postgres/init.go:97`, written `repo.go:426`, read `repo.go:554` |
| JSON | name | `auth/pat.go:155-164` |
| Text | name | `auth/pat.go:166-174` |
| gRPC | `string entity_type = 6` | `internal/proto/auth/v1/auth.proto:47` |

The ordering never leaves the process, so renumbering is safe.

**What actually remains:** rows holding `entity_type = 'clients'` will fail
`ParseEntityType` once the constant is gone.

**DECIDED (following C1): remove `ClientsType` outright.** Migrate or drop the
affected rows — a clients-scoped PAT *should* stop working once clients cease to
exist. Still worth pinning explicit values rather than `iota` so ordering never
becomes load-bearing by accident.

**Decision:** Remove outright; migrate or drop stale scope rows

**Notes:** Original 🔴 rating was based on a wrong assumption about persistence.

---

#### C3 🟡 Is `Serial` the same value as Bootstrap `ExternalID`, and which creates which?

**Blocks:** MG-09, MG-12 · **Ref:** [architecture.md §7.4](./architecture.md#74-two-identity-concepts-to-reconcile)

Both are "the device's external identity". Shipping both surfaces with an
undefined relationship guarantees divergence.

- **A. Creating a gateway creates the enrollment.** Device is primary.
- **B. Enrolling creates the gateway Device.** Bootstrap is primary.
- **C. Independent, linked by equal value**, enforced by validation.
- **D. Independent, unrelated.** No enforced relationship.

**DECIDED: D — independent, for now.**

Device `Serial` and Bootstrap `ExternalID` are **not** required to be the same
value. They may coincide, and often will, but nothing enforces it.

Rationale: they answer different questions. `Serial` identifies a physical device
within a domain; `ExternalID` identifies an enrollment used to fetch a config.
Forcing them equal couples the device model to the bootstrap flow for no
requirement that exists today.

**Consequences**

- MG-09 is **unblocked** — it no longer waits on this reconciliation and can
  freeze the device API independently.
- MG-12 drops the "enforce equal" acceptance criterion and its accompanying test.
- Gateway lookup by serial and by external ID are separate paths; neither implies
  the other.
- **Left open deliberately:** with no enforced link, nothing prevents a gateway
  whose enrollment `ExternalID` and device `Serial` differ confusingly. Operator
  documentation should recommend using the same value even though it is not
  required.

**Decision:** D — independent, no enforced relationship

**Notes:** "For the time being" — revisit if a flow appears that must resolve one
from the other.

---

#### C4 🟡 How is `Serial` uniqueness enforced?

**Blocks:** MG-09, MG-13

A plain attribute has no uniqueness guarantee. Atom has `alias` with a slug
constraint and (apparently) uniqueness semantics
(`001_initial.sql:104-113`).

- **A. Use Atom `alias`.** Uniqueness for free; constrained to a slug pattern
  (`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`) — check real serials fit.
- **B. Plain attribute + application-level uniqueness check.** Racy without a
  constraint.
- **C. Attribute + a unique index** added in Atom.

**DECIDED: unique per Atom tenant (= Magistrala domain), not globally.**

Two domains may each hold a meter with serial `ABC123`; within one domain the
serial identifies exactly one device.

**Consequences**

- Every serial lookup is tenant-scoped. A resolution path that forgets the tenant
  filter would cross-attribute data between customers — the failure is silent, so
  this needs an explicit test, not just care.
- MG-05's device-route resolution must scope by domain, which it already has from
  the topic (`m/<domain>/c/...`).
- MG-13's announce path enforces uniqueness within the announcing gateway's
  tenant.

**Still open — mechanism:** whether to use Atom's `alias`
(`001_initial.sql:104-113`), which gives uniqueness but constrains values to
`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`. **Verify real meter serials fit that
pattern** — many are uppercase alphanumeric, which normalises fine, but some
carry `/`, `.` or spaces, which do not. If they do not fit, a unique index on the
attribute is needed instead.

**Decision:** Unique per tenant. Mechanism pending the slug-pattern check.

**Notes:** Research task — collect real serial formats from target deployments.

---

#### C5 🟡 Accept reserving `d` as a first subtopic segment?

**Blocks:** MG-05

The topic grammar `m/<domain>/c/<channel>[/d/<device>][/<subtopic>]` is ambiguous
unless `d` is reserved as a leading subtopic segment. Breaking change for anyone
publishing to such a subtopic today.

- **A. Accept.** Cheap before 1.0, documented in release notes.
- **B. Use a less collision-prone marker** (`_d`, `dev`).
- **C. Different grammar** — device before channel, or a distinct topic prefix.

**Recommendation:** A. Single-letter markers match the existing `m` and `c`
convention, and the collision is unlikely in practice.

**Decision:**

**Notes:**

---

#### C6 🟡 Does the `Connection` proto need a device field?

**Blocks:** MG-09

`internal/proto/common/v1/common.proto:52` is
`{client_id, channel_id, domain_id, type}`. Are channel connections gateway-level
only, or can a sub-device be connected independently?

- **A. Gateway-level only.** The gateway connects; its devices inherit. Simpler,
  matches the physical reality.
- **B. Per-device connections.** A meter can be connected to a channel its gateway
  is not. More expressive, more state.

**Recommendation:** A. A sub-device has no independent transport, so an
independent connection has no physical meaning.

**Decision:**

**Notes:**

---

#### C7 ⚪ Restore the CLI binary for 1.0?

**Blocks:** MG-11 acceptance testing

`cmd/cli/main.go` was deleted in `16ba29cf4`, so `cli` is an unimported library.
New commands would ship unreachable.

- **A. Restore for 1.0.**
- **B. Leave it; CLI is not a 1.0 surface.**
- **C. Restore, but scoped** to devices/gateways/types only.

**Recommendation:** A or B explicitly. The current state — maintained code that
cannot be run — is the worst of both.

**Decision:**

**Notes:**

---

#### C8 ⚪ Resolve the channel-roles inconsistency before 1.0?

Clients, groups and domains have full role APIs; **channels have none**
(`pkg/sdk/channels.go` has no role methods). 1.0 freezes this.

- **A. Add channel roles.**
- **B. Ship as-is, document.**
- **C. Remove roles from other entities** for consistency.

**Recommendation:** B unless a channel-sharing requirement exists. Flagged because
1.0 makes it permanent.

**Decision:**

**Notes:**

---

### Section D — Provisioning

---

#### D1 🟡 Fleet delivery: rendered into config, or a separate endpoint?

**Blocks:** MG-12

A gateway with 500 meters produces a large encrypted config over a possibly
constrained link.

- **A. Rendered into the bootstrap config.** One fetch, consistent with the
  existing design.
- **B. Separate paginated fleet endpoint.** Scales; a second protocol for the
  agent.
- **C. Rendered up to a threshold**, endpoint beyond it. Two code paths.

**Recommendation:** Measure first. A if a realistic fleet fits comfortably;
otherwise B. Do not build C without evidence.

Expected max devices per gateway: ______

**Decision:**

**Notes:**

---

#### D2 🟡 Confirm: fleet snapshotted, not resolved at render time?

**Blocks:** MG-12

- **A. Snapshot**, refreshed via the existing `RefreshBootstrapBindings` path.
  Preserves "render never calls out"; fleet is stale until refresh.
- **B. Resolve at render.** Always current; breaks that invariant.

**Recommendation:** A, with attachment changes marking the config stale.

**Decision:**

**Notes:**

---

#### D3 🟡 Announce over the Bootstrap endpoint or MQTT?

**Blocks:** MG-13

- **A. Bootstrap HTTP endpoint**, reusing the existing challenge/response auth.
  Request/response, real error reporting.
- **B. Reserved MQTT topic.** No second protocol for the agent; but entity
  creation on the message path with no response channel.

**Recommendation:** A.

**Decision:**

**Notes:**

---

### Section E — Deferred but worth confirming

---

#### E1 ⚪ Telemetry validation against the device type?

**Blocks:** MG-10 documentation

Atom validates device **attributes** against the type schema
(`src/identity/repo.rs:641`). It does **not** validate message payloads — that
path never reaches Atom.

- **A. Attributes only.** Document clearly that telemetry is unvalidated.
- **B. Add ingest-time validation.** Puts a schema lookup on the hot path.
- **C. Validate asynchronously**, flagging non-conforming messages.

**Recommendation:** A for 1.0. This is the most likely user misunderstanding of
the feature, so the documentation is the deliverable.

**Decision:**

**Notes:**

---

#### E2 ⚪ Is the capability model expressive enough?

**Blocks:** MG-10 API freeze

Measurements + commands cover the watermeter case. Multi-channel devices, nested
structures and enumerated states may not fit.

**Action:** model two or three additional real device types against it before
freezing. Which ones? ______

**Decision:**

**Notes:**

---

#### E3 ⚪ `AddGroupMember` on an already-grouped entity — desired semantics?

**Blocks:** MG-03 documentation

Given single membership (A1), what *should* happen?

- **A. Fail** with a clear error; caller must remove first.
- **B. Move** silently to the new group.
- **C. Whatever Atom does**, documented faithfully.

**Recommendation:** A if Atom permits distinguishing it — a silent move is a
surprising way to lose a sharing grant.

**Decision:**

**Notes:**

---

#### E5 🟡 Agent-side device roster — where is it designed?

**Blocks:** end-to-end completeness of MG-12 and MG-13 ·
**Ref:** [architecture.md §10.6](./architecture.md#106-agent-side-device-roster--not-designed)

A3 gave the gateway ownership of local addressing, which makes the agent's local
store a real component that nothing specifies. A gateway must hold across
restarts and cloud outages: the roster (mine, by serial, with type — from
bootstrap), the local address map (serial → BLE/Modbus — from its own scan), and
the join.

- **A. Design it now**, as an `absmach/agent` PRD alongside MG-12/MG-13.
- **B. Defer** — treat the agent as a black box that "figures it out"; ship the
  cloud side and discover the gaps in integration.
- **C. Specify only the contract** — what bootstrap renders and what announce
  accepts — and leave storage entirely to the agent.

**Recommendation:** C now, A before field deployment. The contract is the part
that must be right in MG-12/MG-13; storage is genuinely the agent's business. But
"the agent figures it out" is how the cloud side ends up shipping something the
agent cannot consume.

Sub-questions worth answering with it:

- Does the gateway publish by **serial** or device ID? MG-05 accepts either in
  the `d` segment. If serial works as a route, the agent never stores
  cloud-assigned IDs — a real simplification. Depends on C4's mechanism.
- Cloud-outage behaviour: keep polling and buffer, or stop? Under B1 a pending
  device's data is rejected anyway, so buffering for unapproved devices is waste.
- Should the agent poll discovered-but-unapproved devices at all?

**Decision:**

**Notes:**

---

#### E4 ⚪ Command path — cloud → gateway → sensor

**Ref:** [architecture.md §10](./architecture.md#10-open-questions)

Not designed. The topic grammar accommodates it and device types declare commands,
but routing, addressing, ack semantics and timeout behaviour are unspecified.

- **A. Out of scope for 1.0.** Device types declare commands; nothing dispatches
  them.
- **B. In scope** — needs its own architecture pass and PRD set.

**Recommendation:** A, but decide now: if commands are a 1.0 requirement, the
downlink topic shape should be settled alongside MG-05 rather than bolted on
after the grammar is frozen.

**Decision:**

**Notes:**

---
