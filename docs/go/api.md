# Core API for the window (`orcaton/api`)

**English** · [Deutsch](api.de.md) · [Русский](api.ru.md) · [ქართული](api.ka.md)

The public Go package `app-go/api` is what the graphical window (`app-gui/`, Wails) calls the core through. The CLI `orcaton sync` runs on the same
scenarios (`internal/app`), so the window and the command line give an **identical result**: the test
`api/gui_test.go` compares the API output and report byte by byte with CLI snapshots. The `internal/…` packages are not available from
another module (a Go rule) — the window sees only `api`.

## Connecting
```
app-gui/go.mod:   require orcaton v0.0.0
                  replace orcaton => ../app-go
code:             import "orcaton/api"
```
The core has no external dependencies; the Wails dependencies live only in `app-gui/`. The contract version is
`api.Version` (changes on incompatible edits), the program version is `api.ProductVersion`.

## The window's scenario
| Step | Call | What to show |
|---|---|---|
| start | `api.FFmpegAvailable()` | no FFmpeg — `api.Describe(err)`: the text + an install hint |
| About | `api.GetFFmpegInfo()` → `{ffmpeg, ffprobe, version, old, min}` (`old` — older than `min` = 4.4: warn); `api.GetCacheInfo(dir)` → `{path, exists, entries, bytes, partial}`, `api.ClearCache(dir)` (`""` — the system folder) | FFmpeg version and path, cache size, "clear" |
| open XML/DRT | `p, err := api.Open(path, api.OpenOptions{})` → `p.Info()` | cameras, files (`ok` / `missing` / `offline` / `nomedia`), how many were not found, save formats (`outputFormats` — only the input format: XML → XML, DRT → DRT) |
| find media | `p.FindMedia(ctx, api.FindRequest{Roots, Ask, OnSearch})` | the result of each search (`SearchReport`), the question — a dialog (`Asker`) |
| settings | `api.LoadSettings(project)` → `{options, media, files}`; `api.DefaultSettings()` / `api.DefaultOptions()`, `o.Validate()`; lists — `api.UnsyncedModes`, `api.OrderModes`, `api.SinglesModes`, `api.AudioModes`; save — `api.SaveSettings(path, s)` | a `bad_options` error with the field name (in a file — with the path and key) |
| synchronization | `res, err := api.Sync(ctx, p, api.SyncRequest{Options: o}, onEvent)` | a progress bar from `Event`; "Stop" — `cancel()` of the ctx |
| result | `res.Summary`, `res.Files` | a table: sound / clock / no match / not on disk (`FileResult.Result`), the reason as a code (`reasonCode`), the partner; "before" — `startSec`, "after" — `positionSec` / `positionTC` |
| resources | `Options.Jobs` / `Threads` / `MemLimitMB` (0 — auto: ffmpeg by cores and disk, all cores, ¾ of RAM within 1…7 GB); the result — `res.Summary.Resources` (`ramMB`, `memLimitMB`, `threads`, `jobs`, `disk`, `peakMB`, `ffmpeg` — the version of the FFmpeg that read the sound) | the "Performance" settings; the memory peak — in the result window |
| save | `res.Save(path, api.SaveOptions{Format, KeepPaths}, onEvent)`, `res.SaveReport(csv)` | may be called several times; the window offers only the input format; `.drt` — only for DRT input; DRT → XML — with an explicit `Format: "xmeml"` (the terminal does it that way) |

## "Before / after" timeline by camera
* A timeline row is a **camera** (`ProjectInfo.cameras`: `id`, `title` "6D · V1+A1", `tracks`, `files`);
  a file belongs by `camera`, the file's tracks are `tracks`. `id` is "track|folder"; if the folder holds files with different
  extensions (two cameras in one folder) — "track|folder|.ext", `title` — "6D2 MOV · V2+A2".
* "Before" — `MediaFile.startSec` / `FileResult.startSec`: the start of the file on the input timeline (clip =
  file start + in point), seconds from the start of the sequence; "after" — `FileResult.positionSec`
  (`nil` — the file was removed). The length is the file's `durationSec`.
* Timecode: `SequenceInfo{fpsNum, fpsDen, startSec, startTC}` — the window draws the scale itself;
  ready strings — `MediaFile.startTC`, `FileResult.positionTC`; `api.FormatTC(sec, seq)` — the same
  in Go (the sequence's start TC, no drop-frame — as in the XML output).
* The waveform on clips in v1 is decorative (the API does not hand out the sound feature).

## Texts and translation
The window is localized (English, German, Russian, Georgian); the core's texts are in English. Translate **by code**; the core's text is a fallback and
for "Details": the stage — `Event.stage`; the error — `ErrorInfo.kind`; the file's reason —
`FileResult.reasonCode`: `no_match` (no reliable match by sound), `short_audio`, `no_audio`,
`offline` (not on disk, offline in the editing program), `no_path`, `not_found` (not found by path),
`decode_error`, `conflict` (overlap on its own track), `by_clock` (by the clock, the sound did not confirm),
`duplicate` (a copy of another project file on a track shared with it; in DRT tracks are not added — the copy has no match);
constants — `api.Reason*`. The texts of the CLI and the CSV report do not change.

`Summary.duplicates` — copies of files (the same sound): in XML they stand together with the original, their clips
on overlap go to additional tracks (V/A with a number after the last one); `Summary.timeShift` — of
`timeAudio` confirmed by the shift check (music).

All types are structs with json tags (Wails builds the TypeScript models from them). `Result` keeps
the result in memory: the window holds the pointer in its own struct and hands only `Summary`/`Files` to JS.

## Progress — `Event`
`{stage, title, done, total, file}`; `done == total` — the stage is finished, `total == 0` — no counter.
Stages in order: `search` (media search) → `record_time` (camera clocks) → `features` (sound, per file:
`file`) → `pairs` (pair matching, ~1 % — up to 100 events) → `refine` → `solve` (3 steps) → `layout` →
`write`. The same stream is printed by `orcaton sync --progress json` (a JSON line per event, stderr).
The handler is called sequentially but from worker goroutines: only `runtime.EventsEmit` and return.

## Cancellation
The `ctx` of `Sync` and `FindMedia`. It is checked between stages and inside long ones: recording time — per file,
features — ffmpeg processes are terminated (`exec.CommandContext`), pairs — per series. An error of kind
`cancelled`, no result, the project does not change. Indexing folders during the media search (a disk walk)
cannot be interrupted yet — only between searches.

## The question about missing media — `Asker`
`AskMissing(ctx, missing []MissingFile) (Answer, error)` is called while the list is not empty.
The answer: `{action: "folder", folder}` — search there (if a file is given, its folder is searched), `"skip"` —
the rest go to the very end as "not on disk" (offline in the editing program), `"cancel"` — a `cancelled`
error. The call is blocking: in Wails — an event to the window and waiting for the answer from a channel:
```go
func (a *App) AskMissing(ctx context.Context, ms []api.MissingFile) (api.Answer, error) {
	runtime.EventsEmit(a.ctx, "media:missing", ms)
	select {
	case ans := <-a.answers: // the App.Answer(ans) method is called from JS
		return ans, nil
	case <-ctx.Done():
		return api.Answer{}, ctx.Err()
	}
}
```
In the CLI the terminal stands in its place (`internal/cli/mediafind.go`, `termAsker`).

## Settings — `orcaton.toml`
The window and `orcaton sync` read the same files ([usage.md](usage.md#settings-file-orcatontoml)):
* `api.LoadSettings(project)` — defaults ← the user file (`api.UserSettingsPath()`) ← the file
  next to the project (`api.ProjectSettingsPath(project)`; `""` — only the user). The result is
  `Settings{Options, Media{Map, Search, SearchDepth, KeepPaths, Originals}, Files}`;
  `Media.SearchRoots()` — folders for `FindRequest.Roots` (like `--map FOLDER` + `--search`).
  `Map` → `OpenOptions.PathMap`, `KeepPaths` → `SaveOptions`, `Originals` → `SyncRequest`.
* An error in a file — `bad_options` (the text has the path and key) and `DefaultSettings()`: the window shows
  the error and works with the defaults.
* `api.SaveSettings(path, s)` — writes to the `[sync]` section the values that differ from the defaults (a key
  already written in the file is updated even when it equals the default); comments, other sections and terminal-only
  keys (`progress`) are not touched; the write is atomic; wrong settings — `bad_options`, the file
  does not change. Where to save (user / project) is up to the window.
* `MemLimitMB` is stored in the file in GB (`mem-limit`), like the flag.

## Errors
`api.KindOf(err)` → `cancelled` · `no_ffmpeg` · `not_found` · `bad_format` · `bad_options` ·
`bad_output` · `write` · `failed`; `api.Describe(err)` → `{kind, message, hint}` for the window. The error
text is the same as in the CLI. No FFmpeg when syncing by media — `no_ffmpeg` at once (before,
every file got a "decode error").

## Threads
One `Project` — one operation at a time (`FindMedia` substitutes paths in the project). `Sync` does not change the project
(it works on a copy), `Result.Save` may be called repeatedly.
