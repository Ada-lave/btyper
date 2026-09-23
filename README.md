# btyper

**English** | [Русский](README.ru.md)

Adaptive touch-typing practice in your terminal, inspired by Keybr's learning loop.

## Install and verify

See [all installation channels](docs/install.md) for Homebrew, the
AUR-compatible PKGBUILD, Scoop on Windows, and platform smoke tests. The
project is licensed under [MIT](LICENSE).

Install the latest Linux or macOS release into `~/.local/bin`:

```sh
curl -fsSL https://raw.githubusercontent.com/Ada-lave/btyper/main/install.sh | sh
```

Review [`install.sh`](install.sh) before running it. Set `BTYPER_INSTALL_DIR`
to choose another destination or `BTYPER_VERSION` to install a specific
release, for example `BTYPER_VERSION=v1.0.0`.

Alternatively, build from source using the Go version specified in `go.mod`:

```sh
go build -o /tmp/btyper ./cmd/btyper
/tmp/btyper --version
go install ./cmd/btyper
```

`go install` places the executable in `GOBIN`, or `$(go env GOPATH)/bin` by
default. Add that directory to your PATH.

Development checks: `go test ./...`, `go test -race ./...`, and `go vet ./...`.
CI also builds the executable. Coverage can be collected with
`go test -cover ./...` when the installed Go toolchain includes coverage tools.

## Run

```sh
go run ./cmd/btyper
```

The application opens directly into the main menu. On the first launch, the
training layout follows the detected UI language and can be changed later in
Settings. The interface can be anchored to any of nine terminal positions from
the same screen. Adaptive training calibrates keys in order, then schedules
weak or overdue letters and common letter pairs for spaced review. Progress is
stored locally in an SQLite database under the XDG data directory.

By default, the database is stored at `$XDG_DATA_HOME/btyper/btyper.db`, or
`~/.local/share/btyper/btyper.db` when `XDG_DATA_HOME` is not set. Passing
`--data-dir` selects a different database directory.

Learning lessons show the state of every letter, the current target, and its
confidence. Their practice words are generated from the unlocked vowels and
consonants, so the lesson stays pronounceable and varied without depending on
a large hardcoded dictionary.

Useful options:

```text
--lang PROFILE_ID
--mode adaptive|text
--text path/to/file.txt
--data-dir path/to/data
--version
```

Inside the application use arrow keys or `j`/`k` to navigate. In Settings,
`h`/`l` also change values. These physical shortcuts work with both English
and Russian keyboard layouts. During lessons, Esc opens a pause menu with
continue, restart, and exit actions; Backspace clears a mistake, and Ctrl+R
restarts the generated lesson. Custom text can be pasted into the editor or
opened with Ctrl+O, then started with Ctrl+S.

Custom text must be valid UTF-8 and at most 1 MiB. Both the CLI and file picker
normalize text to NFC and collapse whitespace. Files are read in the background
inside the application. PgUp/PgDn scroll screens that exceed the terminal height.
Shrinking the terminal below 60×16 pauses the lesson; resume explicitly after
enlarging it.

## Learning and metrics

For a detailed description of scheduling, confidence formulas, and the
relationship between letters and bigrams, see
[`docs/adaptive-learning.md`](docs/adaptive-learning.md) (Russian).

WPM is correct characters / 5 / active minutes; CPM is correct characters /
active minutes. Accuracy is successful attempts / all accepted attempts. Input
while an error is awaiting Backspace is ignored. Pauses and time in menus are
excluded; time spent correcting mistakes is included. The first character does
not contribute a zero-latency measurement.

Confidence is the minimum of speed, accuracy and sample scores: the configured
speed and accuracy targets, and 30 attempts per skill. The adaptive scheduler
tracks both letters and common two-letter sequences. After calibration, a skill
moves through review intervals of 1, 3, 7, 14 and 30 days when a lesson meets
the configured speed and accuracy targets; a failed review shortens the
interval. Letter pairs are introduced only after every letter has accumulated
30 observations; the lesson progress panel always names the actual letter or
pair selected by the scheduler. Old `learn` and `improve` settings are migrated
to `adaptive`, while their historical sessions remain available in statistics.
After the letter foundation, adaptive practice also schedules digits, uppercase
letters and punctuation. Custom-text practice measures these skills too.

Choose **Drill a letter or pair** from the menu to practice any letter or
two-letter pair in the current language. Selecting a target does not advance
the adaptive queue; completed Drill sessions are saved under their own mode in
history. When you submit custom text with Ctrl+S, btyper first shows its most
frequent skills and skills that may be difficult based on occurrence count and
your saved confidence. Press Enter to start or Esc to edit the text.

Results show weak letters, target confidence changes and the next exercise's
purpose. A failed result save remains on screen: Enter retries the same attempt
without duplication; Esc explicitly discards the unsaved result. Confirmed
progress is only changed after a successful save.

## Time today and history

The menu, lesson, result and statistics screens show **active practice time for
today**, across both languages and all modes. This includes abandoned and
restarted attempts. The counter starts on the first typed character, excludes
pauses and menus, and splits active intervals at midnight in the local time zone.
It refreshes during practice, saves in the background approximately every five
seconds, and flushes on lesson completion and normal exit. A crash or forced kill
can lose the latest unsaved seconds. A visible storage error means the pending
time has not yet been persisted; subsequent saves retry it.

In session history, L cycles language, M cycles mode, P cycles all time / seven
days / thirty days. Left/right switch pages of 50 sessions. Summary speed and
accuracy are weighted by active duration and attempts, respectively. History
filters use the session start time; the daily practice counter uses actual
active intervals, including those spanning midnight.

The third statistics tab shows 30-day speed, accuracy and key-latency trends.
The fourth tab reviews the latest seven local calendar days against the prior
seven days of active practice. It compares this week's duration-weighted WPM
with the personal baseline from all earlier sessions in the selected language.
The retention summary counts observed skills at review level 3 or higher
(interval of at least seven days) and skills currently due for review.
Bundled EN/RU layouts, learning order, words, and frequent pairs live in
validated JSON profiles inside the binary.
You can import and export your own profiles offline; see the
[language-profile authoring guide](docs/language-profiles.md).
Settings also has a daily active-practice goal (1–120 minutes, default 10).
The menu and practice screens show a neutral in-progress or complete status
and the current streak of completed local calendar days. An incomplete today
does not break yesterday's streak. The goal and daily time survive restart.

## Backup and export

Create a portable, versioned JSON backup or CSV reports without stopping using
the local-first workflow:

```sh
btyper export --format backup --output btyper-backup.json
btyper export --format csv --output-dir reports
btyper import --input btyper-backup.json
```

Import validates the backup and replaces the current dataset in one
transaction. Before replacement, btyper writes a timestamped safety backup to
the active data directory. CSV export creates `sessions.csv`,
`practice_time.csv`, and `skills.csv`.

If a lesson result cannot be saved, leave the result screen open, fix the
storage problem (for example, free disk space or restore write access), then
press Enter to retry. Esc discards that unsaved result. A failed import keeps
the current dataset; the error prints the path of its pre-import safety backup.
To recover from a bad import, close btyper and run
`btyper import --input <safety-backup-path>`. To recover a damaged SQLite
database, close btyper and restore a copy of the entire data directory,
including any `-wal` and `-shm` files. Do not copy a live SQLite database.

## Data and upgrades

The [stable v1.0 contract](docs/stable-contract.md) documents supported
platforms, backup compatibility, migration policy and accessibility checks.

Schema upgrades run transactionally at startup. Existing settings and session
history are preserved. Previously practiced letters restore a contiguous opened
alphabet; historical latency sample counts remain unknown. Legacy daily totals
are reconstructed from completed sessions and attributed to their local start
date because old databases do not contain active intervals or abandoned attempts.

Before upgrading, close btyper and copy the entire data directory to a backup
location, including any SQLite `-wal` and `-shm` files. Restore with btyper closed.
Settings → Reset removes progress, history and daily practice time for both
languages; it preserves preferences. Older binaries must not be used to write
an upgraded database.
