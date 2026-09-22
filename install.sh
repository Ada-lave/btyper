#!/bin/sh

set -eu

repo="Ada-lave/btyper"
version="${BTYPER_VERSION:-latest}"
install_dir="${BTYPER_INSTALL_DIR:-${HOME}/.local/bin}"

command -v curl >/dev/null 2>&1 || {
  echo "btyper installer: curl is required" >&2
  exit 1
}

case "$(uname -s)" in
  Linux) os="linux" ;;
  Darwin) os="darwin" ;;
  *)
    echo "btyper installer: unsupported operating system: $(uname -s)" >&2
    exit 1
    ;;
esac

case "$(uname -m)" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *)
    echo "btyper installer: unsupported architecture: $(uname -m)" >&2
    exit 1
    ;;
esac

if [ "$version" = "latest" ]; then
  release_url="https://github.com/${repo}/releases/latest/download"
else
  case "$version" in v*) ;; *) version="v${version}" ;; esac
  release_url="https://github.com/${repo}/releases/download/${version}"
fi

asset="btyper-${os}-${arch}"
tmp_dir="$(mktemp -d 2>/dev/null || mktemp -d -t btyper)"
trap 'rm -rf "$tmp_dir"' EXIT HUP INT TERM

echo "Downloading ${asset} (${version})..."
curl -fL --retry 3 --proto '=https' --tlsv1.2 \
  "${release_url}/${asset}" -o "${tmp_dir}/${asset}"
curl -fL --retry 3 --proto '=https' --tlsv1.2 \
  "${release_url}/checksums.txt" -o "${tmp_dir}/checksums.txt"

expected="$(awk -v name="$asset" '$2 == name { print $1 }' "${tmp_dir}/checksums.txt")"
if [ -z "$expected" ]; then
  echo "btyper installer: checksum for ${asset} is missing" >&2
  exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
  actual="$(sha256sum "${tmp_dir}/${asset}" | awk '{ print $1 }')"
elif command -v shasum >/dev/null 2>&1; then
  actual="$(shasum -a 256 "${tmp_dir}/${asset}" | awk '{ print $1 }')"
else
  echo "btyper installer: sha256sum or shasum is required" >&2
  exit 1
fi

if [ "$actual" != "$expected" ]; then
  echo "btyper installer: checksum verification failed" >&2
  exit 1
fi

mkdir -p "$install_dir"
install -m 0755 "${tmp_dir}/${asset}" "${install_dir}/btyper"

echo "Installed btyper to ${install_dir}/btyper"
case ":${PATH}:" in
  *":${install_dir}:"*) ;;
  *) echo "Add ${install_dir} to PATH to run btyper from any directory." ;;
esac
