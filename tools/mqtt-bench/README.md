# MQTT benchmarking tool

A simple MQTT (broker) benchmarking tool for Mainflux platform. ( based on github.com/krylovsk/mqtt-benchmark )


The tool supports multiple concurrent clients, publishers and subscribers configurable message size, etc:

```
cd benchmark
go build  -o mqtt-benchmark *.go

> mqtt-benchmark --help
Usage of mqtt-benchmark:
  -broker="tcp://localhost:1883": MQTT broker endpoint as scheme://host:port
  -clients=10: Number of clients to start
  -count=100: Number of messages to send per client
  -format="text": Output format: text|json
  -password="": MQTT password (empty if auth disabled)
  -qos=1: QoS for published messages
  -quiet=false : Suppress logs while running (except errors and the result)
  -size=100: Size of the messages payload (bytes
  -subs=10 number of subscribers
  -pubs=10 number of publishers
  -config=connections.json , file with mainflux channels
  -mtls=false, use mtls
  -ca=ca.crt, use mqtts, pass ca to server validate certificate
```

Two output formats supported: human-readable plain text and JSON.

Before use you need a channels.toml  you can use tools/provision/main.go to create
channels for testing

Example use and output:

```
go build -o mqtt-benchmark *.go

without mtls
./mqtt-benchmark --broker tcp://localhost:1883 --count 100 --size 100  --qos 0 --format text   --subs 100 --pubs 0 --channels channels.toml

with mtls
./mqtt-benchmark --broker tcps://localhost:8883 --count 100 --size 100  --qos 0 --format text   --subs 100 --pubs 0 --channels channels.toml --mtls -ca ca.crt
```


You can use config.toml to create tests with this tool
./mqtt-benchmark --config config.toml it will read params from config.toml


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

For Example

```
./mqtt-benchmark --config tests/fanin.toml
```
