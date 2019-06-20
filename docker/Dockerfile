FROM golang:1.10-alpine AS builder
ARG SVC
ARG GOARCH
ARG GOARM

WORKDIR /go/src/github.com/mainflux/mainflux
COPY . .
RUN apk update \
    && apk add make\
    && make $SVC \
    && mv build/mainflux-$SVC /exe

FROM scratch
COPY --from=builder /exe /
ENTRYPOINT ["/exe"]
