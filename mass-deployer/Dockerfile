# Build Stage
FROM golang:1.21-alpine as builder

WORKDIR /src/rmb-sdk-go

COPY /rmb-sdk-go .

WORKDIR /src/grid-client

COPY /grid-client .

WORKDIR /src/grid-proxy

COPY /grid-proxy .

WORKDIR /src/mass-deployer

COPY /mass-deployer .

RUN go build -o bin/mass-deployer main.go

# Final Stage
FROM alpine:3.19

WORKDIR /app

COPY --from=builder /src/mass-deployer/bin/mass-deployer .

ENTRYPOINT ["./mass-deployer"]
