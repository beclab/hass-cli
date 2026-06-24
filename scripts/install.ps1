# Install hass-cli on Windows by downloading the matching release tarball.
#
#   irm https://raw.githubusercontent.com/bytetrade/hass-cli/main/scripts/install.ps1 | iex
#
# Environment / params:
#   -Version   release version to install (default: latest)
#   -BinDir    install dir (default: %LOCALAPPDATA%\Programs\hass-cli)
[CmdletBinding()]
param(
  [string]$Version = $env:HASS_CLI_VERSION,
  [string]$BinDir = $env:HASS_CLI_BIN_DIR
)

$ErrorActionPreference = 'Stop'
$repo = 'bytetrade/hass-cli'

if (-not $Version) { $Version = 'latest' }

# Resolve version tag.
if ($Version -eq 'latest') {
  $rel = Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases/latest" -Headers @{ 'User-Agent' = 'hass-cli-install' }
  $tag = $rel.tag_name
  if (-not $tag) { throw 'could not resolve latest release tag' }
} elseif ($Version.StartsWith('v')) {
  $tag = $Version
} else {
  $tag = "v$Version"
}

# Map architecture (must match .goreleaser.yaml name_template).
switch ($env:PROCESSOR_ARCHITECTURE) {
  'AMD64' { $goarch = 'amd64' }
  'ARM64' { $goarch = 'arm64' }
  default { throw "unsupported architecture: $env:PROCESSOR_ARCHITECTURE" }
}

$name = "hass-cli-${tag}_windows_${goarch}.tar.gz"
$url = "https://github.com/$repo/releases/download/$tag/$name"

if (-not $BinDir) { $BinDir = Join-Path $env:LOCALAPPDATA 'Programs\hass-cli' }
New-Item -ItemType Directory -Force -Path $BinDir | Out-Null

$tmp = Join-Path $env:TEMP ("hass-cli-" + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Force -Path $tmp | Out-Null
try {
  $tarball = Join-Path $tmp 'hass-cli.tar.gz'
  Write-Host "downloading $url"
  Invoke-WebRequest -Uri $url -OutFile $tarball -Headers @{ 'User-Agent' = 'hass-cli-install' }

  # Windows 10+ ships bsdtar as `tar`, which handles .tar.gz.
  tar -xzf $tarball -C $tmp
  $bin = Join-Path $tmp 'hass-cli.exe'
  if (-not (Test-Path $bin)) { throw 'archive did not contain hass-cli.exe' }
  Copy-Item -Force $bin (Join-Path $BinDir 'hass-cli.exe')
} finally {
  Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
}

Write-Host "installed hass-cli $tag to $BinDir\hass-cli.exe"
$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if ($userPath -notlike "*$BinDir*") {
  Write-Host "note: $BinDir is not on your PATH. Add it with:"
  Write-Host "  setx PATH `"$BinDir;`$env:PATH`""
}
Write-Host "run 'hass-cli init' to configure your Home Assistant connection"
