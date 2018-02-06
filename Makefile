BUILD_DIR=build

all: manager http writer coap
.PHONY: all manager http writer coap

manager:
	go build -o ${BUILD_DIR}/mainflux-manager cmd/manager/main.go

http:
	go build -o ${BUILD_DIR}/mainflux-http cmd/http/main.go

writer:
	go build -o ${BUILD_DIR}/mainflux-writer cmd/writer/main.go

coap:
	go build -o ${BUILD_DIR}/mainflux-coap cmd/coap/main.go


clean:
	rm -rf ${BUILD_DIR}

install:
	cp ${BUILD_DIR}/* $(GOBIN)
