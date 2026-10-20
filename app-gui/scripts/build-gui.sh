#!/usr/bin/env bash
set -euo pipefail
here="$(cd "$(dirname "$0")/.." && pwd)"
root="$(dirname "$here")"
install=0; run=0; native=0
for a in "$@"; do
    case "$a" in
        --install) install=1 ;;
        --run) run=1 ;;
        --native) native=1 ;;
        *) echo "неизвестный ключ: $a" >&2; exit 2 ;;
    esac
done
export PATH="$HOME/.local/node/bin:$HOME/.local/go/bin:$HOME/.local/bin:$PATH"
export PATH="$PATH:$(go env GOPATH 2>/dev/null)/bin"
export GOFLAGS=-buildvcs=false GOTOOLCHAIN=auto
for c in go node npm wails; do
    command -v "$c" >/dev/null || { echo "не найдено: $c — поставьте инструменты: bash scripts/install.sh (или app-gui/scripts/install-gui.sh)" >&2; exit 1; }
done
cd "$here"
[ -f go.sum ] || go mod tidy
wargs=()
[ "$(uname -s)" = "Linux" ] && wargs=(-tags webkit2_41)
universal=0
if [ "$(uname -s)" = Darwin ] && [ "$native" = 0 ]; then
    universal=1
    command -v lipo >/dev/null || { echo "не найдено: lipo — нужны Xcode Command Line Tools (xcode-select --install)" >&2; exit 1; }
    wargs+=(-platform darwin/universal)
fi
wails build -skipbindings -trimpath ${wargs[@]+"${wargs[@]}"}

os="$(go env GOOS)"; arch="$(go env GOARCH)"
if [ "$os" = darwin ]; then bin="$here/build/bin/Orcaton.app/Contents/MacOS"; docs="$here/build/bin/Orcaton.app/Contents/Resources"
else bin="$here/build/bin"; docs="$bin"; fi
mkdir -p "$docs"
if [ "$universal" = 1 ]; then
    for a in amd64 arm64; do
        (cd "$root/app-go" && CGO_ENABLED=0 GOOS=darwin GOARCH=$a go build -trimpath -ldflags "-s -w" -o "$bin/orcaton-$a" ./cmd/orcaton)
    done
    lipo -create -output "$bin/orcaton" "$bin/orcaton-amd64" "$bin/orcaton-arm64"
    rm -f "$bin/orcaton-amd64" "$bin/orcaton-arm64"
    arch=universal
else
    (cd "$root/app-go" && CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o "$bin/orcaton" ./cmd/orcaton)
fi
echo "Команда: $bin/orcaton"
for f in LICENSE NOTICE THIRD-PARTY-NOTICES.md VERSION; do cp -f "$root/$f" "$docs/"; done
ff="$root/packaging/ffmpeg/$os-$arch"
if [ -x "$ff/ffmpeg" ] && [ -x "$ff/ffprobe" ]; then
    cp -f "$ff/ffmpeg" "$ff/ffprobe" "$bin/"
    mkdir -p "$docs/ffmpeg-licenses"
    find "$ff" -maxdepth 1 -type f ! -name ffmpeg ! -name ffprobe -exec cp -f {} "$docs/ffmpeg-licenses/" \;
    echo "FFmpeg вложен: $ff"
else
    echo "FFmpeg не вложен (нет $ff/ffmpeg, ffprobe) — будет искаться в системе"
fi

case "$(uname -s)" in
    Darwin)
        app="$here/build/bin/Orcaton.app"
        xattr -cr "$app"
        for x in "$bin/orcaton" "$bin/ffmpeg" "$bin/ffprobe"; do
            if [ -f "$x" ]; then codesign --force --sign - "$x"; fi
        done
        codesign --force --deep --sign - "$app"
        codesign --verify --deep --strict "$app" || { echo "подпись Orcaton.app не прошла проверку" >&2; exit 1; }
        echo "Подпись: ad-hoc ($(lipo -archs "$bin/orcaton" 2>/dev/null || uname -m))"
        echo "Собрано: $app"
        if [ "$install" = 1 ]; then
            mkdir -p "$HOME/Applications"
            rm -rf "$HOME/Applications/Orcaton.app"
            cp -R "$app" "$HOME/Applications/Orcaton.app"
            app="$HOME/Applications/Orcaton.app"
            echo "Установлено: $app (Launchpad / Finder → Программы пользователя)"
        fi
        [ "$run" = 1 ] && open "$app"
        ;;
    *)
        bin="$here/build/bin/orcaton-gui"
        echo "Собрано: $bin"
        if [ "$install" = 1 ]; then
            apps="$HOME/.local/share/applications"
            mkdir -p "$apps"
            cat > "$apps/orcaton.desktop" <<DESK
[Desktop Entry]
Type=Application
Name=Orcaton
Comment=Multicam audio sync (FCP7 XML, DRT)
Exec="$bin"
Icon=$here/build/appicon.png
Terminal=false
Categories=AudioVideo;Video;
StartupWMClass=orcaton
DESK
            chmod +x "$apps/orcaton.desktop"
            command -v update-desktop-database >/dev/null && update-desktop-database "$apps" >/dev/null 2>&1 || true
            echo "Ярлык: $apps/orcaton.desktop (меню приложений → Orcaton)"
        fi
        [ "$run" = 1 ] && { nohup "$bin" >/dev/null 2>&1 & }
        ;;
esac
exit 0
