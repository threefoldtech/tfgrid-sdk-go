BIN_PATH = bin/main

.PHONY: build

build:
	@go build -o $(BIN_PATH) ./main.go

run:
	@go run main.go up -f ./grid-compose.yaml
