# Release checklist

For a compatible v1.x release, follow these steps from a clean `main` checkout.
Record manual smoke-test results in the release notes or release issue.

1. Review issues and migration impact; update the
   [stable contract](stable-contract.md), both READMEs, profile guide and
   changelog/release notes as needed. If the schema or backup format changes,
   add round-trip and old-version fixture tests before proceeding.
2. Run `just check`. It runs the Go tests, race tests, vet, build, and installer
   shell syntax check. Main CI runs the same recipe; ensure it is green.
3. Perform the [accessibility checklist](stable-contract.md#accessibility-checklist)
   on at least one supported terminal. Export a backup from the previous
   release, import it in the candidate, then export and compare key counts;
   verify failure leaves current data intact. Test an old database copy, never
   the only production copy.
4. Commit all code/docs, push `main`, and wait for green CI. Create and push an
   annotated `vX.Y.Z` tag at that commit. Wait for the release workflow to
   publish Linux, macOS and Windows binaries, `checksums.txt`, and `install.sh`.
   Confirm every asset exists and matches its checksum.
5. Run the [installation smoke test](install.md) for `install.sh` on an
   available supported platform. Do not claim untested OS/CPU pairs as
   smoke-tested. Check `btyper --version` and a fresh launch.
6. Update release notes with changes, compatibility notes, checksums, test
   evidence and any untested paths. Close the completed issues and milestone
   only after their acceptance criteria are met.
