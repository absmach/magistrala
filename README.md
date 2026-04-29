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

[Guide](https://magistrala.absmach.eu/docs/) | [Contributing](CONTRIBUTING.md) | [Website](https://absmach.eu/) | [Chat](https://matrix.to/#/#supermq:matrix.org)
</div>

## Introduction 🌍

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

## 🧩 IoT Platform Framework

We call Magistrala a **framework**, not just a platform.

It is extremely flexible and lets you build systems the way you want — from simple prototypes to complex, large-scale deployments — without forcing you into rigid patterns.

At the same time, it avoids the typical complexity of many IoT platforms, where you need to learn an entirely new set of concepts before you can even get started.

Magistrala is built around a small number of core concepts:
- users
- clients (devices)
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

- Multi-tenant domains for isolating environments
- Users, roles, and organizational hierarchies
- Fine-grained access control (ABAC + RBAC)
- Mutual TLS (X.509) and JWT-based authentication
- Personal Access Tokens (PATs) with scoping and revocation

### 🔌 Connectivity

- Native support for MQTT, HTTP, WebSocket, and CoAP
- Consistent authentication and authorization across protocols
- Designed for both cloud services and constrained devices

### 📦 Device & Application Model

- Device (client) provisioning and lifecycle management
- Channels for grouping and controlling message flow
- Application-level grouping and sharing of clients
- Simple but flexible communication model

### ⚙️ Processing & Automation

- Rules engine for message processing and routing
- Alarms and triggers for reacting to events
- Scheduled actions for time-based workflows
- Event-driven architecture as the foundation

### 📊 Observability

- Audit logs for tracking system activity
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

```bash
git clone https://github.com/absmach/magistrala.git
cd magistrala
make run_latest
```

---

## Upgrade from v0.19.0 to v0.20.0

Before upgrading, back up the Domains, Rules Engine, Reports, Alarms, Auth, and SpiceDB databases.

v0.20.0 adds new domain admin actions for alarms and reports, and it requires existing rules and reports to have their built-in admin roles backfilled. The service database migrations run when the v0.20.0 services start, then the role backfill scripts must be run once.

For the default Docker Compose setup:

```bash
cd docker

docker compose up -d \
  spicedb-db spicedb-migrate spicedb \
  auth-db auth \
  domains-db domains \
  re-db re \
  reports-db reports \
  alarms-db alarms
```

Wait until the services are running. The `auth` service must start successfully because it loads the SpiceDB schema.

From the repository root, run the backfills:

```bash
go run ./scripts/re-backfill-roles/
go run ./scripts/reports-backfill-roles/
```

The scripts are idempotent. If they are interrupted, fix the issue and run them again.

Expected successful summaries:

```text
backfill finished processed=<number> skipped=<number> failed=0
```

After the backfills finish, verify that the services are still running:

```bash
cd docker
docker compose ps re reports alarms domains auth spicedb
```

For non-default deployments, make sure the database and SpiceDB connection settings used by the backfill scripts match your environment before running them.

---

## Usage

```bash
make cli
./build/cli health <service>
```

---

## License

Apache-2.0
