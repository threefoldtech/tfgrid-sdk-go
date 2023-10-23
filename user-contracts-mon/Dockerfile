FROM golang:1.21-alpine

ARG env 

WORKDIR /contracts-mon-bot

COPY . .

RUN go mod tidy

RUN go build -o contractsmon main.go

CMD ./contractsmon -e ${env}
