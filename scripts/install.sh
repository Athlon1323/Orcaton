#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
version="$(tr -d ' \r\n' < "$root/VERSION")"
OS="$(uname -s)"
RUN=0; FF=1; ADDPATH=1
for a in "$@"; do
    case "$a" in
        --run) RUN=1 ;; --no-ffmpeg) FF=0 ;; --no-path) ADDPATH=0 ;;
        *) echo "unknown option: $a" >&2; exit 2 ;;
    esac
done
case "$OS" in Darwin|Linux) ;; *) echo "This OS is not supported (Windows: scripts\\install.ps1)" >&2; exit 1 ;; esac
ORIG_PATH="$PATH"
export PATH="$HOME/.local/go/bin:$HOME/.local/node/bin:$HOME/.local/bin:$PATH"
export GOTOOLCHAIN=auto
MARK="# added by Orcaton installer"
have() { command -v "$1" >/dev/null 2>&1; }
say() { printf '\n== %s ==\n' "$*"; }
die() { echo "!! $*" >&2; exit 1; }
narch() { case "$(uname -m)" in x86_64|amd64) echo "$1" ;; arm64|aarch64) echo "$2" ;; *) die "architecture $(uname -m) is not supported" ;; esac; }
rc() { if [ "$OS" = Darwin ]; then echo "$HOME/.zprofile"; else echo "$HOME/.profile"; fi; }
addrc() { local f; f="$(rc)"; touch "$f"; grep -qsF "$1" "$f" || printf '%s %s\n' "$1" "$MARK" >> "$f"; }
go_ok() { have go && [ "$(go env GOVERSION 2>/dev/null | sed -E 's/^go1\.([0-9]+).*/\1/')" -ge 22 ] 2>/dev/null; }
node_ok() {
    have node || return 1
    local v maj min; v="$(node -v | sed 's/^v//')"; maj="${v%%.*}"; min="$(echo "$v" | cut -d. -f2)"
    [ "$maj" -ge 23 ] 2>/dev/null || { [ "$maj" -eq 22 ] && [ "$min" -ge 12 ]; } || { [ "$maj" -eq 20 ] && [ "$min" -ge 19 ]; }
}
tarball() {
    local url="$1" name="$2" zflag="$3" tmp; tmp="$(mktemp -d)"
    echo "Downloading $url"
    curl -fsSL "$url" | tar -x "$zflag" -C "$tmp" -f - || die "failed to download $url (no network?)"
    rm -rf "${HOME:?}/.local/$name"; mkdir -p "$HOME/.local"; mv "$tmp"/* "$HOME/.local/$name"; rm -rf "$tmp"
}
need_brew() { have brew || for b in /opt/homebrew/bin/brew /usr/local/bin/brew; do [ -x "$b" ] && eval "$("$b" shellenv)" && return; done; have brew || die "Homebrew is required: https://brew.sh — install it and try again"; }
pkg() {
    if   have apt-get; then sudo apt-get update && sudo apt-get install -y "$@"
    elif have dnf;     then sudo dnf install -y "$@"
    elif have pacman;  then sudo pacman -S --needed --noconfirm "$@"
    elif have zypper;  then sudo zypper --non-interactive install "$@"
    else die "no package manager found: install manually: $*"; fi
}

echo "Orcaton $version — installing for user $USER ($OS)"
echo "Missing tools are installed by the OS package manager and from official sites: Go (BSD), Node.js (MIT),"
echo "Wails CLI (MIT), FFmpeg (LGPL/GPL — your distribution's or Homebrew build). Each has its own license;"
echo "by running the installer you accept them. Press Ctrl+C to cancel."

say "1/6 OS tools"
if [ "$OS" = Darwin ]; then
    xcode-select -p >/dev/null 2>&1 || { xcode-select --install || true; die "Xcode Command Line Tools are being installed — wait for it to finish and run the command again"; }
    need_brew; echo "Xcode Command Line Tools and Homebrew: found"
else
    if pkg-config --exists webkit2gtk-4.1 gtk+-3.0 2>/dev/null && have gcc && have curl; then echo "Window libraries: found"
    elif have apt-get; then pkg build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.1-dev curl xz-utils
    elif have dnf;     then pkg gcc gcc-c++ make pkgconf-pkg-config gtk3-devel webkit2gtk4.1-devel curl xz
    elif have pacman;  then pkg base-devel pkgconf gtk3 webkit2gtk-4.1 curl xz
    else die "install manually: a C compiler, pkg-config, GTK 3 (dev), WebKitGTK 4.1 (dev) — then try again"; fi
fi

say "2/6 Go (compiler)"
if go_ok; then echo "Go: $(go env GOVERSION)"
elif [ "$OS" = Darwin ]; then brew install go || brew upgrade go
else
    gv="$(curl -fsSL 'https://go.dev/VERSION?m=text' | head -n 1)"; [ -n "$gv" ] || die "could not determine the Go version (no network?)"
    tarball "https://go.dev/dl/$gv.linux-$(narch amd64 arm64).tar.gz" go -z
fi
go_ok || die "Go not found after installation"
[ -x "$HOME/.local/go/bin/go" ] && addrc 'export PATH="$HOME/.local/go/bin:$PATH"'

say "3/6 Node.js (UI build) and Wails CLI v2.13.0"
if node_ok; then echo "Node.js $(node -v)"
elif [ "$OS" = Darwin ]; then brew install node || brew upgrade node
else
    base="https://nodejs.org/dist/latest-v22.x"
    file="$(curl -fsSL "$base/SHASUMS256.txt" | awk '{print $2}' | grep "linux-$(narch x64 arm64).tar.xz$" | head -n 1)"
    [ -n "$file" ] || die "could not determine the Node.js version (no network?)"
    tarball "$base/$file" node -J
fi
node_ok || die "Node.js 20.19+ / 22.12+ not found after installation"
[ -x "$HOME/.local/node/bin/node" ] && addrc 'export PATH="$HOME/.local/node/bin:$PATH"'
export PATH="$PATH:$(go env GOPATH)/bin"
if have wails; then echo "wails: found"; else go install github.com/wailsapp/wails/v2/cmd/wails@v2.13.0; fi

say "4/6 FFmpeg (audio decoding)"
if [ "$FF" = 0 ]; then echo "Skipped (--no-ffmpeg): the window offers to install FFmpeg on first launch"
elif have ffmpeg && have ffprobe; then echo "FFmpeg: found"
elif [ "$OS" = Darwin ]; then brew install ffmpeg
elif have dnf && ! have apt-get; then pkg /usr/bin/ffmpeg
else pkg ffmpeg; fi

say "5/6 Build"
bash "$root/app-gui/scripts/build-gui.sh" --native
bin="$root/app-gui/build/bin"

say "6/6 Install"
made=()
mkdir -p "$HOME/.local/bin"
if [ "$OS" = Darwin ]; then
    dest="$HOME/Applications/Orcaton.app"; mkdir -p "$HOME/Applications"
    rm -rf "$dest"; cp -R "$bin/Orcaton.app" "$dest"
    cli="$dest/Contents/MacOS/orcaton"; res="$dest/Contents/Resources"
    gui="$dest"
else
    dest="$HOME/.local/share/orcaton"; mkdir -p "$dest"
    cp -f "$bin"/orcaton-gui "$bin"/orcaton "$dest/"
    for f in LICENSE NOTICE THIRD-PARTY-NOTICES.md VERSION; do cp -f "$bin/$f" "$dest/"; done
    [ -d "$bin/ffmpeg-licenses" ] && cp -R "$bin/ffmpeg-licenses" "$dest/"
    [ -x "$bin/ffmpeg" ] && cp -f "$bin/ffmpeg" "$bin/ffprobe" "$dest/"
    cp -f "$root/app-gui/build/appicon.png" "$dest/orcaton.png"
    cli="$dest/orcaton"; res="$dest"; gui="$dest/orcaton-gui"
    apps="$HOME/.local/share/applications"; mkdir -p "$apps"
    cat > "$apps/orcaton.desktop" <<DESK
[Desktop Entry]
Type=Application
Name=Orcaton
Comment=Multicam audio sync (FCP7 XML, DRT)
Exec="$dest/orcaton-gui"
Icon=$dest/orcaton.png
Terminal=false
Categories=AudioVideo;Video;
StartupWMClass=orcaton
DESK
    chmod +x "$apps/orcaton.desktop"
    have update-desktop-database && update-desktop-database "$apps" >/dev/null 2>&1 || true
    made+=("desktop=$apps/orcaton.desktop")
fi
cp -f "$root/scripts/uninstall.sh" "$res/uninstall.sh"
if [ "$ADDPATH" = 1 ]; then
    ln -sfn "$cli" "$HOME/.local/bin/orcaton"; made+=("link=$HOME/.local/bin/orcaton")
    case ":$ORIG_PATH:" in *":$HOME/.local/bin:"*) ;; *) addrc 'export PATH="$HOME/.local/bin:$PATH"' ;; esac
fi
rm -f "$HOME/.local/bin/orcaton-go"
made+=("rc=$(rc)")
printf '%s\n' "${made[@]}" > "$res/install-manifest.txt"

echo
echo "Done: Orcaton $version is installed."
if [ "$OS" = Darwin ]; then echo "  Window: Finder → ~/Applications → Orcaton (or Spotlight / Launchpad: Orcaton)"
else echo "  Window: applications menu → Orcaton"; fi
[ "$ADDPATH" = 1 ] && echo "  Command: orcaton --help (in a new terminal)"
echo "  Uninstall: bash scripts/uninstall.sh"
if [ "$RUN" = 1 ]; then
    if [ "$OS" = Darwin ]; then open "$gui"; else nohup "$gui" >/dev/null 2>&1 & fi
fi
