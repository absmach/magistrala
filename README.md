<div align="center">

  # Magistrala
  
  **A Modern IoT Platform Built on SuperMQ**
  
  **Scalable • Secure • Open-Source**
  
  [![Check License Header](https://github.com/absmach/magistrala/actions/workflows/check-license.yaml/badge.svg?branch=main)](https://github.com/absmach/magistrala/actions/workflows/check-license.yaml)
  [![Continuous Delivery](https://github.com/absmach/magistrala/actions/workflows/build.yaml/badge.svg?branch=main)](https://github.com/absmach/magistrala/actions/workflows/build.yaml)
  [![Go Report Card](https://goreportcard.com/badge/github.com/absmach/magistrala)](https://goreportcard.com/report/github.com/absmach/magistrala)
  [![Coverage](https://codecov.io/gh/absmach/magistrala/graph/badge.svg?token=SEMDAO3L09)](https://codecov.io/gh/absmach/magistrala)
  [![License](https://img.shields.io/badge/license-Apache%202.0-blue?style=flat-square)](LICENSE)
  [![Matrix](https://img.shields.io/matrix/magistrala:matrix.org?style=flat-square)](https://matrix.to/#/#magistrala:matrix.org)
  
  ### [Guide](https://docs.magistrala.abstractmachines.fr) | [Contributing](CONTRIBUTING.md) | [Website](https://abstractmachines.fr/magistrala.html) | [Chat](https://matrix.to/#/#magistrala:matrix.org)

  Made with ❤️ by [Abstract Machines](https://abstractmachines.fr/)

</div>


## Introduction 🌍

Magistrala is a cutting-edge, open-source IoT cloud platform built on top of [SuperMQ](https://github.com/absmach/supermq). It serves as a robust middleware solution for building complex IoT applications. With Magistrala, you can connect and manage IoT devices seamlessly using multi-protocol support, all while ensuring security and scalability.

### Key Benefits:
- **Unified IoT Management**: Connect sensors, actuators, and applications over various network protocols.
- **Scalability and Performance**: Designed to handle enterprise-grade IoT deployments.
- **Secure by Design**: Features such as mutual TLS authentication and fine-grained access control.
- **Open-Source Freedom**: Patent-free, community-driven, and designed for extensibility.


## ✨ Features

- 🏢 **Multi-Tenancy**: Support for managing multiple independent domains seamlessly.
- 👥 **Multi-User Platform**: Unlimited organizational hierarchies and user roles for streamlined collaboration.
- 🌐 **Multi-Protocol Connectivity**: HTTP, MQTT, WebSocket, CoAP, and more (see [contrib repository](https://www.github.com/absmach/mg-contrib) for LoRa and OPC UA).
- 💻 **Device Management and Provisioning**: Including Zero-Touch provisioning for seamless device onboarding.
- 🛡️ **Mutual TLS Authentication (mTLS)**: Secure communication using X.509 certificates.
- 📜 **Fine-Grained Access Control**: Support for ABAC and RBAC policies.
- 💾 **Message Persistence**: Timescale and PostgreSQL support (see [contrib repository](https://www.github.com/absmach/mg-contrib) for Cassandra, InfluxDB, and MongoDB).
- 🔄 **Rules Engine (RE)**: Automate processes with flexible rules for decision-making.
- 🚨 **Alarms and Triggers**: Immediate notifications for critical IoT events.
- 📅 **Scheduled Actions**: Plan and execute tasks at predefined times.
- 📝 **Audit Logs**: Maintain a detailed history of platform activities for compliance and debugging.
- 📊 **Platform Logging and Instrumentation**: Integrated with Prometheus and OpenTelemetry.
- ⚡ **Event Sourcing**: Streamlined architecture for real-time IoT event processing.
- 🐳 **Container-Based Deployment**: Fully compatible with Docker and Kubernetes.
- 🌍 **Edge and IoT Ready**: Agent and Export services for managing remote IoT gateways.
- 🛠️ **Developer Tools**: Comprehensive SDK and CLI for efficient development.
- 🏗️ **Domain-Driven Design**: High-quality codebase and extensive test coverage.


## 🔧 Install

Clone the repository and start the services:

```bash
git clone https://github.com/absmach/magistrala.git
cd magistrala
docker compose -f docker/docker-compose.yaml --env-file docker/.env up
```

Alternatively, use the Makefile for a simpler command:

```bash
make run args=-d
```

## 📤 Usage

#### Using the CLI:

Check the health of a specific service using the CLI:

```bash
make cli
./build/cli health <service>
```

Replace `<service>` with the name of the service you want to check.

#### Using Curl:

Alternatively, use a simple HTTP GET request to check the platform's health:

```bash
curl -X GET http://localhost:8080/health
```

For additional usage examples and advanced configurations, visit the [official documentation](https://docs.magistrala.abstractmachines.fr).


## 📚 Documentation

Complete documentation is available at the [Magistrala official docs page](https://docs.magistrala.abstractmachines.fr).

For CLI usage details, visit the [CLI Documentation](https://docs.magistrala.abstractmachines.fr/cli).


## 🌐 Community and Contributing

Join the community and contribute to the future of IoT middleware:

- [Open Issues](https://github.com/absmach/magistrala/issues)
- [Contribution Guide](CONTRIBUTING.md)
- [Matrix Chat](https://matrix.to/#/#magistrala:matrix.org)


## 📜 License

Magistrala is open-source software licensed under the [Apache-2.0](LICENSE) license. Contributions are welcome and encouraged!


## 💼 Professional Support

Need help deploying Magistrala or integrating it into your systems? Contact **[Abstract Machines](https://abstractmachines.fr/)** for expert guidance and support.
