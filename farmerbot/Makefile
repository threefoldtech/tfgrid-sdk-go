GOPATH := $(shell go env GOPATH)

all: lint test

test: 
	@echo "Running Tests"
	go test -v ./...

build:
	@echo "Running $@"
	@go build -ldflags=\
	"-X 'github.com/threefoldtech/tfgrid-sdk-go/farmerbot/version.Version=$(shell git tag --sort=-version:refname | head -n 1)'"\
	 -o bin/farmerbot main.go


lint: 
	@echo "Running $@"
	golangci-lint run -c ../.golangci.yml --timeout 10m

coverage: clean 
	mkdir coverage
	go test -v -vet=off ./... -coverprofile=coverage/coverage.out
	go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@${GOPATH}/bin/gopherbadger -png=false -md="README.md"
	rm coverage.out

clean:
	rm ./coverage -rf
	rm ./bin -rf
