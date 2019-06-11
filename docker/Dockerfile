FROM golang:1.10-alpine AS builder
ARG SVC_NAME
ARG GOARCH
ARG GOARM

WORKDIR /go/src/github.com/mainflux/mainflux
COPY . .
RUN apk update \
    && apk add make\
    && make $SVC_NAME \
    && mv build/mainflux-$SVC_NAME /exe

FROM scratch
COPY --from=builder /exe /
ENTRYPOINT ["/exe"]
