param([string]$Version = "")
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
$utf8 = New-Object System.Text.UTF8Encoding($false)
$vf = Join-Path $Root "VERSION"
if ($Version) { [IO.File]::WriteAllText($vf, "$Version`n", $utf8) }
$v = ([IO.File]::ReadAllText($vf)).Trim()
if ($v -notmatch '^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$') { throw "VERSION: does not look like a version: '$v'" }
function Patch($rel, $pattern, $replacement, $count) {
    $p = Join-Path $Root $rel
    $s = [IO.File]::ReadAllText($p)
    $re = New-Object System.Text.RegularExpressions.Regex($pattern, "Multiline")
    $s = $re.Replace($s, $replacement, $count)
    [IO.File]::WriteAllText($p, $s, $utf8)
}
Patch "app-go\internal\app\report.go" '^const Version = "[^"]*"' "const Version = `"$v`"" 1
Patch "app-gui\wails.json" '("productVersion": )"[^"]*"' "`${1}`"$v`"" 1
Patch "app-gui\frontend\package.json" '^(  "version"): "[^"]*"' "`${1}: `"$v`"" 1
Patch "app-gui\frontend\package-lock.json" '^(  "version"|      "version"): "[^"]*"' "`${1}: `"$v`"" 2
Write-Host "Version ${v}: VERSION, app.Version, wails.json, package.json, package-lock.json"
Write-Host "Window fixtures: cd app-gui\session; go test -run Fixtures -update"
