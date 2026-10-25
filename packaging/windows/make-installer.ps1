param([ValidateSet("all", "x64", "arm64")][string]$Arch = "all", [switch]$NoBuild)
$ErrorActionPreference = "Stop"
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$Root = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$AppGui = Join-Path $Root "app-gui"
$Bin = Join-Path $AppGui "build\bin"
$Out = Join-Path $AppGui "build\installer"
$Nsi = Join-Path $AppGui "build\windows\installer\orcaton.nsi"
$Version = ([IO.File]::ReadAllText((Join-Path $Root "VERSION"))).Trim()
$Files = @("orcaton-gui.exe", "orcaton.exe", "LICENSE", "NOTICE", "THIRD-PARTY-NOTICES.md", "VERSION")
$GoArch = @{ x64 = "amd64"; arm64 = "arm64" }

function Step($text) { Write-Host "`n== $text ==" -ForegroundColor Cyan }
function Fail($text) { Write-Host $text -ForegroundColor Red; exit 1 }

function PeArch($path) {
    $b = [IO.File]::ReadAllBytes($path)
    $pe = [BitConverter]::ToInt32($b, 0x3C)
    switch ([BitConverter]::ToUInt16($b, $pe + 4)) { 0x8664 { "amd64" } 0xAA64 { "arm64" } default { "other" } }
}

$makensis = (Get-Command "makensis" -ErrorAction SilentlyContinue).Source
if (-not $makensis) {
    foreach ($d in @(${env:ProgramFiles(x86)}, $env:ProgramFiles)) {
        if ($d -and (Test-Path (Join-Path $d "NSIS\makensis.exe"))) { $makensis = Join-Path $d "NSIS\makensis.exe"; break }
    }
}
if (-not $makensis) { Fail "Нет NSIS (makensis). Установите: winget install -e --id NSIS.NSIS — и откройте новое окно PowerShell." }
$nv = (& $makensis /VERSION) -replace '^v', ''
if (-not ($nv -match '^3\.')) { Fail "Нужен NSIS 3, найден $nv ($makensis)" }

$wj = Get-Content (Join-Path $AppGui "wails.json") -Raw | ConvertFrom-Json
if ($wj.info.productVersion -ne $Version) { Fail "Версия расходится: VERSION $Version, wails.json $($wj.info.productVersion) — scripts\set-version.ps1 $Version" }

$native = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }
$archs = if ($Arch -eq "all") { @("x64", "arm64") } else { @($Arch) }
$archs = @($archs | Where-Object { $GoArch[$_] -ne $native }) + @($archs | Where-Object { $GoArch[$_] -eq $native })
if ($NoBuild -and ($archs | Where-Object { $GoArch[$_] -ne $native })) {
    Fail "-NoBuild — только своя архитектура ($native): build\bin одна. Уберите -NoBuild или укажите -Arch."
}
New-Item -ItemType Directory -Force -Path $Out | Out-Null
Write-Host "Orcaton $Version · NSIS $nv · архитектуры: $($archs -join ', ')"

$made = @()
foreach ($a in $archs) {
    $ga = $GoArch[$a]
    $stage = Join-Path $Out "stage-$a"
    if (-not $NoBuild) {
        Step "Сборка $a"
        $backup = $null
        if ($ga -ne $native -and (Test-Path (Join-Path $Bin "orcaton-gui.exe"))) {
            $backup = Join-Path $Out "bin-backup"
            if (Test-Path $backup) { Remove-Item -Recurse -Force $backup }
            New-Item -ItemType Directory -Force -Path $backup | Out-Null
            foreach ($f in @("orcaton-gui.exe", "orcaton.exe")) { if (Test-Path (Join-Path $Bin $f)) { Copy-Item -Force (Join-Path $Bin $f) $backup } }
        }
        try {
            & (Join-Path $AppGui "scripts\build-gui.ps1") -Arch $ga
            if ($LASTEXITCODE -ne 0) { Fail "Сборка $a не удалась" }
            if (Test-Path $stage) { Remove-Item -Recurse -Force $stage }
            New-Item -ItemType Directory -Force -Path $stage | Out-Null
            foreach ($f in $Files) { Copy-Item -Force (Join-Path $Bin $f) $stage }
        } finally {
            if ($backup) {
                Get-ChildItem $backup -File | Copy-Item -Destination $Bin -Force
                Remove-Item -Recurse -Force $backup
            }
        }
    } else {
        if (Test-Path $stage) { Remove-Item -Recurse -Force $stage }
        New-Item -ItemType Directory -Force -Path $stage | Out-Null
        foreach ($f in $Files) {
            if (-not (Test-Path (Join-Path $Bin $f))) { Fail "Нет $Bin\$f — сначала app-gui\scripts\build-gui.ps1" }
            Copy-Item -Force (Join-Path $Bin $f) $stage
        }
    }
    foreach ($f in @("orcaton-gui.exe", "orcaton.exe")) {
        $pa = PeArch (Join-Path $stage $f)
        if ($pa -ne $ga) { Fail "$f — архитектура $pa, ожидалась $ga" }
    }
    $lic = [IO.File]::ReadAllText((Join-Path $stage "LICENSE")) -replace "`r`n", "`n" -replace "`n", "`r`n"
    [IO.File]::WriteAllText((Join-Path $stage "license-page.txt"), $lic, (New-Object System.Text.UTF8Encoding($false)))

    Step "Установщик $a"
    $name = "Orcaton-$Version-windows-$a-setup.exe"
    $setup = Join-Path $Out $name
    if (Test-Path $setup) { Remove-Item -Force $setup }
    & $makensis /V2 /WX /INPUTCHARSET UTF8 "/DVERSION=$Version" "/DARCH=$a" "/DSRC=$stage" "/DOUTFILE=$setup" $Nsi
    if ($LASTEXITCODE -ne 0 -or -not (Test-Path $setup)) { Fail "makensis: ошибка ($a)" }
    $hash = (Get-FileHash -Algorithm SHA256 $setup).Hash.ToLowerInvariant()
    [IO.File]::WriteAllText("$setup.sha256", "$hash  $name`n", (New-Object System.Text.UTF8Encoding($false)))
    Remove-Item -Recurse -Force $stage
    $mb = [math]::Round((Get-Item $setup).Length / 1MB, 1)
    Write-Host "Готово: $setup ($mb МБ)" -ForegroundColor Green
    Write-Host "SHA-256: $hash"
    $made += $setup
}

Write-Host "`nУстановщики: $Out" -ForegroundColor Green
foreach ($m in $made) { Write-Host "  $(Split-Path -Leaf $m)" }
Write-Host "Проверка (PowerShell от имени администратора; лучше в «Песочнице Windows»): packaging\windows\check-installer.ps1"
Write-Host "Не подписаны: при запуске SmartScreen — More info → Run anyway."
