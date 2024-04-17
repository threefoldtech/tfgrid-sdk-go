FROM golang:1.21-alpine as builder

WORKDIR /src/activation-service

COPY /activation-service .

RUN go build -o bin/activation-service main.go

FROM alpine:3.19

WORKDIR /app

COPY --from=builder /src/activation-service/bin/activation-service .

ENTRYPOINT [ "./activation-service" ]

EXPOSE 3000
