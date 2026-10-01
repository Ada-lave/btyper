# btyper

**English** | [Русский](README.ru.md)

Adaptive touch-typing practice in your terminal. Train weak letters and letter
pairs, practice in English or Russian, and track your progress locally.
Inspired by Keybr's learning loop.

![btyper typing practice demo](output.gif)

## Features

- **Adaptive lessons:** start with a small set of letters, then practice weak
  skills and revisit skills due for review.
- **English and Russian:** built-in layouts, translated interface, and optional
  larger dictionaries.
- **Three ways to practice:** adaptive training, focused letter or pair drills,
  and your own text.
- **Visible progress:** speed, accuracy, weak keys, history, daily goals and streaks.
- **Local storage:** no account required; training works offline. Downloads are
  optional, and progress stays on your machine.

## Install and start

On Linux or macOS, review [install.sh](install.sh), then install the latest
release and launch:

```sh
curl -fsSL https://raw.githubusercontent.com/Ada-lave/btyper/main/install.sh | sh
btyper
```

The installer verifies the binary's SHA-256 and installs to `~/.local/bin`.
Ensure that directory is in your `PATH`; you can also launch `~/.local/bin/btyper`
directly. No Go toolchain is needed for release binaries.

Windows amd64 binaries are available on the [releases page](https://github.com/Ada-lave/btyper/releases/latest).
See the [installation guide](docs/install.md) for checksum verification,
custom installation paths and version pinning.

Use a UTF-8 terminal at least **60 columns × 16 rows**. Smaller windows pause
practice. Release builds support Linux and macOS amd64/arm64, and Windows amd64.

## Your first lesson

1. Run `btyper` and choose **Adaptive training**.
2. Type the displayed words. Use Backspace to clear a mistake before continuing.
3. Review your speed, accuracy and weak letters, then start the next lesson.

Training starts with a small alphabet and gradually introduces more letters.
After the letter foundation, it schedules letter pairs and can include numbers,
uppercase letters and punctuation. These categories can be toggled in Settings.
The first launch selects a training language from the detected interface
language; change it in Settings or run `btyper --lang en` / `btyper --lang ru`.

Choose **Drill a letter or pair** for a specific target, or **Custom text** to
practice a passage you paste or open from a UTF-8 file. Statistics show your
history and progress; Settings includes a daily practice goal.

## Controls

| Key | Action |
| --- | --- |
| Arrows or `j` / `k` | Navigate menus |
| Enter | Select an item or continue after a lesson |
| Esc during a lesson | Open the pause menu |
| Backspace | Clear a typing mistake |
| Ctrl+R during a lesson | Restart the lesson |
| Ctrl+K during a lesson | Show or hide the virtual keyboard |
| Ctrl+O / Ctrl+S in the text editor | Open a file / start custom-text practice |
| PgUp / PgDn | Scroll long screens |

Navigation shortcuts work with both English and Russian keyboard layouts.
For CLI options and subcommands:

```sh
btyper -h
btyper dictionary download -h
```

## Optional dictionaries

Built-in vocabulary works immediately. For a larger selection of real words:

```sh
btyper dictionary download            # English and Russian
btyper dictionary download --lang ru  # Russian only
```

Downloaded dictionaries are used on the next launch, including offline. Lessons
still use only unlocked letters. If training uses `--data-dir DIR`, pass the
same option after `download`. Source, filtering and CC BY-SA 4.0 attribution
are documented in the [dictionary guide](docs/dictionaries.md).

## Documentation and backups

- [User guide](docs/user-guide.en.md): metrics, history, settings, backups and recovery.
- [Adaptive learning](docs/adaptive-learning.md) (Russian): skill selection and review scheduling.
- [Custom language profiles](docs/language-profiles.md).
- [Compatibility and data storage](docs/stable-contract.md): the stable contract for v1.x.
- [Release notes](https://github.com/Ada-lave/btyper/releases).

Progress is stored in `$XDG_DATA_HOME/btyper`, or `~/.local/share/btyper` by
default. Use `--data-dir DIR` to choose another directory. To create a portable
backup:

```sh
btyper export --output btyper-backup.json
```

Import replaces the current dataset and creates a safety backup first. See the
user guide before restoring data or upgrading an existing installation.

## Feedback and contributing

[Open an issue](https://github.com/Ada-lave/btyper/issues) for bugs or suggestions.
For bugs, include `btyper --version`, your OS and terminal, steps to reproduce,
and the error message or a screenshot. Feedback on lesson variety and English
or Russian practice is welcome.

## Development

Clone the repository and use the Go version from [go.mod](go.mod) plus `just`:

```sh
git clone https://github.com/Ada-lave/btyper.git
cd btyper
just build
/tmp/btyper --data-dir /tmp/btyper-dev
just check
```

`just run` starts from source; `just install` installs to `GOBIN` or
`$(go env GOPATH)/bin`. Focused checks: `just test`, `just test-race`, `just vet`.
See the [release checklist](docs/release-checklist.md) for publishing releases.

## License

btyper code is licensed under [MIT](LICENSE). Optional downloaded dictionaries
have their own [CC BY-SA 4.0 content license](docs/dictionaries.md).
