FROM golang:1.21-alpine as builder

WORKDIR /src/rmb-sdk-go

COPY /rmb-sdk-go .

WORKDIR /src/grid-client

COPY /grid-client .

WORKDIR /src/grid-proxy

COPY /grid-proxy .

WORKDIR /src/gridify

COPY /gridify .

RUN go build -o bin/gridify main.go

FROM alpine:3.19

WORKDIR /app

COPY --from=builder /src/gridify/bin/gridify .

ENTRYPOINT [ "./gridify" ]
