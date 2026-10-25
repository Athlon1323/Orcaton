param(
    [Parameter(Mandatory = $true)][ValidateSet("Add", "Remove")][string]$Action,
    [Parameter(Mandatory = $true)][string]$Dir,
    [ValidateSet("User", "Machine")][string]$Scope = "User",
    [string[]]$Also = @()
)
$ErrorActionPreference = "Stop"

function Norm([string]$p) {
    $x = [Environment]::ExpandEnvironmentVariables($p.Trim().Trim('"'))
    return $x.TrimEnd('\').ToLowerInvariant()
}

function Update-PathValue([string]$old, [string]$Action, [string]$Dir, [string[]]$Also) {
    $Dir = $Dir.Trim().TrimEnd('\')
    if (-not $Dir) { throw "empty -Dir" }
    $self = Norm $Dir
    $old_dirs = @($Also | Where-Object { $_ } | ForEach-Object { Norm $_ } | Where-Object { $_ -ne $self })
    $parts = New-Object System.Collections.Generic.List[string]
    $found = $false
    foreach ($p in ($old -split ";")) {
        if ($p.Trim() -ne "") {
            $n = Norm $p
            if ($old_dirs -contains $n) { continue }
            if ($n -eq $self) {
                if ($Action -eq "Remove" -or $found) { continue }
                $found = $true
            }
        }
        $parts.Add($p)
    }
    if ($Action -eq "Add" -and -not $found) {
        while ($parts.Count -gt 0 -and $parts[$parts.Count - 1].Trim() -eq "") { $parts.RemoveAt($parts.Count - 1) }
        $parts.Add($Dir)
    }
    return ($parts -join ";")
}
if ($env:ORCATON_PATH_TEST) { return }

try {
    if ($Scope -eq "Machine") {
        $key = [Microsoft.Win32.Registry]::LocalMachine.OpenSubKey("SYSTEM\CurrentControlSet\Control\Session Manager\Environment", $true)
        if (-not $key) { throw "system environment key not found" }
    } else {
        $key = [Microsoft.Win32.Registry]::CurrentUser.CreateSubKey("Environment")
    }
    try {
        $name = $key.GetValueNames() | Where-Object { $_ -ieq "Path" } | Select-Object -First 1
        if ($name) {
            $kind = $key.GetValueKind($name)
            if ($kind -ne [Microsoft.Win32.RegistryValueKind]::ExpandString -and $kind -ne [Microsoft.Win32.RegistryValueKind]::String) {
                throw "user PATH has an unexpected type ($kind) - left unchanged"
            }
            $old = [string]$key.GetValue($name, "", [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)
        } elseif ($Scope -eq "Machine") {
            throw "system PATH not found - left unchanged"
        } else {
            $name = "Path"; $old = ""; $kind = [Microsoft.Win32.RegistryValueKind]::ExpandString
        }

        $new = Update-PathValue $old $Action $Dir $Also

        if ($new -ceq $old) {
            Write-Output "PATH ($Scope): no change"
        } else {
            $key.SetValue($name, $new, $kind)
            if ($Action -eq "Add") { Write-Output "PATH ($Scope): added $Dir" } else { Write-Output "PATH ($Scope): removed $Dir" }
        }
    } finally { $key.Close() }
    exit 0
} catch {
    Write-Output "PATH: error - $($_.Exception.Message)"
    exit 1
}
