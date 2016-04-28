# Mainflux

[![License](https://img.shields.io/badge/license-Apache%20v2.0-blue.svg)](LICENSE) [![Join the chat at https://gitter.im/Mainflux/mainflux](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/Mainflux/mainflux?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

### About
Mainflux is lean open source industrial IoT cloud written in NodeJS.

It allows device, user and application connections over various network protocols, like HTTP, MQTT, WebSocket and CoAP, making a seamless bridge between them. As a consequence, Mainflux represents highly secure and highly optimised M2M platform based on the cutting-edge standards and approaches in the industry.


### Architecture

![Mainflux Architecture](https://github.com/Mainflux/mainflux-doc/blob/master/img/mainfluxArchitecture.jpg "Mainflux Architecture")

### Features
An extensive (and incomplete) list of featureas includes:
- Responsive and scalable architecture based on a set of [Microservices](https://en.wikipedia.org/wiki/Microservices)
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


### Install

Clone the repo:
```bash
git clone https://github.com/Mainflux/mainflux.git
cd mainflux
```
### System Architecture
Mainflux IoT cloud is composed of several components, i.e. microservices:
- Mainflux Core Server
- Mainflux HTTP API Server
- Mainflux MQTT API Server
- Mainflux WebSocket API Server
- NATS PUB/SUB Broker

The following matrix describes the functionality of each GE in the system and gives the location of the code repositories:

| Microservice         | Function               |  GitHub repo                                                             |
| :------------------- |:-----------------------| :------------------------------------------------------------------------|
| Mainflux Core        | Core Server            | [mainflux-core-server](https://github.com/Mainflux/mainflux-core-server) |
| HTTP API Server      | HTTP API Server        | [mainflux-http-server](https://github.com/Mainflux/mainflux-http-server) |
| HTTP MQTT Server     | MQTT API Server        | [mainflux-mqtt-server](https://github.com/Mainflux/mainflux-mqtt-server) |
| HTTP WS Server       | WS API Server          | [mainflux-ws-server](https://github.com/Mainflux/mainflux-ws-server)     |
| NATS                 | PUB/SUB Broker         | [gnatsd](https://github.com/nats-io/gnatsd)                              |

These components are packaged and deployed in a set of Docker containers maintained by Mainflux team, with images uploaded to [Mainflux Docker Hub page](https://hub.docker.com/u/mainflux/).

Docker composition that constitues Mainflux IoT infrastructure is defined in the [`docker-compose.yml`](https://github.com/Mainflux/mainflux/blob/master/docker-compose.yml).

### Deployment
Deployment of Mainflux IoT Cloud is super-easy:
- Get the [`docker-compose.yml`](https://github.com/Mainflux/mainflux-fiware/blob/master/docker-compose.yml)
- Start the composition:
```
docker-compose up
```
This will automatically download Docker images from [Mainflux Docker Hub](https://hub.docker.com/u/mainflux/) and deploy the composition.

If you need to modify these Docker images, you will have to look at appropriate repos in the [Mainflux project GitHub](https://github.com/Mainflux) - look for the repos starting with prefix `mainflux-<protocol>-server`.

### Docker
Apart from main `nodejs` Docker image, Mainflux also uses `mongo` Docker image (database instance is run in a separte generic Docker image).

This is why Mainflux uses [Docker Compose](https://docs.docker.com/compose/install/), to run both `nodejs` and `mongo` images at the same time and make a connection ([container link](https://docs.docker.com/v1.8/userguide/dockerlinks/)) between them.

Executing:
```bash
docker-compose up
```
will automatically build all the images, run Docker containers and create link between them - i.e. it will bring up Mainflux API server + MongoDB ready for use.

For more details and troubleshooting please consult [Docker chapter on Mainflux Wiki](https://github.com/Mainflux/mainflux/wiki/Docker).

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
