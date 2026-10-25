#!/usr/bin/env bash
set -uo pipefail
root="$(cd "$(dirname "$0")/../.." && pwd)"
version="$(tr -d ' \r\n' < "$root/VERSION")"
MAX_GLIBC=2.35
fails=0
ok()   { echo "  [OK] $*"; }
bad()  { echo "  [!!] $*"; fails=$((fails + 1)); }
note() { echo "  [..] $*"; }
have() { command -v "$1" >/dev/null 2>&1; }
[ $# -gt 0 ] || { echo "укажите файлы пакетов (.deb / .rpm)" >&2; exit 2; }

common=(/usr/bin/orcaton /usr/bin/orcaton-gui /usr/share/applications/orcaton.desktop
    /usr/share/icons/hicolor/48x48/apps/orcaton.png /usr/share/icons/hicolor/64x64/apps/orcaton.png
    /usr/share/icons/hicolor/128x128/apps/orcaton.png /usr/share/icons/hicolor/256x256/apps/orcaton.png
    /usr/share/icons/hicolor/512x512/apps/orcaton.png
    /usr/share/doc/orcaton/THIRD-PARTY-NOTICES.md /usr/share/doc/orcaton/VERSION)
want_deb=("${common[@]}" /usr/share/doc/orcaton/copyright /usr/share/doc/orcaton/LICENSE /usr/share/doc/orcaton/NOTICE)
want_rpm=("${common[@]}" /usr/share/licenses/orcaton/LICENSE /usr/share/licenses/orcaton/NOTICE)

same_list() {
    local got="$1"; shift
    local want; want="$(printf '%s\n' "$@" | LC_ALL=C sort)"
    local extra miss
    extra="$(LC_ALL=C comm -13 <(echo "$want") <(LC_ALL=C sort <<<"$got"))"
    miss="$(LC_ALL=C comm -23 <(echo "$want") <(LC_ALL=C sort <<<"$got"))"
    if [ -z "$extra$miss" ]; then ok "состав: $# файлов, как ожидается"
    else
        [ -n "$miss" ] && bad "нет в пакете: $(echo $miss)"
        [ -n "$extra" ] && bad "лишнее в пакете: $(echo $extra)"
    fi
}

check_tree() {
    local d="$1" march="$2" gui="$1/usr/bin/orcaton-gui" cli="$1/usr/bin/orcaton"
    if have readelf; then
        for x in "$gui" "$cli"; do
            local m; m="$(readelf -h "$x" 2>/dev/null | awk -F: '/Machine:/{gsub(/^ +/,"",$2); print $2}')"
            case "$m" in *"$march"*) ok "$(basename "$x"): ELF, $m" ;; *) bad "$(basename "$x"): архитектура «$m», ждали $march" ;; esac
        done
        if readelf -d "$gui" 2>/dev/null | grep -q 'libwebkit2gtk-4.1.so.0'; then ok "окно связано с WebKitGTK 4.1"
        else bad "окно не связано с libwebkit2gtk-4.1.so.0 (собрано без -tags webkit2_41?)"; fi
        local g; g="$(readelf -V "$gui" 2>/dev/null | grep -o 'GLIBC_[0-9.]*' | sed 's/GLIBC_//' | sort -uV | tail -n 1)"
        if [ -z "$g" ]; then note "glibc: версия не найдена"
        elif [ "$(printf '%s\n%s\n' "$g" "$MAX_GLIBC" | sort -V | tail -n 1)" = "$MAX_GLIBC" ]; then ok "glibc: нужна $g (≤ $MAX_GLIBC — Ubuntu 22.04, Debian 12)"
        else note "glibc: нужна $g > $MAX_GLIBC — пакет только для систем с glibc $g+ (не для Ubuntu 22.04 / 24.04, Debian 12)"; fi
    else note "readelf нет (binutils) — архитектура и glibc не проверены"; fi
    if [ "$march" = "$(host_machine)" ]; then
        local v; v="$(ORCATON_FFMPEG_DIR=/nonexistent "$cli" version 2>/dev/null | head -n 1)"
        [ "$v" = "orcaton $version" ] && ok "orcaton version: $v" || bad "orcaton version: «$v», ждали «orcaton $version»"
    else note "orcaton version не запускался (другая архитектура)"; fi
    if have desktop-file-validate; then
        local out; out="$(desktop-file-validate "$d/usr/share/applications/orcaton.desktop" 2>&1)"
        [ -z "$out" ] && ok "ярлык: desktop-file-validate без замечаний" || bad "ярлык: $out"
    else note "desktop-file-validate нет (desktop-file-utils) — ярлык не проверен"; fi
}
host_machine() { case "$(uname -m)" in x86_64|amd64) echo X86-64 ;; aarch64|arm64) echo AArch64 ;; *) uname -m ;; esac; }

for f in "$@"; do
    echo "Проверка $(basename "$f")"
    [ -f "$f" ] || { bad "файла нет: $f"; continue; }
    if [ -f "$f.sha256" ]; then
        (cd "$(dirname "$f")" && sha256sum -c --quiet "$(basename "$f").sha256" >/dev/null 2>&1) && ok "sha256 совпадает" || bad "sha256 не совпадает с $f.sha256"
    else bad "нет $f.sha256"; fi
    tmp="$(mktemp -d)"
    case "$f" in
        *.deb)
            have dpkg-deb || { note "dpkg-deb нет — .deb не проверен"; rm -rf "$tmp"; continue; }
            pkg="$(dpkg-deb --field "$f" Package)"; ver="$(dpkg-deb --field "$f" Version)"; arch="$(dpkg-deb --field "$f" Architecture)"
            [ "$pkg" = orcaton ] && ok "пакет: $pkg $ver $arch" || bad "имя пакета: $pkg"
            [[ "$ver" == "$version"-* ]] || bad "версия $ver не из VERSION ($version)"
            dep="$(dpkg-deb --field "$f" Depends)"
            for d in 'libgtk-3-0t64 | libgtk-3-0' libwebkit2gtk-4.1-0; do
                [[ ", $dep," == *", $d,"* ]] && ok "зависимость: $d" || bad "нет зависимости «$d» (Depends: $dep)"
            done
            [[ "$dep" == *ffmpeg* ]] && bad "FFmpeg — в Depends (должен быть только в Suggests): $dep" || ok "FFmpeg не обязателен (не в Depends)"
            [[ ", $(dpkg-deb --field "$f" Suggests)," == *", ffmpeg,"* ]] && ok "Suggests: ffmpeg" || bad "нет Suggests: ffmpeg"
            note "сопровождающий: $(dpkg-deb --field "$f" Maintainer)"
            list="$(dpkg-deb -c "$f")"
            same_list "$(awk '$1 !~ /^d/ {print $NF}' <<<"$list" | sed 's#^\./#/#')" "${want_deb[@]}"
            notroot="$(awk '$2 != "root/root" {print $NF}' <<<"$list")"
            [ -z "$notroot" ] && ok "владелец файлов: root" || bad "не root: $(echo $notroot)"
            dpkg-deb -x "$f" "$tmp"
            case "$arch" in amd64) m=X86-64 ;; arm64) m=AArch64 ;; *) m="$arch" ;; esac
            check_tree "$tmp" "$m"
            if have lintian; then
                n="$(lintian --no-tag-display-limit "$f" 2>/dev/null | grep -c '^[EW]:' || true)"
                note "lintian: $n замечаний E/W (информация: lintian $f)"
            fi ;;
        *.rpm)
            if ! have rpm || ! have rpm2cpio || ! have cpio; then note "rpm, rpm2cpio или cpio нет — .rpm не проверен (Ubuntu: sudo apt install rpm cpio)"; rm -rf "$tmp"; continue; fi
            q="$(rpm -qp --qf '%{NAME} %{VERSION} %{RELEASE} %{ARCH}' "$f" 2>/dev/null)"
            read -r pkg ver rel arch <<<"$q"
            [ "$pkg" = orcaton ] && ok "пакет: $q" || bad "имя пакета: $pkg"
            [ "$ver" = "$version" ] || bad "версия $ver не из VERSION ($version)"
            req="$(rpm -qp --requires "$f" 2>/dev/null)"
            for d in 'libgtk-3.so.0()(64bit)' 'libwebkit2gtk-4.1.so.0()(64bit)'; do
                grep -qxF "$d" <<<"$req" && ok "зависимость: $d" || bad "нет зависимости «$d»"
            done
            grep -qi ffmpeg <<<"$req" && bad "FFmpeg — в Requires (должен быть только в Suggests)" || ok "FFmpeg не обязателен (не в Requires)"
            rpm -qp --suggests "$f" 2>/dev/null | grep -qxF /usr/bin/ffmpeg && ok "Suggests: /usr/bin/ffmpeg" || bad "нет Suggests: /usr/bin/ffmpeg"
            same_list "$(rpm -qp --qf '[%{FILEMODES:perms} %{FILENAMES}\n]' "$f" 2>/dev/null | awk '$1 !~ /^d/ {print $2}')" "${want_rpm[@]}"
            notroot="$(rpm -qp --qf '[%{FILEUSERNAME}:%{FILEGROUPNAME} %{FILENAMES}\n]' "$f" 2>/dev/null | awk '$1 != "root:root" {print $2}')"
            [ -z "$notroot" ] && ok "владелец файлов: root" || bad "не root: $(echo $notroot)"
            (cd "$tmp" && rpm2cpio "$(realpath "$f")" | cpio -idm --quiet --no-absolute-filenames 2>/dev/null)
            case "$arch" in x86_64) m=X86-64 ;; aarch64) m=AArch64 ;; *) m="$arch" ;; esac
            check_tree "$tmp" "$m" ;;
        *) bad "неизвестный тип файла: $f" ;;
    esac
    rm -rf "$tmp"
done
if [ "$fails" = 0 ]; then echo "Проверка пакетов: ошибок нет"; else echo "Проверка пакетов: ошибок — $fails"; exit 1; fi
