# This script contains commands to be executed by the CI tool.
NPROC=$(nproc)
GO_VERSION=1.13
PROTOC_VERSION=3.11.4
PROTOC_GEN_VERSION=v1.3.3
PROTOC_GOFAST_VERSION=v1.3.1
GRPC_VERSION=v1.27.1

function version_gt() { test "$(printf '%s\n' "$@" | sort -V | head -n 1)" != "$1"; }

update_go() {
	CURRENT_GO_VERSION=$(go version | sed 's/[^0-9.]*\([0-9.]*\).*/\1/')
	if version_gt $GO_VERSION $CURRENT_GO_VERSION; then
		echo "Update go version from $CURRENT_GO_VERSION to $GO_VERSION ..."
		sudo rm -rf /usr/local/go
		wget https://dl.google.com/go/go$GO_VERSION.linux-amd64.tar.gz
		sudo mkdir /usr/local/golang/$GO_VERSION && sudo tar -C /usr/local/golang/$GO_VERSION -xzf go$GO_VERSION.linux-amd64.tar.gz
		rm go$GO_VERSION.linux-amd64.tar.gz

		# remove other Go version from path
		export PATH=$(echo $PATH | sed -e 's|:/usr/local/golang/[1-9.]*/go/bin||')

		sudo ln -fs /usr/local/golang/$GO_VERSION/go/bin/go /usr/local/bin/go

		# setup GOROOT
		export GOROOT="/usr/local/golang/$GO_VERSION/go"

		# add new go installation to PATH
		export PATH="$PATH:/usr/local/golang/$GO_VERSION/go/bin"
	fi
	go version
}

setup_protoc() {
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
}

setup_mf() {
	echo "Setting up Mainflux..."
	MF_PATH=$GOPATH/src/github.com/mainflux/mainflux
	if test $PWD != $MF_PATH; then
		mkdir -p $MF_PATH
		mv ./* $MF_PATH
	fi
	cd $MF_PATH
	for p in $(ls *.pb.go); do
		mv $p $p.tmp
	done
	make proto
	for p in $(ls *.pb.go); do
		if ! cmp -s $p $p.tmp; then
			echo "Proto file and generated Go file $p are out of sync!"
			exit 1
		fi
	done
	make -j$NPROC
}

setup() {
	echo "Setting up..."
	update_go
	setup_protoc
	setup_mf
}

run_test() {
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
