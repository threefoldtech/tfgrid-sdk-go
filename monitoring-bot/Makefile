OUT=$(shell realpath -m bin)
GOPATH=$(shell go env GOPATH)
branch=$(shell git symbolic-ref -q --short HEAD || git describe --tags --exact-match)
revision=$(shell git rev-parse HEAD)
dirty=$(shell test -n "`git diff --shortstat 2> /dev/null | tail -n1`" && echo "*")
ldflags='-w -s -X $(version).Branch=$(branch) -X $(version).Revision=$(revision) -X $(version).Dirty=$(dirty)'

all: lint build test

test:
	go test -v -vet=off ./...

benchmarks:
	go test -v -vet=off ./... -bench=. -count 1 -benchtime=10s -benchmem -run=^#

getverifiers:
	@echo "Installing golangci-lint" && go install github.com/golangci/golangci-lint/cmd/golangci-lint
	go mod tidy

lint:
	@echo "Running $@"
	golangci-lint run -c ../.golangci.yml

build:
	@echo "Running $@"
	@go build -ldflags=\
	"-X 'github.com/threefoldtech/tfgrid-sdk-go/monitoring-bot/cmd.commit=$(shell git rev-parse HEAD)'\
	 -X 'github.com/threefoldtech/tfgrid-sdk-go/monitoring-bot/cmd.version=$(shell git tag --sort=-version:refname | head -n 1)'"\
	 -o bin/monitoring-bot main.go

coverage: clean 
	@echo "Installing gopherbadger" && go get -u github.com/jpoles1/gopherbadger && go install github.com/jpoles1/gopherbadger
	mkdir coverage
	go test -v -vet=off ./... -coverprofile=coverage/coverage.out
	go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@${GOPATH}/bin/gopherbadger -png=false -md="README.md"
	rm coverage.out
	go mod tidy

testrace: verifiers
	go test -v -race -vet=off ./...

run:
	go run main.go

clean:
	rm ./coverage -rf
	rm ./bin -rf
