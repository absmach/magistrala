# Consumers

Consumers provide an abstraction of various `SuperMQ consumers`.
SuperMQ consumer is a generic service that can handle received messages - consume them.
The message is not necessarily a SuperMQ message - before consuming, SuperMQ message can
be transformed into any valid format that specific consumer can understand. For example,
writers are consumers that can take a SenML or JSON message and store it.

Consumers are optional services and are treated as plugins. In order to
run consumer services, core services must be up and running.

For an in-depth explanation of the usage of `consumers`, as well as thorough
understanding of SuperMQ, please check out the [official documentation][doc].

For more information about service capabilities and its usage, please check out
the [API documentation](https://docs.api.supermq.abstractmachines.fr/?urls.primaryName=consumers-notifiers-openapi.yaml).

[doc]: https://docs.supermq.abstractmachines.fr
