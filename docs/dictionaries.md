# Optional dictionaries

```sh
btyper dictionary download                 # English and Russian
btyper dictionary download --lang en
btyper dictionary download --lang ru --data-dir /tmp/btyper-dev
```

The command downloads the English and Russian top-50,000 frequency lists
from [FrequencyWords by Hermit Dave](https://github.com/hermitdave/FrequencyWords),
derived from OpenSubtitles 2018. The content is licensed under
[CC BY-SA 4.0](https://creativecommons.org/licenses/by-sa/4.0/), independently
of btyper's MIT code license. Attribution and the license link are included in
each saved dictionary. These subtitle-derived lists can contain informal
language, names, and profanity; they are not curated school dictionaries.

btyper lowercases words, removes duplicates, and retains words of 2–10 letters
using the built-in language's alphabet. Thus the installed word count is below
50,000. Frequencies are discarded. Only words using unlocked letters enter a
lesson; synthetic words remain available when too few real words qualify.

Files are saved as `dictionaries/en.txt` and `dictionaries/ru.txt` inside the
data directory (`$XDG_DATA_HOME/btyper`, or `~/.local/share/btyper` by default).
They are loaded on the next launch and merged with the built-in vocabulary.
Custom profiles keep their own vocabulary. No network access occurs during
training; without downloaded files, the built-in vocabulary continues to work.

Run the command again to update a dictionary. A failed download leaves its
previous file intact. To return to the built-in vocabulary, remove the relevant
dictionary file. Optional dictionaries are downloadable assets and are not
included in progress backups or built-in profile exports.
