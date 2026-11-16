# Installs the latest WaveTrack release on Windows.
#
#   irm https://raw.githubusercontent.com/lucasrafaldini/wavetrack/main/scripts/install.ps1 | iex
#
# Or, with options, from a downloaded copy:
#   .\install.ps1 -Version 3.1.0 -Destination "C:\Tools\WaveTrack"
#
# The download is checked against the release's checksums.txt before anything is installed.
# Installs for the current user only; no administrator rights needed.

param(
    [string]$Version = "",
    [string]$Destination = (Join-Path $env:LOCALAPPDATA "Programs\WaveTrack")
)

$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"
$Repo = "lucasrafaldini/wavetrack"

if (-not [Environment]::Is64BitOperatingSystem) {
    throw "WaveTrack ships 64-bit Windows builds only."
}
if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") {
    Write-Host "No ARM64 build for Windows yet; installing the x64 build, which runs under emulation."
}

if (-not $Version) {
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" -Headers @{ "User-Agent" = "wavetrack-installer" }
    $Version = $release.tag_name.TrimStart("v")
}
$Version = $Version.TrimStart("v")

$name = "wavetrack_${Version}_windows_amd64"
$base = "https://github.com/$Repo/releases/download/v$Version"
$tmp = Join-Path ([IO.Path]::GetTempPath()) ("wavetrack-" + [Guid]::NewGuid())
New-Item -ItemType Directory -Path $tmp | Out-Null

try {
    Write-Host "Downloading WaveTrack $Version for Windows x64"
    Invoke-WebRequest -Uri "$base/$name.zip" -OutFile (Join-Path $tmp "$name.zip")
    Invoke-WebRequest -Uri "$base/checksums.txt" -OutFile (Join-Path $tmp "checksums.txt")

    $line = Get-Content (Join-Path $tmp "checksums.txt") | Where-Object { $_ -match "\s$([Regex]::Escape("$name.zip"))$" }
    if (-not $line) { throw "$name.zip isn't listed in checksums.txt" }
    $expected = ($line -split "\s+")[0].ToLowerInvariant()
    $actual = (Get-FileHash -Algorithm SHA256 (Join-Path $tmp "$name.zip")).Hash.ToLowerInvariant()
    if ($expected -ne $actual) { throw "Checksum mismatch: expected $expected, got $actual" }
    Write-Host "Checksum verified"

    Expand-Archive -Path (Join-Path $tmp "$name.zip") -DestinationPath $tmp -Force
    New-Item -ItemType Directory -Path $Destination -Force | Out-Null
    Copy-Item -Path (Join-Path $tmp "$name\*") -Destination $Destination -Recurse -Force
    Write-Host "Installed $Destination\wavetrack.exe"
}
finally {
    Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
}

$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if (-not (($userPath -split ";") -contains $Destination)) {
    [Environment]::SetEnvironmentVariable("Path", (($userPath.TrimEnd(";"), $Destination) -join ";"), "User")
    Write-Host "Added $Destination to your PATH (open a new terminal to use it)"
}

Write-Host ""
Write-Host "Next steps"
Write-Host "  wavetrack serve --demo     try it now with a simulated office"
Write-Host "  wavetrack doctor           check capture permission and configuration"
$npcap = Join-Path $env:WINDIR "System32\Npcap\wpcap.dll"
if (-not (Test-Path $npcap)) {
    Write-Host "  Install Npcap from https://npcap.com with 'WinPcap API-compatible mode' checked:"
    Write-Host "  live capture needs it (the demo doesn't)."
}
