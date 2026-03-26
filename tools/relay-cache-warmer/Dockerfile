FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY tools/relay-cache-warmer .

RUN go build -o bin/cache-warmer .

FROM alpine:3.19

COPY --from=builder /app/bin/cache-warmer .

ENTRYPOINT [ "/cache-warmer" ]
