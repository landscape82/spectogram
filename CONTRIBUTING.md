# Contributing

Thanks for considering a contribution to this project.

## Getting started

```bash
go mod tidy
go build ./cmd/...
go test ./... -race -cover
```

See the [Development & Testing](README.md#-development--testing) section of the README for the full set of checks CI runs.

## Submitting changes

1. Fork the repo and create a branch from `main`.
2. Make your changes, keeping `gofmt`, `go vet`, and the test suite passing.
3. Open a pull request describing what changed and why.
