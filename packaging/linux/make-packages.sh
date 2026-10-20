#!/usr/bin/env bash
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
root="$(cd "$here/../.." && pwd)"
NFPM_VERSION=v2.46.3
build=1; kinds=(deb rpm)
for a in "$@"; do
    case "$a" in
        --no-build) build=0 ;;
        --deb) kinds=(deb) ;;
        --rpm) kinds=(rpm) ;;
        *) echo "неизвестный ключ: $a" >&2; exit 2 ;;
    esac
done
die() { echo "!! $*" >&2; exit 1; }
[ "$(uname -s)" = Linux ] || die "make-packages.sh — только на Linux"
export PATH="$HOME/.local/go/bin:$HOME/.local/node/bin:$HOME/.local/bin:$PATH"
command -v go >/dev/null || die "не найдено: go — поставьте инструменты: bash scripts/install.sh"
version="$(tr -d ' \r\n' < "$root/VERSION")"
case "$(uname -m)" in
    x86_64|amd64) arch=amd64 ;;
    aarch64|arm64) arch=arm64 ;;
    *) die "архитектура $(uname -m) не поддерживается" ;;
esac
glibc="$(getconf GNU_LIBC_VERSION 2>/dev/null | awk '{print $2}')"
case "$glibc" in
    2.3[0-5]|2.2*) ;;
    *) echo "Внимание: glibc $glibc новее 2.35 — пакет не запустится на Ubuntu 22.04 / Debian 12; собирайте на Ubuntu 22.04" ;;
esac

[ "$build" = 1 ] && bash "$root/app-gui/scripts/build-gui.sh"
bin="$root/app-gui/build/bin"
for f in orcaton-gui orcaton; do [ -x "$bin/$f" ] || die "нет $bin/$f — запустите без --no-build"; done

nfpm="${NFPM:-}"
if [ -z "$nfpm" ]; then
    tools="${XDG_CACHE_HOME:-$HOME/.cache}/orcaton-build/nfpm-$NFPM_VERSION"
    nfpm="$tools/nfpm"
    if [ ! -x "$nfpm" ]; then
        echo "Ставлю nfpm $NFPM_VERSION в $tools"
        mkdir -p "$tools"
        GOBIN="$tools" GOTOOLCHAIN=auto go install "github.com/goreleaser/nfpm/v2/cmd/nfpm@$NFPM_VERSION"
    fi
fi
[ -x "$nfpm" ] || die "nfpm не найден: $nfpm"

export ORCATON_VERSION="$version" ORCATON_ARCH="$arch"
cd "$root"
for k in "${kinds[@]}"; do
    "$nfpm" package --config packaging/linux/nfpm.yaml --packager "$k" --target "$bin/"
    case "$k" in
        deb) pkg="$bin/orcaton_${version}-1_${arch}.deb" ;;
        rpm) [ "$arch" = amd64 ] && r=x86_64 || r=aarch64; pkg="$bin/orcaton-${version}-1.${r}.rpm" ;;
    esac
    [ -f "$pkg" ] || die "nfpm не создал $pkg"
    (cd "$bin" && sha256sum "$(basename "$pkg")" > "$(basename "$pkg").sha256")
    echo "Готово: $pkg"
    case "$k" in
        deb) if command -v dpkg-deb >/dev/null; then dpkg-deb --field "$pkg" Package Version Architecture Depends; fi ;;
        rpm) if command -v rpm >/dev/null; then rpm -qp --requires "$pkg" 2>/dev/null | grep -v '^rpmlib' || true; fi ;;
    esac
done
echo "Установка: Ubuntu / Debian — sudo apt install ./<файл>.deb; Fedora — sudo dnf install ./<файл>.rpm"
