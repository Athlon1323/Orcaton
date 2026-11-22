param([switch]$Run, [switch]$NoFFmpeg, [switch]$NoPath, [switch]$NoDesktop)
$ErrorActionPreference = "Stop"
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$Root = Split-Path -Parent $PSScriptRoot
$Version = ([IO.File]::ReadAllText((Join-Path $Root "VERSION"))).Trim()
$Dest = Join-Path $env:LOCALAPPDATA "Programs\Orcaton"
$OldCli = Join-Path $env:LOCALAPPDATA "Programs\orcaton-go"
$RegKey = "HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall\Orcaton"
$env:GOTOOLCHAIN = "auto"
if ($env:ProgramFiles -and $Root.StartsWith($env:ProgramFiles, [StringComparison]::OrdinalIgnoreCase)) {
    throw "The project is inside Program Files - the build will fail there without administrator rights. Copy the project folder to a regular location, e.g. $env:USERPROFILE\Projects\Orcaton, and run the script from there."
}

function Have($cmd) { [bool](Get-Command $cmd -ErrorAction SilentlyContinue) }
function Step($text) { Write-Host "`n== $text ==" -ForegroundColor Cyan }
function RegPath {
    $k = [Microsoft.Win32.Registry]::CurrentUser.CreateSubKey("Environment")
    try {
        $n = $k.GetValueNames() | Where-Object { $_ -ieq "Path" } | Select-Object -First 1
        if (-not $n) { return @{ Name = "Path"; Text = ""; Kind = [Microsoft.Win32.RegistryValueKind]::ExpandString } }
        return @{ Name = $n; Kind = $k.GetValueKind($n)
                  Text = [string]$k.GetValue($n, "", [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames) }
    } finally { $k.Close() }
}
function UserPath { @(((RegPath).Text -split ";") | Where-Object { $_ }) }
function Set-UserPath($parts) {
    $cur = RegPath
    $new = (@($parts) -join ";")
    if ($new -ceq (@($cur.Text -split ";" | Where-Object { $_ }) -join ";")) { return }
    $k = [Microsoft.Win32.Registry]::CurrentUser.CreateSubKey("Environment")
    try { $k.SetValue($cur.Name, $new, $cur.Kind) } finally { $k.Close() }
    [Environment]::SetEnvironmentVariable("ORCATON_PATH_REFRESH", "1", "User")
    [Environment]::SetEnvironmentVariable("ORCATON_PATH_REFRESH", $null, "User")
}
function Show($name, $ok, $info) {
    if ($ok) { Write-Host ("[OK] {0,-12} {1}" -f $name, $info) -ForegroundColor Green }
    else     { Write-Host ("[--] {0,-12} {1}" -f $name, $info) -ForegroundColor Yellow }
}
function GoOk {
    if (-not (Have "go")) { return $false }
    $v = (& go env GOVERSION) -replace '^go', '' -replace '[^0-9.].*$', ''
    try { return ([version]$v -ge [version]"1.22") } catch { return $false }
}
function NodeOk {
    if (-not (Have "node")) { return $false }
    try { $v = [version]((node -v) -replace '^v', '') } catch { return $false }
    return ($v.Major -ge 23) -or ($v.Major -eq 22 -and $v.Minor -ge 12) -or ($v.Major -eq 20 -and $v.Minor -ge 19)
}
function WebView2Ok {
    $id = "{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}"
    foreach ($k in @("HKLM:\SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\$id",
                     "HKLM:\SOFTWARE\Microsoft\EdgeUpdate\Clients\$id",
                     "HKCU:\SOFTWARE\Microsoft\EdgeUpdate\Clients\$id")) {
        $pv = (Get-ItemProperty -Path $k -Name pv -ErrorAction SilentlyContinue).pv
        if ($pv -and $pv -ne "0.0.0.0") { return $true }
    }
    return $false
}
function Shortcut($lnk, $exe) {
    $sc = (New-Object -ComObject WScript.Shell).CreateShortcut($lnk)
    $sc.TargetPath = $exe
    $sc.WorkingDirectory = Split-Path $exe
    $sc.IconLocation = "$exe,0"
    $sc.Description = "Orcaton — multicam audio sync"
    $sc.Save()
}

Write-Host "Orcaton $Version - installing for user $env:USERNAME" -ForegroundColor Green

if (Have "go") { $env:Path += ";" + (Join-Path (go env GOPATH) "bin") }

Step "1/3 Checking tools"
$missing = @()
if (GoOk) { Show "Go" $true (go env GOVERSION) } else {
    Show "Go" $false "missing or older than 1.22"; $missing += "Go 1.22+:  winget install --id GoLang.Go -e"
}
if (NodeOk) { Show "Node.js" $true (node -v) } else {
    Show "Node.js" $false "missing or not 20.19+ / 22.12+"; $missing += "Node.js LTS:  winget install --id OpenJS.NodeJS.LTS -e"
}
if (WebView2Ok) { Show "WebView2" $true "found" } else {
    Show "WebView2" $false "missing"; $missing += "WebView2:  winget install --id Microsoft.EdgeWebView2Runtime -e"
}
if (Have "wails") { Show "Wails CLI" $true "found" } else {
    Show "Wails CLI" $false "missing"; $missing += "Wails CLI v2.13.0:  go install github.com/wailsapp/wails/v2/cmd/wails@v2.13.0"
}
if ((Have "ffmpeg") -and (Have "ffprobe")) { Show "FFmpeg" $true "found (optional)" } else {
    Show "FFmpeg" $false "missing (optional - the window offers it on launch; or: winget install --id Gyan.FFmpeg -e)"
}
if ($missing.Count -gt 0) {
    Write-Host "`nSome tools are missing. The script installs nothing itself. Run the commands below (each separately),"
    Write-Host "open a NEW PowerShell window in the project folder and run the script again:" -ForegroundColor Yellow
    foreach ($m in $missing) { Write-Host "  $m" -ForegroundColor Cyan }
    exit 1
}

Step "2/3 Build"
& (Join-Path $Root "app-gui\scripts\build-gui.ps1")
if ($LASTEXITCODE -ne 0) { throw "build failed" }
$Bin = Join-Path $Root "app-gui\build\bin"

Step "3/3 Installing to $Dest"
$MachineKey = "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\Orcaton"
if (Test-Path $MachineKey) {
    $loc = (Get-ItemProperty -Path $MachineKey -ErrorAction SilentlyContinue).InstallLocation
    Write-Host "Warning: Orcaton is already installed by the .exe installer ($loc). This will be a second copy (per user);" -ForegroundColor Yellow
    Write-Host "the orcaton command in a terminal comes from Program Files (system PATH goes first). To remove that one: Settings > Apps > Orcaton." -ForegroundColor Yellow
}
if (Get-Process -Name "orcaton-gui" -ErrorAction SilentlyContinue) { throw "Orcaton is running - close the window and try again" }
New-Item -ItemType Directory -Force -Path $Dest | Out-Null
Get-ChildItem $Bin -Force | Copy-Item -Destination $Dest -Recurse -Force
Copy-Item -Force (Join-Path $PSScriptRoot "uninstall.ps1") $Dest
Remove-Item -Force (Join-Path $Dest "uninstall.exe") -ErrorAction SilentlyContinue
$exe = Join-Path $Dest "orcaton-gui.exe"
$made = @()
$start = Join-Path ([Environment]::GetFolderPath("Programs")) "Orcaton.lnk"
Shortcut $start $exe; $made += "shortcut=$start"
if (-not $NoDesktop) {
    $desk = Join-Path ([Environment]::GetFolderPath("Desktop")) "Orcaton.lnk"
    Shortcut $desk $exe; $made += "shortcut=$desk"
}
$parts = UserPath | Where-Object { $_ -ne $OldCli }
if (-not $NoPath -and -not ($parts -contains $Dest)) { $parts += $Dest }
Set-UserPath $parts
if (-not $NoPath) { $made += "path=$Dest" }
if (Test-Path $OldCli) { Remove-Item -Recurse -Force $OldCli -ErrorAction SilentlyContinue }
$size = [int]((Get-ChildItem $Dest -Recurse -File | Measure-Object Length -Sum).Sum / 1KB)
if (Test-Path $RegKey) { Remove-Item -Recurse -Force $RegKey }
New-Item -Path $RegKey -Force | Out-Null
$props = @{ DisplayName = "Orcaton"; DisplayVersion = $Version; Publisher = "Athlon1323"; InstallLocation = $Dest;
    DisplayIcon = "$exe,0"; NoModify = 1; NoRepair = 1; EstimatedSize = $size;
    UninstallString = "powershell.exe -NoProfile -ExecutionPolicy Bypass -File `"$Dest\uninstall.ps1`"" }
foreach ($k in $props.Keys) {
    $type = if ($props[$k] -is [int]) { "DWord" } else { "String" }
    New-ItemProperty -Path $RegKey -Name $k -Value $props[$k] -PropertyType $type -Force | Out-Null
}
$made += "registry=$RegKey"
[IO.File]::WriteAllLines((Join-Path $Dest "install-manifest.txt"), [string[]]$made, (New-Object System.Text.UTF8Encoding($false)))

Write-Host "`nDone: Orcaton $Version is installed." -ForegroundColor Green
Write-Host "  Window: Start menu > Orcaton$(if (-not $NoDesktop) { ' (or the desktop shortcut)' })"
if (-not $NoPath) { Write-Host "  Command: orcaton --help (in a NEW terminal window)" }
Write-Host "  Uninstall: Settings > Apps > Orcaton, or scripts\uninstall.ps1"
if ($Run) { Start-Process $exe }
