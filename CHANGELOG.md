# Changelog

Versions follow `MAJOR.MINOR.PATCH`; the single source of the version is the file [VERSION](VERSION).

## 0.0.46 — 2026-10-26

* Windows installer (`.exe`, x64 and arm64): installs for all users into Program Files, asks for the folder, adds `orcaton` to the system PATH, and removes the old per-user copy.
* Linux packages (`.deb`, `.rpm`) and macOS `.dmg` build scripts; FFmpeg is not a package dependency.
* Uninstall also removes the window's WebView2 data.

## 0.0.45 — 2026-10-20 — first alpha

First build for alpha testing (not published yet).

* One package: the window `orcaton-gui` and the command-line tool **`orcaton`** (formerly `orcaton-go`)
  in one folder, with `LICENSE`, `NOTICE`, `THIRD-PARTY-NOTICES.md`.
* FFmpeg is not bundled: when it is missing, the window offers **Install FFmpeg** (winget on Windows,
  Homebrew on macOS, the package manager on Linux) and shows the command first. FFmpeg 4.4 or newer;
  an older one gets a warning. `orcaton version` and the sync summary show the FFmpeg version.
* FFmpeg is looked up in `ORCATON_FFMPEG_DIR`, next to the program, in PATH and where winget, scoop,
  Chocolatey and Homebrew put it.
* One version everywhere: CLI `orcaton version`, the About window, Windows file properties.
* Sync: audio → camera window → camera clock → unsynced in camera order; FCP7 XML and DRT in and out;
  window in English, German, Russian and Georgian.
