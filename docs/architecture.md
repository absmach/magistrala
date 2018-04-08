## Components

Mainflux IoT platform is comprised of the following services:

| Service                                                                   | Description                                                             |
|:--------------------------------------------------------------------------|:------------------------------------------------------------------------|
| [manager](https://github.com/mainflux/mainflux/tree/master/manager)       | Manages platform entities, and auth concerns                            |
| [http-adapter](https://github.com/mainflux/mainflux/tree/master/http)     | Provides an HTTP interface for accessing communication channels         |
| [normalizer](https://github.com/mainflux/mainflux/tree/master/normalizer) | Normalizes SenML messages and generates the "processed" messages stream |

> The following diagram is an (obsolete) overview of platform architecture

![arch](img/architecture.jpg)

## Domain model

The platform is built around 3 main entities: **users**, **clients** and **channels**.

`User` represents the real (human) user of the system. It is represented via its
e-mail and password, which he uses as platform access credentials in order to obtain
an access token. Once logged into the system, user can manage his resources in
CRUD fashion (i.e. channels and clients), and define access control policies
between them.

`Device` is used to represent any device that connects to Mainflux. It is a
generic model that describes any client device of the system.

`Application` is very similar to the `Device` and is represented by the same
`Client` structure (just with different `type` info). Application represents
an end-user application that communicates with devices through Mainflux, and
can be running somewhere in the cloud, locally on the PC or on the mobile phone.
Usually it acquires data from sensor measurement and displays it on various
dashboards.

`Channel` represents a communication channel. It serves as message topic that
can be consumed by all of the clients connected to it.

## Messaging

Mainflux uses [NATS](https://nats.io) as its messaging backbone, due to its
lightweight and performant nature. You can treat its *subjects* as physical
representation of Mainflux channels, where subject name is constructed using
channel unique identifier.

In general, there is no constrained put on content that is being exchanged
through channels. However, in order to be post-processed and normalized,
messages should be formatted using [SenML](https://tools.ietf.org/html/draft-ietf-core-senml-08).
