#!/usr/bin/env bash
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
root="$(cd "$here/../.." && pwd)"
NFPM_VERSION=2.46.3
NFPM_SHA_x86_64=9170d25ea056d7329def38134b2bcad98d02d221f8331610dcd619e84d28c565
NFPM_SHA_arm64=7871d72bb1035924eab56031f962ae3f004b1f7695151789ef018bdbc507a2fa
die() { echo "!! $*" >&2; exit 1; }
export GOTOOLCHAIN=go1.25.14
[ "$(id -u)" != 0 ] || die "запускайте без sudo: bash packaging/linux/make-packages.sh — пароль для пакетов ОС скрипт спросит сам (иначе инструменты встанут в /root, а файлы проекта станут принадлежать root)"
build=1; setup=1; release=1; kinds=(deb rpm)
while [ $# -gt 0 ]; do
    case "$1" in
        --no-build) build=0 ;;
        --no-setup) setup=0 ;;
        --deb) kinds=(deb) ;;
        --rpm) kinds=(rpm) ;;
        --release) shift; release="${1:-}"; [[ "$release" =~ ^[1-9][0-9]*$ ]] || die "--release: нужно целое число ≥ 1" ;;
        *) die "неизвестный ключ: $1" ;;
    esac
    shift
done
[ "$(uname -s)" = Linux ] || die "make-packages.sh — только на Linux"
for c in curl tar sha256sum; do command -v "$c" >/dev/null || die "не найдено: $c"; done
version="$(tr -d ' \r\n' < "$root/VERSION")"
case "$(uname -m)" in
    x86_64|amd64) arch=amd64; narch=x86_64; nsha=$NFPM_SHA_x86_64 ;;
    aarch64|arm64) arch=arm64; narch=arm64; nsha=$NFPM_SHA_arm64 ;;
    *) die "архитектура $(uname -m) не поддерживается" ;;
esac
echo "Orcaton $version-$release, пакеты: ${kinds[*]} ($arch)"
glibc="$(getconf GNU_LIBC_VERSION 2>/dev/null | awk '{print $2}')"
case "$glibc" in
    2.3[0-5]|2.2*) ;;
    *) echo "Внимание: glibc этой системы $glibc новее 2.35 (Ubuntu 22.04). Какая glibc нужна самому пакету — строка «glibc» в проверке"
       echo "         ниже: ≤ 2.35 — пакет подходит и для Ubuntu 22.04 / 24.04, Debian 12; выше — только для систем с ней и новее" ;;
esac

if [ "$build" = 1 ]; then
    if ! bash "$here/setup-build.sh" --check >/dev/null 2>&1; then
        if [ "$setup" = 1 ]; then bash "$here/setup-build.sh"
        else bash "$here/setup-build.sh" --check || true; die "не хватает инструментов сборки — запустите без --no-setup"; fi
    fi
    bash "$root/app-gui/scripts/build-gui.sh"
fi
bin="$root/app-gui/build/bin"
for f in orcaton-gui orcaton; do [ -x "$bin/$f" ] || die "нет $bin/$f — запустите без --no-build"; done

nfpm="${NFPM:-}"
if [ -z "$nfpm" ]; then
    tools="${XDG_CACHE_HOME:-$HOME/.cache}/orcaton-build/nfpm-$NFPM_VERSION"
    nfpm="$tools/nfpm"
    if [ ! -x "$nfpm" ]; then
        url="https://github.com/goreleaser/nfpm/releases/download/v$NFPM_VERSION/nfpm_${NFPM_VERSION}_Linux_${narch}.tar.gz"
        echo "Скачиваю nfpm $NFPM_VERSION: $url"
        tmp="$(mktemp -d)"
        curl -fsSL -o "$tmp/nfpm.tar.gz" "$url" || { rm -rf "$tmp"; die "не удалось скачать nfpm (нет сети?)"; }
        echo "$nsha  $tmp/nfpm.tar.gz" | sha256sum -c --quiet - || { rm -rf "$tmp"; die "nfpm: контрольная сумма не совпала — файл не используется"; }
        mkdir -p "$tools"
        tar -xzf "$tmp/nfpm.tar.gz" -C "$tools" nfpm LICENSE.md
        rm -rf "$tmp"
    fi
fi
[ -x "$nfpm" ] || die "nfpm не найден: $nfpm"

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
    [ -f "$pkg" ] || die "nfpm не создал $pkg"
    (cd "$bin" && sha256sum "$(basename "$pkg")" > "$(basename "$pkg").sha256")
    echo "Собрано: $pkg"
    pkgs+=("$pkg")
done
bash "$here/check-packages.sh" "${pkgs[@]}"
echo
echo "Установка: Ubuntu / Debian — sudo apt install ./<файл>.deb; Fedora — sudo dnf install ./<файл>.rpm"
[ -f "$here/test-package.sh" ] && echo "Проверка установки и удаления на тестовой машине: bash packaging/linux/test-package.sh <файл>"
exit 0
