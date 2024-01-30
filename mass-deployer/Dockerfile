# Build Stage
FROM golang:1.21-alpine as builder

WORKDIR /src/rmb-sdk-go

COPY /rmb-sdk-go .

WORKDIR /src/grid-client

COPY /grid-client .

WORKDIR /src/grid-proxy

COPY /grid-proxy .

WORKDIR /src/mass-deployer

COPY /mass-deployer/go.mod .

RUN go mod download

COPY /mass-deployer/ .

RUN go build -o bin/tfrobot main.go

# Final Stage
FROM alpine:3.19

WORKDIR /app

COPY --from=builder /src/mass-deployer/bin/tfrobot .

ENTRYPOINT ["./tfrobot"]
