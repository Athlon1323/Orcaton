$ErrorActionPreference = "Stop"
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$AppGo = Split-Path -Parent $PSScriptRoot
$Exe = Join-Path $AppGo "bin\orcaton.exe"

$need = -not (Test-Path $Exe)
if (-not $need) {
    $built = (Get-Item $Exe).LastWriteTimeUtc
    $newer = Get-ChildItem -Path $AppGo -Recurse -File -Include *.go, go.mod |
             Where-Object { $_.LastWriteTimeUtc -gt $built } | Select-Object -First 1
    $need = [bool]$newer
}
if ($need) {
    if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
        Write-Host "Нет Go для сборки: app-go\scripts\install.ps1 (или https://go.dev/dl/)" -ForegroundColor Red
        exit 3
    }
    & (Join-Path $PSScriptRoot "build.ps1") | Out-Null
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}
& $Exe @args
exit $LASTEXITCODE
