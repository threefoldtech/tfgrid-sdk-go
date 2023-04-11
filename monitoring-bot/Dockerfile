FROM golang:1.19-alpine

WORKDIR /monitoring-bot

COPY . .

RUN go mod tidy

RUN go build -o tfgridmon main.go 

CMD ./tfgridmon -e .env -w wallets.json
