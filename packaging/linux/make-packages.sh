#!/usr/bin/env bash
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
root="$(cd "$here/../.." && pwd)"
NFPM_VERSION=2.46.3
NFPM_SHA_x86_64=9170d25ea056d7329def38134b2bcad98d02d221f8331610dcd619e84d28c565
NFPM_SHA_arm64=7871d72bb1035924eab56031f962ae3f004b1f7695151789ef018bdbc507a2fa
die() { echo "!! $*" >&2; exit 1; }
export GOTOOLCHAIN=go1.25.14
[ "$(id -u)" != 0 ] || die "run without sudo: bash packaging/linux/make-packages.sh — the script asks for the OS package password itself (otherwise tools land in /root and project files become owned by root)"
build=1; setup=1; release=1; kinds=(deb rpm)
while [ $# -gt 0 ]; do
    case "$1" in
        --no-build) build=0 ;;
        --no-setup) setup=0 ;;
        --deb) kinds=(deb) ;;
        --rpm) kinds=(rpm) ;;
        --release) shift; release="${1:-}"; [[ "$release" =~ ^[1-9][0-9]*$ ]] || die "--release: an integer >= 1 is required" ;;
        *) die "unknown option: $1" ;;
    esac
    shift
done
[ "$(uname -s)" = Linux ] || die "make-packages.sh runs on Linux only"
for c in curl tar sha256sum; do command -v "$c" >/dev/null || die "not found: $c"; done
version="$(tr -d ' \r\n' < "$root/VERSION")"
case "$(uname -m)" in
    x86_64|amd64) arch=amd64; narch=x86_64; nsha=$NFPM_SHA_x86_64 ;;
    aarch64|arm64) arch=arm64; narch=arm64; nsha=$NFPM_SHA_arm64 ;;
    *) die "architecture $(uname -m) is not supported" ;;
esac
echo "Orcaton $version-$release, packages: ${kinds[*]} ($arch)"
glibc="$(getconf GNU_LIBC_VERSION 2>/dev/null | awk '{print $2}')"
case "$glibc" in
    2.3[0-5]|2.2*) ;;
    *) echo "Warning: this system's glibc $glibc is newer than 2.35 (Ubuntu 22.04). The glibc the package actually needs is the \"glibc\" line in the check"
       echo "         below: <= 2.35 — the package also fits Ubuntu 22.04 / 24.04, Debian 12; higher — only systems with that glibc or newer" ;;
esac

if [ "$build" = 1 ]; then
    if ! bash "$here/setup-build.sh" --check >/dev/null 2>&1; then
        if [ "$setup" = 1 ]; then bash "$here/setup-build.sh"
        else bash "$here/setup-build.sh" --check || true; die "build tools are missing — run without --no-setup"; fi
    fi
    bash "$root/app-gui/scripts/build-gui.sh"
fi
bin="$root/app-gui/build/bin"
for f in orcaton-gui orcaton; do [ -x "$bin/$f" ] || die "no $bin/$f — run without --no-build"; done

nfpm="${NFPM:-}"
if [ -z "$nfpm" ]; then
    tools="${XDG_CACHE_HOME:-$HOME/.cache}/orcaton-build/nfpm-$NFPM_VERSION"
    nfpm="$tools/nfpm"
    if [ ! -x "$nfpm" ]; then
        url="https://github.com/goreleaser/nfpm/releases/download/v$NFPM_VERSION/nfpm_${NFPM_VERSION}_Linux_${narch}.tar.gz"
        echo "Downloading nfpm $NFPM_VERSION: $url"
        tmp="$(mktemp -d)"
        curl -fsSL -o "$tmp/nfpm.tar.gz" "$url" || { rm -rf "$tmp"; die "failed to download nfpm (no network?)"; }
        echo "$nsha  $tmp/nfpm.tar.gz" | sha256sum -c --quiet - || { rm -rf "$tmp"; die "nfpm: checksum mismatch — the file is not used"; }
        mkdir -p "$tools"
        tar -xzf "$tmp/nfpm.tar.gz" -C "$tools" nfpm LICENSE.md
        rm -rf "$tmp"
    fi
fi
[ -x "$nfpm" ] || die "nfpm not found: $nfpm"

export ORCATON_VERSION="$version" ORCATON_ARCH="$arch" ORCATON_RELEASE="$release"
cd "$root"
pkgs=()
for k in "${kinds[@]}"; do
    case "$k" in
        deb) pkg="$bin/orcaton_${version}-${release}_${arch}.deb" ;;
        rpm) pkg="$bin/orcaton-${version}-${release}.${narch/arm64/aarch64}.rpm" ;;
    esac
    rm -f "$pkg" "$pkg.sha256"
    "$nfpm" package --config packaging/linux/nfpm.yaml --packager "$k" --target "$bin/" >/dev/null
    [ -f "$pkg" ] || die "nfpm did not create $pkg"
    (cd "$bin" && sha256sum "$(basename "$pkg")" > "$(basename "$pkg").sha256")
    echo "Built: $pkg"
    pkgs+=("$pkg")
done
bash "$here/check-packages.sh" "${pkgs[@]}"
echo
echo "Install: Ubuntu / Debian — sudo apt install ./<file>.deb; Fedora — sudo dnf install ./<file>.rpm"
[ -f "$here/test-package.sh" ] && echo "Install and removal check on a test machine: bash packaging/linux/test-package.sh <file>"
exit 0
