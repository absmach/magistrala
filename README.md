# Mainflux

[![License](https://img.shields.io/badge/license-Apache%20v2.0-blue.svg)](LICENSE)
[![Join the chat at https://gitter.im/Mainflux/mainflux](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/Mainflux/mainflux?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

> **NOTE**
>
> Mainflux is currently under heavy refactoring, moving to `go-kit` framework and `Cassandra` database for `v1.0.0` release.
> Release ETA: end of September
>
> **Docker composition will be unusable during this period - use direct install**

### About
Mainflux is modern massively-scalable and [highly-secured](#security) open source and patent-free IoT cloud platform written in Go, based on a set of [microservices](#architecture).

It allows device, user and application connections over various network protocols, like HTTP, MQTT, WebSocket and CoAP, making a seamless bridge between them. It is used as the IoT middleware for building complex IoT solutions.

![gophersBanner](https://github.com/mainflux/doc/blob/master/docs/img/gopherBanner.jpg)

Mainflux is built with <3 by [Mainflux company](https://www.mainflux.com/) and community contributors.

### Architecture
Mainflux IoT cloud is composed of several components, i.e. microservices:

| Link          | Description           |
|:--------------|:----------------------|
| [http-adapter](https://github.com/mainflux/http-adapter) | HTTP message API server |
| [manager](https://github.com/mainflux/manager) | Service for managing platform resources, including auth |
| [message-writer](https://github.com/mainflux/message-writer) | Worker behind NATS that writes messages into Cassandra DB |
| [mqtt-adapter](https://github.com/mainflux/mqtt-adapter) | MQTT PUB/SUB Broker (with WebSocket support) |
| [mainflux-coap](https://github.com/mainflux/mainflux-coap) | CoAP Server |
| [mainflux-ui](https://github.com/mainflux/mainflux-ui)     | System Dashboard in Angular 2 Material |
| [mainflux-cli](https://github.com/mainflux/mainflux-cli)   | Interactive command-line interface |
| [Cassandra](https://github.com/apache/cassandra)           | System Database |
| [NATS](https://github.com/nats-io/gnatsd)                  | System event bus |
| [NGINX](https://github.com/nginx/nginx)                    | Reverse Proxy with Auth forwarding |

![arch](https://github.com/mainflux/doc/blob/master/docs/img/architecture.jpg)

Docker composition that constitues Mainflux IoT infrastructure is defined in the [`docker-compose.yml`](https://github.com/Mainflux/mainflux/blob/master/docker-compose.yml).

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
- Enhanced and fine-grained security with Access Control Lists
- Easy deployment and high system scalability via [Docker](https://www.docker.com/) images
- Clear project roadmap, extensive development ecosystem and highly skilled developer community
- And many more

### Roadmap
- [x] Use `go-kit` microservice framework
- [x] Switch to `Cassandra`
- [x] Use Docker multi-stage builds
- [ ] Enable service discovery (`Consul` or `etcd`)
- [ ] Finish `Dashflux` (Mainflux UI) MVP
- [ ] Release `v1.0.0` (ETA: end of September)
- [ ] Deploy public cloud
- [ ] E2E tests and benchmarks
- [ ] Ansible and Terraform deployment scripts
- [ ] Kubernetes deployment procedure

Project task management is done via [GitHub Projects](https://github.com/mainflux/mainflux/projects/2), through issues opened in this repo.

### Documentation
Mainflux documentation can be found [here](http://mainflux.readthedocs.io).

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
