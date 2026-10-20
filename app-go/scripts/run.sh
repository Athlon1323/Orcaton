#!/usr/bin/env bash
set -euo pipefail
APPGO="$(cd "$(dirname "$0")/.." && pwd)"
EXE="$APPGO/bin/orcaton"
export PATH="$HOME/.local/go/bin:$PATH"
if [ ! -x "$EXE" ] || [ -n "$(find "$APPGO/cmd" "$APPGO/internal" "$APPGO/go.mod" -newer "$EXE" \( -name '*.go' -o -name go.mod \) -print -quit 2>/dev/null)" ]; then
    command -v go >/dev/null 2>&1 || { echo "Нет Go для сборки: bash app-go/scripts/install.sh (или https://go.dev/dl/)" >&2; exit 3; }
    bash "$APPGO/scripts/build.sh" >/dev/null
fi
exec "$EXE" "$@"
