param([switch]$Purge, [switch]$Yes)
$ErrorActionPreference = "Stop"
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$Dest = Join-Path $env:LOCALAPPDATA "Programs\Orcaton"
$RegKey = "HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall\Orcaton"
$Data = @((Join-Path $env:APPDATA "orcaton"), (Join-Path $env:LOCALAPPDATA "orcaton"))

if (-not (Test-Path (Join-Path $Dest "orcaton-gui.exe")) -and -not (Test-Path $RegKey)) {
    Write-Host "Orcaton не установлен ($Dest)."; exit 0
}
if (-not $Yes) {
    $what = if ($Purge) { "программу, ярлыки, настройки и кэш" } else { "программу и ярлыки (настройки и кэш останутся)" }
    $a = Read-Host "Удалить Orcaton — $what? [y/N]"
    if ($a -notmatch '^(y|yes|д|да)$') { Write-Host "Отменено."; exit 0 }
}
if (Get-Process -Name "orcaton-gui" -ErrorAction SilentlyContinue) { throw "Orcaton сейчас открыт — закройте окно и повторите" }

$manifest = Join-Path $Dest "install-manifest.txt"
$items = if (Test-Path $manifest) { Get-Content $manifest } else {
    @("shortcut=" + (Join-Path ([Environment]::GetFolderPath("Programs")) "Orcaton.lnk"),
      "shortcut=" + (Join-Path ([Environment]::GetFolderPath("Desktop")) "Orcaton.lnk"), "path=$Dest", "registry=$RegKey")
}
foreach ($it in $items) {
    $kind, $val = $it -split "=", 2
    switch ($kind) {
        "shortcut" {
            if (Test-Path $val) {
                $t = (New-Object -ComObject WScript.Shell).CreateShortcut($val).TargetPath
                if ($t -and $t.StartsWith($Dest, [StringComparison]::OrdinalIgnoreCase)) { Remove-Item -Force $val; Write-Host "Ярлык: $val" }
            }
        }
        "path" {
            $parts = @(([Environment]::GetEnvironmentVariable("Path", "User") -split ";") | Where-Object { $_ -and $_ -ne $val })
            [Environment]::SetEnvironmentVariable("Path", ($parts -join ";"), "User")
            Write-Host "PATH: убрано $val"
        }
        "registry" { if ($val -eq $RegKey -and (Test-Path $val)) { Remove-Item -Recurse -Force $val; Write-Host "«Приложения»: запись удалена" } }
    }
}
if ($Purge) {
    foreach ($d in $Data) { if (Test-Path $d) { Remove-Item -Recurse -Force $d; Write-Host "Удалено: $d" } }
} else {
    Write-Host "Настройки и кэш остались: $($Data -join ', ') (удалить: -Purge)"
}
Set-Location $env:TEMP
Remove-Item -Recurse -Force $Dest -ErrorAction SilentlyContinue
if (Test-Path $Dest) {
    Start-Process -WindowStyle Hidden cmd.exe -ArgumentList "/c timeout /t 2 >nul & rd /s /q `"$Dest`""
}
Write-Host "`nOrcaton удалён. Go, Node.js, Wails и FFmpeg не тронуты (удалить при желании: winget uninstall <имя>)." -ForegroundColor Green
