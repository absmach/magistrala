# Mainflux Lite

[![License](https://img.shields.io/badge/license-Apache%20v2.0-blue.svg)](LICENSE) [![Join the chat at https://gitter.im/Mainflux/mainflux](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/Mainflux/mainflux?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

### About
Mainflux is lean open source industrial IoT cloud written in NodeJS.

It allows device, user and application connections over various network protocols, like HTTP, MQTT, WebSocket and CoAP, making a seamless bridge between them. As a consequence, Mainflux represents highly secure and highly optimised M2M platform based on the cutting-edge standards and approaches in the industry.


### Architecture

![Mainflux Architecture](https://github.com/Mainflux/mainflux-doc/blob/master/img/mainfluxArchitecture.jpg "Mainflux Architecture")

### Features
An extensive (and incomplete) list of featureas includes:
- Set of clean APIs, Swagger documented: HTTP RESTful, MQTT, WebSocket and CoAP
- Set of client libraries for many HW platforms in several programming languages: C/C++, JavaScript and Python
- Device management and provisioning and OTA FW updates
- UNIX-like permissions for device sharing
- Highly secured connections via TLS and DTLS
- User authentication via [JSON Web Tokens](http://jwt.io/)
- Responsive and scalable ModgoDB database
- Modern architecture based on micro-services
- [LwM2M](http://goo.gl/rHjLZQ) standard compliance via [Coreflux](https://github.com/Mainflux/coreflux)
- Partial [oneM2M](http://www.onem2m.org/) compliance
- Easy deployment and high system scalability via Docker images
- Clear project roadmap, extensive development ecosystem and highly skilled developer community
- And many more


### Install

Clone the repo:
```bash
git clone https://github.com/Mainflux/mainflux.git
cd mainflux
```
Install Node modules:
```bash
npm install
```

Run Gulp Task:
```bash
gulp
```

> N.B. Mainflux has a MongoDB dependency. Database path and port can be defined in the [config](https://github.com/Mainflux/mainflux/tree/master/config) files.
> 
> To avoid installation of MongoDB on the local system in order to deploy Mainflux you can use Docker image,
> as explained below.

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

Swagger-generated API reference can be foud at [http://mainflux.com/apidoc](http://mainflux.com/apidoc).

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
