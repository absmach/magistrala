# Mainflux

[![License](http://img.shields.io/:license-mit-blue.svg)](http://doge.mit-license.org) [![Join the chat at https://gitter.im/Mainflux/mainflux](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/Mainflux/mainflux?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

### About
Mainflux is lean open source MIT licensed industrial IoT cloud written in NodeJS.

It allows device, user and application connections over various network protocols, like HTTP, MQTT, WebSocket and CoAP, making a seamless bridge between them. As a consequence, Mainflux represents highly secure and highly optimised M2M platform based on the cutting-edge standards and approaches in the industry.


### Architecture

![AltTxt](http://we-io.net/img/MainfluxDiagram.png "Mainflux Architecture")

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
- Professional support via [Mainflux](http://mainflux.com) company
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

### Docker
Apart from main `nodejs` docker image, `Mainflux` also uses `mongo` Docker image (database instance is run in a separte generic Docker image).

This is why Mainflux uses [Docekr Compose](https://docs.docker.com/compose/install/), to run both `nodejs` and `mongo` images at the same time and make a connection between them:
```bash
    docker-compose up
```

### License
MIT
