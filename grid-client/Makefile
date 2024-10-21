PWD := $(shell pwd)
GOPATH := $(shell go env GOPATH)
integration_tests := $(shell go run scripts/scrap_tests/scrap_tests.go)

all: verifiers test

test: 
	@echo "Running Tests"
	go test -v `go list ./... | grep -v integration_tests`

integration:
	@echo "Running integration tests"
	@for test in ${integration_tests} ; do \
		go test -v ./integration_tests -run $$test ; \
	done

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
