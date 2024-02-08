# Relay Cache Warmer

Relay Cache Warmer is a software used to warm Relay's Redis cache with twins fetched from GraphQl periodically to avoid the Relay slowdown due to fetching twins from TFChain on RMB calls.

## Usage

Run:

```bash
cache-warmer --interval 10 --graphql https://graphql.grid.tf/graphql --redis-url redis://localhost:6379
```

## Build

You need Go(1.21) and make.
Run:

```bash
make build
```

## Flags

```text
  --graphql string
        graphql url (default "https://graphql.grid.tf/graphql")
  --interval uint
        cache warming interval (default 10)
  --redis-password string
        redis password
  --redis-url string
        redis url (default "redis://localhost:6379")
  --version
        print version and exit
```
