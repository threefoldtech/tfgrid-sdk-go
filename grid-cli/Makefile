PWD := $(shell pwd)
GOPATH := $(shell go env GOPATH)

all: verifiers test

test: 
	@echo "Running Tests"
	go test -v ./...

coverage: clean 
	@echo "Installing gopherbadger" && go get -u github.com/jpoles1/gopherbadger && go install github.com/jpoles1/gopherbadger
	mkdir coverage
	go test -v -vet=off ./... -coverprofile=coverage/coverage.out
	go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@${GOPATH}/bin/gopherbadger -png=false -md="README.md"
	rm coverage.out
	go mod tidy
	
clean:
	rm ./coverage -rf
	rm ./bin -rf

getverifiers:
	@echo "Installing golangci-lint" && go install github.com/golangci/golangci-lint/cmd/golangci-lint
	go mod tidy

lint:
	@echo "Running $@"
	golangci-lint run -c ../.golangci.yml

build:
	@echo "Running $@"
	@go build -ldflags=\
	"-X 'github.com/threefoldtech/tfgrid-sdk-go/grid-cli/cmd.commit=$(shell git rev-parse HEAD)'\
	 -X 'github.com/threefoldtech/tfgrid-sdk-go/grid-cli/cmd.version=$(shell git tag --sort=-version:refname | head -n 1)'"\
	 -o bin/tfcmd main.go
