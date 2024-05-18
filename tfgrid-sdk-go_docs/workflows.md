# Workflows

## Grid-cli

- `grid-cli` has 3 workflows
  
### Lint workflow

Runs `gofmt` and `golangci-lint` with every push.

### Test workflow

Runs `unit tests` with every push.

### Release workflow

Uses `go-releaser` on every push with tag that starts with `v`

---

## Grid-client

- `grid-client` has 2 workflows
  
### Lint workflow

Runs `gofmt` and `golangci-lint` with every push.

### Test workflow

Runs `unit tests` with every push.

### Integration test workflow

Runs `integration tests` daily

---

## Grid-proxy

- `grid-proxy` has 4 workflows
  
### Build and lint workflow

Runs `golangci-lint` and builds gridproxy server and docker image with every push.

### Unit test workflow

Runs `unit tests` with every push.

### Integration test workflow

Runs `integration tests` with every push.

### Release workflow

Uses `docker` and runs on every published release

---

## Gridify

- `gridify` has 3 workflows
  
### Lint workflow

Runs `gofmt` and `golangci-lint` with every push.

### Test workflow

Runs `unit tests` with every push.

### Release workflow

Uses `go-releaser` on every push with tag that starts with `v`

---

## Monitoring bot

- `monitoring bot` has 3 workflows
  
### Lint workflow

Runs `gofmt` and `golangci-lint` with every push.

### Test workflow

Runs `unit tests` with every push.

### Release workflow

Uses `go-releaser` on every push with tag that starts with `v`

---

## RMB-sdk-go

- `rmb-sdk-go` has a workflow
  
### Lint and test workflow

Runs `gofmt`, `golangci-lint` and `unit tests` with every push.
