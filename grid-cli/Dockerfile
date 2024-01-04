FROM golang:1.21-alpine as builder

WORKDIR /src/rmb-sdk-go

COPY /rmb-sdk-go .

WORKDIR /src/grid-client

COPY /grid-client .

WORKDIR /src/grid-proxy

COPY /grid-proxy .

WORKDIR /src/grid-cli

COPY /grid-cli .

RUN go build -o bin/grid-cli main.go

FROM alpine:3.19

WORKDIR /app

COPY --from=builder /src/grid-cli/bin/grid-cli .

ENTRYPOINT [ "./grid-cli" ]
