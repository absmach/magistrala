# Bootstrap Profile and Template Plan

## Goal

Decouple the Bootstrap service from direct clients and channels ownership and move it toward a user-managed device profile and configuration template model.

Bootstrap should own device onboarding configuration, external bootstrap credentials, template rendering, and resource references. Clients, channels, certificates, and connection lifecycle should remain owned by their respective services or by a provisioning layer.

Users should be able to:

1. Create or upload multiple device profiles.
2. Use each profile as a configuration template.
3. Join one or more devices/enrollments to a profile.
4. Link actual clients, channels, certificates, or other resources to the named slots required by the template.
5. Bootstrap a device by rendering the selected profile with the device context and linked resources.

## Current Problem

Bootstrap is tightly coupled to clients and channels:

- It creates or fetches clients during config creation.
- It validates and stores channel details.
- It connects and disconnects clients to channels.
- It mirrors clients and channels state through local tables and event consumers.
- It renders bootstrap responses from a config object that mixes identity, credentials, channels, certs, and raw content.

This works for the current flow, but it makes Bootstrap less flexible for device profiles, static inventory, non-MQTT devices, imported resources, or alternative provisioning backends.

## Target Boundary

Bootstrap owns:

- Device profiles.
- Bootstrap enrollments/configs.
- External IDs and external keys.
- Template rendering.
- Bootstrap state.
- Resource binding snapshots.

Clients service owns:

- Client identity.
- Client credentials.
- Client lifecycle.

Channels service owns:

- Channels.
- Client-channel connections.
- Publish/subscribe authorization.

Provisioning owns:

- Creating, finding, or linking clients/channels/certs from profile intent.
- Refreshing Bootstrap binding snapshots when resource data changes.

## Core Concepts

### Device Profile

A user-created or uploaded configuration template for a type or class of device.

The profile is not a device and does not directly own clients or channels. It declares what the rendered configuration should look like and which named resource slots must be provided before a device can use it.

Example fields:

- `id`
- `domain_id`
- `name`
- `description`
- `template_format`
- `content_template`
- `defaults`
- `binding_slots`
- `version`

Example:

```json
{
  "name": "edge-gateway-v2",
  "description": "Gateway profile for MQTT telemetry and command handling.",
  "template_format": "go-template",
  "defaults": {
    "mqtt_url": "tcp://localhost:1883",
    "heartbeat_interval": "30s"
  },
  "binding_slots": [
    {
      "name": "mqtt_client",
      "type": "client",
      "required": true,
      "fields": ["id", "secret"]
    },
    {
      "name": "telemetry",
      "type": "channel",
      "required": true,
      "fields": ["id", "name", "topic"]
    },
    {
      "name": "commands",
      "type": "channel",
      "required": false,
      "fields": ["id", "name", "topic"]
    }
  ]
}
```

The profile can be created through an API request or uploaded as a JSON/YAML/TOML document containing the template and binding slot declarations.

### Enrollment

A per-device or per-fleet bootstrap config instance joined to a profile.

Example fields:

- `id`
- `domain_id`
- `profile_id`
- `external_id`
- `external_key`
- `state`
- `render_context`

The `render_context` stores per-device values such as serial number, firmware channel, site, asset tag, or user-defined variables.

Example:

```json
{
  "profile_id": "profile_gateway_v2",
  "external_id": "gw-001",
  "external_key": "generated-or-user-provided-key",
  "render_context": {
    "serial": "GW001",
    "site": "belgrade-lab",
    "firmware_channel": "stable"
  }
}
```

### Binding Slot

A named placeholder declared by a profile and referenced by the template.

Examples:

- `mqtt_client`
- `telemetry`
- `commands`
- `device_cert`

The profile declares the slot names and expected resource type. Device enrollments then bind actual resources to those slots.

### Resource Binding Snapshot

A Bootstrap-owned snapshot of external resources needed for rendering.

Example fields:

- `config_id`
- `slot`
- `type`
- `resource_id`
- `snapshot`
- `secret_snapshot`
- `updated_at`

Example:

```json
{
  "slot": "telemetry",
  "type": "channel",
  "resource_id": "ch_123",
  "snapshot": {
    "id": "ch_123",
    "name": "Telemetry",
    "topic": "devices/gw-001/telemetry"
  }
}
```

Secrets should be stored encrypted separately or in an encrypted snapshot.

Binding is explicit. A user, provisioning workflow, or API client maps profile slots to actual resources:

```json
{
  "bindings": [
    {
      "slot": "mqtt_client",
      "type": "client",
      "resource_id": "cli_123"
    },
    {
      "slot": "telemetry",
      "type": "channel",
      "resource_id": "ch_123"
    },
    {
      "slot": "commands",
      "type": "channel",
      "resource_id": "ch_456"
    }
  ]
}
```

## Template Filling Model

Rendering must be deterministic and must only read Bootstrap-owned data.

Allowed during render:

- Device profile.
- Enrollment/config row.
- Profile defaults.
- Enrollment render context.
- Stored resource binding snapshots.
- Stored encrypted secret snapshots after decryption.

Not allowed during render:

- SDK calls.
- gRPC calls.
- Direct clients/channels queries.
- Event store reads.
- Implicit provisioning.

This keeps device bootstrap fast, testable, and independent from clients/channels availability.

## Render Context

Use a typed render context rather than exposing database rows directly to templates.

```go
type RenderContext struct {
    Device   DeviceContext
    Vars     map[string]any
    Bindings map[string]BindingContext
}

type DeviceContext struct {
    ID         string
    ExternalID string
    DomainID   string
}

type BindingContext struct {
    Type     string
    ID       string
    Snapshot map[string]any
    Secret   map[string]any
}
```

Example template:

```toml
[agent.mqtt]
url = "{{ .Vars.mqtt_url }}"
client_id = "{{ (index .Bindings "mqtt_client").ID }}"
username = "{{ (index .Bindings "mqtt_client").ID }}"
password = "{{ index (index .Bindings "mqtt_client").Secret "secret" }}"

[export.telemetry]
channel = "{{ (index .Bindings "telemetry").ID }}"
topic = "{{ index (index .Bindings "telemetry").Snapshot "topic" }}"

[agent.heartbeat]
interval = "{{ .Vars.heartbeat_interval }}"
```

## Provisioning Flow

Provisioning may call clients/channels/certs. Rendering may not.

Suggested flow:

1. User creates or uploads a device profile template.
2. User creates a device enrollment and selects the profile.
3. User or automation links actual clients/channels/certs to the profile's binding slots.
4. Bootstrap validates that all required slots are bound.
5. Bootstrap stores binding snapshots for deterministic rendering.
6. Device calls bootstrap with `external_id` and `external_key`.
7. Bootstrap validates the key, loads the stored profile/enrollment/bindings, renders the template, and returns the config.

Provisioning can be optional:

- Manual mode: user links existing clients/channels to slots.
- Assisted mode: Bootstrap calls a resolver/provisioner to create or find resources and then stores snapshots.
- Hybrid mode: user binds some slots manually and lets the resolver fill the rest.

## Binding Flow

The profile defines required slots, but the enrollment decides which actual resources fill those slots.

Manual binding example:

```text
Profile: edge-gateway-v2
Slots:
- mqtt_client -> client
- telemetry -> channel
- commands -> channel

Enrollment: gw-001
Bindings:
- mqtt_client -> client cli_123
- telemetry -> channel ch_123
- commands -> channel ch_456
```

The binding operation should:

1. Load the enrollment and its profile.
2. Verify the slot exists in the profile.
3. Verify the resource type matches the slot type.
4. Validate the resource through its owning service at binding time.
5. Store a snapshot of only the fields needed for rendering.
6. Mark the enrollment as renderable only when all required slots are bound.

Binding-time validation may call clients/channels. Render-time must not.

Pseudo-code:

```go
func (s service) BindResources(ctx context.Context, id string, req BindResourcesRequest) error {
    profile, enrollment := s.loadProfileAndEnrollment(ctx, id)

    bindings, err := s.resolver.Resolve(ctx, ResolveRequest{
        Profile:    profile,
        Enrollment: enrollment,
        Requested:  req.Bindings,
    })
    if err != nil {
        return err
    }

    if err := profile.ValidateRequiredSlots(bindings); err != nil {
        return err
    }

    return s.bindingStore.Save(ctx, id, bindings)
}

func (s service) Bootstrap(ctx context.Context, externalID, externalKey string) ([]byte, error) {
    enrollment := s.authEnrollment(ctx, externalID, externalKey)
    profile := s.profileStore.Retrieve(ctx, enrollment.ProfileID)
    bindings := s.bindingStore.Retrieve(ctx, enrollment.ID)

    return s.renderer.Render(profile, enrollment, bindings)
}
```

## Interfaces

Introduce interfaces around the new boundary.

```go
type BindingResolver interface {
    Resolve(ctx context.Context, req ResolveRequest) ([]BindingSnapshot, error)
}

type BindingStore interface {
    Save(ctx context.Context, configID string, bindings []BindingSnapshot) error
    Retrieve(ctx context.Context, configID string) ([]BindingSnapshot, error)
}

type ProfileStore interface {
    Save(ctx context.Context, profile Profile) (Profile, error)
    Retrieve(ctx context.Context, domainID, profileID string) (Profile, error)
}

type EnrollmentStore interface {
    Save(ctx context.Context, enrollment Enrollment) (Enrollment, error)
    RetrieveByExternalID(ctx context.Context, externalID string) (Enrollment, error)
}

type Renderer interface {
    Render(profile Profile, enrollment Enrollment, bindings []BindingSnapshot) ([]byte, error)
}
```

Adapters can implement resource resolution for current Magistrala clients/channels first, then later support other backends.

## API Shape

Initial endpoints can be additive:

- `POST /{domainID}/bootstrap/profiles`
- `POST /{domainID}/bootstrap/profiles/upload`
- `GET /{domainID}/bootstrap/profiles/{profileID}`
- `PATCH /{domainID}/bootstrap/profiles/{profileID}`
- `GET /{domainID}/bootstrap/profiles/{profileID}/slots`
- `POST /{domainID}/bootstrap/enrollments`
- `PATCH /{domainID}/bootstrap/enrollments/{id}/profile`
- `PUT /{domainID}/bootstrap/enrollments/{id}/bindings`
- `GET /{domainID}/bootstrap/enrollments/{id}/bindings`
- `POST /{domainID}/bootstrap/enrollments/{id}/provision`
- `POST /{domainID}/bootstrap/enrollments/{id}/bindings/refresh`
- `POST /{domainID}/bootstrap/profiles/{profileID}/render-preview`

Existing config endpoints can remain as compatibility wrappers during migration.

## Template Engine Rules

Start with Go `text/template` for simplicity.

Rules:

- Use `missingkey=error`.
- Provide only allowlisted helper functions.
- Validate rendered output when the profile declares a structured format such as JSON, TOML, or YAML.
- Fail bootstrap when required bindings or variables are missing.
- Do not allow templates to call arbitrary functions or resolve resources dynamically.
- Do not store secrets in profile defaults.
- Inject secrets only from encrypted binding snapshots at render time.

## Migration Plan

1. Add profile and binding snapshot models without changing current behavior.
2. Add profile upload/create APIs.
3. Add enrollment-to-profile assignment.
4. Add explicit binding APIs for mapping profile slots to clients/channels/certs.
5. Add a renderer that can render the current `Config.Content` as a template using existing config fields.
6. Move clients/channels validation and optional creation behind a `BindingResolver`.
7. Store resolved clients/channels values as binding snapshots.
8. Change bootstrap response rendering to use stored snapshots.
9. Keep current config API behavior through compatibility wrappers.
10. Add explicit provision and refresh operations.
11. Remove Bootstrap's dependency on local clients/channels replica tables once snapshots cover the needed render data.
12. Remove clients/channels event consumers from Bootstrap unless they are only used for snapshot refresh.

## Testing Strategy

Renderer tests:

- Missing variable fails.
- Missing binding fails.
- Missing secret fails.
- Valid template renders deterministic output.
- Secure bootstrap encrypts rendered output.
- JSON/TOML/YAML templates validate when format is declared.

Provisioning tests:

- Resolver creates or links expected resources.
- Resolver stores snapshots without render-time service calls.
- Partial provisioning failure does not leave a config marked ready.
- Refresh updates snapshots.

Binding tests:

- Unknown profile slot fails.
- Wrong resource type for a slot fails.
- Missing required slot keeps enrollment non-renderable.
- Optional slot can be omitted.
- Manual binding stores the expected snapshot.

Service tests:

- Bootstrap succeeds when clients/channels services are unavailable but snapshots exist.
- Bootstrap fails when required snapshots are missing.
- Existing config endpoints keep current behavior during compatibility phase.

Repository tests:

- Profiles are scoped by domain.
- Bindings are scoped by config and domain.
- Secret snapshots are encrypted at rest.
- Binding refresh is idempotent.

## First Practical Step

The smallest useful first step in this repository is:

1. Add a profile/template model and repository.
2. Add profile create/upload APIs.
3. Add enrollment-to-profile assignment.
4. Add a binding snapshot repository.
5. Add manual binding APIs for linking slots to clients/channels.
6. Add a pure renderer behind the existing `ConfigReader` concept.
7. Make current config content render from stored config data and bindings.
8. Keep current client/channel calls outside the renderer.

This gives the project a clean path toward profiles without breaking existing bootstrap behavior.
