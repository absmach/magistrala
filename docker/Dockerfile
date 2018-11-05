FROM golang:1.10-alpine AS builder
ARG SVC_NAME

WORKDIR /go/src/github.com/mainflux/mainflux
COPY . .
RUN apk update \
    && apk add make protobuf git \
    && go get -u github.com/golang/protobuf/protoc-gen-go \
    && go get github.com/gogo/protobuf/protoc-gen-gofast \
    && go get -u google.golang.org/grpc \
    && make proto \
    && make $SVC_NAME \
    && mv build/mainflux-$SVC_NAME /exe

FROM scratch
COPY --from=builder /exe /
ENTRYPOINT ["/exe"]
