.DEFAULT_GOAL := all

.PHONY: help
help: ## this help message
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: all
all: deps test ## get deps and run tests

.PHONY: deps
deps: ## install deps
	go get -t ./...

.PHONY: test
test: ## run tests
	go test -v ./...

.PHONY: clean
clean: ## clean up
	go clean
	rm -f coverage.out

.PHONY: format
format: ## format code
	go fmt -x *.go

.PHONY: run
run: ## run for debug
	@go run cmd/bean/main.go
