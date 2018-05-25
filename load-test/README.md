# Load Test

This project contains platform's load tests.

## Prerequisites

To run the tests you must have [OpenJDK8](http://openjdk.java.net/install/) and
[SBT](https://www.scala-sbt.org/1.0/docs/Setup.html) installed on your machine.

## Configuration

Tests are configured to use variables from `JAVA_OPTS` presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable | Description                              | Default               |
|----------|------------------------------------------|-----------------------|
| users    | Users service URL                        | http://localhost:8180 |
| things   | Things service URL                       | http://localhost:8182 |
| http     | HTTP adapter service URL                 | http://localhost:8185 |
| ws       | WebSocket adapter service URL            | localhost:8186        |
| requests | Number of requests to be sent per second | 100                   |

## Usage

This project contains three simulations:

- `CreateAndRetrieveThings`
- `PublishHttpMessages`
- `PublishWsMessages`

To run all tests you will have to run following commands:

```bash
cd <path_to_mainflux_project>/load-test
sbt gatling:test
```

### Things creation and retrieval

`CreateAndRetrieveThings` contains load tests for creating and retrieving things.
Execute the following command to run the suite:

```bash
sbt "gatling:testOnly com.mainflux.loadtest.CreateAndRetrieveThings"
```

### Message publishing over HTTP adapter

`PublishHttpMessages` contains load tests for publishing messages over HTTP.
Execute the following command to run the suite:

```bash
sbt "gatling:testOnly com.mainflux.loadtest.PublishHttpMessages"
```

### Message publishing over WebSocket adapter

`PublishWsMessages` contains load tests for publishing messages over WebSocket.
Execute the following command to run the suite:

```bash
sbt "gatling:testOnly com.mainflux.loadtest.PublishWsMessages"
```
