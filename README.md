# btyper

Adaptive touch-typing practice in your terminal, inspired by Keybr's learning loop.

## Run

```sh
go run ./cmd/btyper
```

The application opens directly into the main menu. On the first launch, the
training layout follows the detected UI language and can be changed later in
Settings. "Learn letters" unlocks keys gradually; "Improve skill" first
calibrates your current ability and then focuses on weak keys. Progress is
stored locally in an SQLite database under the XDG data directory.

Useful options:

```text
--lang en|ru
--mode learn|improve|text
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
