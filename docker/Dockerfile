# Copyright (c) Abstract Machines
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.26.1-alpine3.22 AS builder
ARG SVC
ARG GOARCH
ARG GOARM
ARG VERSION
ARG COMMIT
ARG TIME
ARG BUILD_TAGS

WORKDIR /go/src/github.com/absmach/magistrala
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${GOARCH} GOARM=${GOARM} \
    go build \
    -trimpath \
    -mod=readonly \
    -buildvcs=false \
    -tags "${BUILD_TAGS}" \
    -ldflags "-s -w \
    -X 'github.com/absmach/magistrala.BuildTime=${TIME}' \
    -X 'github.com/absmach/magistrala.Version=${VERSION}' \
    -X 'github.com/absmach/magistrala.Commit=${COMMIT}'" \
    -o /exe cmd/${SVC}/main.go

FROM scratch
# Certificates are needed so that mailing util can work.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /exe /
ENTRYPOINT ["/exe"]
