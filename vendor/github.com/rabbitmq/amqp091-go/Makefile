.DEFAULT_GOAL := list

# Insert a comment starting with '##' after a target, and it will be printed by 'make' and 'make list'
list: ## list Makefile targets
	@echo "The most used targets: \n"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'


fmt: ## Run go fmt against code
	go fmt ./...


vet: ## Run go vet against code
	go vet ./...

tests: ## Run all tests and requires a running rabbitmq-server
	env AMQP_URL=amqp://guest:guest@127.0.0.1:5672/ go test -v -tags integration
