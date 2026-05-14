# Magistrala to Atom Core Migration Plan

## Summary

Atom will become the source of truth for Magistrala core identity, catalog, and access-control data. Magistrala should stop running separate `domains`, `users`, `clients`, `groups`, and `channels` services and replace them with one small Atom-backed `core` proxy service.

This is a breaking cleanup migration. There is no temporary compatibility layer for old role fields. Existing Magistrala data backfill is out of scope for this phase and will be planned separately at the end.

## Current Status

The migration has moved the main domain/user/client/group/channel runtime path to a single Atom-backed Magistrala `core` service.

Completed runtime direction:

- `core` is the active service for:
  - domains
  - users
  - clients
  - groups
  - channels
- `core` stores and reads these objects through Atom instead of local Magistrala service databases.
- `core` exposes compatibility HTTP and gRPC endpoints so existing Magistrala-style clients can keep calling the familiar routes while the backing implementation is Atom.
- `core` has Atom authentication and Atom PDP authorization middleware for protected endpoints.
- `core` has callout middleware and event publishing middleware.
- Old split-service implementations for domains, users, clients, groups, channels, bootstrap, provision, and roles were moved out of the active Go package tree into `_legacy/`.
- Old split-service command entrypoints were moved out of the active command tree into `_legacy/cmd`.
- SpiceDB runtime wiring and active SpiceDB packages were removed.
- Magistrala roles are intentionally removed from the active public contract.
- Rules, reports, and alarms keep their service-owned databases but maintain Atom resource projections for searchable authorized listing.
- A small Astro demo UI was added under `demo-ui/` for quickly exercising the Atom-backed `core` API without Postman.

Latest cleanup completed:

- Removed old Magistrala `auth`, `auth-db`, and `auth-redis` services from the default Docker Compose stack.
- Removed `auth` from `core` and other `depends_on` references.
- Removed old stopped Docker containers:
  - `magistrala-auth`
  - `magistrala-auth-db`
  - `magistrala-auth-redis`
- Removed `auth` from the Makefile build service list.
- Removed `auth` from Makefile API-test service list.
- Removed the old Auth API Schemathesis workflow step.
- Removed auth gRPC certificate generation from the default Makefile certificate target.
- Verified Docker Compose no longer lists `auth`, `auth-db`, or `auth-redis`.
- Verified:
  - `go test ./core ./internal/atom`

Important distinction:

- `fluxmq-auth` still exists in Docker Compose. This is FluxMQ broker authorization, not the removed Magistrala `auth` service.

Known remaining old-auth references:

- Several non-core services still contain `MG_AUTH_GRPC_*` environment wiring and Go imports through shared `pkg/authn`, `pkg/authz`, or old auth client packages.
- Those references are the next migration/fix area after removing the runtime `magistrala-auth` container.
- Removing the source-level `auth/` package immediately will currently break broad compilation because services such as journal, certs, readers, alarms, reports, and SDK tests still import old auth-related packages.

## Target Architecture

Run one Magistrala `core` service instead of five services:

- Remove separate runtime services:
  - `cmd/domains`
  - `cmd/users`
  - `cmd/clients`
  - `cmd/groups`
  - `cmd/channels`
- Add:
  - `cmd/core`
  - `core/`
- `core` exposes the required HTTP/gRPC endpoints for domains, users, clients, groups, and channels.
- `core` has no local Postgres database for these objects.
- `core` calls Atom for storage, credentials, search/listing, and authorization.
- Other Magistrala services call `core` or Atom instead of five separate core services.

## Object Mapping

| Magistrala object | Atom object | Ownership |
| --- | --- | --- |
| Domain | Tenant | Atom stores name, route, status, tags, metadata, timestamps. |
| User | Entity | Atom stores identity metadata and credentials. |
| Client | Entity | Atom stores client identity metadata and client credentials. |
| Group | Group | Atom stores group metadata; hierarchy fields stay in attributes until Atom has first-class hierarchy support. |
| Channel | Resource `kind=channel` | Atom stores searchable channel catalog data and access policy target. |
| Rule | Resource `kind=rule` projection | Magistrala keeps full rule config; Atom stores searchable authorized projection. |
| Alarm | Resource `kind=alarm` projection | Magistrala keeps full alarm record; Atom stores searchable authorized projection. |
| Report | Resource `kind=report` projection | Magistrala keeps full report config/template; Atom stores searchable authorized projection. |

## Roles And SpiceDB Removal

Remove Magistrala `pkg/roles` completely. It was needed because SpiceDB could do authorization checks but did not provide good searchable authorized listing. Atom now owns both authorization and searchable access-controlled listing.

Remove:

- `pkg/roles`
- `roles.RoleManager` from service interfaces
- `roles.Repository` from repositories
- `[]roles.RoleProvision` return values from service methods
- built-in role provisioning code
- role tables/migrations from core service databases
- role event encode/decode paths
- role API endpoints
- role mocks
- role-related tests
- `role_id`, `role_name`, `roles`, `actions`, `access_type`, `access_provider_id`, `access_provider_role_id`, `access_provider_role_name`, and `access_provider_role_actions` from public structs and API responses

Remove SpiceDB-related code and deployment:

- `pkg/policies/spicedb`
- `pkg/spicedb`
- SpiceDB schema decoder usage
- SpiceDB config/env values
- SpiceDB Docker/Compose services
- SpiceDB permissions files when they are only used for role/policy provisioning
- duplicated local policy writes that existed only to support listing/search

Replace with Atom:

- Atom capabilities
- Atom roles only if Atom-side role grouping is useful
- Atom policy bindings
- Atom PDP checks
- Atom list/search APIs with authorization applied

## Public API Cleanup

This migration intentionally removes old role compatibility fields from public responses. API clients should not receive or send Magistrala role fields after the core migration.

Allowed response data should be object data only. If a caller needs access information, expose Atom-native concepts through new explicit endpoints such as:

- effective capabilities
- access checks
- authorized list/search

Do not preserve old role-shaped fields.

## Completed

- Added `internal/atom` package with:
  - config loading for `ATOM_*` flags
  - HTTP client
  - projection types
  - tenant/entity/group/resource mapping helpers
  - unit tests
- Added Atom projection decorators for:
  - domains
  - users
  - clients
  - groups
  - channels
  - rules
  - reports
  - alarms create/update/delete
- Added initial `core` service skeleton:
  - `core/` package
  - `cmd/core`
  - HTTP health/version endpoints
  - typed Atom-backed HTTP handlers for domains, users, clients, groups, and channels
  - Atom-backed compatibility gRPC services for domains, users, clients, groups, and channels
  - gRPC server startup with reflection
  - Atom config validation through `internal/atom`
- Removed raw Atom reverse proxy routing from `core`; public HTTP routes now translate Magistrala shapes explicitly.
- Client gRPC authentication now resolves shareable client keys from Atom device metadata instead of requiring a private credential introspection endpoint.
- Wired rule, report, and alarm Atom projection decorators into their service binaries. These services keep their own databases and maintain Atom resource projections for listing.
- Added `core` to the main Docker Compose stack and repointed shared domain/user/client/group/channel HTTP and gRPC client URLs in `docker/.env` to `core`.
- Repointed provisioning URLs for users, clients, and channels to `core`.
- Repointed API test targets for users, clients, domains, channels, and groups to the single `core` HTTP port.
- Moved the old `domains`, `users`, `clients`, `groups`, `channels` services and their DB/cache containers behind a `legacy-core` Docker Compose profile so they are not part of the default runtime stack.
- Repointed `journal` runtime dependency from the old `domains` service to `core`.
- Repointed nginx domain/user/client/group/channel HTTP routes, `/health`, and `/metrics` to `core`.
- Repointed `make run_addons` bootstrap startup from `domains` to `core`.
- Repointed GitHub API-test workflow core-object URLs to `localhost:9000`.
- Updated GitHub API-test path filters so core-object API tests run when `core`, `cmd/core`, or `internal/atom` changes.
- Updated GitHub API-test path filters so rule/alarm/report API tests run when their service, command, or Atom projection code changes.
- Repointed Prometheus addon default scrape target from old users/clients services to `magistrala-core:9000`.
- Updated service README endpoint examples so remaining domain/client/channel/group gRPC references use `core:7000`.
- Added Atom-backed `POST /users/tokens/issue` compatibility in `core`.
- Added compatibility handlers for token refresh/revoke/list refresh tokens in `core`.
- Added explicit `501` handlers in `core` for removed role endpoints and unsupported invitation/email-verification/password-reset flows.
- Added Atom-backed user email/profile-picture updates in `core`.
- Added `/metrics` to `core` so nginx and Prometheus can target the single service.
- Deleted old split service command entrypoints:
  - `cmd/domains`
  - `cmd/users`
  - `cmd/clients`
  - `cmd/groups`
  - `cmd/channels`
- Updated CI test path filters to use `core`/`cmd/core` instead of deleted split service command paths.
- Added Atom-write mode constructors for rules and reports that skip legacy Magistrala role provisioning.
- Rules and reports now avoid connecting to SpiceDB at startup.
- Removed rule/report role-management HTTP route registration from active rule/report handlers.
- Removed role provisioning and role-manager middleware/event wrappers from the active rules and reports service chains.
- Removed `[]roles.RoleProvision` from the active rules service create contract.
- Removed rule role provisioning from the downstream rule event consumer used by alarms.
- Removed the remaining role-manager consumer dependency from the downstream rule event decoder.
- Removed built-in role setup from rule/report service constructors and command startup.
- Removed remaining rule/report repository role SQL paths:
  - `RetrieveByIDWithRoles`
  - `ListUserRules`
  - `ListUserReportsConfig`
  - rule/report role-table migrations
- Removed rule/report role/access fields from internal DTO mapping.
- Regenerated rule/report mocks after removing the stale repository role methods.
- Removed alarm user-list role SQL path and made alarm listing use the normal service repository path gated by Atom authorization.
- Regenerated alarm mocks after removing `ListUserAlarms`.
- Added shared Atom PDP authorization helper in `internal/atom`.
- Added Atom authorization middleware constructors for rules, reports, and alarms.
- Rules, reports, and alarms now skip authz/domain authorization gRPC clients and call Atom PDP directly.
- Added Atom-backed policy evaluator for `auth`.
- `cmd/auth` now skips SpiceDB startup and uses Atom PDP.
- Removed SpiceDB startup/import fallback from active `cmd/auth`, `cmd/re`, and `cmd/reports`.
- Removed obsolete SpiceDB env fields from active rule/report command config.
- Removed SpiceDB, SpiceDB migration, and SpiceDB Postgres containers from Docker Compose.
- Removed default Docker Compose dependencies from `auth`, rules, alarms, and reports to SpiceDB/SpiceDB migration.
- Added `ATOM_*` environment wiring to active `auth`, rules, alarms, and reports Docker Compose services.
- Removed unused SpiceDB environment variables and schema mounts from active `auth`, rules, alarms, and reports Docker Compose services.
- Added Atom-backed `policies.Service` implementation for Bootstrap's authorized client listing path.
- `cmd/bootstrap` now uses Atom for policy listing and no longer imports or initializes SpiceDB.
- Deleted legacy SpiceDB packages:
  - `pkg/policies/spicedb`
  - `pkg/spicedb`
- Removed SpiceDB environment variables from `docker/.env`.
- Removed remaining SpiceDB references from Go code, `go.mod`, `go.sum`, and default Docker runtime files.
- Removed rule/report hidden role fields from SDK DTOs.
- Known Atom model gap: group update/status/hierarchy routes return `501` until Atom groups support attributes/status/update semantics.
- Updated `Makefile` `SERVICES` to build `core` instead of the five old core service binaries.
- Removed old role/access fields from public JSON output for:
  - domains
  - clients
  - channels
  - groups
  - rules
  - reports
- Updated SDK response models and tests so removed role/access fields are not treated as public JSON contract.
- Made `core` self-contained for domain/user/client/group/channel request and response DTOs; it no longer imports the old split service packages for public shapes.
- Replaced shared gRPC client setup to use generated protobuf clients directly instead of importing old split service gRPC wrappers.
- Replaced the domain authorization status interface with a small package-local status type so active authorization code no longer imports the old `domains` package.
- Removed reports/rules migration dependency on old domain Postgres migrations; domain storage now belongs to Atom.
- Removed old domain event-store subscriptions from active rules, reports, and alarms command startup.
- Moved legacy split-service implementations out of the active Go package tree into `_legacy/oldservices`:
  - domains
  - users
  - clients
  - groups
  - channels
  - bootstrap
  - provision
  - `pkg/roles`
- Moved legacy command entrypoints out of the active Go package tree into `_legacy/cmd`:
  - `cmd/bootstrap`
  - `cmd/provision`
  - `cmd/cli`
- Gated old CLI/SDK tests that depend on removed split services behind the `oldservices` build tag.
- Added Atom-authenticated HTTP guards to active `core` domain/user/client/group/channel routes.
- Added Atom PDP checks for protected `core` HTTP endpoints using the caller Bearer token.
- Added `core` callout execution before protected operations through `MG_CORE_CALLOUT_*`.
- Added `core` event publishing after successful protected operations through `MG_ES_URL`.
- Kept public exceptions for health/version/metrics, token issue/refresh/revoke/list, and explicitly unsupported compatibility routes.
- Added this migration plan document.
- Verified current repo with:
  - `go test ./pkg/sdk`
  - `go test ./core ./internal/atom`
  - `go test ./internal/atom ./domains ./users ./clients ./channels ./groups ./re ./reports ./alarms`
  - `go test ./...`
  - `go test ./core ./cmd/core ./internal/atom`
  - `make core`
  - `go test ./domains ./clients ./channels ./groups ./re ./reports ./core`
  - `go test ./core ./cmd/core ./internal/atom`
  - `go test ./alarms ./alarms/middleware ./alarms/consumer`
  - `go test ./cmd/re ./cmd/reports ./cmd/alarms ./re ./reports ./alarms`
  - `go test ./re ./reports ./cmd/re ./cmd/reports`
  - `go test ./re/api ./reports/api ./re ./reports`
  - `go test ./...`
  - `go test ./internal/atom ./re/middleware ./reports/middleware ./alarms/middleware ./cmd/re ./cmd/reports ./cmd/alarms`
  - `go test ./re ./reports ./alarms ./cmd/re ./cmd/reports ./cmd/alarms ./internal/atom`
  - `go test ./...`
  - `go test ./internal/atom ./auth ./cmd/auth`
  - default `docker compose --env-file docker/.env -f docker/docker-compose.yaml config` contains no SpiceDB services
  - `go test ./cmd/auth ./cmd/re ./cmd/reports ./cmd/alarms ./internal/atom`
  - `go test ./...`
  - `docker compose --env-file docker/.env -f docker/docker-compose.yaml config`
  - `go test ./re ./re/api ./re/events ./re/middleware ./reports ./reports/events ./reports/middleware ./cmd/re ./cmd/reports`
  - `go test ./pkg/re/events/consumer ./alarms ./cmd/alarms`
  - `go test ./re ./re/postgres ./re/api ./re/events ./re/middleware ./reports ./reports/postgres ./reports/api ./reports/events ./reports/middleware`
  - `go test ./alarms ./alarms/postgres ./alarms/middleware ./cmd/alarms`
  - `go test ./cmd/auth ./cmd/re ./cmd/reports ./auth ./re ./reports ./internal/atom`
  - `go test ./...`
  - `go test ./internal/atom ./bootstrap ./bootstrap/events/producer ./cmd/bootstrap`
  - `go test ./...`
  - `go mod tidy`
  - `go test ./...`
  - `go test ./pkg/sdk ./re ./reports`
  - `go test ./pkg/sdk`
  - `go test ./core`
  - `go test ./core ./api/http ./pkg/grpcclient ./pkg/authz/authsvc ./pkg/domains ./pkg/domains/grpcclient`
  - `go list ./...`
  - `go test ./...`
  - `go test ./core ./internal/atom ./cmd/core`
  - `go test ./...`
  - `go test ./...`
  - `go test ./re ./re/middleware ./cmd/re ./reports ./reports/middleware ./cmd/reports`
  - `go test ./...`
  - `go test ./pkg/re/events/consumer ./alarms ./cmd/alarms`

## Next Phases

### Phase 1: Create Core Service Skeleton

Status: completed for the initial buildable skeleton, typed HTTP Atom adapters, and first-pass compatibility gRPC adapters for internal Magistrala callers.

- Add `cmd/core`.
- Add `core/` package.
- Start one HTTP server and one gRPC server.
- Load Atom config through `internal/atom`.
- Register grouped HTTP routes for:
  - domains
  - users
  - clients
  - groups
  - channels
- Register compatibility gRPC services needed by other Magistrala services, but implement them through Atom/core logic.

### Phase 2: Move Core Writes To Atom

Status: started. HTTP writes for domains/users/clients/channels go to Atom. Client shareable keys are stored on Atom device metadata for MQTT/reader key resolution. Group create/delete go to Atom; group update/status/hierarchy needs Atom model support.

- Domain create/update/status/delete writes Atom tenants.
- User create/update/status/delete writes Atom entities and credentials.
- Client create/update/status/delete writes Atom entities and credentials.
- Group create/update/status/delete writes Atom groups.
- Channel create/update/status/delete writes Atom resources.
- Remove role provisioning from all these flows.
- Remove local Postgres writes for these objects.

### Phase 3: Move Core Reads To Atom

Status: started. HTTP reads/lists for domains/users/clients/groups/channels come from Atom. gRPC retrieve/list helpers also read from Atom.

- Domain list/search/read comes from Atom tenants.
- User list/search/read comes from Atom entities.
- Client list/search/read comes from Atom entities.
- Group list/search/read comes from Atom groups.
- Channel list/search/read comes from Atom resources.
- Remove Redis caches that only support old core lookup flows.

### Phase 4: Replace Authorization

- Status: mostly complete for active runtime. Public role/access response fields are hidden from JSON. Auth, rules, reports, alarms, and bootstrap now call Atom-backed authorization/listing paths and no longer initialize SpiceDB. Default Docker Compose no longer contains SpiceDB services or env. Legacy SpiceDB packages are deleted. Active rules and reports no longer provision Magistrala roles, no longer compute built-in roles at startup, and no longer wrap role-manager middleware/events. Rule/report/alarm repository role-list SQL paths and stale generated mocks have been removed. Remaining work is to move the legacy core entity packages and SDK role endpoint helpers off `pkg/roles`, then delete `pkg/roles`.

- Replace SpiceDB policy checks with Atom PDP checks.
- Replace authorization middleware internals with Atom client calls.
- Remove `pkg/policies/spicedb`. Done.
- Remove duplicated local policy writes. Started; active rule/report/alarm paths are done.
- Remove permissions schema decoding tied to SpiceDB/roles. SpiceDB decoder is deleted; role permission files still need deployment/doc review.

### Phase 5: Simplify Rules, Alarms, And Reports

Status: started. Rule and report create/update/delete projections are wired into runtime commands. In Atom-write mode, rule/report startup no longer requires SpiceDB. Rule/report create paths no longer provision Magistrala roles, active rule/report middleware and event wrappers no longer embed role-manager behavior, and command startup no longer builds legacy built-in role definitions. Rule/report repository role-list SQL paths are removed. Alarm create/update/delete projections are wired into runtime commands after changing alarm creation to return the inserted alarm for projection, and alarm listing no longer depends on rule/domain role tables.

- Keep full rule/alarm/report data in their service databases.
- Store searchable projection in Atom resources.
- Create/update/delete still happens through the owning service.
- List/search first queries Atom for authorized IDs, then hydrates from the service DB.
- Remove legacy role HTTP routes from rule/report APIs. Rule and report route registration is complete; OpenAPI cleanup still needs review.

### Phase 6: Remove Old Services

Status: started. Build list, shared Docker client URLs, provision URLs, API test targets, default Compose runtime, nginx routes, and CI filters now target `core`. Old split service command entrypoints are deleted. Old service packages still exist as DTO/API/test surfaces while downstream imports and role/SpiceDB dependencies are removed safely.

- Delete old runtime commands:
  - `cmd/domains`
  - `cmd/users`
  - `cmd/clients`
  - `cmd/groups`
  - `cmd/channels`
- Remove old service packages or keep only API DTOs that are still used by `core`.
- Remove old Postgres repositories and migrations for core data.
- Remove old mocks generated only for deleted interfaces.
- Update Makefile service lists.
- Update Docker Compose and deployment manifests to run `core` instead of five services. Default Docker Compose runtime is complete; other deployment manifests still need review.
- Update dependent env vars so readers, writers, FluxMQ, notifications, rules, alarms, reports, certs, journal, bootstrap, and provision point to `core`. Docker `.env` is complete; other environment templates still need review.

### Phase 7: Final Backfill Plan

Existing Magistrala-to-Atom backfill is planned later. That plan must include:

- idempotent import
- count comparison
- referential integrity checks
- policy/capability migration
- rollback procedure
- cutover validation

## Test Plan

Unit tests:

- Atom config parsing.
- Atom HTTP client behavior.
- Atom mapping helpers.
- Core adapter mapping for all public request/response shapes.
- No role fields in public response structs.

Functional tests:

- Core HTTP endpoints create/read/update/delete through Atom.
- Core gRPC endpoints satisfy downstream service calls.
- Core list/search uses Atom authorization.
- No local Postgres writes for domains/users/clients/groups/channels.

Integration tests:

- Run Magistrala core + Atom + required remaining services.
- Create domain, user, client, group, and channel and verify Atom state.
- Verify Atom PDP allow/deny behavior.
- Verify rules/reports/alarms projections.
- Verify readers/writers/FluxMQ can still resolve clients/channels/domains through core.

End-to-end tests:

- Login/create user through Atom-backed core.
- Create tenant/domain.
- Create client and channel.
- Connect client/channel.
- List/search authorized core objects.
- Create/update/delete rule/report/alarm and list through Atom-backed authorized projection.

Removal tests:

- `go test ./...` passes after deleting roles and SpiceDB.
- No imports remain for `pkg/roles`, `pkg/policies/spicedb`, or `pkg/spicedb`.
- No deployment references remain for SpiceDB or the five old core services.
