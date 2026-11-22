# Working in the terminal (`orcaton`)

**English** · [Deutsch](usage.de.md) · [Русский](usage.ru.md) · [ქართული](usage.ka.md)

The command is **`orcaton`** (or `app-go/scripts/run.*` without installing). The window does the same
([docs/gui/user.md](../gui/user.md)) — the table of buttons and options is there.

## Synchronization
1. Your editing program: one camera or recorder — one track; export the timeline
   (*File → Export → Timeline…*) → **FCP 7 XML V5** (`.xml`) or **DRT** (`.drt`).
2. Run (in a **new** terminal window after installing; media somewhere else — `--map`, see below):
   ```
   orcaton sync "Timeline 1.xml" "Timeline 1-out.xml" --report "Timeline 1-out.csv"
   orcaton sync "Timeline 1.drt" "Timeline 1-out.drt"          # DRT → DRT (the report is written by itself)
   orcaton sync "Timeline 1.drt" "out.drt" --map "D:\media\project"   # the media is here now (search by name)
   ```
3. Your editing program: import the result (*File → Import → Timeline…*).

| Clip label | Meaning |
|---|---|
| no label | synchronized **by sound** (frame accuracy) |
| **Mango** | placed **by the camera clock** — the sound did not confirm it (≈1 s) |
| **Rose** | no match found — placed in shooting order |

The order of a camera's files is by the number in the name (MVI_0001, C0001, 00001…), then by shooting time.
Files without a match go in that order of their own camera, next to their neighbours (also inside synced
stretches); files that are not on disk go to the very end; there are no gaps. In a `.drt` only clip positions, the duration and the
timeline name change; the Rose/Mango labels are not written to `.drt` yet — the statuses are in `*_report.csv`.

## `sync` options
| Option | Default | Purpose |
|---|---|---|
| `--report FILE.csv` | — (for `.drt` — `OUT_report.csv`) | report for every file |
| `--map FOLDER` / `--map "D:/=G:/"` | `$ORCATON_PATH_MAP` | where the media is on this computer: a folder with the sources (search by name) and/or a replacement of the start of paths; `;` — several |
| `--search FOLDER` | — | the same as `--map FOLDER`; may be repeated |
| `--search-depth N` | `6` | how many levels of subfolders to search |
| `-y`, `--yes` | off | do not ask about files not found — put them at the end of the timeline at once |
| `--keep-paths` | off | with `--map`/`--search`: read from the new paths, write the old ones to the result |
| `--originals FOLDER` | — | the timeline has proxies: where the cameras' originals are (shooting time); may be repeated |
| `--unsynced` | `order_time` | `order_time` (in the camera's order, also inside stretches) · `edges` (never inside a stretch; after the last stretch — at the end) · `end` · `remove` |
| `--max-gap` / `--keep-gap` | `0` / `0` | which gaps to remove and to what length to shorten them, sec |
| `--order-by` | `name` | order of a camera's files: `name` — the number in the name (MVI_0001, C0001, 00001…), then shooting time · `time` — shooting time / TC, then the name |
| `--singles` | `sequential` | files without a match: `sequential` — do not run at the same time as other files without a match by sound (they stand next to their camera's neighbours, time is inserted when needed; the sync does not change) · `stacked` — each in its camera's order, different cameras may run at the same time |
| `--label` / `--time-label` | `Rose` / `Mango` | label colors |
| `--min-conf` / `--min-ratio` | `12` / `1.5` | strictness of the sound match |
| `--no-auto-threshold` | — | do not raise the threshold up to the project's noise |
| `--min-overlap` / `--max-offset` | `2` / — | minimum overlap and search window, sec |
| `--audio-mode` | `mix` | `mix` · `first_stream` · `first_channel` |
| `--no-file-order` / `--intra-lane` / `--no-record-time` | — | turn off the camera order / compare inside a track / do not use the clocks |
| `-j N` | `0` (auto) | how many files to decode in parallel; auto — half of the cores (2…16), on an HDD — up to 4 |
| `--threads N` | `0` (all cores) | correlation threads — fewer if your editing program runs at the same time |
| `--mem-limit GB` | `0` (auto) | memory ceiling: auto — ¾ of RAM, from 1 to 7 GB; minimum 0.25 GB |
| `--cache FOLDER` | system | folder of the feature cache |
| `--to xmeml\|drt` | by extension | output format |
| `--config FILE` / `--no-config` | `orcaton.toml` | your own settings file instead of the `orcaton.toml` next to the timeline / do not read settings files ([below](#settings-file-orcatontoml)) |
| `--brief` / `--json` | off | the result as one line / as JSON |
| `--progress auto\|text\|json\|none` | `auto` | progress on stderr: `auto` — a line, only if stderr is a terminal and there is no `--brief`/`--json`; `json` — one event per line (`stage`, `title`, `done`, `total`, `file` — the same stream as the window gets, [api.md](api.md)) |

Pair correlation uses all processor cores (the result does not depend on the number of cores).
Memory: parallel correlations are limited by the ceiling (`--mem-limit`), the sound feature is computed as
the file is decoded — a project of about 400 files and 100 GB of media fits in ~0.5 GB. What was chosen and
how much memory is used — the `Resources:` line in the `sync` summary (RAM → cap, threads, ffmpeg, disk
type, FFmpeg version; the process's peak memory). `orcaton version` — the program and FFmpeg versions (path; older than
4.4 — a warning).

## Settings file `orcaton.toml`
Default values of the `sync` options can be kept in a file — the window reads the same file
([docs/gui/user.md](../gui/user.md)). Layers, each next one wins:
1. the defaults from the table above;
2. the **user file** — for all projects: Windows `%AppData%\orcaton\orcaton.toml` · macOS
   `~/Library/Application Support/orcaton/orcaton.toml` · Linux `~/.config/orcaton/orcaton.toml`
   (`ORCATON_CONFIG` — your own path, `none` — do not read);
3. **`orcaton.toml` next to the timeline** (in the folder of the input `.xml`/`.drt`) or the file from `--config`;
4. a command-line flag; for `map` the `ORCATON_PATH_MAP` variable wins over the file.
```
orcaton config "Timeline 1.drt"          # which files are read and what is in effect
orcaton config --init                    # a user file template (all keys commented out)
orcaton config "Timeline 1.drt" --init   # a template next to the timeline
```
```toml
[sync]
map = 'D:\media\project'        # Windows paths — in single quotes
min-conf = 10.0
jobs = 4                        # HDD
originals = ['Sony', 'D:/Originals']   # relative paths — from the folder of this file
```
* Keys are the `sync` flag names without `--` (`min_conf` also works): `map`, `search`, `search-depth`,
  `keep-paths`, `originals`, `unsynced`, `order-by`, `singles`, `label`, `time-label`, `min-conf`, `min-ratio`,
  `auto-threshold`, `min-overlap`, `max-offset` (0 — no limit), `audio-mode`, `file-order`,
  `intra-lane`, `record-time`, `max-gap`, `keep-gap`, `jobs`, `threads`, `mem-limit` (GB), `cache`,
  `progress`. The `--no-…` flags are `false` of the key without `no-` in the file (`record-time = false`). Input, output and
  report are different for every run and do not go into the file.
* A typo in a key, a wrong type or a value out of range is an error with the file and key name, code 2
  (it never silently falls back to the default). Orcaton does not read or touch other sections of the file.
* `sync` prints a `Settings: <files>` line if any file was read.

## Other commands
| Command | What it does |
|---|---|
| `inspect FILE` | summary: sequences, tracks, FPS, labels, folders |
| `clips FILE [-t V1] [--csv OUT.csv]` | list of clips (seconds and TC) |
| `convert IN OUT [--map …]` | XML ⇄ model ⇄ XML, DRT → XML, moving paths |
| `cache [info\|clear\|prune] [--older-than 30d] [--dry-run] [--json]` | the feature cache (no subcommand — `info`) |
| `probe MEDIA`, `offset A B` | file data; the shift of sound B relative to A |
| `config [PROJECT] [--config FILE] [--init]` | `orcaton.toml` settings files and the values in effect; `--init` — a template (never overwrites) |

Exit codes: `0` — success; `1` — a runtime error (for example, the result could not be written); `2` — an input or
option error, including one in `orcaton.toml` (numbers out of range — the range is shown in `--help`, for example `--jobs INTEGER [1<=x<=256]`);
`3` — FFmpeg not found (when syncing by media — at once, before the analysis); `130` — `sync` interrupted
by Ctrl+C (ffmpeg processes finish, no output is written); `70` — an internal program error (send the message and the stack to the developer).

## Media not in place
A timeline stores full paths to files. If the project was edited on another computer or the media
was moved, point to the folder where it is now:
```
orcaton sync "Timeline 1.drt" "out.drt" --map "D:\media\project"
orcaton sync "Timeline 1.xml" "out.xml" --search "E:\Camera A" --search "F:\Sound"
```
* Files are searched **by name** (case-insensitive, NFC) in all subfolders down to `--search-depth` (6 —
  enough for `PRIVATE/M4ROOT/CLIP` of Sony). Files found in place are not touched.
* The same names in different folders (`1/C0992.MP4` and `3/C0992.MP4`) are told apart by the folders at the end of
  the original path (`…/project/3/C0992.MP4` → `…/3/`). If they cannot be told apart, the file is not substituted and
  a list of variants is printed — name the folder more exactly.
* The old form `--map "D:/media/project/=D:/media/project/"` (replacing the start of a path) works as before;
  both forms can be mixed: `--map "D:/=G:/;E:\Sound"`.
* **What was not found** — in a terminal the program shows a list and asks: a folder to search further
  (you can drag it into the window), **Enter** — skip, `q` — cancel. Skipped files go to the
  very end of the timeline with their old path — in your editing program they are offline and are relinked later
  (*Relink Clips*). Without a terminal (a script, `--yes`, `--brief`, `--json`) there is no question — they are skipped at once, the
  list is printed as a warning.
* The XML result gets the paths that were found (your editing program sees the media at once; `--keep-paths` — the old ones). In a
  `.drt` paths do not change (only positions are patched) — after the import relink the media in your editing program.
* Media search is a feature of `sync`.

## Cache
The sound feature of each file is computed once and kept in the cache: Windows
`%LOCALAPPDATA%\orcaton\cache` · macOS `~/Library/Caches/orcaton` · Linux `~/.cache/orcaton` (or
`ORCATON_CACHE_DIR`, `--cache`). If a sync was interrupted, a repeat takes the finished work from the cache.
The commands work from any folder; only Orcaton cache files are deleted:
```
orcaton cache                         # where the cache is and how much it takes (= cache info)
orcaton cache clear                   # clear everything (features are recomputed at the next sync)
orcaton cache prune --older-than 30d  # only what has not been used for long and *.tmp leftovers
```

## Environment variables
| Variable | Purpose |
|---|---|
| `ORCATON_PATH_MAP` | default `--map` (folders and/or rules) |
| `ORCATON_CACHE_DIR` | cache folder |
| `ORCATON_CONFIG` | user settings file instead of the system folder; `none` — do not read |
| `ORCATON_FFMPEG_DIR` | folder with ffmpeg/ffprobe — wins over everything; then: the program folder, PATH, usual folders |
| `ORCATON_CPUPROFILE` | (for developers) write a CPU profile for `go tool pprof` |

## Report
`--report` writes a CSV (`;`, UTF-8 with BOM — opens in Excel): for every file — the status, whom it
matched, the confidence (z), the recording time and the reason if it is not synchronized. For a `.drt` output the report
`OUT_report.csv` is always created — labels are not written to `.drt` yet.

## Questions and answers
**DRT: where are the Rose/Mango labels?** They are not written to `.drt` yet — the statuses are in `*_report.csv`. If you need
labels on the timeline, choose the `.xml` output.

**DRT from XML?** Not possible: a `.drt` is written by patching the source `.drt`. Export the timeline as `.drt`.
DRT → XML works.

**The media is on another drive, the result is needed on the original computer.** `--map "D:/Project/=G:/Project/"`
(or `--map FOLDER`) + `--keep-paths` — read from the new paths, write the old ones to the result.

**The timeline has proxies (ProRes from your editing program).** Proxies have no shooting time (theirs is the render time) —
point to the originals: `--originals "G:\Sony_card"`. Without the time, files without a match are placed worse. The shooting time
is also taken from the file name (`DJI_20261003231144_0085_D.MP4`; recorder `031026_191644_TrMic.WAV` — the order of the
date is chosen by the other files) and from the recorder's WAV (BWF).

**A file "found no match" (Rose).** Causes: the camera did not hear the common sound, the overlap with the others is
less than 2 s, silence. If you have the originals, `--originals` will place it by the camera clock (Mango).
The reason for every file is in the report.

**How many tracks are allowed?** No limit; time grows with the number of files (hundreds — minutes).

**Slow the first time.** FFmpeg reads the files in full; the number of parallel decoders is chosen by the
cores and the disk type (HDD — up to 4). If your editing program runs at the same time — lower `--threads`.
