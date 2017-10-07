# Mainflux

[![build][ci-badge]][ci-url]
[![go report card][grc-badge]][grc-url]
[![license][license]](LICENSE)
[![chat][gitter-badge]][gitter]

![banner][banner]

Mainflux is modern, scalable, secure open source and patent-free IoT cloud platform written in Go.

It accepts user, device, and application connections over various network protocols (i.e. HTTP,
MQTT, WebSocket, CoAP), thus making a seamless bridge between them. It is used as the IoT middleware
for building complex IoT solutions.

For more details, check out the [official documentation][docs].

## Features

- Protocol bridging (i.e. HTTP, MQTT, WebSocket, CoAP)
- Device management and provisioning
- Linearly scalable [data storage][cassandra]
- Fine-grained access control
- Platform logging and instrumentation support
- Container-based deployment using [Docker][docker]

## Quickstart

#### Docker

To start the docker composition, execute the [startup script](bin/mainflux-docker.sh) from `bin`
directory:

```
./mainflux-docker.sh start  
```

If the command successfully completes, you can verify that the all images are up & running by
executing the following command in the terminal window:

```
docker ps
```

The composition can be terminated in the following way:

```
./mainflux-docker.sh stop
```

#### From sources

To download all the sources, and place them in appropriate locations (i.e. $GOPATH), use the
[installation script](bin/mainflux-install.sh). Once it completes, the script will provide the
instructions on how to finish the manual installation (i.e. install the required infrastructure).

## Contributing

Thank you for your interest in Mainflux and wish to contribute!

1. Take a look at our [open issues](https://github.com/mainflux/mainflux/issues).
2. Checkout the [contribution guide](CONTRIBUTING.md) to learn more about our style and conventions.
3. Make your changes compatible to our workflow.

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
[cassandra]: http://cassandra.apache.org
