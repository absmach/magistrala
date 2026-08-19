# Magistrala GraphQL API

Magistrala uses Atom GraphQL for identity, domain, client, channel, group, role,
and policy management. The old HTTP APIs for `users`, `clients`, `channels`,
`groups`, and `domains` are no longer served by Magistrala as standalone
service endpoints, so their OpenAPI specifications were removed.

OpenAPI remains in `apidocs/openapi/` only for HTTP endpoints that Magistrala
still serves directly.

## Endpoint

Atom exposes GraphQL at:

```text
POST /graphql
```

In the default local stack this is available through the Atom service:

```text
http://localhost:8080/graphql
```

Authenticated requests use the same bearer token returned by Atom login.

The IDs in the examples are placeholders; replace them with IDs from your tenant before running the operations.

```bash
TOKEN=$(curl -s -X POST http://localhost:8080/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"identifier": "admin", "secret": "12345678", "kind": "password"}' \
  | jq -r .token)
```

Then send GraphQL requests with:

```bash
curl -s http://localhost:8080/graphql \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"query":"{ tenants { items { id name alias status } } }"}'
```

## Concept Mapping

| Magistrala concept | Atom primitive |
| --- | --- |
| Domain | Tenant |
| User | Entity with `kind = human` |
| Client | Entity with `kind = device` or `service` |
| Channel | Resource with `kind = "channel"` |
| Group | Group |
| Client key | Password credential or API key credential |
| Client certificate | Certificate credential |
| Role member set | Principal group or role assignment |
| Client-channel connection | Role assignment or direct policy |

## Common Operations

### Login

```graphql
mutation {
  login(input: {
    identifier: "admin"
    secret: "12345678"
    kind: "password"
  }) {
    token
    entityId
    sessionId
    expiresAt
  }
}
```

### Create A Domain

```graphql
mutation {
  createTenant(input: {
    name: "factory-a"
    alias: "factory-a"
  }) {
    id
    name
    alias
    status
  }
}
```

### List Domains

```graphql
query {
  tenants {
    items {
      id
      name
      alias
      status
    }
    total
  }
}
```

### Create A User

```graphql
mutation {
  createEntity(input: {
    profileId: "human-profile-id"
    name: "Alice"
    alias: "alice"
    tenantId: "domain-tenant-id"
    attributes: {
      email: "alice@example.com"
    }
  }) {
    id
    kind
    profileId
    name
    alias
    tenantId
  }
}
```

### Create A Client

```graphql
mutation {
  createEntity(input: {
    profileId: "client-profile-id"
    name: "meter-001"
    alias: "meter-001"
    tenantId: "domain-tenant-id"
  }) {
    id
    kind
    profileId
    name
    alias
    tenantId
  }
}
```

### Create A Channel

```graphql
mutation {
  createResource(input: {
    kind: "channel"
    name: "telemetry"
    alias: "telemetry"
    tenantId: "domain-tenant-id"
  }) {
    id
    kind
    name
    alias
    tenantId
  }
}
```

### Create A Group

```graphql
mutation {
  createGroup(input: {
    name: "field-devices"
    groupType: "object"
    tenantId: "domain-tenant-id"
  }) {
    id
    name
    groupType
    tenantId
  }
}
```

### Assign A Client To A Group

```graphql
mutation {
  addEntityToObjectGroup(
    entityId: "client-entity-id"
    objectGroupId: "group-id"
  ) {
    id
    objectGroupIds
  }
}
```

### Assign A Channel To A Group

```graphql
mutation {
  addResourceToObjectGroup(
    resourceId: "channel-resource-id"
    objectGroupId: "group-id"
  ) {
    id
    objectGroupIds
  }
}
```

### Check Publish Authorization

```graphql
mutation {
  authzCheck(input: {
    subjectId: "client-entity-id"
    action: "publish"
    resourceId: "channel-resource-id"
  }) {
    allowed
    reason
  }
}
```

## Source Of Truth

The Atom repository owns the GraphQL schema and generated introspection surface.
For details, see:

- Atom documentation: https://github.com/absmach/atom/tree/main/docs/content/docs/
- Magistrala mapping: https://github.com/absmach/atom/blob/main/docs/content/docs/magistrala-on-atom.mdx
- GraphQL implementation: https://github.com/absmach/atom/tree/main/src/graphql/
