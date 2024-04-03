# Writers

Writers provide an implementation of various `message writers`.
Message writers are services that normalize (in `SenML` format)
Magistrala messages and store them in specific data store.

Writers are optional services and are treated as plugins. In order to
run writer services, core services must be up and running. For more info
on the platform core services with its dependencies, please check out
the [Docker Compose][compose] file.

For an in-depth explanation of the usage of `writers`, as well as thorough
understanding of Magistrala, please check out the [official documentation][doc].

[doc]: https://docs.magistrala.abstractmachines.fr
[compose]: ../docker/docker-compose.yml
