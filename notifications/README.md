# Notifications Service

The Notifications Service is responsible for sending email notifications when domain invitation events occur in the SuperMQ platform.

## Overview

This service listens to invitation events from the domains service and sends email notifications to users when:
- They are invited to join a domain (`invitation.send`)
- Someone accepts their domain invitation (`invitation.accept`)
- Someone rejects their domain invitation (`invitation.reject`)

The service uses gRPC to fetch user information from the users service and sends styled email notifications using SMTP.

## Features

- **Event-Driven**: Listens to invitation events from the event store (NATS/RabbitMQ)
- **gRPC Integration**: Fetches user details (name, email) from the users service
- **Beautiful Email Templates**: Styled HTML email templates with Magistrala branding (#083662)
- **Configurable**: Email server settings and templates are fully configurable

## Architecture

```
domains service → event store → notifications service → users service (gRPC)
                                         ↓
                                    SMTP Server → Email Recipients
```

## Configuration

The service is configured using environment variables:

### General Configuration
- `MG_NOTIFICATIONS_LOG_LEVEL` - Log level (default: "info")
- `MG_NOTIFICATIONS_INSTANCE_ID` - Instance ID for the service
- `MG_NOTIFICATIONS_DOMAIN_ALT_NAME` - Alternative name for domains such as, say, workspaces or tenants (default: "domains")
- `MG_ES_URL` - Event store URL (default: "nats://localhost:4222")

### Email Configuration
- `MG_EMAIL_HOST` - SMTP server host (default: "localhost")
- `MG_EMAIL_PORT` - SMTP server port (default: "25")
- `MG_EMAIL_USERNAME` - SMTP username
- `MG_EMAIL_PASSWORD` - SMTP password
- `MG_EMAIL_FROM_ADDRESS` - From email address (default: "noreply@supermq.com")
- `MG_EMAIL_FROM_NAME` - From name (default: "SuperMQ Notifications")

### Template Configuration
- `MG_EMAIL_INVITATION_TEMPLATE` - Path to invitation email template
- `MG_EMAIL_ACCEPTANCE_TEMPLATE` - Path to acceptance email template
- `MG_EMAIL_REJECTION_TEMPLATE` - Path to rejection email template

### gRPC Configuration (Users Service)
- `MG_USERS_GRPC_URL` - Users service gRPC URL
- `MG_USERS_GRPC_TIMEOUT` - gRPC request timeout
- `MG_USERS_GRPC_CLIENT_CERT` - Client certificate path
- `MG_USERS_GRPC_CLIENT_KEY` - Client key path
- `MG_USERS_GRPC_SERVER_CA_CERTS` - Server CA certificates path

## Running the Service

```bash
go run cmd/notifications/main.go
```

Or build and run:

```bash
go build -o notifications cmd/notifications/main.go
./notifications
```

## Email Templates

The service includes three beautifully styled email templates with Magistrala branding:

1. **Invitation Sent** (`invitation-sent-email.tmpl`) - Blue gradient header (#083662)
2. **Invitation Accepted** (`invitation-accepted-email.tmpl`) - Green gradient header
3. **Invitation Rejected** (`invitation-rejected-email.tmpl`) - Red gradient header

All templates are responsive and include:
- Professional styling
- Gradient headers
- Clear call-to-action sections
- Magistrala branding

## Testing

Run the tests:

```bash
go test ./notifications/... -v
```

To run email integration tests (requires SMTP server):

```bash
MG_RUN_EMAIL_TESTS=true go test ./notifications/emailer -v
```

## Development

The service consists of:
- `notifier.go` - Main service interface
- `emailer/emailer.go` - Email notification implementation
- `events/consumer.go` - Event consumer for invitation events
- `cmd/notifications/main.go` - Service entry point
- Tests with mocks for unit testing

## Dependencies

- Users service (gRPC) - for fetching user information
- Event store (NATS/RabbitMQ) - for receiving invitation events
- SMTP server - for sending emails
