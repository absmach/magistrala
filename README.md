<div align="center">

  # SuperMQ
  
  **Planetary event-driven infrastructure**
  
  **Made with ❤️ by [Abstract Machines](https://abstractmachines.fr/)**
  
  [![Build Status](https://github.com/absmach/supermq/actions/workflows/build.yml/badge.svg?branch=main)](https://github.com/absmach/supermq/actions/workflows/build.yml)
  [![Check License Header](https://github.com/absmach/supermq/actions/workflows/check-license.yaml/badge.svg?branch=main)](https://github.com/absmach/supermq/actions/workflows/check-license.yaml)
  [![Check Generated Files](https://github.com/absmach/supermq/actions/workflows/check-generated-files.yml/badge.svg?branch=main)](https://github.com/absmach/supermq/actions/workflows/check-generated-files.yml)
  [![Go Report Card](https://goreportcard.com/badge/github.com/absmach/supermq)](https://goreportcard.com/report/github.com/absmach/supermq)
  [![Coverage](https://codecov.io/gh/absmach/supermq/graph/badge.svg?token=nPCEr5nW8S)](https://codecov.io/gh/absmach/supermq)
  [![License](https://img.shields.io/badge/license-Apache%20v2.0-blue.svg)](LICENSE)
 [![Matrix](https://img.shields.io/matrix/supermq%3Amatrix.org?label=Chat&style=flat&logo=matrix&logoColor=white)](https://matrix.to/#/#supermq:matrix.org)
  
  ### [Guide](https://docs.supermq.abstractmachines.fr) | [Contributing](CONTRIBUTING.md) | [Website](https://abstractmachines.fr/) | [Chat](https://matrix.to/#/#supermq:matrix.org)

</div>



## Introduction 📖

SuperMQ is a distributed, highly scalable, and secure open-source cloud platform for messaging and event-driven architecture (EDA). It is a planetarily distributed, highly scalable, and secure platform that serves as a robust foundation for building advanced real-time and reactive systems.

## Why SuperMQ Stands Out 🚀

SuperMQ bridges the gap between various network protocols (HTTP, MQTT, WebSocket, CoAP, and more) to provide a seamless messaging experience. Whether you're working on IoT solutions, real-time data pipelines, or event-driven systems, SuperMQ has you covered. 🌐✨

## Key Features 🌟

- **Multi-Protocol Connectivity**: HTTP, MQTT, WebSocket, CoAP, and more! 🌉
- **Secure by Design**: Mutual TLS (mTLS) with X.509 Certificates, JWT support, and multi-protocol authorization. 🔒
- **Fine-Grained Access Control**: Support for ABAC and RBAC policies. 📜
- **Multi-Tenant**: Manage multiple domains seamlessly. 🏢
- **Multi-User**: Unlimited organizational hierarchies for user management. 👥
- **Application Management**: Group and share messaging clients for streamlined operations. 📱
- **Ease of Use**: Simple and powerful communication channel management, grouping, and sharing. ✨
- **Personal Access Tokens (PATs)**: Scoped and revocable tokens for enhanced security. 🔑
- **Observability**: Integrated logging and instrumentation with Prometheus and OpenTelemetry. 📈
- **Event Sourcing**: Build robust and scalable architectures. ⚡
- **Edge and IoT Ready**: Supports MQTT and CoAP protocols for seamless IoT gateway and sensor communication and management. 🌍
- **Developer-Friendly**: SDKs, CLI tools, and comprehensive documentation to get you started. 👩‍💻👨‍💻
- **Production-Ready**: Container-based deployment using Docker and Kubernetes. 🐳☸️

## Installation 🛠️

Clone the repository and start SuperMQ services:

```bash
git clone https://github.com/absmach/supermq.git
cd supermq
docker compose -f docker/docker-compose.yml --env-file docker/.env up
```

Or use the [Makefile](Makefile) for a simpler command:

```bash
make run
```

For production deployments, check our [Kubernetes guide](https://docs.supermq.abstractmachines.fr/kubernetes). ⚙️

### Usage 📤📥

#### Using the CLI:

```bash
make cli
./build/supermq-cli status
```

This command retrieves the status of the SuperMQ server and outputs it to the console.

#### Using HTTP with Curl:

```bash
curl -X GET http://localhost:8080/status
```

This request fetches the server status over HTTP and provides a JSON response.

See our [CLI documentation](https://docs.supermq.abstractmachines.fr/cli) for more details.

## Documentation 📚

The official documentation is hosted at [SuperMQ docs page](https://docs.supermq.abstractmachines.fr).

Documentation is auto-generated, check out the instructions in the [docs repository](https://github.com/absmach/supermq-docs).
If you spot an error or a need for corrections, please let us know - or even better: send us a PR! 💌

## Community and Contributing 🤝

Thank you for your interest in SuperMQ and the desire to contribute!

1. Take a look at our [open issues](https://github.com/absmach/supermq/issues). The [good-first-issue](https://github.com/absmach/supermq/labels/good-first-issue) label is specifically for issues that are great for getting started.
2. Checkout the [contribution guide](CONTRIBUTING.md) to learn more about our style and conventions.
3. Make your changes compatible to our workflow.

Join our community:

- [Matrix Room](https://matrix.to/#/#supermq\:matrix.org)

## Professional Support 💼

Need help deploying SuperMQ or integrating it into your system? Reach out to **[Abstract Machines](https://abstractmachines.fr/)** for professional support and guidance.

## License 📜

SuperMQ is open-source software licensed under the [Apache License 2.0](LICENSE). Contributions are welcome!

## Acknowledgments 🙌

Special thanks to the amazing contributors who make SuperMQ possible. Check out the [MAINTAINERS](MAINTAINERS) file to see the team behind the magic.

Ready to build the future of messaging and event-driven systems? Let's get started! 🚀

