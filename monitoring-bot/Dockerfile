FROM golang:1.21-alpine as builder

WORKDIR /src/rmb-sdk-go

COPY /rmb-sdk-go .

WORKDIR /src/grid-proxy

COPY /grid-proxy .

WORKDIR /src/monitoring-bot

COPY /monitoring-bot .

RUN go build -o bin/monitoring-bot main.go

FROM alpine:3.19

WORKDIR /app

COPY --from=builder /src/monitoring-bot/bin/monitoring-bot .

ENTRYPOINT [ "./monitoring-bot" ]
