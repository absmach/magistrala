# MQTT benchmarking tool

A simple MQTT (broker) benchmarking tool for Mainflux platform. ( based on github.com/krylovsk/mqtt-benchmark )


The tool supports multiple concurrent clients, publishers and subscribers configurable message size, etc:

```
cd benchmark
go build  -o mqtt-bench *.go

> mqtt-bench --help
Flags:
  -b, --broker string     address for mqtt broker, for secure use tcps and 8883 (default "tcp://localhost:1883")
      --ca string         CA file (default "ca.crt")
      --channels string   config file for channels (default "channels.toml")
  -g, --config string     config file default is config.toml (default "config.toml")
  -n, --count int         Number of messages sent per publisher (default 100)
  -f, --format string     Output format: text|json (default "text")
  -h, --help              help for mqtt-bench
      --msg string        messg to be sent, SENML (default "{\"n\":\"current\",\"t\":-4,\"v\":1.3}")
  -m, --mtls              Use mtls for connection
      --pubs int          Number of publishers (default 10)
  -q, --qos int           QoS for published messages, values 0 1 2
      --quiet             Supress messages
  -r, --retain            Retain mqtt messages
  -s, --size int          Size of message payload bytes (default 100)
  -t, --skipTLSVer        Skip tls verification
      --subs int          Number of subscribers (default 10)

```

Two output formats supported: human-readable plain text and JSON.

Before use you need a channels.toml  you can use tools/provision/main.go to create
channels for testing

Example use and output:

```
go build -o mqtt-bench *.go

without mtls
./mqtt-bench --broker tcp://localhost:1883 --count 100 --size 100  --qos 0 --format text   --subs 100 --pubs 0 --channels channels.toml

with mtls
./mqtt-bench --broker tcps://localhost:8883 --count 100 --size 100  --qos 0 --format text   --subs 100 --pubs 0 --channels channels.toml --mtls -ca ca.crt
```


You can use config.toml to create tests with this tool
```
./mqtt-bench --config config.toml it will read params from config.toml
```
```
broker_url = "tcp://localhost:1883"
qos = 2
message_size =100
message_count =100
publishers_num =3
subscribers_num =1
format = "text"
quiet = true
mtls = false
skiptlsver = true
ca_file = "ca.crt"
channels_file = "channels.toml"
```
You can use script which will run series of tests using mqtt-bench
```
cd tools/mqtt-bench/scripts
./mqtt-bench.sh mainflux mainflux.com channels.toml

```

For Example

```
./mqtt-benchmark --config tests/fanin.toml
```
