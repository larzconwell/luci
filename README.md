# `luci`

[![Lint/Test](https://github.com/larzconwell/luci/workflows/test-lint/badge.svg)](https://github.com/larzconwell/luci/actions)
[![Package Reference](https://pkg.go.dev/badge/github.com/larzconwell/luci.svg)](https://pkg.go.dev/github.com/larzconwell/luci)

Go module to create web services quickly and painlessly.

# Using `luci`

## Installing
```shell
go get github.com/larzconwell/luci
```

## Example
See [the example project](https://github.com/larzconwell/luci/tree/main/example) for a simple example using all of the functionality `luci` provides.

# Developing `luci`

## Prerequisites
- Go 1.21
- `golangci-lint` 1.55

## Setting up hooks
```shell
ln -s $(pwd)/.hooks/pre-commit .git/hooks/pre-commit
```

## Testing and linting
Make targets exist for testing and linting
- `make test`
- `make test-race`
- `make lint`
