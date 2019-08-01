FROM erlang:21-alpine AS builder
WORKDIR /mainflux

COPY ./mqtt/verne .
RUN apk add --no-cache git && \
  ./rebar3 compile

FROM vernemq/vernemq:1.8.0-alpine
WORKDIR /mainflux
COPY --from=builder --chown=10000:10000 /mainflux .
