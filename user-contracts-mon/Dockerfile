FROM golang:1.21-alpine

WORKDIR /contracts-mon-bot

COPY . .

WORKDIR /contracts-mon-bot/user-contracts-mon

RUN go mod tidy

RUN go build -o contractsmon main.go

CMD ./contractsmon -e .env 
