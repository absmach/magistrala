.PHONY: test
test:
	go test -mod=vendor -v --race -covermode=atomic -coverprofile cover.txt ./...

.PHONY: cover-html
cover-html: test
	go tool cover -html cover.txt -o cover.html

.PHONY: lint
lint:
	golangci-lint cache clean && golangci-lint run --enable-all --disable misspell --disable funlen --disable gofumpt --disable ireturn --disable cyclop --disable lll --disable gosec --disable gochecknoglobals --disable paralleltest --disable wsl --disable gocognit --disable depguard

godoc-serve:
	godoc -http=:6060

.PHONY: install-precommit
install-precommit:
	pre-commit install

.PHONY: precommit
precommit:
	pre-commit run --all-files -v

.PHONY: update-precommit
update-precommit:
	pre-commit autoupdate
