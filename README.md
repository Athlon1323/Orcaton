# Orcaton

Orcaton synchronizes multicam video and separately recorded sound **by audio**. It reads a timeline
exported from your editor (FCP7 XML or DRT), finds where every clip belongs by comparing the sound, and
writes the timeline back with the clips lined up — in the same format.

* Any number of cameras, phones and audio recorders; the whole file is the unit, clips may be trimmed.
* Audio first; then the camera's own clock and the shooting order for files that cannot be matched by sound.
* A desktop window (Windows, macOS, Linux) and the `orcaton` command for scripts — the same engine.
* Interface in English, Deutsch, Русский, ქართული.

Status: **alpha** (version in [VERSION](VERSION), changes in [CHANGELOG.md](CHANGELOG.md)).

## Install

The installer builds Orcaton from source and installs it for the current user — no administrator rights
needed. It needs Go, Node.js and the Wails CLI (Windows — also WebView2):

* **Windows:** the script installs nothing by itself. It checks the tools and, if something is missing,
  prints the exact commands (`winget install …`, `go install …`); run them, open a new PowerShell window and
  run the script again.
* **macOS, Linux:** the script installs what is missing with the system package manager (Homebrew; apt / dnf —
  Linux asks for your password only to install system libraries).

| OS | Command (in the project folder) |
|---|---|
| Windows 10/11 | `powershell -ExecutionPolicy Bypass -File scripts\install.ps1 -Run` |
| macOS | `bash scripts/install.sh --run` |
| Linux | `bash scripts/install.sh --run` |

The tools are free software under their own licenses (Go — BSD, Node.js — MIT, Wails — MIT, FFmpeg —
LGPL/GPL, WebView2 — Microsoft). They are installed as separate programs and are not part of Orcaton; by
installing them you accept their licenses.

Result: the **Orcaton** app (Start menu / Applications / application menu) and the `orcaton` command.
Options: `--no-ffmpeg` (macOS, Linux), `-NoPath` / `--no-path`, `-NoDesktop` (Windows).
Details and troubleshooting — [docs/quickstart.md](docs/quickstart.md).

**FFmpeg** reads the audio. It is a separate free program and is not part of Orcaton; the installer
sets it up with the system package manager on macOS and Linux, and the window offers **Install FFmpeg**
if it is missing (on Windows — the window only).

## Uninstall

| OS | Command |
|---|---|
| Windows | Settings → Apps → Orcaton → Uninstall, or `powershell -ExecutionPolicy Bypass -File scripts\uninstall.ps1` |
| macOS, Linux | `bash scripts/uninstall.sh` |

Only Orcaton itself is removed (program, shortcuts, the `orcaton` command). Go, Node.js, Wails and
FFmpeg stay — other programs may use them. `-Purge` / `--purge` also removes settings and the audio cache.

## Use

The window: open a timeline → **Synchronize** → **Save**. Guide — [docs/gui/user.md](docs/gui/user.md)
(also [Deutsch](docs/gui/user.de.md), [Русский](docs/gui/user.ru.md), [ქართული](docs/gui/user.ka.md)).

The command line:

```
orcaton sync "Timeline 1.xml" "Timeline 1 - synced.xml"
orcaton --help
```

Full reference — [docs/go/usage.md](docs/go/usage.md); the engine API used by the window —
[docs/go/api.md](docs/go/api.md). Both are also available in Deutsch, Русский and ქართული (`*.de.md`, `*.ru.md`, `*.ka.md`).

## Source layout

```
app-go/    engine and the orcaton command (Go, no external dependencies)
app-gui/   desktop window (Wails v2 + Svelte 5 / TypeScript)
scripts/   install, uninstall, set-version
docs/      user guides
```

## License

Apache License 2.0 — see [LICENSE](LICENSE) and [NOTICE](NOTICE). Third-party components and their
licenses — [THIRD-PARTY-NOTICES.md](THIRD-PARTY-NOTICES.md).
