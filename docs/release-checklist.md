# Release checklist

For a compatible v1.x release, follow these steps from a clean `main` checkout.
Record manual smoke-test results in the release notes or release issue.

1. Review issues and migration impact; update the
   [stable contract](stable-contract.md), both READMEs, profile guide and
   changelog/release notes as needed. If the schema or backup format changes,
   add round-trip and old-version fixture tests before proceeding.
2. Run `go test ./...`, `go test -race ./...`, `go vet ./...`,
   `go build ./cmd/btyper`, `bash -n install.sh`, and
   `bash -n packaging/aur/PKGBUILD`. Run `makepkg --printsrcinfo` on Arch and
   `ruby -c Formula/btyper.rb`. Ensure main CI is green.
3. Perform the [accessibility checklist](stable-contract.md#accessibility-checklist)
   on at least one supported terminal. Export a backup from the previous
   release, import it in the candidate, then export and compare key counts;
   verify failure leaves current data intact. Test an old database copy, never
   the only production copy.
4. Commit all code/docs, push `main`, and wait for green CI. Create and push an
   annotated `vX.Y.Z` tag at that commit. Wait for the release workflow to
   publish Linux, macOS and Windows binaries, `checksums.txt`, and `install.sh`.
   Confirm every asset exists and matches its checksum.
5. Update `Formula/btyper.rb`, `packaging/aur/PKGBUILD`, and
   `bucket/btyper.json` to the new version and exact published asset SHA-256
   values. Pin the AUR license source to the tagged `LICENSE` and its hash.
   Regenerate `packaging/aur/.SRCINFO` with `makepkg --printsrcinfo` if
   submitting to AUR. Commit and push these package updates.
6. Run the [installation smoke tests](install.md) for install.sh, Homebrew,
   AUR-compatible PKGBUILD and Scoop on their respective platforms, or record
   the CI job that exercises the path. Do not claim untested OS/CPU pairs as
   smoke-tested. Check `btyper --version` and a fresh launch.
7. Update release notes with changes, compatibility notes, checksums, test
   evidence and any untested paths. Close the completed issues and milestone
   only after their acceptance criteria are met. If a central Homebrew/AUR or
   Scoop registry submission is desired, publish there separately; these
   recipes work from this repository without central listing.
