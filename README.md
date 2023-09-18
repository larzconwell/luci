# `luci`

[![Lint/Test](https://github.com/larzconwell/luci/workflows/test-lint/badge.svg)](https://github.com/larzconwell/luci/actions)

Go module to create web services quickly and painlessly.

# Developing `luci`

## Prerequisites
- Go
- `golangci-lint`

## Setting up hooks
```
ln -s $(pwd)/.hooks/pre-commit .git/hooks/pre-commit
```

## Testing and linting
Make targets exist for testing and linting
- `make test`
- `make test-race`
- `make lint`
