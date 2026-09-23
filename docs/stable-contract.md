# Stable v1.0 contract

## Platforms and interface

Release binaries target Linux amd64/arm64, macOS amd64/arm64, and Windows
amd64. Other operating systems and CPUs may work from source but are not
release-tested. A UTF-8 terminal of at least 60 columns by 16 rows is required
for lessons; a smaller terminal pauses practice. All progress is local. The
default database is `$XDG_DATA_HOME/btyper/btyper.db`, or
`~/.local/share/btyper/btyper.db` when XDG_DATA_HOME is unset. Use `--data-dir`
to choose another location. Custom text accepts UTF-8 up to 1 MiB.

## Persistence and migrations

The v1.0 series preserves user settings, completed sessions, skills, practice
time and imported profiles when upgrading an older database. Schema migrations
run transactionally at startup. Newer binaries may migrate the database; do
not use an older binary to write it afterward. Before any upgrade, close the
app and copy the complete data directory, including SQLite `-wal` and `-shm`
files. Restore only while btyper is closed.

Portable JSON backups use format version 1. v1.0 accepts its own format-1
exports and the legacy format-1 backups covered by regression tests. Imports
validate before replacing current data in one transaction and create a
timestamped pre-import safety backup. Unknown future backup format versions
are rejected, rather than silently misread. CSV exports are reports, not an
import format. Imported language profiles live beside the database; export
those separately with `btyper profile export --id ID --output profile.json`.
See [profile authoring](language-profiles.md). Before changing the backup
format or dropping support for old data, add fixtures and document a migration
path in the release notes.

## Accessibility checklist

For each release, verify with EN and RU interfaces:

- Navigate menu, settings, lesson pause and result without a mouse using
  arrows/Enter/Esc; `j`/`k` and `h`/`l` work with both keyboard layouts.
- Select the monochrome theme and ensure state is communicated in words or
  symbols, not by color alone. Hide the keyboard with Ctrl+K and confirm practice remains
  usable.
- Resize to 60×16 and below; clipped content must be scrollable with PgUp/PgDn
  where appropriate, and practice must pause below the minimum size.
- Confirm focus, errors, pending result-save retry, and destructive reset
  confirmation are visible. No irreversible action should be triggered by
  color alone.
- Try custom text with Cyrillic and combining marks; review readable output
  on a UTF-8 terminal. Terminal screen readers and high-contrast settings
  vary by platform, so record the terminal and test result in release notes.

## User guide

The [English README](../README.md) and [Russian README](../README.ru.md)
cover startup, navigation, adaptive/Drill/custom-text modes, statistics,
daily goals, backup and restore. See [installation](install.md) for every
channel, [adaptive scheduling](adaptive-learning.md) for learning semantics,
and [profile authoring](language-profiles.md) for custom layouts.
