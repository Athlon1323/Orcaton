# Changelog

Versions follow `MAJOR.MINOR.PATCH`; the single source of the version is the file [VERSION](VERSION).

## 0.0.47 — 2026-11-22

Bug-fix release: better sync for long shoots with a separate audio recorder.

* Recorder WAV files (BWF) now get their recording time: the `date` and `creation_time` tags are read together.
* Recording time is also taken from the file name (DJI, phones, recorders: `…_YYYYMMDDHHMMSS`, `YYYYMMDD_HHMMSS`,
  six-digit recorder dates) — even when the media is not on the disk; an ambiguous date is resolved by the file's
  date tag or by the other files of the same camera.
* A recorder that writes every take into its own folder is treated as one camera (common order and clock).
* Groups of files are joined when one recorder take runs continuously through both of them, even if the
  camera clocks disagree.
* Groups that still cannot be joined are placed in shooting order (by the clocks of shared cameras), so the
  morning no longer ends up after the evening.
* One false audio match between a short clip and a long recorder take no longer puts the morning on top of the
  evening recording: groups are joined by a continuous recorder take first, and a camera's clock in a joined
  group follows the majority of its files. The ceremony and the preparation clips are no longer pushed to the
  end of the timeline; all events go in shooting order.
* A camera clock that rests on a single uncertain match no longer joins groups on its own (a second camera
  must agree); such a match far off the camera clock is dropped and the clip is placed by its clock. Groups
  that cannot be joined are put in shooting order even when another group stands between them, and the result
  is the same from run to run.
* A long take that the recorder cut into 2 GB / 4 GB files stays one continuous recording even when the first
  file's time stamp is early (record standby); the clips filmed under it are no longer shifted.
* All messages of the command-line tool and of the install, build and packaging scripts are now in English (the window stays in English, German, Russian and Georgian).
* The command-line guide (`docs/go/usage.md`) and the API guide (`docs/go/api.md`) are now in English, German, Russian and Georgian, like the window guides.
* The message for a file that is not on the disk no longer names a particular editor.

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
