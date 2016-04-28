# Mainflux

[![License](https://img.shields.io/badge/license-Apache%20v2.0-blue.svg)](LICENSE) [![Join the chat at https://gitter.im/Mainflux/mainflux](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/Mainflux/mainflux?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

### About
Mainflux is an open source and patent-free IoT cloud platform based on [microservices](https://en.wikipedia.org/wiki/Microservices).

It allows device, user and application connections over various network protocols, like HTTP, MQTT, WebSocket and CoAP, making a seamless bridge between them. It is used as the IoT middleware for building complex IoT solutions.

![Cloud Architecture](https://github.com/Mainflux/mainflux-doc/blob/master/img/cloudArchitecture.jpg)

Mainflux is built with <3 by Mainflux team and community contributors.

### Features
An extensive (and incomplete) list of featureas includes:
- Responsive and scalable architecture based on a set of [microservices](https://en.wikipedia.org/wiki/Microservices)
- Set of clean APIs, Swagger documented: HTTP RESTful, MQTT, WebSocket and CoAP
- SDK - set of client libraries for many HW platforms in several programming languages: C/C++, JavaScript, Go and Python
- Device management and provisioning and OTA FW updates
- Highly secured connections via TLS and DTLS
- Standardized [NGSI](http://technical.openmobilealliance.org/Technical/technical-information/release-program/current-releases/ngsi-v1-0) model representation
- Enhanced and fine-grained security via [Reverse Proxy](https://en.wikipedia.org/wiki/Reverse_proxy), [OAuth 2.0](http://oauth.net/2/) [identity management](https://en.wikipedia.org/wiki/Identity_management) and [RBAC](https://en.wikipedia.org/wiki/Role-based_access_control) Authorization Server.
- [LwM2M](http://goo.gl/rHjLZQ) standard compliance
- [oneM2M](http://www.onem2m.org/) adapter
- Easy deployment and high system scalability via [Docker](https://www.docker.com/) images
- Clear project roadmap, extensive development ecosystem and highly skilled developer community
- And many more

### Install/Deploy
Installation and deployment of Mainflux IoT cloud is super-easy:
- Clone the repo:
```bash
git clone https://github.com/Mainflux/mainflux.git && cd mainflux
```

- Start the Docker composition:
```bash
docker-compose up
```

This will automatically download Docker images from [Mainflux Docker Hub](https://hub.docker.com/u/mainflux/) and deploy the composition.

If you need to modify these Docker images, you will have to look at appropriate repos in the [Mainflux project GitHub](https://github.com/Mainflux) - look for the repos starting with prefix `mainflux-<protocol>-server`.

### System Architecture
Mainflux IoT cloud is composed of several components, i.e. microservices:
- Mainflux Core
- HTTP API Server
- MQTT API Server
- WebSocket API Server
- NATS PUB/SUB Broker

The following matrix describes the functionality of each GE in the system and gives the location of the code repositories:

| Microservice         | Function               |  GitHub repo                                                             |
| :------------------- |:-----------------------| :------------------------------------------------------------------------|
| Mainflux Core        | Core Server            | [mainflux-core-server](https://github.com/Mainflux/mainflux-core-server) |
| HTTP API Server      | HTTP API Server        | [mainflux-http-server](https://github.com/Mainflux/mainflux-http-server) |
| MQTT API Server      | MQTT API Server        | [mainflux-mqtt-server](https://github.com/Mainflux/mainflux-mqtt-server) |
| WS API Server        | WS API Server          | [mainflux-ws-server](https://github.com/Mainflux/mainflux-ws-server)     |
| NATS                 | PUB/SUB Broker         | [nats-io/gnatsd](https://github.com/nats-io/gnatsd)                      |

These components are packaged and deployed in a set of Docker containers maintained by Mainflux team, with images uploaded to [Mainflux Docker Hub page](https://hub.docker.com/u/mainflux/).

Docker composition that constitues Mainflux IoT infrastructure is defined in the [`docker-compose.yml`](https://github.com/Mainflux/mainflux/blob/master/docker-compose.yml).

### Documentation
Development documentation can be found on our [Mainflux GitHub Wiki](https://github.com/Mainflux/mainflux/wiki).

### Community
#### Mailing lists
- [mainflux-dev](https://groups.google.com/forum/#!forum/mainflux-dev) - developers related. This is discussion about development of Mainflux IoT cloud itself.
- [mainflux-user](https://groups.google.com/forum/#!forum/mainflux-user) - general discussion and support. If you do not participate in development of Mainflux cloud infrastructure, this is probably what you're looking for.

#### IRC
[Mainflux Gitter](https://gitter.im/Mainflux/mainflux?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

#### Twitter
[@mainflux](https://twitter.com/mainflux)

### License
[Apache License, version 2.0](LICENSE)
