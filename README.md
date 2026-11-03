![luci](https://raw.githubusercontent.com/larzconwell/luci/main/luci.svg)

[![License](https://img.shields.io/badge/License-BSD_2--Clause_+_Patent-blue.svg)](https://github.com/larzconwell/luci/blob/main/LICENSE)
[![Lint/Test](https://github.com/larzconwell/luci/actions/workflows/checks.yml/badge.svg)](https://github.com/larzconwell/luci/actions)
[![Package Reference](https://pkg.go.dev/badge/github.com/larzconwell/luci.svg)](https://pkg.go.dev/github.com/larzconwell/luci)

`luci` is a Go module to create web services quickly and painlessly.

# Using `luci`

## Installing
```shell
go get github.com/larzconwell/luci/v2
```

## Example
See [the example project](https://github.com/larzconwell/luci/tree/main/example) for a simple example using all of the functionality `luci` provides.

Running `mise example` will also build and run the example project.

# Developing `luci`

1. `mise install` - Install build dependencies
1. `mise setup` - Setup githooks
1. `mise lint` - Lint code
1. `mise test` - Run tests
1. `mise testrace` - Run tests with race detector enabled
