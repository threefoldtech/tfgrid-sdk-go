.DEFAULT_GOAL := help
PQ_HOST = $(shell docker inspect -f '{{range.NetworkSettings.Networks}}{{.IPAddress}}{{end}}' postgres)
PQ_CONTAINER = postgres
count = 3

install-swag: 
	@go install github.com/swaggo/swag/cmd/swag@v1.8.12;

.PHONY: docs
docs: install-swag ## Create the swagger docs
	@go mod vendor; 
	@$(shell go env GOPATH)/bin/swag init -g internal/explorer/server.go --parseVendor;
	@rm -rf vendor;

build: ## Bulil the project
	@cd cmds/proxy_server && CGO_ENABLED=0 GOOS=linux go build -ldflags "-w -s -X main.GitCommit=$(shell git describe --always --abbrev=0) -extldflags '-static'"  -o server

db-start: ## Start postgres server on a docker container
	@docker run --rm --name $(PQ_CONTAINER) \
		-e POSTGRES_USER=postgres \
		-e POSTGRES_PASSWORD=postgres \
		-e POSTGRES_DB=tfgrid-graphql \
		-d \
		postgres

db-fill: ## Fill the database with a randomly generated data
	@echo "Loading...   It takes some time."
	@cd ./tools/db &&\
	go run . \
		--postgres-host $(PQ_HOST) \
		--postgres-db tfgrid-graphql \
		--postgres-password postgres \
		--postgres-user postgres \
		--reset \
		--seed 13

db-update:
	@echo "Updating node uptimes"
	@psql postgresql://postgres:postgres@$(PQ_HOST):5432/tfgrid-graphql < ./internal/explorer/db/helpers.sql

db-dump: ## Load a dump of the database 		(Args: `p=<path/to/file.sql`)
	@docker cp $(p) postgres:/dump.sql;
	@docker exec $(PQ_CONTAINER) bash -c "psql -U postgres  -d tfgrid-graphql < ./dump.sql"

db-stop: ## Stop the database container if running
	@if [ ! "$(shell docker ps | grep '$(PQ_CONTAINER)' )" = "" ]; then \
		docker stop postgres; \
	fi
db-refill: db-stop db-start sleep db-fill
server-start: ## Start the proxy server			(Args: `m=<MNEMONICS>`)
	@go run cmds/proxy_server/main.go \
		-no-cert \
		-no-indexer \
		--address :8080 \
		--log-level debug \
		--sql-log-level 4 \
		--postgres-host $(PQ_HOST) \
		--postgres-db tfgrid-graphql \
		--postgres-password postgres \
		--postgres-user postgres \
		--mnemonics "$(m)" ;

all-start: db-stop db-start sleep db-fill server-start ## Full start of the database and the server (Args: `m=<MNEMONICS>`)

sleep:
	@sleep 5

test-queries: ## Run all queries tests
	@cd tests/queries/ &&\
	go test -v \
		-parallel 20 \
		-no-modify \
		--seed 13 \
		--postgres-host $(PQ_HOST) \
		--postgres-db tfgrid-graphql \
		--postgres-password postgres \
		--postgres-user postgres \
		--endpoint http://localhost:8080 \
		-count $(count) 

test-query: ## Run specific test query 			(Args: `t=TestName`).
	@cd tests/queries/ &&\
	go test -v \
		-parallel 10 \
		-no-modify \
		--seed 13 \
		--postgres-host $(PQ_HOST) \
		--postgres-db tfgrid-graphql \
		--postgres-password postgres \
		--postgres-user postgres \
		--endpoint http://localhost:8080 \
		-count $(count) \
		-run $(t)

test-unit: ## Run only unit tests
	@go test -v ./pkg/client &&\
	go test ./internal/explorer/db/ -timeout 0

test-all: test-unit test-queries ## Run all unit/queries tests

.PHONY: help
help:
	@printf "%s\n" "Avilable targets:"
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m  make %-15s\033[0m %s\n", $$1, $$2}'

getverifiers:
	@echo "Installing staticcheck" && go get -u honnef.co/go/tools/cmd/staticcheck && go install honnef.co/go/tools/cmd/staticcheck
	@echo "Installing gocyclo" && go get -u github.com/fzipp/gocyclo/cmd/gocyclo && go install github.com/fzipp/gocyclo/cmd/gocyclo
	@echo "Installing deadcode" && go get -u github.com/remyoudompheng/go-misc/deadcode && go install github.com/remyoudompheng/go-misc/deadcode
	@echo "Installing misspell" && go get -u github.com/client9/misspell/cmd/misspell && go install github.com/client9/misspell/cmd/misspell
	@echo "Installing golangci-lint" && go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.50.1
	go mod tidy

verifiers: fmt lint cyclo deadcode spelling staticcheck

checks: verifiers

fmt:
	@echo "Running $@"
	@gofmt -d .

lint:
	@echo "Running $@"
	@$(shell go env GOPATH)/bin/golangci-lint run

cyclo:
	@echo "Running $@"
	@$(shell go env GOPATH)/bin/gocyclo -over 100 .

deadcode:
	@echo "Running $@"
	@$(shell go env GOPATH)/bin/deadcode -test $(shell go list ./...) || true

spelling:
	@echo "Running $@"
	@$(shell go env GOPATH)/bin/misspell -i monitord -error `find .`

staticcheck:
	@echo "Running $@"
	@$(shell go env GOPATH)/bin/staticcheck -- ./...

bench:
	@cd tests/queries/ &&\
	go test -v -bench Bench -run notests -count 5\
		--seed 13 \
		--postgres-host $(PQ_HOST) \
		--postgres-db tfgrid-graphql \
		--postgres-password postgres \
		--postgres-user postgres \
		--endpoint http://localhost:8080