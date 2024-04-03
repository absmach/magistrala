# MQTT Benchmarking Tool

A simple MQTT benchmarking tool for Magistrala platform.

It connects Magistrala things as subscribers over a number of channels and
uses other Magistrala things to publish messages and create MQTT load.

Magistrala things used must be pre-provisioned first, and Magistrala `provision` tool can be used for this purpose.

## Installation

```
cd tools/mqtt-bench
make
```

## Usage

The tool supports multiple concurrent clients, publishers and subscribers configurable message size, etc:

```
./mqtt-bench --help
Tool for extensive load and benchmarking of MQTT brokers used within Magistrala platform.
Complete documentation is available at https://docs.magistrala.abstractmachines.fr

Usage:
  mqtt-bench [flags]

Flags:
  -b, --broker string     address for mqtt broker, for secure use tcps and 8883 (default "tcp://localhost:1883")
      --ca string         CA file (default "ca.crt")
  -c, --config string     config file for mqtt-bench (default "config.toml")
  -n, --count int         Number of messages sent per publisher (default 100)
  -f, --format string     Output format: text|json (default "text")
  -h, --help              help for mqtt-bench
  -m, --magistrala string   config file for Magistrala connections (default "connections.toml")
      --mtls              Use mtls for connection
  -p, --pubs int          Number of publishers (default 10)
  -q, --qos int           QoS for published messages, values 0 1 2
      --quiet             Supress messages
  -r, --retain            Retain mqtt messages
  -z, --size int          Size of message payload bytes (default 100)
  -t, --skipTLSVer        Skip tls verification
  -t, --timeout           Timeout mqtt messages (default 10000)
```

Two output formats supported: human-readable plain text and JSON.

Before use you need a `mgconn.toml` - a TOML file that describes Magistrala connection data (channels, thingIDs, thingKeys, certs).
You can use `provision` tool (in tools/provision) to create this TOML config file.

```bash
go run tools/mqtt-bench/cmd/main.go -u test@magistrala.com -p test1234 --host http://127.0.0.1 --num 100 > tools/mqtt-bench/mgconn.toml
```

Example use and output

Without mtls:

```
go run tools/mqtt-bench/cmd/main.go --broker tcp://localhost:1883 --count 100 --size 100 --qos 0 --format text --pubs 10 --magistrala tools/mqtt-bench/mgconn.toml
```

With mtls
go run tools/mqtt-bench/cmd/main.go --broker tcps://localhost:8883 --count 100 --size 100 --qos 0 --format text --pubs 10 --magistrala tools/mqtt-bench/mgconn.toml --mtls -ca docker/ssl/certs/ca.crt

```

You can use `config.toml` to create tests with this tool:

```

go run tools/mqtt-bench/cmd/main.go --config tools/mqtt-bench/config.toml

```

Example of `config.toml`:

```

[mqtt]
[mqtt.broker]
url = "tcp://localhost:1883"

[mqtt.message]
size = 100
format = "text"
qos = 2
retain = true

[mqtt.tls]
mtls = false
skiptlsver = true
ca = "ca.crt"

[test]
pubs = 3
count = 100

[log]
quiet = false

[magistrala]
connections_file = "mgconn.toml"

```

Based on this, a test scenario is provided in `templates/reference.toml` file.
```
