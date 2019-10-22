all: test integration

test:
	go test ./...

integration:
	go test -v -tags=integration ./uatest/...

install-py-opcua:
	pip3 install opcua
