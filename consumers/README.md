# Consumers

Consumers provide an abstraction for various “SuperMQ consumers”.

A consumer is a generic plugin‑style service that handles received messages — for example, writing them to a database, sending notifications, or transforming them. Before consuming, messages from SuperMQ can be transformed (e.g. to JSON or SenML) to match what a specific consumer expects.

This service (Notifiers) is optional — to use it, core services must be running (e.g. message broker + clients + channels etc.).

## Concepts & Consumer Types

### Consumer Interfaces

The service supports two main consumer interfaces in code:

- **BlockingConsumer** — a synchronous consumer interface. Such a consumer processes the incoming message and returns an error if something goes wrong.  
- **AsyncConsumer** — an asynchronous consumer interface. The consumer receives messages and processes them asynchronously; errors can be monitored via an error channel returned by `Errors()`.  

A consumer implementation may wrap message parsing or transformation logic (e.g. converting to SenML/JSON) before invoking its own consume logic.

### Message Flow

When a subscriber receives messages from the message broker:

1. Messages may be transformed (e.g. via a transformer for SenML or JSON) based on configuration.  
2. The transformed message is passed to a consumer — either synchronously (BlockingConsumer) or asynchronously (AsyncConsumer).  
3. The consumer handles the message (e.g. storing to DB, sending notifications, writing files, etc.).

Consumers are decoupled from core messaging logic, making them flexible and pluggable.

## Supported Consumers

The following consumer plugins are supported within the [Magistrala](https://github.com/absmach/magistrala) repository:

| Consumer | Type        | Description | Link |
|----------|-------------|-------------|------|
| **SMPP** | Notifier    | Sends SMS messages via SMPP | [smpp consumer](https://github.com/absmach/magistrala/tree/main/consumers/notifiers/smpp) |
| **SMTP** | Notifier    | Sends email notifications via SMTP | [smtp consumer](https://github.com/absmach/magistrala/tree/main/consumers/notifiers/smtp) |
| **Postgres** | Writer | Stores messages in a PostgreSQL database | [postgres writer](https://github.com/absmach/magistrala/tree/main/consumers/writers/postgres) |
| **Timescale** | Writer | Stores messages in TimescaleDB (optimized for time-series) | [timescale writer](https://github.com/absmach/magistrala/tree/main/consumers/writers/timescale) |

> Each consumer has its own README with deployment instructions, configuration, and usage examples.

## Notifier API (for Notifications)

The Notifiers service exposes an HTTP API to manage subscriptions and send notifications when messages are consumed. The API supports:

- Creating subscriptions
- Listing and filtering subscriptions
- Viewing a subscription by ID
- Deleting a subscription

### Available Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/subscriptions` | POST | Create a new subscription (topic + contact) |
| `/subscriptions` | GET | List subscriptions (with optional filters) |
| `/subscriptions/{id}` | GET | Retrieve a specific subscription by ID |
| `/subscriptions/{id}` | DELETE | Remove a subscription |

### Example: Create Subscription

```bash
curl -X POST http://localhost:9014/subscriptions \
  -H "Authorization: Bearer <user_access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "some/topic/subtopic",
    "contact": "user@example.com"
  }'
  ```

For more information about service capabilities and its usage, please check out the [API documentation](https://docs.api.magistrala.absmach.eu/?urls.primaryName=api%2Fnotifiers.yaml).

## Best Practices

- **Use async consumers** when message processing is long or non-blocking (e.g. writing to external APIs).  
- **Always read from error channels** in async consumers to avoid silent failures.  
- **Use message transformers** to adapt data format (SenML or JSON) to consumer needs.  
- **Configure each consumer independently**, allowing clear isolation and debugging.  
- **Monitor subscription usage** and prune stale entries regularly.

## Versioning & Health Check

If the consumer exposes a `/health` endpoint, use it for service monitoring.

```bash
curl -X GET http://localhost:<port>/health \
  -H "accept: application/health+json"
```

Example response:

```bash
{
  "status": "pass",
  "version": "0.15.1",
  "description": "notifiers service",
  "build_time": "YYYY‑MM‑DDTHH:MM:SSZ"
}
```

For an in-depth explanation of the usage of `consumers`, as well as thorough
understanding of SuperMQ, please check out the [official documentation][doc].

[doc]: https://docs.supermq.absmach.eu/
