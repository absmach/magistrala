# Load Test

This SBT project contains load tests written for mainflux platform.

## Setup

In order to run load tests you must have [openjdk8](http://openjdk.java.net/install/) and [sbt](https://www.scala-sbt.org/1.0/docs/Setup.html) installed on your machine.

## Configuration

Tests are configured to use variables from `JAVA_OPTS` presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable | Description                              | Default               |
|----------|------------------------------------------|-----------------------|
| users    | Users service URL                        | http://localhost:8180 |
| clients  | Clients service URL                      | http://localhost:8182 |
| http     | HTTP adapter service URL                 | http://localhost:8185 |
| requests | Number of requests to be sent per second | 100                   |

## Usage

This project contains two simulations:

- `PublishSimulation`
- `CreateAndRetrieveClientSimulation`

To run all tests you will have to run following commands:

```bash
cd <path_to_mainflux_project>/load-test
sbt gatling:test
```

### Publish Simulation

`PublishSimulation` contains load tests for publishing messages. To run this test use following command:

```bash
sbt "gatling:testOnly com.mainflux.loadtest.simulations.PublishSimulation"
```

### Create And Retrieve Client Simulation

`CreateAndRetrieveClientSimulation` contains load tests for creating and retrieving clients. To run this test use following command:

```bash
sbt "gatling:testOnly com.mainflux.loadtest.simulations.CreateAndRetrieveClientSimulation"
```
