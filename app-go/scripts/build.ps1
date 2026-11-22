param([switch]$All)
$ErrorActionPreference = "Stop"
$AppGo = Split-Path -Parent $PSScriptRoot
$env:GOTOOLCHAIN = "local"
$env:GOFLAGS = "-buildvcs=false"
$env:CGO_ENABLED = "0"
Push-Location $AppGo
try {
    New-Item -ItemType Directory -Force -Path "bin" | Out-Null
    go build -trimpath -ldflags "-s -w" -o "bin\orcaton.exe" ./cmd/orcaton
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    Write-Host "Built: $AppGo\bin\orcaton.exe"
    $inst = Join-Path $env:LOCALAPPDATA "Programs\orcaton\orcaton.exe"
    if (Test-Path $inst) {
        try {
            Copy-Item -Force "bin\orcaton.exe" $inst -ErrorAction Stop
            Write-Host "Updated the orcaton command: $inst"
        } catch {
            Write-Host "orcaton not updated (is the program running?): $inst" -ForegroundColor Yellow
        }
    }
    if ($All) {
        New-Item -ItemType Directory -Force -Path "dist" | Out-Null
        foreach ($t in @("windows/amd64", "windows/arm64", "darwin/arm64", "darwin/amd64", "linux/amd64", "linux/arm64")) {
            $os, $arch = $t -split "/"
            $env:GOOS = $os; $env:GOARCH = $arch
            $ext = if ($os -eq "windows") { ".exe" } else { "" }
            go build -trimpath -ldflags "-s -w" -o "dist\orcaton-$os-$arch$ext" ./cmd/orcaton
            if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
            Write-Host "  dist\orcaton-$os-$arch$ext"
        }
        Remove-Item Env:GOOS, Env:GOARCH
    }
} finally { Pop-Location }
exit 0
