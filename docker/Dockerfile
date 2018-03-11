FROM golang:1.9-alpine AS builder
ARG SVC_NAME
RUN apk update \
    && apk add git \
    && go get github.com/mainflux/mainflux \
    && cd /go/src/github.com/mainflux/mainflux/cmd/$SVC_NAME \
    && CGO_ENABLED=0 GOOS=linux go build -ldflags "-s" -a -installsuffix cgo -o /exe

FROM scratch
COPY --from=builder /exe /
ENTRYPOINT ["/exe"]
