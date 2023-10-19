# Brokers Docker Compose

Mainflux supports configurable MQTT broker and Message broker.

## MQTT Broker

Mainflux supports VerneMQ and Nats as an MQTT broker.

## Message Broker

Mainflux supports NATS and RabbitMQ as a message broker.

## Profiles

This directory contains 4 docker-compose profiles for running Mainflux with different combinations of MQTT and message brokers.

The profiles are:

- `vernemq_nats` - VerneMQ as an MQTT broker and Nats as a message broker
- `vernemq_rabbitmq` - VerneMQ as an MQTT broker and RabbitMQ as a message broker
- `nats_nats` - Nats as an MQTT broker and Nats as a message broker
- `nats_rabbitmq` - Nats as an MQTT broker and RabbitMQ as a message broker

The following command will run VerneMQ as an MQTT broker and Nats as a message broker:

```bash
MF_MQTT_BROKER_TYPE=vernemq MF_MESSAGE_BROKER_TYPE=nats make run
```

The following command will run VerneMQ as an MQTT broker and RabbitMQ as a message broker:

```bash
MF_MQTT_BROKER_TYPE=vernemq MF_MESSAGE_BROKER_TYPE=rabbitmq make run
```
