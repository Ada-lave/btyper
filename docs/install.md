# Installation channels

Official release binaries are built for Linux amd64/arm64, macOS amd64/arm64,
and Windows amd64. They are checksum-protected in the release's `checksums.txt`.
No public Homebrew tap, AUR package, or Scoop bucket has been submitted to a
central registry; the recipes below are maintained in this repository.

## Linux and macOS: install.sh

Inspect `install.sh`, then run:

```sh
curl -fsSL https://raw.githubusercontent.com/Ada-lave/btyper/main/install.sh | sh
btyper --version
```

The script verifies the release SHA-256 and installs to `~/.local/bin` by
default. Set `BTYPER_VERSION=v1.0.1` to pin a version and
`BTYPER_INSTALL_DIR` to change the destination. Ensure it is in `PATH`.

Smoke test: install into a new temporary directory with
`BTYPER_INSTALL_DIR="$(mktemp -d)"`, run its `btyper --version`, and compare
the binary's `sha256sum` with the named line in the release's `checksums.txt`.

## Homebrew (Linux or macOS)

```sh
brew tap ada-lave/btyper https://github.com/Ada-lave/btyper.git
brew install ada-lave/btyper/btyper
btyper --version
```

The formula installs the appropriate released binary and verifies its SHA-256.
Smoke test on each supported OS/CPU pair: run the commands above,
`brew test ada-lave/btyper/btyper`, and confirm `btyper --version` matches the
formula version. This repository's CI tests the Linux amd64 path.

## Arch Linux (AUR-compatible PKGBUILD)

This package is not yet published on aur.archlinux.org. On an Arch Linux
machine, clone this repository, then:

```sh
cd packaging/aur
makepkg --syncdeps --clean
sudo pacman -U ./btyper-bin-*.pkg.tar.zst
btyper --version
```

If `makepkg` creates a `.pkg.tar.xz` instead, use that file. Inspect the
PKGBUILD before running it. `makepkg` verifies the binary and license hashes.
Smoke test: run `makepkg --check --syncdeps` (if supported by the local
makepkg), install the built package, check `btyper --version`, and remove it
with `sudo pacman -R btyper-bin` after testing. The package supports x86_64 and
aarch64. Generate `.SRCINFO` with `makepkg --printsrcinfo > .SRCINFO` when
publishing the recipe to AUR.

## Windows amd64 (Scoop)

With Scoop already installed:

```powershell
scoop bucket add btyper https://github.com/Ada-lave/btyper.git
scoop install btyper/btyper
btyper --version
```

This repository is the bucket. Scoop downloads the Windows release executable,
renames it to `btyper.exe`, and verifies its SHA-256. Smoke test on Windows:
run the commands above in PowerShell and confirm the reported version matches
`bucket/btyper.json`; then `scoop uninstall btyper` if desired. Alternatively,
download `btyper-windows-amd64.exe` from GitHub Releases and compare its
SHA-256 to `checksums.txt` before running.

## Source

With the Go version in `go.mod` and `just`, run `just install` from the repo.
The local source build reports `dev` for `--version` unless built with release
linker flags. Run `just check` before using a source checkout as a release.
