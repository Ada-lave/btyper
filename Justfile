# Common development and release checks. Run `just --list` to see recipes.
set shell := ["bash", "-cu"]

default:
    @just --list

run:
    go run ./cmd/btyper

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

vet:
    go vet ./...

format:
    go fmt ./...

syntax:
    bash -n install.sh packaging/aur/PKGBUILD
    ruby -c Formula/btyper.rb

recipes:
    python3 packaging/verify.py
    if command -v makepkg >/dev/null 2>&1; then diff -u packaging/aur/.SRCINFO <(cd packaging/aur && makepkg --printsrcinfo); else echo 'Skipping makepkg metadata check (makepkg is unavailable)'; fi

srcinfo:
    cd packaging/aur && makepkg --printsrcinfo > .SRCINFO

check: test test-race vet build syntax recipes
