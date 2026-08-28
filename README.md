<div align="center">

# Magistrala

### A Modern IoT Platform Framework for Scalable IoT

**Made with ❤ by [Abstract Machines](https://absmach.eu/)**

[![Build Status](https://github.com/absmach/magistrala/actions/workflows/build.yaml/badge.svg?branch=main)](https://github.com/absmach/magistrala/actions/workflows/build.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/absmach/magistrala)](https://goreportcard.com/report/github.com/absmach/magistrala)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/absmach/magistrala)
[![Check License Header](https://github.com/absmach/magistrala/actions/workflows/check-license.yaml/badge.svg?branch=main)](https://github.com/absmach/magistrala/actions/workflows/check-license.yaml)
[![Check Generated Files](https://github.com/absmach/magistrala/actions/workflows/check-generated-files.yaml/badge.svg?branch=main)](https://github.com/absmach/magistrala/actions/workflows/check-generated-files.yaml)
[![Coverage](https://codecov.io/gh/absmach/magistrala/graph/badge.svg?token=nPCEr5nW8S)](https://codecov.io/gh/absmach/magistrala)
[![License](https://img.shields.io/badge/license-Apache%20v2.0-blue.svg)](LICENSE)
[![Matrix](https://img.shields.io/matrix/supermq%3Amatrix.org?label=Chat&style=flat&logo=matrix&logoColor=white)](https://matrix.to/#/#supermq:matrix.org)

💡 [Get Started](#-quick-start)&ensp;•&ensp;🌐 [Website](https://www.absmach.eu/products/magistrala/)&ensp;•&ensp;📚 [Documentation](https://www.absmach.eu/docs/magistrala)&ensp;•&ensp;🤝 [Contributing](CONTRIBUTING.md)&ensp;•&ensp;💬 [Chat](https://matrix.to/#/#supermq:matrix.org)

</div>

## 🚀 Quick Start

Get Magistrala running locally in a few commands:

```bash
git clone https://github.com/absmach/magistrala.git
cd magistrala
make run_latest
```

This brings up the full stack — messaging, identity, and all core services — via Docker Compose. See [Installation](#installation) for what each step does, how certificates and secrets are generated, and how to point the stack at a different deployment.

## 💡 Getting Started

Once the stack is up, head to the [documentation](https://www.absmach.eu/docs/magistrala) to learn how to:

- Create a workspace and invite users
- Provision devices and connect them over MQTT, HTTP, CoAP, or WebSocket
- Create channels and start publishing and subscribing to messages
- Define access policies and roles for fine-grained control
- Build automations with the rules engine and alarms (Enterprise Edition)

## 🌍 Introduction

Magistrala is an open-source IoT platform built for engineers who need full control over their messaging, device management, and data pipelines.

It is built on top of [FluxMQ](https://github.com/absmach/fluxmq), a modern message broker designed for both messaging and event streams. Magistrala provides everything around it: identity, access control, device provisioning, data processing, and observability.

IoT systems usually involve brokers, databases, rule engines, and custom services. Magistrala does not pretend those pieces disappear. It provides a coherent framework for integrating them into a single system with a consistent model for identity, access control, messaging, and observability.

**What it is:**
- An event-driven IoT middleware platform
- A unified control plane for devices, users, and data
- A foundation for building scalable IoT systems

**What it is not:**
- Not just an MQTT broker
- Not a black-box SaaS
- Not tied to a single cloud or vendor

---

## 🚀 Key Benefits

- **A Coherent System, Not a Mess of Integrations**
  Build IoT systems from multiple components without ending up with fragmented security, messaging, and operations.

- **Event-Driven at the Core**
  Everything is built around events — enabling real-time processing, streaming, and scalable data flows.

- **Protocol-Native, Not Forced Abstractions**
  MQTT, HTTP, WebSocket, and CoAP are treated as first-class citizens, each with their own semantics.

- **Security Built Into the Model**
  Identity, authentication, and authorization are part of the system design — not bolted on later.

- **Flexible by Design**
  Start simple or build complex systems — without changing platforms or rewriting your architecture.

- **Runs Where You Need It**
  Cloud, edge, or hybrid — no vendor lock-in, no hidden dependencies.
---
## ✨ Features

Magistrala provides a complete set of building blocks for IoT systems — from device connectivity to data processing and observability — without forcing a rigid architecture.

### 🔐 Identity & Access

- Multi-tenant workspaces for isolating environments
- Users, roles, and organizational hierarchies
- Fine-grained access control (ABAC + RBAC)
- Mutual TLS (X.509) and JWT-based authentication
- Personal Access Tokens (PATs) with scoping and revocation

### 🔌 Connectivity

- Native support for MQTT, HTTP, WebSocket, and CoAP
- Consistent authentication and authorization across protocols
- Designed for both cloud services and constrained devices

### 📦 Device & Application Model

- Device provisioning and lifecycle management
- Channels for grouping and controlling message flow
- Application-level grouping and sharing of devices
- Simple but flexible communication model

### ⚙️ Processing & Automation

- Rules engine for message processing and routing (Enterprise Edition)
- Alarms and triggers for reacting to events (Enterprise Edition)
- Scheduled actions for time-based workflows
- Event-driven architecture as the foundation

### 📊 Observability

- Audit logs for tracking system activity (Enterprise Edition)
- Metrics and tracing via Prometheus and OpenTelemetry
- Built-in visibility into system behavior and data flows

### 🚀 Deployment & Operations

- Container-native (Docker, Kubernetes)
- Designed for cloud, edge, and hybrid deployments
- Works with external storage and processing systems
- Scales from small setups to production environments

### 🧑‍💻 Developer Experience

- CLI and SDKs for fast integration
- Straightforward APIs and concepts
- Documentation focused on getting you running quickly

---

## Installation

A fresh clone carries no generated secrets. Two sets have to exist before the
stack can start — certificates and keys the internal services authenticate
with, and the Atom service tokens each service presents to Atom. `make
run_latest` produces the first set itself but expects the second to be there
already, which is why the token step comes first in [Quick Start](#-quick-start).

### Certificates, broker secret and trace key

Generated by `make run_latest`, or on demand:

```bash
make check_certs
```

This creates whatever is missing and leaves anything already present alone:

| Path                                                        | What it is                                                                |
| ----------------------------------------------------------- | ------------------------------------------------------------------------- |
| `docker/ssl/certs/fluxmq-service-server.{crt,key}`          | Server certificate for FluxMQ's mTLS service listener                     |
| `docker/ssl/certs/re-fluxmq-client.{crt,key}`               | Client certificate whose URI SAN identifies the Rules Engine              |
| `docker/ssl/certs/timescale-writer-fluxmq-client.{crt,key}` | Client certificate whose URI SAN identifies the Timescale writer          |
| `docker/ssl/certs/postgres-writer-fluxmq-client.{crt,key}`  | Client certificate whose URI SAN identifies the Postgres writer           |
| `docker/ssl/certs/fluxmq-auth-fluxmq-client.{crt,key}`      | Client certificate whose URI SAN identifies the publish proxy            |
| `docker/fluxmq/secrets/re-current`                          | Rules Engine principal secret, from `MG_RE_BROKER_SECRET`                 |
| `docker/fluxmq/secrets/timescale-writer-current`            | Timescale writer secret, from `MG_TIMESCALE_WRITER_BROKER_SECRET`         |
| `docker/fluxmq/secrets/postgres-writer-current`             | Postgres writer secret, from `MG_POSTGRES_WRITER_BROKER_SECRET`           |
| `docker/fluxmq/secrets/fluxmq-auth-current`                 | Publish proxy secret, from `MG_FLUXMQ_BROKER_SECRET`                     |
| `docker/re/secrets/trace.key`                               | HMAC key the Rules Engine signs its loop-detection traces with            |

Internal services reach the broker as *local principals* rather than as ordinary
clients: each presents a client certificate whose URI SAN names it, plus a SASL
secret, and the broker grants it only what it needs — the Rules Engine consumes
`m`, republishes under it, and feeds the `writers` and `alarms` streams; the
writers only subscribe to `writers`; the publish proxy that serves the UI's
HTTP publish endpoint only publishes under `m.`. The principals are declared in
`docker/fluxmq/node{1,2,3}.yaml`, and adding a service means adding an entry
there alongside its certificate and secret.

Being a local principal is also what preserves a message's origin. The broker
stamps its own transport protocol and identity on anything published over a
connection it does not trust, so a message relayed to the writers over the plain
AMQP listener would be stored as `protocol: amqp` with the relaying service as
its publisher. A `service`-role principal on the mTLS listener may state the
origin instead, and the protocol the device actually published with survives to
the database.

The certificates are issued by the development CA committed at
`docker/ssl/certs/ca.crt`, so no extra setup is needed for a local run. The
generated material is gitignored.

The server certificate is issued for `fluxmq` and `fluxmq-node{1,2,3}`, which
covers both this Compose stack and a single-node deployment. Point any
`MG_*_BROKER_URL` at a host outside that set and the service fails its TLS
verification with `certificate is valid for ...`; add the name to
`FLUXMQ_SERVICE_SERVER_CERT_CONFIG` in `docker/ssl/Makefile` and reissue:

```bash
rm -f docker/ssl/certs/fluxmq-service-server.* \
  docker/ssl/certs/re-fluxmq-client.* \
  docker/ssl/certs/timescale-writer-fluxmq-client.* \
  docker/ssl/certs/postgres-writer-fluxmq-client.*
make -C docker/ssl fluxmq_service_certs
```

`make check_certs` skips certificates that already exist, so stale certificates
have to be removed rather than merely re-running the target.

Each local-principal secret must stay equal to the corresponding value in
`docker/.env`; a mismatch fails that service's broker authentication. After
changing one, re-run its target:

| Variable                            | Target                                    |
| ------------------------------------ | ----------------------------------------- |
| `MG_RE_BROKER_SECRET`               | `fluxmq_service_secret`                   |
| `MG_TIMESCALE_WRITER_BROKER_SECRET` | `timescale_writer_fluxmq_service_secret`  |
| `MG_POSTGRES_WRITER_BROKER_SECRET`  | `postgres_writer_fluxmq_service_secret`   |
| `MG_FLUXMQ_BROKER_SECRET`           | `fluxmq_auth_fluxmq_service_secret`       |

`trace.key` is created once and preserved on later runs — replacing it while
messages are in flight would invalidate the rule traces they already carry, so
delete it only deliberately. Every Rules Engine replica must read the same key.

Start the stack through `make run_latest` rather than calling `docker compose
up` directly. Compose creates a missing bind-mount source as an empty
*directory*, so bringing up `re` or `fluxmq` before these files exist leaves the
containers failing against a directory where they expect a key.

### Atom service tokens

Generated on demand: `make run_latest` runs `scripts/generate-atom-secrets.sh`
through the `docker/.env.tokens` prerequisite, so the first run mints them for
you. To do it explicitly:

```bash
make atom-secrets
```

The script writes two gitignored files from the same random material:
`docker/atom-bootstrap.yaml`, bind-mounted into the Atom container so Atom
hashes the credentials at first boot, and `docker/.env.tokens`, which supplies
one `MG_ATOM_TOKEN_*` per consumer — `MG_ATOM_TOKEN_FLUXMQ_AUTH`,
`MG_ATOM_TOKEN_FLUXMQ_NODE{1,2,3}`, `MG_ATOM_TOKEN_JOURNAL`,
`MG_ATOM_TOKEN_NOTIFICATIONS`, `MG_ATOM_TOKEN_RE`, `MG_ATOM_TOKEN_ALARMS`,
`MG_ATOM_TOKEN_REPORTS`, `MG_ATOM_TOKEN_TIMESCALE_READER`,
`MG_ATOM_TOKEN_POSTGRES_READER`, and `MG_ATOM_TOKEN_BOOTSTRAP`. No init
container is involved: services come up already holding credentials Atom has
accepted.

Rotate with `make atom-secrets-rotate`, which regenerates both files, restarts
Atom, and reminds you to restart the downstream services. Anything that resets
Atom's database also needs a rotation — the old tokens do not survive it.

---

## Usage

```bash
make cli
./build/cli login admin 12345678
./build/cli --token "$ATOM_ADMIN_TOKEN" workspaces list
```

The CLI reads the same `ATOM_URL` and `ATOM_SERVICE_TOKEN`/`ATOM_ADMIN_TOKEN`
variables the services use, so a shell that has sourced `docker/.env` needs no
extra endpoint or token flags. CLI requests use a separate
`MG_CLI_ATOM_TIMEOUT` setting and default to a 90s interactive timeout.

See [cli/README.md](cli/README.md) for the Atom GraphQL-backed command reference.

---

## 🧩 IoT Platform Framework

We call Magistrala a **framework**, not just a platform.

It is extremely flexible and lets you build systems the way you want — from simple prototypes to complex, large-scale deployments — without forcing you into rigid patterns.

At the same time, it avoids the typical complexity of many IoT platforms, where you need to learn an entirely new set of concepts before you can even get started.

Magistrala is built around a small number of main concepts:
- users
- devices
- channels
- messages
- policies

Most engineers are already familiar with these ideas, so you can start building immediately.

You can keep things simple:
- connect devices
- send messages
- store data

Or you can go deeper:
- define complex access control policies
- build event-driven pipelines
- integrate custom processing and automation

Magistrala scales with your needs — simple when you want it, powerful when you need it.

---

## Atom Integration Model

Magistrala uses **Atom** as the backend for identity, authorization, and the catalog.

Atom is the source of truth for:
- workspaces
- users
- devices
- channels
- groups
- roles
- access policies

Magistrala services such as rules, alarms, and reports remain Magistrala services, but they use Atom for identity and authorization.

Current Docker deployments use the Atom image configured by `ATOM_IMAGE` in `docker/.env`. For compatibility with the current Magistrala integration, the generated `MG_ATOM_TOKEN_*` service credentials are unscoped Atom access tokens. Scoped Atom access tokens should not be used for these service env vars until Magistrala stops using owner-wide Atom listing APIs such as `authorizedObjectIds` in service policy paths.

### Core Entity Mapping

| Magistrala concept | Atom concept                 | Meaning                                                            |
| ------------------- | ----------------------------- | -------------------------------------------------------------------- |
| Workspace           | Tenant                       | Isolation boundary for one organization, project, or environment   |
| User                | Entity with kind `human`     | A person who logs in and uses the UI/API                           |
| Device              | Entity with kind `device`    | A device or application that sends/receives data                   |
| Channel             | Resource with kind `channel` | A messaging/data path that devices can publish or subscribe to     |
| Group               | Group                        | A collection of users, devices, channels, or other grouped objects |

In simple terms:

```text
MG Workspace = Atom Tenant
MG User      = Atom Human Entity
MG Device    = Atom Device Entity
MG Channel   = Atom Channel Resource
MG Group     = Atom Group
```

### Actions, Permission Blocks, Roles, and Assignments

Atom access control has these basic parts:

| Atom word        | Simple meaning                | Example                                                   |
| ----------------- | ------------------------------ | ------------------------------------------------------------- |
| Action            | One permission verb           | `read`, `write`, `delete`, `role.manage`, `policy.manage` |
| Permission Block  | Where actions apply           | all channels in workspace `d1` can `read`, `publish`         |
| Role              | A bundle of permission blocks | `tenant-admin` bundles workspace, role, and member access    |
| Role Assignment   | Who gets a role               | give `user1` the `tenant-admin` role                      |

Read an assignment like this:

```text
Give <who> this <role>.
The role contains permission blocks that say where and what.
```

Example:

```text
Give user1 the tenant-admin role on workspace d1.
```

That means:

```text
user1 can use the tenant-admin permissions inside workspace d1.
```

### How MG Roles Work With Atom

MG UI shows actions such as:
- read
- update
- delete
- manage roles
- add/remove members
- publish
- subscribe

These are mapped to Atom actions:

| MG action                    | Atom action     |
| ----------------------------- | ----------------- |
| view/read                    | `read`          |
| create/update/edit/connect   | `write`         |
| delete/remove                | `delete`        |
| manage roles                 | `role.manage`   |
| add/remove members or access | `policy.manage` |
| channel publish              | `publish`       |
| channel subscribe            | `subscribe`     |

So when MG UI checks:

```text
Can user1 manage roles for client1?
```

Atom checks:

```text
Does user1 have role.manage on device1, or on the workspace that contains device1?
```

When MG UI checks:

```text
Can user1 add a member to channel1?
```

Atom checks:

```text
Does user1 have policy.manage on channel1, or on the workspace that contains channel1?
```

### Practical Rule

If a user is workspace admin, they usually receive a tenant-scoped role in Atom.

That tenant-scoped role can allow them to manage objects inside the workspace:
- devices
- channels
- groups
- rules
- alarms
- reports

For narrower access, create object-scoped roles. For example:

```text
Give user2 a reader role only on channel1.
```

Then user2 can read only that channel, not the whole workspace.

---

## License

Apache-2.0
