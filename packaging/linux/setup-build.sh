#!/usr/bin/env bash
set -euo pipefail
WAILS=v2.13.0
GO_PIN=go1.25.14
CHECK=0
for a in "$@"; do case "$a" in --check) CHECK=1 ;; *) echo "unknown option: $a" >&2; exit 2 ;; esac; done
export PATH="$HOME/.local/go/bin:$HOME/.local/node/bin:$HOME/.local/bin:$PATH"
export GOTOOLCHAIN="$GO_PIN"
[ "$(id -u)" != 0 ] || { echo "!! run without sudo — the script asks for the OS package password itself" >&2; exit 1; }
have() { command -v "$1" >/dev/null 2>&1; }
die() { echo "!! $*" >&2; exit 1; }
narch() { case "$(uname -m)" in x86_64|amd64) echo "$1" ;; arm64|aarch64) echo "$2" ;; *) die "architecture $(uname -m) is not supported" ;; esac; }
os_ok() { have gcc && have pkg-config && have curl && have xz && pkg-config --exists gtk+-3.0 webkit2gtk-4.1 2>/dev/null; }
go_ok() { have go && [ "$(go env GOVERSION 2>/dev/null)" = "$GO_PIN" ]; }
node_ok() {
    have node || return 1
    local v maj min; v="$(node -v | sed 's/^v//')"; maj="${v%%.*}"; min="$(echo "$v" | cut -d. -f2)"
    [ "$maj" -ge 23 ] 2>/dev/null || { [ "$maj" -eq 22 ] && [ "$min" -ge 12 ]; } || { [ "$maj" -eq 20 ] && [ "$min" -ge 19 ]; }
}
wails_bin() { if have wails; then command -v wails; elif have go && [ -x "$(go env GOPATH)/bin/wails" ]; then echo "$(go env GOPATH)/bin/wails"; fi; }
wails_ok() { local w; w="$(wails_bin)"; [ -n "$w" ] && "$w" version 2>/dev/null | grep -q "$WAILS" && go version "$w" 2>/dev/null | grep -q ": $GO_PIN\$"; }
tarball() {
    local url="$1" name="$2" zflag="$3" tmp; tmp="$(mktemp -d)"
    echo "Downloading $url"
    curl -fsSL "$url" | tar -x "$zflag" -C "$tmp" -f - || { rm -rf "$tmp"; die "failed to download $url (no network?)"; }
    rm -rf "${HOME:?}/.local/$name"; mkdir -p "$HOME/.local"; mv "$tmp"/* "$HOME/.local/$name"; rm -rf "$tmp"
}
row() { if "$2"; then echo "[OK] $1"; else echo "[--] $1"; miss=1; fi; }

if [ "$CHECK" = 1 ]; then
    miss=0
    row "OS packages (gcc, pkg-config, GTK 3, WebKitGTK 4.1 — dev, curl, xz)" os_ok
    row "Go $GO_PIN (via GOTOOLCHAIN) $(go env GOVERSION 2>/dev/null || true)" go_ok
    row "Node.js 20.19+ / 22.12+ $(node -v 2>/dev/null || true)" node_ok
    row "Wails CLI $WAILS" wails_ok
    exit "$miss"
fi

echo "Orcaton build tools: OS packages, Go, Node.js, Wails CLI — each with its own license. Press Ctrl+C to cancel."
echo "== 1/4 OS packages =="
if os_ok; then echo "found"
elif have apt-get; then sudo apt-get update && sudo apt-get install -y build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.1-dev curl xz-utils
elif have dnf; then sudo dnf install -y gcc gcc-c++ make pkgconf-pkg-config gtk3-devel webkit2gtk4.1-devel curl xz
else die "apt or dnf is required; install manually: a C compiler, pkg-config, GTK 3 (dev), WebKitGTK 4.1 (dev), curl, xz"; fi
os_ok || die "OS packages not found after installation (pkg-config --exists gtk+-3.0 webkit2gtk-4.1)"

echo "== 2/4 Go =="
if go_ok; then echo "$(go env GOVERSION)"
else
    if have go; then go version >/dev/null 2>&1 || die "Go $(command -v go) could not switch to $GO_PIN (needs Go 1.21+ and network access)"
    else tarball "https://go.dev/dl/$GO_PIN.linux-$(narch amd64 arm64).tar.gz" go -z; fi
fi
go_ok || die "Go not found after installation"

echo "== 3/4 Node.js =="
if node_ok; then echo "$(node -v)"
else
    base="https://nodejs.org/dist/latest-v22.x"
    file="$(curl -fsSL "$base/SHASUMS256.txt" | awk '{print $2}' | grep "linux-$(narch x64 arm64).tar.xz$" | head -n 1)"
    [ -n "$file" ] || die "could not determine the Node.js version (no network?)"
    tarball "$base/$file" node -J
fi
node_ok || die "Node.js 20.19+ / 22.12+ not found after installation"

echo "== 4/4 Wails CLI $WAILS =="
if wails_ok; then echo "found"; else go install "github.com/wailsapp/wails/v2/cmd/wails@$WAILS"; fi
wails_ok || die "Wails CLI $WAILS not found after installation"
echo "Build tools are ready."
