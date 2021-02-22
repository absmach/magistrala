# This script contains commands to be executed by the CI tool.
NPROC=$(nproc)
GO_VERSION=1.14.4
PROTOC_VERSION=3.12.3
PROTOC_GEN_VERSION=v1.4.2
PROTOC_GOFAST_VERSION=v1.3.1
GRPC_VERSION=v1.29.1

function version_gt() { test "$(printf '%s\n' "$@" | sort -V | head -n 1)" != "$1"; }

update_go() {
    CURRENT_GO_VERSION=$(go version | sed 's/[^0-9.]*\([0-9.]*\).*/\1/')
    if version_gt $GO_VERSION $CURRENT_GO_VERSION; then
        echo "Updating go version from $CURRENT_GO_VERSION to $GO_VERSION ..."
        sudo rm -rf /usr/local/go
        sudo rm -rf /usr/local/golang
        sudo rm -rf /usr/bin/go
        wget https://dl.google.com/go/go$GO_VERSION.linux-amd64.tar.gz
        tar -xvf go$GO_VERSION.linux-amd64.tar.gz
        rm go$GO_VERSION.linux-amd64.tar.gz
        sudo mv go /usr/local
        export GOROOT=/usr/local/go
        export GOPATH=/home/runner/go/src
        export GOBIN=/home/runner/go/bin
        mkdir -p $GOPATH
        mkdir $GOBIN
        # remove other Go version from path
        export PATH=$PATH:/usr/local/go/bin:$GOBIN
    fi
    go version
}

setup_protoc() {
    # Execute `go get` for protoc dependencies outside of project dir.
    cd ..
    export GO111MODULE=on
    echo "Setting up protoc..."
    PROTOC_ZIP=protoc-$PROTOC_VERSION-linux-x86_64.zip
    curl -0L https://github.com/google/protobuf/releases/download/v$PROTOC_VERSION/$PROTOC_ZIP -o $PROTOC_ZIP
    unzip -o $PROTOC_ZIP -d protoc3
    sudo mv protoc3/bin/* /usr/local/bin/
    sudo mv protoc3/include/* /usr/local/include/
    rm -f PROTOC_ZIP

    go get -u github.com/golang/protobuf/protoc-gen-go@$PROTOC_GEN_VERSION \
            github.com/gogo/protobuf/protoc-gen-gofast@$PROTOC_GOFAST_VERSION \
            google.golang.org/grpc@$GRPC_VERSION

    export PATH=$PATH:/usr/local/bin/protoc
    cd mainflux
}

setup_mf() {
    echo "Setting up Mainflux..."
    for p in $(ls *.pb.go); do
        mv $p $p.tmp
    done
    for p in $(ls pkg/*/*.pb.go); do
        mv $p $p.tmp
    done
    make proto
    for p in $(ls *.pb.go); do
        if ! cmp -s $p $p.tmp; then
            echo "Proto file and generated Go file $p are out of sync!"
            exit 1
        fi
    done
    for p in $(ls pkg/*/*.pb.go); do
        if ! cmp -s $p $p.tmp; then
            echo "Proto file and generated Go file $p are out of sync!"
            exit 1
        fi
    done
    make -j$NPROC
}

setup_lint() {
    # binary will be $(go env GOBIN)/golangci-lint
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOBIN) v1.24.0

}

setup() {
    echo "Setting up..."
    update_go
    setup_protoc
    setup_mf
    setup_lint
}

run_test() {
    echo "Running lint..."
    golangci-lint run --no-config --disable-all --enable=golint
    echo "Running tests..."
    echo "" > coverage.txt
    for d in $(go list ./... | grep -v 'vendor\|cmd'); do
        GOCACHE=off
        go test -mod=vendor -v -race -tags test -coverprofile=profile.out -covermode=atomic $d
        if [ -f profile.out ]; then
            cat profile.out >> coverage.txt
            rm profile.out
        fi
    done
}

push() {
    if test -n "$BRANCH_NAME" && test "$BRANCH_NAME" = "master"; then
        echo "Pushing Docker images..."
        make -j$NPROC latest
    fi
}

set -e
setup
run_test
push
