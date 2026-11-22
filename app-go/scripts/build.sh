#!/usr/bin/env bash
set -euo pipefail
APPGO="$(cd "$(dirname "$0")/.." && pwd)"
export PATH="$HOME/.local/go/bin:$PATH"
export GOTOOLCHAIN=local
export GOFLAGS=-buildvcs=false
export CGO_ENABLED=0
cd "$APPGO"
mkdir -p bin
go build -trimpath -ldflags "-s -w" -o bin/orcaton ./cmd/orcaton
echo "Built: $APPGO/bin/orcaton"
inst="$HOME/.local/bin/orcaton"
if [ -f "$inst" ]; then
    if cp -f bin/orcaton "$inst.new" 2>/dev/null && mv -f "$inst.new" "$inst"; then
        echo "Updated the orcaton command: $inst"
    else
        rm -f "$inst.new" 2>/dev/null || true; echo "orcaton not updated: $inst"
    fi
fi
if [ "${1:-}" = "--all" ]; then
    mkdir -p dist
    for t in windows/amd64 windows/arm64 darwin/arm64 darwin/amd64 linux/amd64 linux/arm64; do
        os="${t%/*}"; arch="${t#*/}"; ext=""; [ "$os" = windows ] && ext=".exe"
        GOOS="$os" GOARCH="$arch" go build -trimpath -ldflags "-s -w" -o "dist/orcaton-$os-$arch$ext" ./cmd/orcaton
        echo "  dist/orcaton-$os-$arch$ext"
    done
fi
