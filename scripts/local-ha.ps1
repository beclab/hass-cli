# local-ha.ps1 - spin up a throwaway Home Assistant and smoke-test hass-cli.
#
# Usage:
#   ./scripts/local-ha.ps1            # start, smoke test, tear down
#   ./scripts/local-ha.ps1 -KeepRunning   # leave the container up for manual testing
param(
    [int]$Port = 8123,
    [string]$Name = "ha-test",
    [switch]$KeepRunning
)
$ErrorActionPreference = "Stop"
$base = "http://localhost:$Port"

Write-Host "1) starting HA container..."
docker rm -f $Name 2>$null | Out-Null
docker run -d --name $Name -p "${Port}:8123" ghcr.io/home-assistant/home-assistant:stable | Out-Null

Write-Host "2) waiting for onboarding endpoint..."
$ready = $false
for ($i = 0; $i -lt 90; $i++) {
    try { Invoke-RestMethod "$base/api/onboarding" -TimeoutSec 3 | Out-Null; $ready = $true; break }
    catch { Start-Sleep 3 }
}
if (-not $ready) { throw "HA did not become ready in time" }

Write-Host "3) creating owner user -> auth_code..."
$clientId = "$base/"
$body = @{ name = "Test"; username = "admin"; password = "admin1234"; client_id = $clientId; language = "en" } | ConvertTo-Json
$auth = Invoke-RestMethod "$base/api/onboarding/users" -Method Post -ContentType "application/json" -Body $body

Write-Host "4) exchanging auth_code -> access_token..."
$form = "grant_type=authorization_code&code=$($auth.auth_code)&client_id=$([uri]::EscapeDataString($clientId))"
$tok = Invoke-RestMethod "$base/auth/token" -Method Post -ContentType "application/x-www-form-urlencoded" -Body $form

$env:HASS_SERVER = $base
$env:HASS_TOKEN = $tok.access_token
Write-Host "HASS_SERVER=$env:HASS_SERVER  (token expires in $($tok.expires_in)s)"

Write-Host "5) building + smoke testing hass-cli..."
go build -o hass-cli.exe .
$cmds = @(
    @("ping", "-o", "json", "ping"),
    @("config", "-o", "json", "config", "get"),
    @("state-list", "-o", "json", "state", "list"),
    @("service-list", "-o", "json", "service", "list"),
    @("service-describe", "-o", "json", "service", "describe", "homeassistant.turn_on"),
    @("registry-area", "-o", "json", "registry", "area", "list"),
    @("raw-ws", "-o", "json", "raw", "ws", "get_config"),
    @("system-health", "-o", "json", "system", "health"),
    @("system-repairs", "-o", "json", "system", "repairs")
)
foreach ($c in $cmds) {
    $label = $c[0]
    $args = $c[1..($c.Length - 1)]
    Write-Host "--- $label ---"
    & .\hass-cli.exe @args
}

if (-not $KeepRunning) {
    Write-Host "6) tearing down..."
    docker rm -f $Name | Out-Null
}
