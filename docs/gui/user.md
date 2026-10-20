# Orcaton with a window — user guide

**English** · [Deutsch](user.de.md) · [Русский](user.ru.md) · [ქართული](user.ka.md)

> **Status: alpha.** The window already works: open a timeline → find media →
> synchronize → save. Items marked *(later)* are not in the window yet. The terminal does all the same —
> [docs/go/usage.md](../go/usage.md) (in Russian). **The terminal stays**: the window and
> `orcaton sync` run on the same core and give the same result.

## How to open the window
Orcaton with a window is a **separate desktop app**: it opens its own program window (taskbar icon,
Alt+Tab, the usual minimize/close buttons). It is **not a website or a browser page**: everything the
program shows is inside its own window. If you open Orcaton in a browser (for example the address
`http://localhost:…` printed by the terminal during development), the page shows the warning "Open
Orcaton as an app" and does nothing: a browser can neither choose files on your computer nor run a
synchronization.

**Installation** — one command in a terminal in the project folder; step by step for Windows, macOS and
Ubuntu — [docs/quickstart.md](../quickstart.md):
```powershell
powershell -ExecutionPolicy Bypass -File scripts\install.ps1 -Run      # Windows 10/11
```
```bash
bash scripts/install.sh --run                                        # macOS, Ubuntu / Linux
```
The command installs what is missing (Go, Node.js, Wails, FFmpeg; on Linux the window libraries), builds
Orcaton from source, installs it for your user and opens it. Running it again updates the program;
settings stay.

**Starting** — like any program, without a terminal:

| OS | Where Orcaton is | Program file |
|---|---|---|
| Windows | **Orcaton** in the Start menu and on the desktop | `%LOCALAPPDATA%\Programs\Orcaton\orcaton-gui.exe` |
| macOS | Launchpad / Spotlight "Orcaton", Finder → `~/Applications` | `~/Applications/Orcaton.app`; from the `.dmg` — `/Applications/Orcaton.app` |
| Ubuntu / Linux | applications menu → **Orcaton** | `~/.local/share/orcaton/orcaton-gui`; from the package — `/usr/bin/orcaton-gui` |

**Uninstall** — Windows: Settings → Apps → Orcaton (or `scripts\uninstall.ps1`); macOS, Linux:
`bash scripts/uninstall.sh`. Only Orcaton is removed; Go, Node.js, Wails and FFmpeg stay.

## Language
At the bottom of the left column (globe icon) — **English**, **Deutsch**, **Русский**, **ქართული**. The
labels change immediately, without a restart; the choice is remembered between launches (file `gui.json`
next to your `orcaton.toml`). English is the default. Below, button names are as in the English window.

## You choose the files and folders
The program chooses nothing for you: every path is a **system dialog** of Windows / macOS / Linux (the
same as in other programs), where you browse folders and pick what you need. Closing the dialog
(Cancel) does nothing.

| Where in the window | What opens | What you choose |
|---|---|---|
| Import → the **Drop XML / DRT or browse…** box (click anywhere in it), the large box in the middle of the empty window, **Replace…** | "Open" dialog filtered to `*.xml; *.drt` | the timeline file (the folder — via the dialog); you can also drag the file into the window |
| Import → **+ Add folder** | folder dialog | the folder with the source files; search by file name in all its subfolders |
| **N files not found → Locate…** and the "media files not found" dialog → **Choose folder…** | folder dialog | the folder with the missing files (Skip — don't search, Cancel — abort) |
| **Save timeline** (at the top after synchronizing) | "Save as" dialog | folder and name of the output; format as the input (`.xml` → `.xml`, `.drt` → `.drt`); suggested `<input> - synced.<ext>` next to the input — you can change both folder and name |
| **Save report (CSV)** | "Save as" dialog | folder and name of the report; suggested `<output>_report.csv` |
| Settings → **Cache folder → Choose…** | folder dialog | the audio cache folder (empty — system folder) |
| toast **Show in folder**, Developer tools → **Open logs folder** | Explorer / Finder | — (only shows the folder) |

Without a dialog the program takes paths only from what you specified earlier yourself: search folders
and settings from `orcaton.toml` (Settings, `orcaton config`) and media paths stored in the timeline.

## In short: from the video editor and back
1. **Video editor:** one camera or recorder — one track; export the timeline
   (*File → Export → Timeline…*) → **FCP 7 XML V5** (`.xml`) or **DRT** (`.drt`).
2. **Orcaton:** drag the file into the window or press **Import → browse…** and choose it in the
   dialog. If some media are not found — **+ Add folder** and point the dialog to the folder with the
   source files.
3. Press **Synchronize** (the green button bottom left). Watch the progress at the top; **■** (Stop
   synchronization) — cancel (nothing changes).
4. **Save timeline** → choose the folder and name in the dialog (suggested `Timeline 1 - synced.xml` or
   `.drt` — in the input format, next to it); optionally **Save report (CSV)**.
5. **Video editor:** import the saved file (*File → Import → Timeline…*).

## The window
```
┌───────── progress: step, file, %, elapsed / left · after synchronizing — summary and Save ──────────┐
│ About              │  Timeline | Files     Before | After     zoom · Fit                             │
│ Settings           │                                                                                │
│ Import  ◀ default  │  timeline: a row per camera, clips in the colour of their state                 │
│ ── Developer tools │  or the file table                                                             │
│ English ▾          │                                                                                │
│ [ Synchronize ] ■  │  legend-filter: Matched by audio · By camera clock · No match · Not on disk    │
└────────────────────┴────────────────────────────────────────────────────────────────────────────────┘
```
* **Left column** — sections: **Import** (what we synchronize), **Settings**, **About**,
  **Developer tools** (debugging; visible while the program is in pre-alpha). At the bottom — the language
  and **Synchronize** / **■**.
* **Right** — two views with a switch: **Timeline** (a row per camera; camera = track + folder, e.g.
  "6D · V1+A1") and **Files** (table). **Before / After** — original places or the result. Before
  synchronizing **Files** opens, after — **Timeline After**; afterwards the window remembers your choice.

## Import: timeline and media
* File: name, format, fps, tracks, clips, files. **Replace…** — open another one.
* **Media folders** — folders to search for source files if the paths in the timeline are not found
  (project from another computer, moved to another drive). Search by file name in all subfolders (up to
  6 levels, Sony `PRIVATE/M4ROOT/CLIP` is found). Identical names in different folders are told apart by
  the folders at the end of the original path; if they cannot be told apart, the window shows the options.
* **Camera originals** — if the timeline holds proxies (ProRes from the editor): folders with the camera
  originals, to take the recording time (proxies don't have it). *(later in the window; for now — the
  `originals` key in `orcaton.toml`, which the window reads.)*
* **Media status:** found / not found / offline. Not found > 0 — a yellow box with **Locate…**.

**The "N media files not found" dialog** (also appears before synchronizing if something is missing):
**Choose folder…** — search one more folder; **Skip** — the rest goes to the very end of the timeline
as "not on disk" (offline in the editor, relink later); **Cancel** — don't start.

## Result
At the top — "✓ Done in 37 s" and counters. Every file has a colour **and** a symbol:

| In the window | In the editor after import | Meaning |
|---|---|---|
| ✓ Matched by audio (green) | no label | synchronized **by audio**, frame-accurate |
| ◷ By camera clock (amber) | label **Mango** | placed **by the camera clock** — audio did not confirm (≈ 1 s) |
| ✕ No match (pink) | label **Rose** | no match found — placed by recording time |
| ⌀ Not on disk (dashed) | offline clip at the end | the file is not on disk |

* The order of a camera's files — by the number in the name (MVI_0001, C0001, 00001…), then by
  recording time; numbers of different cameras are not compared (6D and 6D2 number separately).
* Files without a match never play at the same time as other files without an audio match (setting
  **Unmatched layout**, default *One after another*): they go next to their camera neighbours, time is
  added if needed; synced blocks are not changed. *Stacked by camera* — as before.
* Files without a match — in this camera order, next to their neighbours (including inside a synced
  block); only files missing on disk go to the very end (before synchronizing the window asks where to
  look for them: **Choose folder…** or **Skip** — then to the end); no gaps.
* Click a file — a card: camera, position (TC), duration, partner, confidence, recording time,
  **reason** (why not by audio).
* **Files**: sorting by columns, search by name, filter by result — start with **No match**.
* **Synchronize** again — a new run with the current settings (a repeat is fast: audio comes from the cache).

## Saving
* **Save timeline** — in the input format, without conversion: an opened `.xml` is saved as `.xml`, an
  opened `.drt` — as `.drt` (only clip positions, duration and the timeline name change; everything else
  stays as it was). DRT to XML can be converted in the terminal (`orcaton sync in.drt out.xml`). You
  choose the folder and name in the "Save as" dialog; suggested `<input> - synced.<ext>` next to the input.
  The **Keep original media paths** switch (Settings) — write the original paths (if the result will be
  opened on the original computer).
* Rose/Mango labels are not written to `.drt` yet — the window offers to save the report.
* **Save report (CSV)** — a table for every file (opens in Excel). **Show in folder** — open the folder.

## Settings
Each field is the same as a terminal option (hint next to the field). **Reset to defaults** — restore
everything. An invalid value is highlighted under the field. Settings are kept between launches — right
after editing, in your `orcaton.toml` (the path is at the bottom of the section).

| Group · field in the window | `orcaton sync` option | Default |
|---|---|---|
| Unmatched files · Placement | `--unsynced order_time\|edges\|end\|remove` | order_time (In camera order) |
| Unmatched files · Camera order | `--order-by name\|time` | name (File number first) |
| Unmatched files · Unmatched layout | `--singles sequential\|stacked` | sequential (One after another) |
| Unmatched files · Label / Clock-only label | `--label`, `--time-label` | Rose, Mango |
| Audio matching · Audio source | `--audio-mode mix\|first_stream\|first_channel` | mix (Mix all channels) |
| Audio matching · Min confidence / Raise threshold to project noise | `--min-conf`, `--no-auto-threshold` | 12, on |
| Audio matching · Min ratio / Min overlap / Max offset | `--min-ratio`, `--min-overlap`, `--max-offset` | 1.5, 2 s, no limit |
| Camera clock · Use recording time / Keep file order within a camera / Compare clips on the same track | `--no-record-time`, `--no-file-order`, `--intra-lane` | on, on, off |
| Gaps · Remove gaps longer than / Keep | `--max-gap`, `--keep-gap` | 0 (all), 0 |
| Performance · Parallel decoders / Threads / Memory limit | `-j`, `--threads`, `--mem-limit` | auto: ½ of cores (HDD ≤ 4), all cores, ¾ of RAM 1…7 GB |
| Performance · Cache folder | `--cache` | system |
| Import · Media folders | `--map FOLDER`, `--search` | — |
| Import · Camera originals | `--originals` | — |
| Saving · Keep original media paths | `--keep-paths` | off |
| Save report | `--report` | always for `.drt` |

Window and terminal settings are shared: your `orcaton.toml` and the one next to the timeline
([usage.md](../go/usage.md#файл-настроек-orcatontoml), in Russian); what you change in the window also
applies to `orcaton sync`. The window language is not a terminal setting — it lives in `gui.json`.

## About and Developer tools
* **About:** program version, FFmpeg (path and version; a warning if older than 4.4), audio cache (folder, size, **Clear cache**),
  licenses. macOS: **Command line** — **Install command-line tool** / **Remove command-line tool**: the `orcaton`
  command for Terminal (`/usr/local/bin/orcaton`, a link to the program inside Orcaton.app; macOS asks for your password).
  Shown when Orcaton is started from the Applications folder.
* **Developer tools:** synchronization event log (copy), window logs folder
  (`%LocalAppData%\orcaton\logs`), cache.

## Terminal only
`inspect`, `clips`, `convert`, `probe`, `offset`, `config`, `cache`, `--json` / `--progress json` for scripts — [docs/go/usage.md](../go/usage.md) (in Russian).

## Questions
**The window says "FFmpeg is not installed".** Orcaton reads audio with FFmpeg — a free program that is
installed separately. Press **Install FFmpeg**: the window shows the exact command first and installs FFmpeg
with the system package manager — winget on Windows, Homebrew on macOS, apt / dnf / pacman / zypper on Linux
(asks for your password). Then it checks again by itself. If the button is not offered (no winget, no
Homebrew, or no graphical password prompt), run the shown command in a terminal and press **Check again**;
on a Mac without Homebrew, install Homebrew first (brew.sh). You can also point Orcaton at any folder with
FFmpeg using the variable `ORCATON_FFMPEG_DIR`. Any FFmpeg 4.4 or newer works; with an older one the window
shows a warning. A timeline can be opened without FFmpeg, synchronized — not. Other installation and
start-up errors — [quickstart.md](../quickstart.md#if-something-goes-wrong).

**Slow the first time.** The audio of every file is read in full once, then it comes from the cache. If
the video editor is running at the same time — lower **Threads**.

**Many "No match".** Look at the reason in the card: no common audio, overlap < 2 s, silence. For
proxies set **Camera originals** — the files will be placed by the camera clock.

*(later)* recent projects on the empty screen, real audio waveform on clips.
