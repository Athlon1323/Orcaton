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
[ $# -gt 0 ] || { echo "specify package files (.deb / .rpm)" >&2; exit 2; }

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
    if [ -z "$extra$miss" ]; then ok "contents: $# files, as expected"
    else
        [ -n "$miss" ] && bad "missing from package: $(echo $miss)"
        [ -n "$extra" ] && bad "extra in package: $(echo $extra)"
    fi
}

check_tree() {
    local d="$1" march="$2" gui="$1/usr/bin/orcaton-gui" cli="$1/usr/bin/orcaton"
    if have readelf; then
        for x in "$gui" "$cli"; do
            local m; m="$(readelf -h "$x" 2>/dev/null | awk -F: '/Machine:/{gsub(/^ +/,"",$2); print $2}')"
            case "$m" in *"$march"*) ok "$(basename "$x"): ELF, $m" ;; *) bad "$(basename "$x"): architecture \"$m\", expected $march" ;; esac
        done
        if readelf -d "$gui" 2>/dev/null | grep -q 'libwebkit2gtk-4.1.so.0'; then ok "window links WebKitGTK 4.1"
        else bad "window does not link libwebkit2gtk-4.1.so.0 (built without -tags webkit2_41?)"; fi
        local g; g="$(readelf -V "$gui" 2>/dev/null | grep -o 'GLIBC_[0-9.]*' | sed 's/GLIBC_//' | sort -uV | tail -n 1)"
        if [ -z "$g" ]; then note "glibc: version not found"
        elif [ "$(printf '%s\n%s\n' "$g" "$MAX_GLIBC" | sort -V | tail -n 1)" = "$MAX_GLIBC" ]; then ok "glibc: needs $g (<= $MAX_GLIBC — Ubuntu 22.04, Debian 12)"
        else note "glibc: needs $g > $MAX_GLIBC — the package only runs on systems with glibc $g+ (not Ubuntu 22.04 / 24.04, Debian 12)"; fi
    else note "no readelf (binutils) — architecture and glibc not checked"; fi
    if [ "$march" = "$(host_machine)" ]; then
        local v; v="$(ORCATON_FFMPEG_DIR=/nonexistent "$cli" version 2>/dev/null | head -n 1)"
        [ "$v" = "orcaton $version" ] && ok "orcaton version: $v" || bad "orcaton version: \"$v\", expected \"orcaton $version\""
    else note "orcaton version not run (different architecture)"; fi
    if have desktop-file-validate; then
        local out; out="$(desktop-file-validate "$d/usr/share/applications/orcaton.desktop" 2>&1)"
        [ -z "$out" ] && ok "shortcut: desktop-file-validate has no findings" || bad "shortcut: $out"
    else note "no desktop-file-validate (desktop-file-utils) — shortcut not checked"; fi
}
host_machine() { case "$(uname -m)" in x86_64|amd64) echo X86-64 ;; aarch64|arm64) echo AArch64 ;; *) uname -m ;; esac; }

for f in "$@"; do
    echo "Checking $(basename "$f")"
    [ -f "$f" ] || { bad "file not found: $f"; continue; }
    if [ -f "$f.sha256" ]; then
        (cd "$(dirname "$f")" && sha256sum -c --quiet "$(basename "$f").sha256" >/dev/null 2>&1) && ok "sha256 matches" || bad "sha256 does not match $f.sha256"
    else bad "no $f.sha256"; fi
    tmp="$(mktemp -d)"
    case "$f" in
        *.deb)
            have dpkg-deb || { note "no dpkg-deb — .deb not checked"; rm -rf "$tmp"; continue; }
            pkg="$(dpkg-deb --field "$f" Package)"; ver="$(dpkg-deb --field "$f" Version)"; arch="$(dpkg-deb --field "$f" Architecture)"
            [ "$pkg" = orcaton ] && ok "package: $pkg $ver $arch" || bad "package name: $pkg"
            [[ "$ver" == "$version"-* ]] || bad "version $ver is not from VERSION ($version)"
            dep="$(dpkg-deb --field "$f" Depends)"
            for d in 'libgtk-3-0t64 | libgtk-3-0' libwebkit2gtk-4.1-0; do
                [[ ", $dep," == *", $d,"* ]] && ok "dependency: $d" || bad "missing dependency \"$d\" (Depends: $dep)"
            done
            [[ "$dep" == *ffmpeg* ]] && bad "FFmpeg is in Depends (must be in Suggests only): $dep" || ok "FFmpeg is optional (not in Depends)"
            [[ ", $(dpkg-deb --field "$f" Suggests)," == *", ffmpeg,"* ]] && ok "Suggests: ffmpeg" || bad "missing Suggests: ffmpeg"
            note "maintainer: $(dpkg-deb --field "$f" Maintainer)"
            list="$(dpkg-deb -c "$f")"
            same_list "$(awk '$1 !~ /^d/ {print $NF}' <<<"$list" | sed 's#^\./#/#')" "${want_deb[@]}"
            notroot="$(awk '$2 != "root/root" {print $NF}' <<<"$list")"
            [ -z "$notroot" ] && ok "file owner: root" || bad "not root: $(echo $notroot)"
            dpkg-deb -x "$f" "$tmp"
            case "$arch" in amd64) m=X86-64 ;; arm64) m=AArch64 ;; *) m="$arch" ;; esac
            check_tree "$tmp" "$m"
            if have lintian; then
                n="$(lintian --no-tag-display-limit "$f" 2>/dev/null | grep -c '^[EW]:' || true)"
                note "lintian: $n E/W findings (details: lintian $f)"
            fi ;;
        *.rpm)
            if ! have rpm || ! have rpm2cpio || ! have cpio; then note "rpm, rpm2cpio or cpio missing — .rpm not checked (Ubuntu: sudo apt install rpm cpio)"; rm -rf "$tmp"; continue; fi
            q="$(rpm -qp --qf '%{NAME} %{VERSION} %{RELEASE} %{ARCH}' "$f" 2>/dev/null)"
            read -r pkg ver rel arch <<<"$q"
            [ "$pkg" = orcaton ] && ok "package: $q" || bad "package name: $pkg"
            [ "$ver" = "$version" ] || bad "version $ver is not from VERSION ($version)"
            req="$(rpm -qp --requires "$f" 2>/dev/null)"
            for d in 'libgtk-3.so.0()(64bit)' 'libwebkit2gtk-4.1.so.0()(64bit)'; do
                grep -qxF "$d" <<<"$req" && ok "dependency: $d" || bad "missing dependency \"$d\""
            done
            grep -qi ffmpeg <<<"$req" && bad "FFmpeg is in Requires (must be in Suggests only)" || ok "FFmpeg is optional (not in Requires)"
            rpm -qp --suggests "$f" 2>/dev/null | grep -qxF /usr/bin/ffmpeg && ok "Suggests: /usr/bin/ffmpeg" || bad "missing Suggests: /usr/bin/ffmpeg"
            same_list "$(rpm -qp --qf '[%{FILEMODES:perms} %{FILENAMES}\n]' "$f" 2>/dev/null | awk '$1 !~ /^d/ {print $2}')" "${want_rpm[@]}"
            notroot="$(rpm -qp --qf '[%{FILEUSERNAME}:%{FILEGROUPNAME} %{FILENAMES}\n]' "$f" 2>/dev/null | awk '$1 != "root:root" {print $2}')"
            [ -z "$notroot" ] && ok "file owner: root" || bad "not root: $(echo $notroot)"
            (cd "$tmp" && rpm2cpio "$(realpath "$f")" | cpio -idm --quiet --no-absolute-filenames 2>/dev/null)
            case "$arch" in x86_64) m=X86-64 ;; aarch64) m=AArch64 ;; *) m="$arch" ;; esac
            check_tree "$tmp" "$m" ;;
        *) bad "unknown file type: $f" ;;
    esac
    rm -rf "$tmp"
done
if [ "$fails" = 0 ]; then echo "Package check: no errors"; else echo "Package check: $fails error(s)"; exit 1; fi
