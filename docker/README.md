# Docker Composition

Configure environment variables and run Mainflux Docker Composition.

*Note**: `docker-compose` uses `.env` file to set all environment variables. Ensure that you run the command from the same location as .env file.

## Installation

Follow the [official documentation](https://docs.docker.com/compose/install/).


## Usage

Run following commands from project root directory.

```
docker-compose -f docker/docker-compose.yml up
```

```
docker-compose -f docker/addons/<path>/docker-compose.yml  up
```

