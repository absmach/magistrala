# This script contains commands to be executed by the CI tool.
NPROC=$(nproc)

setup_protoc() {
	echo "Setting up protoc..."
	PROTOC_ZIP=protoc-3.6.1-linux-x86_64.zip
	curl -0L https://github.com/google/protobuf/releases/download/v3.6.1/$PROTOC_ZIP -o $PROTOC_ZIP
	unzip -o $PROTOC_ZIP -d protoc3
	sudo mv protoc3/bin/* /usr/local/bin/
	sudo mv protoc3/include/* /usr/local/include/
	rm -f PROTOC_ZIP
	go get -u github.com/golang/protobuf/protoc-gen-go \
		github.com/gogo/protobuf/protoc-gen-gofast \
		google.golang.org/grpc
	
	git -C $GOPATH/src/github.com/golang/protobuf/protoc-gen-go checkout v1.3.1
	go install github.com/golang/protobuf/protoc-gen-go
	
	git -C $GOPATH/src/github.com/gogo/protobuf/protoc-gen-gofast checkout v1.2.1
	go install github.com/gogo/protobuf/protoc-gen-gofast

	git -C $GOPATH/src/google.golang.org/grpc checkout v1.20.1
	go install google.golang.org/grpc

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
	setup_protoc
	setup_mf
}

run_test() {
	echo "Running tests..."
	echo "" > coverage.txt;
	for d in $(go list ./... | grep -v 'vendor\|cmd'); do
		GOCACHE=off
		go test -v -race -tags test -coverprofile=profile.out -covermode=atomic $d
		if [ -f profile.out ]; then
			cat profile.out >> coverage.txt
			rm profile.out
		fi
	done
}

install_qemu() {
	echo "Installing qemu..."
	MF_PATH=$GOPATH/src/github.com/mainflux/mainflux
	cd $MF_PATH
	sudo apt-get update && sudo apt-get -y install qemu-user-static
	wget https://github.com/multiarch/qemu-user-static/releases/download/v2.11.1/qemu-arm-static.tar.gz  \
		&& tar -xzf qemu-arm-static.tar.gz \
		&& rm qemu-arm-static.tar.gz
	sudo cp qemu-arm-static /usr/bin/
}

push() {
	if test -n "$BRANCH_NAME" && test "$BRANCH_NAME" = "master"; then
		echo "Pushing Docker images..."
		make -j$NPROC latest
		docker system prune -a -f
		install_qemu
		GOARCH=arm GOARM=7 make -j$NPROC latest
		export DOCKER_CLI_EXPERIMENTAL=enabled
		make latest_manifest
	fi
}

set -e
setup
run_test
push
