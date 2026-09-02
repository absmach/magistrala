# Messaging

`messaging` package defines `Publisher`, `Subscriber` and an aggregate `Pubsub` interface.

`Subscriber` interface defines methods used to subscribe to a message broker such as MQTT, FluxMQ or NATS.

`Publisher` interface defines methods used to publish messages to a message broker such as MQTT, FluxMQ or NATS.

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

Addressing a queue is not the same as one existing. Each stream is captured by its own `$queue/<name>/#` binding in the broker configuration, and a publication matching no binding is dropped without an error — a failed or absent capture never fails the publish. A new `$queue/<name>` namespace therefore needs its queue declared in `docker/fluxmq/node{1,2,3}.yaml` before anything is published to it. The bindings are deliberately disjoint, so that a message lands in exactly one stream rather than also accumulating in the reserved `mqtt` queue; `docker/fluxmq/config_test.go` holds that invariant.

### Stream queues

On startup, publishers and pubsub clients normally declare a durable stream queue named after their prefix. Stream subscribers use consumer groups, so each group receives every message exactly once. The default stream queue is named `m`. `InternalMetadata` instead requires that stream to be pre-provisioned by the broker and never attempts to create or modify it.

### Subscription

`Subscribe` attaches to the durable stream queue via a consumer group filtered by topic. Optionally (when `DirectTopicIngress` is enabled), it also subscribes to the raw MQTT topic so that messages published directly by MQTT clients — bypassing the queue — are also received. A deployment using `InternalMetadata` must authorize the requested subscriptions explicitly; the Rules Engine local principal authorizes only pre-provisioned stream `m`.

### Message origin

A message carries the protocol it was published with (`mqtt`, `http`, `coap`, …) and the identity of its publisher. Both are broker-controlled: on a publication from an untrusted connection the broker overwrites them with the transport and identity of that connection, so a service that consumes a device message and republishes it — into the `writers` stream, for instance — turns every one of them into `protocol: amqp` published by that service.

`InternalMetadata` is what avoids this. A connection authenticated as a `service`-role local principal on the mTLS listener may relay the origin protocol, publisher, `created` timestamp and metadata it received rather than having its own stamped on. Any service that republishes messages someone else authored has to use it, and its principal needs a `permissions.publish` entry for the destination.

`Message.DeviceId` is a separate, opt-in identity: "whose data this is" rather than "who sent it". It is set only when a publish is already known to be about exactly one device — a directly-connected device, or a service republishing a value it computed for one specific meter — and is relayed the same way as publisher/protocol. It is never set from a raw gateway batch, which is never provably single-device; a batch's per-record attribution is resolved downstream, from the payload, by `pkg/transformers/senml` and `pkg/transformers/json`.

### Options

| Option                                 | Description                                                                                       |
| -------------------------------------- | ------------------------------------------------------------------------------------------------- |
| `Prefix(p)`                            | Set topic prefix (default: `m`)                                                                   |
| `ConnectionName(n)`                    | Human-readable broker connection name                                                             |
| `DirectTopicIngress()`                 | Also consume raw MQTT topic messages (subscriber only)                                            |
| `InternalMetadata(cert, key, ca)`      | Require mTLS, carry reserved internal metadata, and use a broker-provisioned stream                 |
