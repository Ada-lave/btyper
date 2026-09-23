# Common development and release checks. Run `just --list` to see recipes.
set shell := ["bash", "-cu"]

default:
    @just --list

run:
    go run ./cmd/btyper

fmt:
    find . -path './submodules' -prune -o -type f -name '*.go' -exec gofmt -w {} +

fmt-check:
    @files="$(find . -path './submodules' -prune -o -type f -name '*.go' -exec gofmt -l {} +)"; test -z "$files" || { echo "Go files are not formatted:" >&2; echo "$files" >&2; echo "Run 'just fmt' to fix them." >&2; exit 1; }

vet:
    go vet ./...

lint:
    golangci-lint run ./...

tidy:
    go mod tidy

check: fmt vet lint test test-race vet build syntax

build:
    go build -o /tmp/btyper ./cmd/btyper

install:
    go install ./cmd/btyper

test:
    go test ./...

test-race:
    go test -race ./...

coverage:
    go test -cover ./...

format:
    go fmt ./...

syntax:
    bash -n install.sh
