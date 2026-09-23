#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

build_dir="$(mktemp -d)"
trap 'rm -rf "$build_dir"' EXIT

go test ./...
go test -race ./...
go vet ./...
go build -o "$build_dir/btyper" ./cmd/btyper

bash -n install.sh packaging/aur/PKGBUILD
ruby -c Formula/btyper.rb
python3 packaging/verify.py

if command -v makepkg >/dev/null 2>&1; then
  diff -u packaging/aur/.SRCINFO <(cd packaging/aur && makepkg --printsrcinfo)
else
  echo 'Skipping makepkg metadata check (makepkg is unavailable)'
fi
