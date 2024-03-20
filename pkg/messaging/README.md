# Messaging

`messaging` package defines `Publisher`, `Subscriber` and an aggregate `Pubsub` interface. 

`Subscriber` interface defines methods used to subscribe messages from a message broker. Currently, supported brokers are MQTT, NATS, RabbitMQ, and Kafka.

`Publisher` interface defines methods used to publish messages to a message broker. Currently, supported brokers are MQTT, NATS, RabbitMQ, and Kafka.

`Pubsub` interface is composed of `Publisher` and `Subscriber` interface and can be used to send messages to as well as to receive messages from a message broker.
