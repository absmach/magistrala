# Copyright (c) Abstract Machines
# SPDX-License-Identifier: Apache-2.0

# Build a standalone atom-migration binary so the tool can be run without
# `go run` and without the Go toolchain on the host.
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /atom-migration ./tools/atom-migration

FROM alpine:3.22
RUN adduser -D -u 10001 migrator
WORKDIR /work
COPY --from=build /atom-migration /usr/local/bin/atom-migration
USER migrator
ENTRYPOINT ["atom-migration"]
