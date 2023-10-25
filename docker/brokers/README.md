# Brokers Docker Compose

Magistrala supports configurable MQTT broker and Message broker.

## MQTT Broker

Magistrala supports VerneMQ and Nats as an MQTT broker.

## Message Broker

Magistrala supports NATS and RabbitMQ as a message broker.

## Profiles

This directory contains 4 docker-compose profiles for running Magistrala with different combinations of MQTT and message brokers.

The profiles are:

- `vernemq_nats` - VerneMQ as an MQTT broker and Nats as a message broker
- `vernemq_rabbitmq` - VerneMQ as an MQTT broker and RabbitMQ as a message broker
- `nats_nats` - Nats as an MQTT broker and Nats as a message broker
- `nats_rabbitmq` - Nats as an MQTT broker and RabbitMQ as a message broker

The following command will run VerneMQ as an MQTT broker and Nats as a message broker:

```bash
MG_MQTT_BROKER_TYPE=vernemq MG_MESSAGE_BROKER_TYPE=nats make run
```

The following command will run VerneMQ as an MQTT broker and RabbitMQ as a message broker:

```bash
MG_MQTT_BROKER_TYPE=vernemq MG_MESSAGE_BROKER_TYPE=rabbitmq make run
```
