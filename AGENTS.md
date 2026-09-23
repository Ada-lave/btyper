# Repository Guidelines

## Project Structure & Module Organization

`cmd/btyper` contains the CLI entry point. Core code lives under `internal/`: `ui` handles terminal screens and key input, `trainer` generates and scores lessons, `application` coordinates services, `storage` owns SQLite and backups, `domain` defines shared types, and `i18n` contains English and Russian messages. Built-in keyboard layouts are JSON files in `internal/trainer/profiles/`. Keep tests beside the code they cover as `*_test.go`. User guides and release notes live in `docs/`; the maintained release installer is `install.sh`.

## Build, Test, and Development Commands

- `just run` starts the app from source. For persistence experiments, run `go run ./cmd/btyper --data-dir /tmp/btyper-dev` to avoid personal data.
- `just build` creates `/tmp/btyper`; `just install` installs the Go binary.
- `just test`, `just test-race`, and `just vet` run the focused Go checks. `just coverage` reports package coverage.
- `just check` runs the full local and CI check set. It does not require package-manager tooling.

## Coding Style & Naming Conventions

Use standard Go formatting (`gofmt`) and tabs in Go source. Keep package names short and lowercase. Name exported identifiers in `MixedCaps` and tests `TestBehaviorName`. Follow existing screen and service patterns before introducing new layers. Update both `internal/i18n/active.en.toml` and `active.ru.toml` for user-facing text.

## Testing Guidelines

Use Go's `testing` package. Add focused regression tests for behavior changes, especially keyboard layouts, lesson state, storage migrations, and backup compatibility. There is no fixed coverage threshold; use `go test -cover ./...` when coverage matters. For database tests, use temporary data rather than a personal btyper directory.

## Commits & Pull Requests

Recent commits commonly use concise prefixes such as `feat:`, `fix:`, `docs:`, `ci:`, and `chore:`. Describe the observable change in the imperative. Pull requests should explain behavior, mention affected platforms or data formats, link relevant issues, and report the checks run. Include terminal captures when changing visible UI. Follow `docs/release-checklist.md` for tags and release assets.
