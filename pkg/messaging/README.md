# Messaging

`messaging` package defines `Publisher`, `Subscriber` and an aggregate `Pubsub` interface.

`Subscriber` interface defines methods used to subscribe to a message broker such as MQTT or NATS or RabbitMQ.

`Publisher` interface defines methods used to publish messages to a message broker such as MQTT or NATS or RabbitMQ.

`Pubsub` interface is composed of `Publisher` and `Subscriber` interface and can be used to send messages to as well as to receive messages from a message broker.

## FluxMQ backend

The `fluxmq` sub-package implements the messaging interfaces against a FluxMQ AMQP broker.

### Topic routing

Publish routing depends on the topic and the publisher prefix.

| Condition                                     | Destination                                                      |
| --------------------------------------------- | ---------------------------------------------------------------- |
| Topic starts with `$queue/`                   | Durable stream queue — queue name is everything after the prefix |
| Publisher prefix is **not** the default (`m`) | Durable stream queue — queue name is `<prefix>/<subtopic>`       |
| Publisher prefix is the default (`m`)         | Regular MQTT topic — `<prefix>/<subtopic>`                       |

The `$queue/` prefix lets any publisher force delivery into the durable stream queue regardless of its own prefix. This is used internally (e.g. by `writers`, `alarms`) to guarantee at-least-once delivery through the broker's stream.

### Stream queues

On startup, publishers and pubsub clients normally declare a durable stream queue named after their prefix. Stream subscribers use consumer groups, so each group receives every message exactly once. The default stream queue is named `m`. `InternalMetadata` instead requires that stream to be pre-provisioned by the broker and never attempts to create or modify it.

### Subscription

`Subscribe` attaches to the durable stream queue via a consumer group filtered by topic. Optionally (when `DirectTopicIngress` is enabled), it also subscribes to the raw MQTT topic so that messages published directly by MQTT clients — bypassing the queue — are also received. A deployment using `InternalMetadata` must authorize the requested subscriptions explicitly; the Rules Engine local principal authorizes only pre-provisioned stream `m`.

### Options

| Option                                 | Description                                                                                       |
| -------------------------------------- | ------------------------------------------------------------------------------------------------- |
| `Prefix(p)`                            | Set topic prefix (default: `m`)                                                                   |
| `ConnectionName(n)`                    | Human-readable broker connection name                                                             |
| `DirectTopicIngress()`                 | Also consume raw MQTT topic messages (subscriber only)                                            |
| `InternalMetadata(cert, key, ca)`      | Require mTLS, carry reserved internal metadata, and use a broker-provisioned stream                 |
