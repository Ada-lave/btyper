#!/usr/bin/env python3
"""Check package recipes against the published release checksum manifest."""

import json
import pathlib
import re
import urllib.request


ROOT = pathlib.Path(__file__).resolve().parents[1]
formula = (ROOT / "Formula/btyper.rb").read_text()
pkgbuild = (ROOT / "packaging/aur/PKGBUILD").read_text()
scoop = json.loads((ROOT / "bucket/btyper.json").read_text())

version = re.search(r'^  version "([^"]+)"$', formula, re.M).group(1)
assert re.search(rf"^pkgver={re.escape(version)}$", pkgbuild, re.M)
assert scoop["version"] == version
assert 'license "MIT"' in formula
assert "license=('MIT')" in pkgbuild
assert scoop["license"] == "MIT"

release_url = f"https://github.com/Ada-lave/btyper/releases/download/v{version}/"
with urllib.request.urlopen(release_url + "checksums.txt") as response:
    lines = response.read().decode().splitlines()
checksums = {name: sha for sha, name in (line.split() for line in lines)}

formula_assets = list(re.finditer(r'url "([^"]+/([^/"]+))", using: :nounzip\s+sha256 "([0-9a-f]{64})"', formula))
assert len(formula_assets) == 4
for match in formula_assets:
    url, asset, sha = match.groups()
    assert url == release_url + asset
    assert checksums[asset] == sha, asset

for arch, asset in (("x86_64", "btyper-linux-amd64"), ("aarch64", "btyper-linux-arm64")):
    sha = re.search(rf"^sha256sums_{arch}=\('([0-9a-f]{{64}})'\)$", pkgbuild, re.M).group(1)
    assert checksums[asset] == sha, asset
assert f"/v{version}/LICENSE" in pkgbuild
srcinfo = (ROOT / "packaging/aur/.SRCINFO").read_text()
assert f"\tpkgver = {version}" in srcinfo
for asset in ("btyper-linux-amd64", "btyper-linux-arm64"):
    assert checksums[asset] in srcinfo

windows = scoop["architecture"]["64bit"]
assert windows["url"] == release_url + "btyper-windows-amd64.exe#/btyper.exe"
assert windows["hash"] == checksums["btyper-windows-amd64.exe"]
assert scoop["bin"] == "btyper.exe"
print(f"Package recipes match v{version} release checksums")
