# This script contains commands to be executed by the CI tool.

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
			echo "Proto file and generated Go file $p are out of cync!"
			exit 1
		fi
	done
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

push() {
	if test -n "$BRANCH_NAME" && test "$BRANCH_NAME" = "master"; then
		echo "Pushing Docker images..."
		make latest
	fi
}

set -e
setup
run_test
push
