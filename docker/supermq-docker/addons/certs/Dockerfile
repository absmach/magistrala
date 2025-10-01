# Copyright (c) Abstract Machines
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.25-alpine AS builder

ARG SVC
ARG GOARCH
ARG GOARM
ARG VERSION
ARG COMMIT
ARG TIME

WORKDIR /go/src/github.com/absmach/certs

COPY . .

RUN apk update \
    && apk add make upx\
    && make $SVC \
    && upx build/$SVC \
    && mv build/$SVC /exe

FROM scratch

# Required for certs service
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /exe /
ENTRYPOINT ["/exe"]
