param([switch]$Run, [switch]$Shortcut)
$ErrorActionPreference = "Stop"
$AppGui = Split-Path -Parent $PSScriptRoot
$Root = Split-Path -Parent $AppGui
foreach ($c in @("go", "node", "npm")) {
    if (-not (Get-Command $c -ErrorAction SilentlyContinue)) {
        Write-Host "Не найдено: $c — см. app-gui\scripts\dev-gui.ps1" -ForegroundColor Red
        exit 1
    }
}
if (-not (Get-Command "wails" -ErrorAction SilentlyContinue)) {
    $gobin = Join-Path (go env GOPATH) "bin"
    if (Test-Path (Join-Path $gobin "wails.exe")) { $env:PATH = "$gobin;$env:PATH" }
    else { Write-Host "Нет Wails CLI: go install github.com/wailsapp/wails/v2/cmd/wails@v2.13.0" -ForegroundColor Red; exit 1 }
}
$env:GOFLAGS = "-buildvcs=false"
$env:GOTOOLCHAIN = "auto"
Push-Location $AppGui
try {
    if (-not (Test-Path "go.sum")) {
        go mod tidy
        if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    }
    wails build -skipbindings -trimpath
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    $exe = Join-Path $AppGui "build\bin\orcaton-gui.exe"
    Write-Host "Собрано: $exe"
    $bin = Split-Path $exe
    $arch = (go env GOARCH)
    Push-Location (Join-Path $Root "app-go")
    try {
        $env:CGO_ENABLED = "0"
        go build -trimpath -ldflags "-s -w" -o (Join-Path $bin "orcaton.exe") ./cmd/orcaton
        if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    } finally { Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue; Pop-Location }
    Write-Host "Команда: $(Join-Path $bin 'orcaton.exe')"
    foreach ($f in @("LICENSE", "NOTICE", "THIRD-PARTY-NOTICES.md", "VERSION")) { Copy-Item -Force (Join-Path $Root $f) $bin }
    $ff = Join-Path $Root "packaging\ffmpeg\windows-$arch"
    if ((Test-Path (Join-Path $ff "ffmpeg.exe")) -and (Test-Path (Join-Path $ff "ffprobe.exe"))) {
        Copy-Item -Force (Join-Path $ff "ffmpeg.exe"), (Join-Path $ff "ffprobe.exe") $bin
        $lic = Join-Path $bin "ffmpeg-licenses"
        New-Item -ItemType Directory -Force -Path $lic | Out-Null
        Get-ChildItem $ff -File | Where-Object { $_.Extension -ne ".exe" } | Copy-Item -Destination $lic -Force
        Write-Host "FFmpeg вложен: $ff"
    } else {
        Write-Host "FFmpeg не вложен (нет $ff\ffmpeg.exe, ffprobe.exe) — будет искаться в системе" -ForegroundColor Yellow
    }
    if ($Shortcut) {
        foreach ($dir in @([Environment]::GetFolderPath("Desktop"), [Environment]::GetFolderPath("Programs"))) {
            $lnk = Join-Path $dir "Orcaton.lnk"
            $sc = (New-Object -ComObject WScript.Shell).CreateShortcut($lnk)
            $sc.TargetPath = $exe
            $sc.WorkingDirectory = Split-Path $exe
            $sc.IconLocation = "$exe,0"
            $sc.Description = "Orcaton — multicam audio sync"
            $sc.Save()
            Write-Host "Ярлык: $lnk"
        }
    }
    if ($Run) { Start-Process $exe }
} finally {
    Pop-Location
}
