# This script contains commands to be executed by the CI tool.

setup_protoc() {
	echo "Setup protoc..."
	PROTOC_ZIP=protoc-3.6.1-linux-x86_64.zip
	curl -0L https://github.com/google/protobuf/releases/download/v3.6.1/$PROTOC_ZIP -o $PROTOC_ZIP
	unzip -o $PROTOC_ZIP -d protoc3
	sudo mv protoc3/bin/* /usr/local/bin/
	sudo mv protoc3/include/* /usr/local/include/
	rm -f PROTOC_ZIP
	go get -u github.com/golang/protobuf/protoc-gen-go \
		github.com/gogo/protobuf/protoc-gen-gofast \
		google.golang.org/grpc
	export PATH=$PATH:/usr/local/bin/protoc
}

setup_mf() {
	echo "Setup Mainflux..."
	go get -d github.com/mainflux/mainflux
	cd $GOPATH/src/github.com/mainflux/mainflux
	make proto
}

setup() {
	echo "Setting up..."
	setup_protoc
	setup_mf
}

run_test() {
	echo "Running tests..."
	set -e; echo "" > coverage.txt; for d in $(go list ./... | grep -v 'vendor\|cmd'); do GOCACHE=off go test -v -race -tags test -coverprofile=profile.out -covermode=atomic $d; if [ -f profile.out ]; then cat profile.out >> coverage.txt; rm profile.out; fi done
}

push() {
	echo "Pushing Docker images..."
	if test -n "$BRANCH_NAME" && test "$BRANCH_NAME" = "master"; then
	make latest
	fi
}

setup
run_test
push
