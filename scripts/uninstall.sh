#!/usr/bin/env bash
set -euo pipefail
OS="$(uname -s)"
PURGE=0; YES=0
for a in "$@"; do
    case "$a" in --purge) PURGE=1 ;; --yes|-y) YES=1 ;; *) echo "unknown option: $a" >&2; exit 2 ;; esac
done
MARK="# added by Orcaton installer"
if [ "$OS" = Darwin ]; then
    dest="$HOME/Applications/Orcaton.app"; res="$dest/Contents/Resources"
    data=("$HOME/Library/Application Support/orcaton" "$HOME/Library/Caches/orcaton")
else
    dest="$HOME/.local/share/orcaton"; res="$dest"
    data=("${XDG_CONFIG_HOME:-$HOME/.config}/orcaton" "${XDG_CACHE_HOME:-$HOME/.cache}/orcaton")
fi
[ -e "$dest" ] || { echo "Orcaton is not installed ($dest)."; exit 0; }
if [ "$YES" = 0 ]; then
    if [ "$PURGE" = 1 ]; then what="the program, shortcut, settings and cache"; else what="the program and shortcut (settings and cache stay)"; fi
    read -r -p "Remove Orcaton — $what? [y/N] " ans
    case "$ans" in y|Y|yes|YES) ;; *) echo "Cancelled."; exit 0 ;; esac
fi
items=()
if [ -f "$res/install-manifest.txt" ]; then
    while IFS= read -r line; do [ -n "$line" ] && items+=("$line"); done < "$res/install-manifest.txt"
else items=("link=$HOME/.local/bin/orcaton" "desktop=$HOME/.local/share/applications/orcaton.desktop"); fi
for it in ${items[@]+"${items[@]}"}; do
    kind="${it%%=*}"; val="${it#*=}"
    case "$kind" in
        link)
            if [ -L "$val" ] && [[ "$(readlink "$val")" == "$dest"* ]]; then rm -f "$val"; echo "Command: $val"; fi ;;
        desktop)
            if [ -f "$val" ] && grep -qs "Name=Orcaton" "$val"; then rm -f "$val"; echo "Shortcut: $val"; fi ;;
        rc)
            if [ -f "$val" ] && grep -qsF "$MARK" "$val"; then
                echo "Profile $val: PATH lines marked \"$MARK\" are kept (shared folders; remove them manually if you like)"
            fi ;;
    esac
done
case "$dest" in */Orcaton.app|*/.local/share/orcaton) rm -rf "$dest"; echo "Removed: $dest" ;; *) echo "!! unexpected folder $dest — not removing" >&2 ;; esac
if [ "$PURGE" = 1 ]; then
    for d in "${data[@]}"; do [ -e "$d" ] && { rm -rf "$d"; echo "Removed: $d"; }; done
else
    echo "Settings and cache are kept: ${data[*]} (to remove: --purge)"
fi
echo
echo "Orcaton removed. Go, Node.js, Wails and FFmpeg are untouched."
