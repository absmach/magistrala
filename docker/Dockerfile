###
# Mainflux Dockerfile
###

FROM golang:alpine
MAINTAINER Mainflux

###
# Install
###

RUN apk update && apk add git && rm -rf /var/cache/apk/*

# Copy the local package files to the container's workspace.
ADD .. /go/src/github.com/mainflux/mainflux

RUN mkdir -p /etc/mainflux
COPY config/config-docker.toml /etc/mainflux/config.toml

# Get and install the dependencies
RUN go get github.com/mainflux/mainflux

###
# Run main command from entrypoint and parameters in CMD[]
###
CMD ["/etc/mainflux/config.toml"]

# Run mainflux command by default when the container starts.
ENTRYPOINT ["/go/bin/mainflux"]

