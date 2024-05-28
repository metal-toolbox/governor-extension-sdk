all: lint test
PHONY: test coverage lint golint clean vendor docker-up docker-down unit-test
GOOS=linux
# use the working dir as the app name, this should be the repo name
APP_NAME=$(shell basename $(CURDIR))

test: | unit-test

unit-test:
	@echo Running unit tests...
	@go test -cover -short -tags testtools ./...

lint: golint

golint: | vendor
	@echo Linting Go files...
	@golangci-lint run --build-tags "-tags testtools"

bin/${APP_NAME}:
	@go mod download
	@CGO_ENABLED=0 GOOS=linux go build -mod=readonly -v -o bin/${APP_NAME}

build: bin/${APP_NAME}

clean: docker-clean
	@echo Cleaning...
	@rm -f app
	@rm -rf ./dist/
	@rm -rf coverage.out
	@go clean -testcache

vendor:
	@go mod download
	@go mod tidy
