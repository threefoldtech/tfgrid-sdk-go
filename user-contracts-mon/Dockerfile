FROM golang:1.21-alpine as builder

WORKDIR /src

COPY . .

WORKDIR /src/user-contracts-mon

RUN go mod tidy 

RUN go build -o contractsmon main.go


FROM alpine

WORKDIR /app

COPY --from=builder /src/user-contracts-mon/contractsmon .

COPY --from=builder /src/user-contracts-mon/.env .

CMD ./contractsmon -e .env 

