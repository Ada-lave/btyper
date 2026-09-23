# Installation

The maintained installation channel for Linux and macOS is the checksum-
verified `install.sh` script. Package-manager releases are not provided.

## Linux and macOS

Inspect the script, then install the latest release:

```sh
curl -fsSL https://raw.githubusercontent.com/Ada-lave/btyper/main/install.sh | sh
btyper --version
```

The script verifies the release SHA-256 and installs to `~/.local/bin` by
default. Set `BTYPER_VERSION=v1.0.2` to pin a version and
`BTYPER_INSTALL_DIR` to change the destination. Ensure the install directory is
in `PATH`.

Smoke test on Linux or macOS: install into a new temporary directory with
`BTYPER_INSTALL_DIR="$(mktemp -d)"`, run its `btyper --version`, and compare the
binary's SHA-256 with the named line in the release's `checksums.txt`.

## Windows

Release binaries include Windows amd64. Download `btyper-windows-amd64.exe`
from the [latest release](https://github.com/Ada-lave/btyper/releases/latest)
and verify its SHA-256 against `checksums.txt` before running it. No Windows
installer is maintained at this time.

## Source

With the Go version in `go.mod` and `just`, run `just install` from the repo.
The local source build reports `dev` for `--version` unless built with release
linker flags. Run `just check` before using a source checkout as a release.
