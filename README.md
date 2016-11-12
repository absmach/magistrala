![gophersBanner](https://github.com/mainflux/mainflux-doc/blob/master/img/gopherBanner.jpg)

# Mainflux

[![License](https://img.shields.io/badge/license-Apache%20v2.0-blue.svg)](LICENSE)
[![Join the chat at https://gitter.im/Mainflux/mainflux](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/Mainflux/mainflux?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

### About
Mainflux is modern massively-scalable and [highly-secured](#security) open source and patent-free IoT cloud platform written in Go and Erlang, based on a set of [microservices](#architecture).

It allows device, user and application connections over various network protocols, like HTTP, MQTT, WebSocket and CoAP, making a seamless bridge between them. It is used as the IoT middleware for building complex IoT solutions.

![Cloud Architecture](https://github.com/Mainflux/mainflux-doc/blob/master/img/cloudArchMonochrome.jpg)

Mainflux is built with <3 by Mainflux [team](MAINTAINERS) and community contributors.

### Architecture
Mainflux IoT cloud is composed of several components, i.e. microservices:
- [Mainflux Core (HTTP API Server and Admin)](https://github.com/mainflux/mainflux-core)
- [Authentication and Authorization Server](https://github.com/mainflux/mainflux-auth-server)
- [MQTT PUB/SUB Broker (with WebSocket and CoAP support)](https://github.com/mainflux/emqttd-docker)
- [Mongo Database](https://github.com/mongodb/mongo)
- [PEP Proxy](https://github.com/mainflux/mainflux-pep-proxy)
- [Mainflux UI](https://github.com/mainflux/mainflux-ui)

Docker composition that constitues Mainflux IoT infrastructure is defined in the [`docker-compose.yml`](https://github.com/Mainflux/mainflux/blob/master/docker-compose.yml).

### Security
For professional deployments Mainflux is usually combined with [Mainflux Authentication and Authorization Server](https://github.com/mainflux/mainflux-auth-server) which adds fine-grained security based on customizable API keys.

Mainflux Auth Server also provides user accounts and device and application access control with simple customizable scheme based on scoped JWTs.

### Install/Deploy
 
- Clone the repo:
```bash
git clone https://github.com/Mainflux/mainflux.git && cd mainflux
```

- Start the Docker composition:
```bash
docker-compose up
```

This will automatically download Docker images from [Mainflux Docker Hub](https://hub.docker.com/u/mainflux/) and deploy the composition of Mianflux microservices.

### Features
An extensive (and incomplete) list of features includes:
- Responsive and scalable architecture based on a set of [microservices](https://en.wikipedia.org/wiki/Microservices)
- Set of clean APIs: HTTP RESTful, MQTT, WebSocket and CoAP
- SDK - set of client libraries for many HW platforms in several programming languages: C/C++, JavaScript, Go and Python
- Device management and provisioning and OTA FW updates
- Highly secured connections via TLS and DTLS
- Enhanced and fine-grained security via deployment-ready [Mainflux Authentication and Authorization Server](https://github.com/mainflux/mainflux-auth-server) with Access Control scheme based on customizable API keys and scoped JWT
- Easy deployment and high system scalability via [Docker](https://www.docker.com/) images
- Clear project roadmap, extensive development ecosystem and highly skilled developer community
- And many more

### Documentation
Development documentation can be found on our [Mainflux GitHub Wiki](https://github.com/Mainflux/mainflux/wiki).

### Community
#### Mailing list
[mainflux](https://groups.google.com/forum/#!forum/mainflux) Google group

For quick questions and suggestions you can also use GitHub Issues.

#### IRC
[Mainflux Gitter](https://gitter.im/Mainflux/mainflux?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

#### Twitter
[@mainflux](https://twitter.com/mainflux)

### Authors
Main architect and BDFL of Mainflux project is [@drasko](https://github.com/drasko). Additionaly, initial version of Mainflux was architectured and crafted by [@janko-isidorovic](https://github.com/janko-isidorovic), [@nmarcetic](https://github.com/nmarcetic) and [@mijicd](https://github.com/mijicd).

Maintainers are listed in [MAINTAINERS](MAINTAINERS) file.

Contributors are listed in [CONTRIBUTORS](CONTRIBUTORS) file.

### License
[Apache License, version 2.0](LICENSE)
