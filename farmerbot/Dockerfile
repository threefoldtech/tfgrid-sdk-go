FROM golang:1.21-alpine as builder

WORKDIR /src/rmb-sdk-go

COPY /rmb-sdk-go .

WORKDIR /src/farmerbot

COPY /farmerbot .

RUN go build -o bin/farmerbot main.go

FROM alpine:3.19

WORKDIR /app

COPY --from=builder /src/farmerbot/bin/farmerbot .

COPY --from=builder /src/farmerbot/.env .

COPY --from=builder /src/farmerbot/config.yml .

CMD ./farmerbot run -c config.yml -e .env

