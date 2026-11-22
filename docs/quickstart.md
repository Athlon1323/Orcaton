# Orcaton — installation and first run (Windows, macOS, Ubuntu)

**English** · [Deutsch](quickstart.de.md) · [Русский](quickstart.ru.md) · [ქართული](quickstart.ka.md)

Orcaton is a **desktop app with its own window**: it opens like any other program (its own window, an
icon in the taskbar or Dock). No web browser is needed. Everything below is step by step and needs no
terminal experience: you use the terminal **once** — to paste a single command. After that, Orcaton
starts from a shortcut.

What you need: the project folder on your computer, an internet connection during installation,
10–20 minutes and about 3 GB of free space (compilers and libraries for the build). Installers may ask
for permission: Windows — the window "Do you want to allow this app to make changes to your device?" →
**Yes**; macOS and Ubuntu — the password of your user account (the characters are not shown while
typing — that is normal). Running the install command again is safe: what is already installed is not
installed again.

## Windows 10 / 11
1. **Open PowerShell in the project folder.** In File Explorer go to the project folder, right-click an
   empty spot → **Open in Terminal** (Windows 11). In Windows 10 — hold **Shift**, right-click →
   **Open PowerShell window here**. The prompt must show the path to the project folder, for example
   `PS D:\Projects\Orcaton>`.
2. **Check what is already installed.** Paste the command (right-click inside the window pastes) and press **Enter**:
   ```powershell
   powershell -ExecutionPolicy Bypass -File scripts\install.ps1
   ```
   The script installs nothing by itself: it only checks Go, Node.js, WebView2 and the Wails CLI. If something
   is missing, it prints the commands. Run them **one at a time** (if installer windows pop up — **Yes**):
   ```powershell
   winget install --id GoLang.Go -e
   winget install --id OpenJS.NodeJS.LTS -e
   winget install --id Microsoft.EdgeWebView2Runtime -e
   go install github.com/wailsapp/wails/v2/cmd/wails@v2.13.0
   ```
   Then open a **new** PowerShell window in the project folder and run the command from step 2 again.
   FFmpeg is optional: if it is missing, the window offers to install it on first start.
3. **Install and build.** When all tools are in place, run:
   ```powershell
   powershell -ExecutionPolicy Bypass -File scripts\install.ps1 -Run
   ```
   The script builds the app, creates the shortcuts and opens the window. Lines `== 2/3 Build ==`
   and `== 3/3 Installing to … ==` appear.
4. **Done** when the **Orcaton** window opens and the terminal shows a green `Done: Orcaton … is installed`. The
   **Orcaton** shortcut is now on the desktop and in the Start menu; you can close the terminal. From now
   on — just the shortcut.

### Windows: ready-made installer (.exe)
If you received Orcaton as a file `Orcaton-<version>-windows-x64-setup.exe` (most PCs) or
`Orcaton-<version>-windows-arm64-setup.exe` (Windows on ARM, for example Snapdragon laptops), nothing needs
to be built: Go, Node.js and Wails are not needed. Windows 10 or 11.
1. **Run the installer** (double-click). Orcaton is free and is not signed with a paid certificate, so
   Windows may show "Windows protected your PC" → **More info** → **Run anyway**. Then Windows asks
   "Do you want to allow this app to make changes to your device?" → **Yes** (administrator rights: Orcaton is
   installed for all users of the computer).
2. **Options:** *Add the orcaton command to PATH* and *Desktop shortcut* (both on); the Start menu
   shortcut is always created (for all users).
3. **Folder:** `C:\Program Files\Orcaton` by default; **Browse…** — another folder. On the last page — *Run Orcaton*.
4. **FFmpeg.** If the window shows **FFmpeg is not installed** — press **Install FFmpeg** (it uses winget).

* Update — run the new installer (close Orcaton first; the installer asks if it is still open). Settings stay.
* An earlier copy of Orcaton for your user only (`%LOCALAPPDATA%\Programs\Orcaton` — an older installer or
  `scripts\install.ps1`) is removed by the installer automatically; settings stay — nothing to remove by hand.
* Check the downloaded file: in PowerShell `(Get-FileHash Orcaton-…-setup.exe).Hash` must print the same value
  as the `.sha256` file next to it (upper / lower case does not matter).
* With **Smart App Control** turned on (Windows 11), unsigned programs do not start at all; Orcaton then works
  only with it turned off (Windows Security → App & browser control → Smart App Control).

## macOS (Apple Silicon and Intel)
1. **Homebrew** (a package manager; skip if you already have it). Open **Terminal** (Spotlight:
   ⌘ + Space → "Terminal") and run the command from <https://brew.sh> (one line, it asks for your Mac
   password). At the end Homebrew prints "Next steps" — run the two commands shown (`echo …` and `eval …`).
2. **Terminal in the project folder:** type `cd ` (with a space), drag the project folder from Finder
   into the Terminal window and press **Enter**.
3. **Install command:**
   ```bash
   bash scripts/install.sh --run
   ```
   If macOS offers to install the **Command Line Tools** — click "Install", wait until it finishes and
   **run the same command again**.
4. **Done** when the **Orcaton** window opens. The app is in **~/Applications** (your user's
   Applications folder): find it via Launchpad or Spotlight ("Orcaton") and keep it in the Dock
   (right-click the icon → Options → Keep in Dock).

### macOS: ready-made package (.dmg)
If you received Orcaton as a file `Orcaton-<version>-macos-universal.dmg` (one package for Apple Silicon
and Intel; macOS 12 Monterey or newer), nothing needs to be built or installed in the terminal:
1. **Copy the app.** Double-click the `.dmg` → a window with **Orcaton** and the **Applications** folder
   opens → drag **Orcaton** onto **Applications**. Then eject the disk (Finder sidebar → ⏏).
2. **First launch.** Orcaton is free and is not signed with a paid Apple certificate, so macOS stops the
   first launch. Open Orcaton from Applications → the message "Apple could not verify…" (or "cannot be
   opened") → **Done** / **OK**. Open **System Settings → Privacy & Security**, scroll down to "Orcaton was
   blocked…" → **Open Anyway** → your Mac password → **Open**. On macOS 12–14 it is enough to right-click
   Orcaton → **Open** → **Open**. This is needed once (and again after installing a new version).
3. **FFmpeg.** If the window shows **FFmpeg is not installed** — press **Install FFmpeg** (it uses
   Homebrew, <https://brew.sh>; without Homebrew the window shows what to do).
4. **Terminal command (optional):** **About → Command line → Install command-line tool** (macOS asks for
   your password). After that `orcaton` works in a new Terminal window.

Update — drag the new Orcaton onto Applications → **Replace**. To check the downloaded file: in Terminal
`shasum -a 256 Orcaton-…dmg` must print the same value as the `.sha256` file next to it.

## Ubuntu 22.04 / 24.04 (and other Linux)
1. **Terminal in the project folder:** in Files open the project folder, right-click an empty spot →
   **Open in Terminal**.
2. **Install command** (asks for your password — the characters are not shown, that is normal):
   ```bash
   bash scripts/install.sh --run
   ```
   The script installs the window libraries (GTK, WebKitGTK), FFmpeg, Go, Node.js and Wails, builds the
   app and adds a shortcut.
3. **Done** when the **Orcaton** window opens. The shortcut is in the applications menu (Super key →
   "Orcaton"); you can add it to favourites (right-click → Add to Favourites).
Fedora and Arch — the same command (the script uses `dnf` / `pacman`).

### Linux: ready-made package (.deb / .rpm)
If you received Orcaton as a package file, nothing needs to be built. The package manager installs Orcaton
together with the window libraries (GTK 3, WebKitGTK 4.1); on a desktop system they are usually already
there, so the package also installs without internet. **FFmpeg** is not part of the package (as on macOS):
install it from your distribution afterwards — see below. Open a terminal in the folder with the file and
run (asks for your password):

| System | File | Command |
|---|---|---|
| Ubuntu 22.04 / 24.04 / 26.04, Debian 12, Linux Mint 21+ | `orcaton_<version>-1_amd64.deb` | `sudo apt install ./orcaton_<version>-1_amd64.deb` |
| Fedora 39 or newer | `orcaton-<version>-1.x86_64.rpm` | `sudo dnf install ./orcaton-<version>-1.x86_64.rpm` |

* `./` at the beginning of the file name is needed — otherwise apt / dnf looks for the package online.
* Orcaton is not signed: if dnf refuses because of a missing signature, add `--nogpgcheck` to the command.
* After that: applications menu → **Orcaton**; in a terminal — the `orcaton` command.
* **FFmpeg:** on the first start the window shows **FFmpeg is not installed** with the command → **Install FFmpeg**
  (asks for your password; needs internet). Or in a terminal: `sudo apt install ffmpeg` (Fedora: `sudo dnf install /usr/bin/ffmpeg`).
* If Orcaton was installed earlier with `bash scripts/install.sh`, remove that copy first:
  `bash scripts/uninstall.sh` — otherwise the menu keeps opening the old one.
* Update — the same command with the new file. Check the downloaded file: `sha256sum -c <file>.sha256`.

## First synchronization
0. **Window language:** at the bottom of the left column (globe icon) choose **English**, **Deutsch**,
   **Русский** or **ქართული** — the labels change immediately and the choice is remembered. English is
   the default; the button names below are as in the English window.
1. **Video editor:** each camera / recorder on its own track; export the timeline
   (*File → Export → Timeline…*) → **DRT (.drt)** or **FCP 7 XML V5 (.xml)**.
2. **Orcaton:** click the **"Drop XML / DRT or browse…"** box — the usual file dialog opens; choose the
   exported timeline (or simply drag the file into the Orcaton window).
3. If the media are not where they were during editing — **+ Add folder** and choose the folder with
   the source files (search by name in all subfolders). You can also do it later: before synchronizing,
   Orcaton asks by itself.
4. Press the green **Synchronize** (bottom left). Progress is shown at the top; **■** — stop.
5. **Save timeline** (at the top) — choose the folder and name in the "Save as" dialog.
6. **Video editor:** import the saved file (*File → Import → Timeline…*). No label — matched by audio,
   **Mango** — by camera clock, **Rose** — no match found.

Every button in detail — [docs/gui/user.md](gui/user.md).

## Update the app (after code changes)
Run the same install command again in the project folder: it rebuilds Orcaton and replaces the installed
copy (close the Orcaton window first). Settings stay.

## Uninstall
| OS | How |
|---|---|
| Windows | Settings → Apps → **Orcaton** → Uninstall, or `powershell -ExecutionPolicy Bypass -File scripts\uninstall.ps1` |
| Windows, from the `.exe` | Settings → Apps → **Orcaton** → Uninstall; tick *Also remove settings and cache* to delete them as well |
| macOS / Linux | `bash scripts/uninstall.sh` |
| Linux, from the `.deb` / `.rpm` | `sudo apt remove orcaton` / `sudo dnf remove orcaton`; FFmpeg and the libraries stay, settings — in `~/.config/orcaton` |
| macOS, from the `.dmg` | About → **Remove command-line tool** (if installed), then drag Orcaton from Applications to the Trash; settings stay in `~/Library/Application Support/orcaton` |

Only Orcaton is removed: the program, its shortcuts and the `orcaton` command. Go, Node.js, Wails and
FFmpeg stay (other programs may use them). Add `-Purge` / `--purge` to also delete settings and the audio
cache.

## If something goes wrong
| What you see | What to do |
|---|---|
| `…cannot be loaded because running scripts is disabled` | paste the whole command, including `powershell -ExecutionPolicy Bypass -File` at the beginning |
| `winget` is not recognized (Windows 10) | Microsoft Store → "App Installer" → Update, then repeat |
| `не найдено: go / node` ("not found") or `… not recognized` after installation | close the terminal, open a **new** one in the project folder and repeat the command (PATH gets refreshed) |
| the window opens white / empty (Windows 10) | the WebView2 engine is missing: `winget install Microsoft.EdgeWebView2Runtime`, then the shortcut again |
| the window shows **FFmpeg is not installed** | press **Install FFmpeg** in that window (or run the shown command in a terminal), then **Check again** |
| Windows: "Windows protected your PC" when starting the installer | **More info** → **Run anyway** (Orcaton is not signed with a paid certificate) |
| macOS: "Apple could not verify…", "cannot be opened because the developer cannot be verified" | System Settings → Privacy & Security → **Open Anyway** (macOS 12–14: right-click Orcaton → Open → Open) — see "macOS: ready-made package" |
| macOS: "Orcaton is damaged and can't be opened" | download the `.dmg` again and check `shasum -a 256`; if the value matches and the message repeats — Terminal: `xattr -dr com.apple.quarantine /Applications/Orcaton.app` |
| Ubuntu: `libwebkit2gtk-4.1` not found | Ubuntu older than 22.04 is not supported: 22.04 or newer is needed |
| anything else | Developer tools → **Copy log** in the Orcaton window and the text from the terminal — to the developer |

## Without the window: terminal
The same features are available as the `orcaton` command (installed by the same installation) — for
scripts, analysis and batch work: `orcaton --help`; all options — [docs/go/usage.md](go/usage.md).
