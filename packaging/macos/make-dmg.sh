#!/usr/bin/env bash
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
root="$(cd "$here/../.." && pwd)"
build=1
for a in "$@"; do
    case "$a" in
        --no-build) build=0 ;;
        *) echo "unknown option: $a" >&2; exit 2 ;;
    esac
done
die() { echo "!! $*" >&2; exit 1; }
[ "$(uname -s)" = Darwin ] || die "make-dmg.sh runs on macOS only (hdiutil, codesign)"
for c in hdiutil codesign lipo ditto shasum osascript; do command -v "$c" >/dev/null || die "not found: $c"; done
version="$(tr -d ' \r\n' < "$root/VERSION")"

[ "$build" = 1 ] && bash "$root/app-gui/scripts/build-gui.sh"
app="$root/app-gui/build/bin/Orcaton.app"
[ -d "$app" ] || die "no $app — run without --no-build"
codesign --verify --deep --strict "$app" || die "the signature of $app is broken — rebuild: bash app-gui/scripts/build-gui.sh"
gui="$(lipo -archs "$app/Contents/MacOS/orcaton-gui")"; cli="$(lipo -archs "$app/Contents/MacOS/orcaton")"
case "$gui" in
    *x86_64*arm64*|*arm64*x86_64*) kind=universal ;;
    arm64) kind=arm64 ;;
    *) kind=x86_64 ;;
esac
case "$cli" in *x86_64*arm64*|*arm64*x86_64*) [ "$kind" = universal ] || die "window — $gui, command — $cli" ;; esac
[ "$kind" != universal ] && echo "Warning: the .app is not universal ($gui) — build without --native"
vol="Orcaton $version"
out="$root/app-gui/build/bin/Orcaton-$version-macos-$kind.dmg"

tmp="$(mktemp -d)"; mnt=""
cleanup() { if [ -n "$mnt" ]; then hdiutil detach "$mnt" -quiet -force 2>/dev/null || true; fi; rm -rf "$tmp"; }
trap cleanup EXIT
stage="$tmp/stage"; mkdir -p "$stage/.background"
ditto "$app" "$stage/Orcaton.app"
ln -s /Applications "$stage/Applications"
cp "$here/dmg-background.png" "$stage/.background/background.png"
cat > "$stage/Read Me.txt" <<TXT
Orcaton $version for macOS 12 or newer (Apple Silicon and Intel)

1. Install: drag Orcaton onto the Applications folder.

2. First launch: Orcaton is free and is not signed with a paid Apple certificate, so macOS
   stops the first launch.
   - Open Orcaton from Applications. macOS says it could not verify the app: press Done (or OK).
   - Open System Settings > Privacy & Security, scroll down to "Orcaton was blocked..."
     and press Open Anyway, enter your password, then Open.
   - macOS 12-14: it is enough to right-click Orcaton > Open > Open.
   This is needed once, and again after installing a new version.

3. FFmpeg (reads audio): if the window says FFmpeg is not installed, press Install FFmpeg
   (it uses Homebrew, https://brew.sh).

4. Terminal command (optional): About > Command line > Install command-line tool.
   Then "orcaton --help" works in a new Terminal window.

Update: drag the new Orcaton onto Applications > Replace.
Uninstall: About > Remove command-line tool (if installed), then move Orcaton from
Applications to the Trash. Settings stay in ~/Library/Application Support/orcaton.

Check the download: shasum -a 256 <file>.dmg must match the .sha256 file.
License: Apache License 2.0, (c) 2026 Athlon1323. FFmpeg is a separate program, not included.
TXT

rw="$tmp/rw.dmg"
hdiutil create -quiet -volname "$vol" -srcfolder "$stage" -fs HFS+ -format UDRW -ov "$rw"
[ -d "/Volumes/$vol" ] && die "disk \"$vol\" is already mounted — eject it and try again"
mnt="$(hdiutil attach "$rw" -readwrite -noverify -noautoopen | awk -F'\t' '/\/Volumes\//{print $NF; exit}')"
[ -n "$mnt" ] || die "failed to mount the image"
if ! osascript <<OSA
tell application "Finder"
    tell disk "$vol"
        open
        set current view of container window to icon view
        set toolbar visible of container window to false
        set statusbar visible of container window to false
        set the bounds of container window to {200, 120, 860, 520}
        set opts to the icon view options of container window
        set arrangement of opts to not arranged
        set icon size of opts to 112
        set text size of opts to 13
        set background picture of opts to file ".background:background.png"
        set position of item "Orcaton.app" of container window to {170, 170}
        set position of item "Applications" of container window to {490, 170}
        set position of item "Read Me.txt" of container window to {330, 310}
        close
        open
        update without registering applications
        delay 2
        close
    end tell
end tell
OSA
then
    echo "Warning: Finder did not lay out the window (no Terminal → Finder permission?) — image without background"
fi
chmod -Rf go-w "$mnt" || true
sync
hdiutil detach "$mnt" -quiet || hdiutil detach "$mnt" -force -quiet
mnt=""
rm -f "$out"
hdiutil convert "$rw" -quiet -format UDZO -imagekey zlib-level=9 -o "$out"
hdiutil verify -quiet "$out"
(cd "$(dirname "$out")" && shasum -a 256 "$(basename "$out")" > "$(basename "$out").sha256")
echo "Done: $out"
echo "SHA-256: $(cut -d' ' -f1 < "$out.sha256")"
