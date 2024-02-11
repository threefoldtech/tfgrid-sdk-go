GOPATH := $(shell go env GOPATH)

all: verifiers

getverifiers:
	@echo "Installing golangci-lint" && go install github.com/golangci/golangci-lint/cmd/golangci-lint
	go mod tidy

lint:
	@echo "Running $@"
	golangci-lint run -c ../.golangci.yml

build:
	@echo "Running $@"
	@go build -ldflags=\
	"-X 'main.commit=$(shell git rev-parse HEAD)'\
	 -X 'main.version=$(shell git tag --sort=-version:refname | head -n 1)'"\
	 -o bin/cache-warmer .

clean:
	rm ./bin -rf

