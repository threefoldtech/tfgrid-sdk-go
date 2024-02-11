FROM golang:1.21-alpine as builder

WORKDIR /app

COPY . .

RUN go build -o bin/cache-warmer .

FROM alpine:3.19

COPY --from=builder /app/bin/cache-warmer .

ENTRYPOINT [ "/cache-warmer" ]