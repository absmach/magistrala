# Nats Docker Profiles

This directory contains the docker-compose profiles for running Nats as an MQTT broker. It is separated from the main profile at `../docker-compose.yml` because of name conflicts with the Nats message broker.

The configuration is the same as for the main profile, except that the MQTT broker is set to `nats` instead of `vernemq`.

The profiles are:

- `nats_nats.yml` - Nats as an MQTT broker and Nats as a message broker
- `nats_rabbit.yml` - Nats as an MQTT broker and RabbitMQ as a message broker

They are automatically included in the main profile, so you can run them depending on the profile you want to use:

The following command will run Nats as an MQTT broker and Nats as a message broker:

```bash
MG_MQTT_BROKER_TYPE=nats MG_MESSAGE_BROKER_TYPE=nats make run
```

The following command will run Nats as an MQTT broker and RabbitMQ as a message broker:

```bash
MG_MQTT_BROKER_TYPE=nats MG_MESSAGE_BROKER_TYPE=rabbit make run
```
