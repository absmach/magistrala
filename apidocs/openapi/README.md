# Magistrala OpenAPI Specifications

This folder contains OpenAPI specifications for HTTP APIs that remain part of
Magistrala's open-source services. Specifications for services moved to
Magistrala Enterprise Edition are maintained separately.

View specification in Swagger UI at [docs.api.magistrala.absmach.eu](https://docs.api.magistrala.absmach.eu)

Workspace, user, device, channel, and group management now goes through Atom
GraphQL rather than standalone Magistrala HTTP services. Those outdated OpenAPI
specifications were removed; see `../graphql/` for the GraphQL replacement
reference.

## Native HTTP messaging

`http.yaml` describes the FluxMQ native `POST /m/{workspacePrefix}/c/{channelPrefix}`
route. In the default Magistrala Nginx deployment, prefix it with `/http/`.
This is separate from the user-authenticated
`POST /{workspaceID}/channels/{channelID}/messages` UI publishing endpoint,
which returns `202 Accepted`.

Native publication returns `200` with `{"status":"ok"}` after broker publication.
It accepts opaque payloads without JSON/SenML or Content-Type validation. SenML
validation and persistence happen downstream according to configured rules and
consumers; HTTP success does not guarantee a Reader row. Confirm persistence
through Reader when an application needs that guarantee. Clients targeting SenML
processing should send valid SenML with `Content-Type: application/senml+json`.
