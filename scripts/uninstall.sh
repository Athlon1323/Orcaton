#!/usr/bin/env bash
set -euo pipefail
OS="$(uname -s)"
PURGE=0; YES=0
for a in "$@"; do
    case "$a" in --purge) PURGE=1 ;; --yes|-y) YES=1 ;; *) echo "неизвестный ключ: $a" >&2; exit 2 ;; esac
done
MARK="# added by Orcaton installer"
if [ "$OS" = Darwin ]; then
    dest="$HOME/Applications/Orcaton.app"; res="$dest/Contents/Resources"
    data=("$HOME/Library/Application Support/orcaton" "$HOME/Library/Caches/orcaton")
else
    dest="$HOME/.local/share/orcaton"; res="$dest"
    data=("${XDG_CONFIG_HOME:-$HOME/.config}/orcaton" "${XDG_CACHE_HOME:-$HOME/.cache}/orcaton")
fi
[ -e "$dest" ] || { echo "Orcaton не установлен ($dest)."; exit 0; }
if [ "$YES" = 0 ]; then
    if [ "$PURGE" = 1 ]; then what="программу, ярлык, настройки и кэш"; else what="программу и ярлык (настройки и кэш останутся)"; fi
    read -r -p "Удалить Orcaton — $what? [y/N] " ans
    case "$ans" in y|Y|yes|д|да) ;; *) echo "Отменено."; exit 0 ;; esac
fi
items=()
if [ -f "$res/install-manifest.txt" ]; then
    while IFS= read -r line; do [ -n "$line" ] && items+=("$line"); done < "$res/install-manifest.txt"
else items=("link=$HOME/.local/bin/orcaton" "desktop=$HOME/.local/share/applications/orcaton.desktop"); fi
for it in ${items[@]+"${items[@]}"}; do
    kind="${it%%=*}"; val="${it#*=}"
    case "$kind" in
        link)
            if [ -L "$val" ] && [[ "$(readlink "$val")" == "$dest"* ]]; then rm -f "$val"; echo "Команда: $val"; fi ;;
        desktop)
            if [ -f "$val" ] && grep -qs "Name=Orcaton" "$val"; then rm -f "$val"; echo "Ярлык: $val"; fi ;;
        rc)
            if [ -f "$val" ] && grep -qsF "$MARK" "$val"; then
                echo "Профиль $val: строки PATH с пометкой «$MARK» оставлены (общие папки; можно удалить вручную)"
            fi ;;
    esac
done
case "$dest" in */Orcaton.app|*/.local/share/orcaton) rm -rf "$dest"; echo "Удалено: $dest" ;; *) echo "!! неожиданная папка $dest — не удаляю" >&2 ;; esac
if [ "$PURGE" = 1 ]; then
    for d in "${data[@]}"; do [ -e "$d" ] && { rm -rf "$d"; echo "Удалено: $d"; }; done
else
    echo "Настройки и кэш остались: ${data[*]} (удалить: --purge)"
fi
echo
echo "Orcaton удалён. Go, Node.js, Wails и FFmpeg не тронуты."
