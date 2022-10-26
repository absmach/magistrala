FROM ubuntu:22.04 AS build
RUN apt-get update \
    && apt-get install -y gcc make git curl file
RUN git clone https://github.com/udhos/update-golang.git \
    && cd update-golang \
    && ./update-golang.sh \
    && ln -s /usr/local/go/bin/go /usr/bin/go
WORKDIR $GOPATH/src/github.com/plgd-dev/go-coap
COPY go.mod go.sum ./
RUN go mod download
COPY . .