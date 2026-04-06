# Standalone packages

The `pkg` directory (the current directory) contains a set of standalone packages that can be imported and used by external applications. The packages are specifically meant for the development of Magistrala based back-end applications and implement common tasks needed by the programmatic operation of the Magistrala platform.

## Using the packages

Fetch any package directly from the module:

```bash
go get github.com/absmach/magistrala/pkg/authn
```

Then import it in your code:

```go
import "github.com/absmach/magistrala/pkg/authn"
```

## Package map (selected)

| Package | Purpose |
| --- | --- |
| `authn`, `authz`, `oauth2`, `policies` | Authentication and authorization helpers, middleware, and policy utilities. |
| `grpcclient` | TLS-aware gRPC client setup with health checks and timeouts. |
| `server` | HTTP, gRPC, and COAP server bootstrap utilities (TLS, graceful shutdown). |
| `postgres` | PostgreSQL connector with migrations helpers. |
| `events` | Event store client abstractions and subscriber utilities. |
| `prometheus` | Metrics collectors for request counts/latency. |
| `jaeger`, `tracing` | OpenTelemetry tracing configuration and instrumentation helpers. |
| `channels`, `clients`, `groups`, `domains`, `roles` | Shared types and helpers for core Magistrala domain services. |
| `messaging`, `connections`, `callout` | Messaging DTOs, connection types, and outbound callout helpers. |
| `sdk` | Go SDK for interacting with Magistrala services. |
| `errors` | Error wrappers with consistent error typing. |
| `uuid`, `ulid`, `sid` | ID generators. |
| `transformers`, `svcutil` | Generic data transformation and service utilities. |

For detailed package-level docs, run `go doc` on the desired package or browse the [source here](https://magistrala.absmach.eu/docs/.).
