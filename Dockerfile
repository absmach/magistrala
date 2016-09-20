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
ADD . /go/src/github.com/mainflux/mainflux-lite

RUN mkdir -p /config/lite
COPY config/config-docker.yml /config/lite/config.yml

# Get and install the dependencies
RUN go get github.com/mainflux/mainflux-lite

###
# Run main command from entrypoint and parameters in CMD[]
###
CMD ["/config/lite/config.yml"]

# Run mainflux command by default when the container starts.
ENTRYPOINT ["/go/bin/mainflux-lite"]

