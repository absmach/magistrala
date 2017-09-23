# Mainflux

[![build][ci-badge]][ci-url]
[![go report card][grc-badge]][grc-url]
[![license][license]](LICENSE)
[![chat][gitter-badge]][gitter]

![banner][banner]

Mainflux is modern, massively-scalable, highly-secured open source and patent-free IoT cloud
platform written in Go.

It allows device, user and application connections over various network protocols, like HTTP, MQTT,
WebSocket, and CoAP, making a seamless bridge between them. It is used as the IoT middleware for
building complex IoT solutions.

For more details, check out the [official documentation][docs].

## Features

An extensive (and incomplete) list of features includes:
- Responsive and scalable microservice architecture
- Set of clean APIs: HTTP RESTful, MQTT, WebSocket and CoAP
- SDK - set of client libraries for many HW platforms in several programming languages: C/C++, JavaScript, Go and Python
- Device management and provisioning and OTA FW updates
- Highly secured connections via TLS and DTLS
- Enhanced and fine-grained security with Access Control Lists
- Easy deployment and high system scalability via [Docker][docker] images
- Clear project roadmap, extensive development ecosystem and highly skilled developer community
- And many more

## Architecture

TBD

## Quickstart

#### Docker
- Clone the repo:
```bash
git clone https://github.com/mainflux/mainflux.git
```

- Go to `mainflux/docker` dir:
```
cd mainflux/docker
```

- Use [`mainflux-docker.sh`](docker/mainflux-docker.sh) script to start the Docker composition:
```bash
./mainflux-docker.sh start
```

Once started, the script will download and start Docker images required by the composition.

#### From sources
Use script [`install_sources.sh`](install_sources.sh).

This will create `./mainflux_sources` dir, git-clone all the sources from GitHub repos and place them in appropriate destination (Go code goes to $GOPATH, symlinks are created).

It will also give you the instructions how to finish the installation manually.

## Community

- [Google group][forum]
- [Gitter][gitter]
- [Twitter][twitter]

[banner]: https://github.com/mainflux/doc/blob/master/docs/img/gopherBanner.jpg
[ci-badge]: https://semaphoreci.com/api/v1/mainflux/mainflux/branches/master/badge.svg
[ci-url]: https://semaphoreci.com/mainflux/mainflux
[docs]: http://mainflux.readthedocs.io
[docker]: https://www.docker.com
[forum]: https://groups.google.com/forum/#!forum/mainflux
[gitter]: https://gitter.im/mainflux/mainflux?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge
[gitter-badge]: https://badges.gitter.im/Join%20Chat.svg
[grc-badge]: https://goreportcard.com/badge/github.com/mainflux/mainflux
[grc-url]: https://goreportcard.com/report/github.com/mainflux/mainflux
[license]: https://img.shields.io/badge/license-Apache%20v2.0-blue.svg
[twitter]: https://twitter.com/mainflux
