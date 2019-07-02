# KEP tools

## kepview

`kepview` is a command that interfaces with [Kubernetes Enhancement Proposals](https://github.com/kubernetes/enhancements).

[![asciicast](https://asciinema.org/a/GySrSLkHeVaOrj2afNtXtYlEV.svg)](https://asciinema.org/a/GySrSLkHeVaOrj2afNtXtYlEV)

## kepval

`kepval` is a tool that checks the YAML metadata in a KEP and returns validation
errors.

## Getting started

1. Clone the enhancements `git clone https://github.com/kubernetes/enhancements.git`
2. Install `kepview`: `go get github.com/chuckha/kepview/cmd/kepview`
3. Install `kepval`: `go get github.com/chuckha/kepview/cmd/kepval`
3. Run `kepview`
4. Run `kepval <path to kep.md>`

## Development

1. Follow the getting started guide
2. Run the tests with `go test -cover ./...`
