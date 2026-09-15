# btyper

Adaptive touch-typing practice in your terminal, inspired by Keybr's learning loop.

## Run

```sh
go run ./cmd/btyper
```

The first launch asks for English QWERTY or Russian ЙЦУКЕН and whether to learn
from scratch or calibrate an existing skill. Progress is stored locally in an
SQLite database under the XDG data directory.

Useful options:

```text
--lang en|ru
--mode learn|improve|text
--text path/to/file.txt
--data-dir path/to/data
--version
```

Inside the application use arrow keys and Enter to navigate. During lessons,
Esc pauses, Backspace clears a mistake, and Ctrl+R restarts the generated lesson.
Custom text can be pasted into the editor or opened with Ctrl+O, then started
with Ctrl+S.
