#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"
if [ $# -ge 1 ]; then printf '%s\n' "$1" > VERSION; fi
v="$(tr -d ' \r\n' < VERSION)"
[[ "$v" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$ ]] || { echo "VERSION: does not look like a version: '$v'" >&2; exit 2; }
sed -i.bak -E "s/^const Version = \"[^\"]*\"/const Version = \"$v\"/" app-go/internal/app/report.go
sed -i.bak -E "s/(\"productVersion\": )\"[^\"]*\"/\1\"$v\"/" app-gui/wails.json
sed -i.bak -E "1,6s/^(  \"version\"): \"[^\"]*\"/\1: \"$v\"/" app-gui/frontend/package.json
sed -i.bak -E "1,12s/^(  \"version\"|      \"version\"): \"[^\"]*\"/\1: \"$v\"/" app-gui/frontend/package-lock.json
rm -f app-go/internal/app/report.go.bak app-gui/wails.json.bak app-gui/frontend/package.json.bak app-gui/frontend/package-lock.json.bak
echo "Version $v: VERSION, app.Version, wails.json, package.json, package-lock.json"
echo "Window fixtures: cd app-gui/session && go test -run Fixtures -update"
