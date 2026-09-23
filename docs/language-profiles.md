# Authoring a language profile

btyper uses portable JSON profiles for the keyboard layout, learning order,
vowels, vocabulary, and frequent letter pairs. Profiles are local files; no
account or network service is involved.

Start from a built-in profile:

```sh
btyper profile export --id en --output my-layout.json
```

Edit the JSON, give it a new `id` (for example `en_colemak`) and a readable
`name`, then import and select it:

```sh
btyper profile import --input my-layout.json
btyper profile list
btyper --lang en_colemak
```

You can also select the profile under Settings → Training language. To use a
different data directory, pass `--data-dir DIR` after `import`, `export`, or
`list`. A profile can be exported later with
`btyper profile export --id en_colemak --output shared.json`.

## Fields

| Field | Meaning |
| --- | --- |
| `id` | Stable lowercase ASCII ID, 1–32 characters: `a-z`, `0-9`, `_`, `-`. Changing it creates a different profile and a separate progress history. `en` and `ru` are reserved. |
| `name` | Display name for an imported profile, up to 80 bytes. Built-in profiles use `name_id` for translated names. |
| `unlock_order` | Each lowercase letter exactly once, in learning order; 6–128 letters. |
| `vowels` | Nonempty subset of `unlock_order` used to build pronounceable synthetic words. |
| `rows` | Keyboard rows. Together they must contain every letter in `unlock_order` exactly once. |
| `finger_groups` | Eight strings assigning each letter exactly once, in order: left pinky, ring, middle, index; right index, middle, ring, pinky. |
| `words` | 1–10,000 lowercase words of up to 32 letters, using only letters in `unlock_order`. |
| `frequent_pairs` | 1–64 unique, two-letter patterns from `unlock_order`, in the order they should enter adaptive practice. |

The built-in files in `internal/trainer/profiles/` are complete examples.
Digits, uppercase forms of the profile letters, and common punctuation are
added automatically to the curriculum. The profile's `vowels` field controls
synthetic words for that language.

## Validation and safety

Import validates every field before writing anything. Errors name the field
and, where possible, its index, such as `profile.words[3]`. Imported files are
stored under the active data directory's `profiles/` folder. Import will not
overwrite an installed profile with the same ID; give a revision a new ID or
move the old file out of that folder while btyper is closed. A failed import
leaves installed profiles and the SQLite database intact. Keep the JSON file
with your other backups; database backup exports do not include user-profile
files. An invalid installed profile prevents startup until its JSON is fixed
or moved aside, and the error identifies the file and field.
