---
name: btyper-release
description: Prepare, publish, and verify a btyper release from this repository. Use for release planning, version tags, GitHub release assets, package recipe updates, and release smoke tests; not for routine feature work.
---

# btyper release

Use the current `docs/release-checklist.md`, `docs/install.md`, `docs/stable-contract.md`, `.github/workflows/release.yml`, and package recipes as the source of truth. This skill captures the v1.0.0 release sequence; check those files for changes before each release.

## Establish the release state

- Inspect the working tree, branch, tags, remote, current version, milestone/issues, and CI. Preserve unrelated local changes. Prepare the candidate from the intended clean `main` commit; do not fold unrelated uncommitted work into it.
- Determine the target version and release scope from issues and compatibility impact. Update the stable contract, English and Russian READMEs, profile guide, and release notes when the changes warrant it. For schema or backup format changes, add round-trip and old-version fixture coverage.

## Verify the candidate

- Run `go test ./...`, `go test -race ./...`, `go vet ./...`, `go build ./cmd/btyper`, `bash -n install.sh`, `bash -n packaging/aur/PKGBUILD`, and `ruby -c Formula/btyper.rb`. On Arch, check `makepkg --printsrcinfo`. Verify main CI is green.
- Follow the accessibility checklist in `docs/stable-contract.md` on a supported terminal. On copies of real data, test backup export from the previous release, import into the candidate, export again, and compare key counts; verify failed import leaves existing data intact. Record manual results and untested paths.

## Publish in dependency order

1. Commit candidate code and documentation, push `main`, and wait for green CI. Create and push an annotated `vX.Y.Z` tag at that exact commit. The tag triggers `.github/workflows/release.yml` to build Linux amd64/arm64, macOS amd64/arm64, and Windows amd64 assets and publish `checksums.txt` and `install.sh`.
2. Wait for the release workflow. Verify every expected asset exists and matches its named SHA-256 line in `checksums.txt`. Stop if an asset is absent or a hash differs; do not substitute guessed hashes.
