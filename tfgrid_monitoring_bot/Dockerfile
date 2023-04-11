FROM golang:1.19-alpine

WORKDIR /tfgrid_monitoring_bot

COPY . .

RUN go mod tidy

RUN go build -o tfgridmon main.go 

CMD ./tfgridmon -e .env -w wallets.json
