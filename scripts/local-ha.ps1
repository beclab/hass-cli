# local-ha.ps1 - spin up a throwaway Home Assistant and run P0+P1 integration tests.
#
# Usage:
#   ./scripts/local-ha.ps1                 # start, test, tear down
#   ./scripts/local-ha.ps1 -KeepRunning    # leave the container up afterwards
param(
    [int]$Port = 8123,
    [string]$Name = "ha-test",
    [switch]$KeepRunning
)
$ErrorActionPreference = "Stop"
$base = "http://localhost:$Port"
$script:pass = 0
$script:fail = 0
$tmp = Join-Path $env:TEMP "hasscli-itest"
New-Item -ItemType Directory -Force -Path $tmp | Out-Null

function J([string]$file, [string]$json) {
    $p = Join-Path $tmp $file
    Set-Content -Path $p -Value $json -Encoding ascii
    return "@$p"
}

# Check runs hass-cli with the given args and asserts the output contains $expect.
# Some commands intentionally exit non-zero (e.g. the no-Supervisor guard), so we
# relax ErrorActionPreference around the call and assert only on output text.
function Check([string]$label, [string]$expect, [string[]]$cliArgs) {
    $prev = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    $out = (& .\hass-cli.exe @cliArgs 2>&1 | Out-String)
    $ErrorActionPreference = $prev
    if ($out -match [regex]::Escape($expect)) {
        Write-Host ("  PASS  {0}" -f $label) -ForegroundColor Green
        $script:pass++
    } else {
        Write-Host ("  FAIL  {0}" -f $label) -ForegroundColor Red
        Write-Host ("        expected to contain: {0}" -f $expect)
        Write-Host ("        got: {0}" -f ($out.Trim() -replace "\s+", " ").Substring(0, [Math]::Min(200, $out.Trim().Length)))
        $script:fail++
    }
}

Write-Host "1) starting HA container..."
$ErrorActionPreference = "SilentlyContinue"; docker rm -f $Name 2>$null | Out-Null; $ErrorActionPreference = "Stop"
docker run -d --name $Name -p "${Port}:8123" ghcr.io/home-assistant/home-assistant:stable | Out-Null

Write-Host "2) waiting for onboarding endpoint..."
$ready = $false
for ($i = 0; $i -lt 90; $i++) {
    try { Invoke-RestMethod "$base/api/onboarding" -TimeoutSec 3 | Out-Null; $ready = $true; break }
    catch { Start-Sleep 2 }
}
if (-not $ready) { throw "HA did not become ready in time" }

Write-Host "3) onboarding -> access token..."
$clientId = "$base/"
$body = @{ name = "Test"; username = "admin"; password = "admin1234"; client_id = $clientId; language = "en" } | ConvertTo-Json
$auth = Invoke-RestMethod "$base/api/onboarding/users" -Method Post -ContentType "application/json" -Body $body
$form = "grant_type=authorization_code&code=$($auth.auth_code)&client_id=$([uri]::EscapeDataString($clientId))"
$tok = Invoke-RestMethod "$base/auth/token" -Method Post -ContentType "application/x-www-form-urlencoded" -Body $form
$env:HASS_SERVER = $base
$env:HASS_TOKEN = $tok.access_token

Write-Host "4) building hass-cli..."
go build -ldflags "-X main.version=0.1.0" -o hass-cli.exe .

Write-Host "`n=== P0: core ==="
Check "ping"                "API running."   @("ping", "-o", "json")
Check "config get"          "version"        @("config", "get", "-o", "json")
Check "state list"          "entity_id"      @("state", "list", "-o", "json")
Check "service list"        "domain"         @("service", "list", "-o", "json")
Check "service describe"    "brightness_pct" @("service", "describe", "light.turn_on", "-o", "json")
Check "registry area list"  "area_id"        @("registry", "area", "list", "-o", "json")
Check "raw ws get_config"   "version"        @("raw", "ws", "get_config", "-o", "json")
Check "system health"       "homeassistant"  @("system", "health", "-o", "json")
Check "system repairs"      "issues"         @("system", "repairs", "-o", "json")

Write-Host "`n=== P1: registry mutate (id positional + @file) ==="
Check "area create"  "hc_area"      @("registry", "area", "create", "--data", (J "area.json" '{"name":"HC Area"}'), "-o", "json")
Check "area update"  "HC Renamed"   @("registry", "area", "update", "hc_area", "--data", (J "area2.json" '{"name":"HC Renamed"}'), "-o", "json")
Check "area delete"  "success"      @("registry", "area", "delete", "hc_area", "-o", "json")

Write-Host "`n=== P1: helpers CRUD + operate via service call ==="
Check "input_boolean create" "hc_flag"  @("helper", "input_boolean", "create", "--data", (J "ib.json" '{"name":"HC Flag"}'), "-o", "json")
Check "input_boolean list"   "HC Flag"  @("helper", "input_boolean", "list", "-o", "json")
Check "input_boolean turn_on" "on"      @("service", "call", "input_boolean.turn_on", "--arguments", "entity_id=input_boolean.hc_flag", "-o", "json")
Check "counter create"       "hc_count" @("helper", "counter", "create", "--data", (J "ct.json" '{"name":"HC Count","initial":0,"step":1}'), "-o", "json")
Check "counter increment"    "1"        @("service", "call", "counter.increment", "--arguments", "entity_id=counter.hc_count", "-o", "json")
Check "input_number create"  "hc_level" @("helper", "input_number", "create", "--data", (J "num.json" '{"name":"HC Level","min":0,"max":100,"step":5}'), "-o", "json")
Check "input_number set"     "35"       @("service", "call", "input_number.set_value", "--data", (J "setnum.json" '{"entity_id":"input_number.hc_level","value":35}'), "-o", "json")
Check "input_select create"  "hc_mode"  @("helper", "input_select", "create", "--data", (J "sel.json" '{"name":"HC Mode","options":["home","away"]}'), "-o", "json")
Check "input_select option"  "away"     @("service", "call", "input_select.select_option", "--data", (J "selopt.json" '{"entity_id":"input_select.hc_mode","option":"away"}'), "-o", "json")

Write-Host "`n=== P1: notifications + assist ==="
Check "persistent_notification" "[]" @("service", "call", "persistent_notification.create", "--data", (J "pn.json" '{"title":"hc","message":"itest","notification_id":"hc1"}'), "-o", "json")
Check "conversation/process" "response" @("raw", "ws", "conversation/process", "--data", (J "cv.json" '{"text":"what time is it"}'), "-o", "json")

Write-Host "`n=== P2: integrations + backups ==="
Check "integration list"  "entry_id"     @("integration", "list", "-o", "json")
Check "backup agents"     "backup.local" @("backup", "agents", "-o", "json")
Check "backup list"       "idle"         @("backup", "list", "-o", "json")
# reload the sun config entry (always present on a fresh instance)
$sunEntry = ((& .\hass-cli.exe integration list -o json | ConvertFrom-Json) | Where-Object { $_.domain -eq "sun" }).entry_id
Check "integration get"    "loaded"      @("integration", "get", $sunEntry, "-o", "json")
Check "integration reload" "require_restart" @("integration", "reload", $sunEntry, "-o", "json")
# full backup lifecycle: create -> wait -> appears -> delete
Check "backup create"     "backup_job_id" @("backup", "create", "--data", (J "bk.json" '{"agent_ids":["backup.local"],"name":"itest"}'), "-o", "json")
for ($i = 0; $i -lt 15; $i++) {
    $bl = & .\hass-cli.exe backup list -o json | ConvertFrom-Json
    if ($bl.backups.Count -ge 1) { break }
    Start-Sleep 2
}
$bid = (& .\hass-cli.exe backup list -o json | ConvertFrom-Json).backups[0].backup_id
Check "backup appeared"   "itest"        @("backup", "get", $bid, "-o", "json")
Check "backup delete"     "agent_errors" @("backup", "delete", $bid, "-o", "json")

Write-Host "`n=== P2: config flow (add an integration end to end) ==="
Check "flow handlers"  "random"   @("integration", "flow", "handlers", "--type", "helper", "-o", "json")
Check "flow progress"  "["        @("integration", "flow", "progress", "-o", "json")
# Drive the 'random' helper flow: start (menu) -> choose sensor -> submit name -> create_entry
$fid = (& .\hass-cli.exe integration flow start random -o json | ConvertFrom-Json).flow_id
& .\hass-cli.exe integration flow step $fid --data (J "fl1.json" '{"next_step_id":"sensor"}') -o json | Out-Null
Check "flow create_entry" "create_entry" @("integration", "flow", "step", $fid, "--data", (J "fl2.json" '{"name":"itest random"}'), "-o", "json")
$randEntry = ((& .\hass-cli.exe integration list --domain random -o json | ConvertFrom-Json) | Select-Object -First 1).entry_id
Check "flow entry created" "loaded" @("integration", "get", $randEntry, "-o", "json")
& .\hass-cli.exe integration delete $randEntry -o json | Out-Null

Write-Host "`n=== P2: system maintenance (hardware / analytics / labs) ==="
Check "system hardware"   "hardware"    @("system", "hardware", "-o", "json")
Check "system labs"       "preview_feature" @("system", "labs", "-o", "json")
Check "system analytics"  "preferences" @("system", "analytics", "-o", "json")
Check "analytics set"     "statistics"  @("system", "analytics", "set", "--data", (J "an.json" '{"base":true,"statistics":true}'), "-o", "json")

Write-Host "`n=== P3: Lovelace dashboards (create -> save config -> get -> delete) ==="
Check "dashboard list (empty)" "[" @("lovelace", "dashboard", "list", "-o", "json")
$dash = & .\hass-cli.exe lovelace dashboard create --data (J "dash.json" '{"url_path":"hc-itest","title":"HC Itest"}') -o json | ConvertFrom-Json
Check "dashboard created" "hc-itest" @("lovelace", "dashboard", "list", "-o", "json")
$viewYaml = Join-Path $tmp "view.yaml"
Set-Content -Path $viewYaml -Value "views:`n  - title: Home`n    cards:`n      - type: markdown`n        content: itest" -Encoding ascii
& .\hass-cli.exe lovelace config save --dashboard hc-itest --file $viewYaml -o json | Out-Null
Check "config get roundtrip" "itest" @("lovelace", "config", "get", "--dashboard", "hc-itest", "-o", "yaml")
Check "resource list" "[" @("lovelace", "resource", "list", "-o", "json")
& .\hass-cli.exe lovelace dashboard delete $dash.id -o json | Out-Null

Write-Host "`n=== P3: Assist pipelines ==="
Check "pipeline list" "preferred_pipeline" @("assist", "pipeline", "list", "-o", "json")
Check "pipeline get preferred" "conversation_engine" @("assist", "pipeline", "get", "-o", "json")
Check "assist languages" "languages" @("assist", "languages", "-o", "json")
Check "assist devices" "[" @("assist", "devices", "-o", "json")

Write-Host "`n=== P3: category registry / energy / statistics ==="
Check "category list" "[" @("registry", "category", "list", "--scope", "automation", "-o", "json")
$cat = & .\hass-cli.exe registry category create --scope automation --data (J "cat.json" '{"name":"HC Cat"}') -o json | ConvertFrom-Json
Check "category created" "HC Cat" @("registry", "category", "list", "--scope", "automation", "-o", "json")
& .\hass-cli.exe registry category delete $cat.category_id --scope automation -o json | Out-Null
& .\hass-cli.exe energy prefs save --data (J "ep.json" '{"energy_sources":[],"device_consumption":[]}') -o json | Out-Null
Check "energy prefs get" "energy_sources" @("energy", "prefs", "get", "-o", "json")
Check "energy info" "cost_sensors" @("energy", "info", "-o", "json")
Check "statistics info" "recording" @("statistics", "info", "-o", "json")
Check "statistics list" "[" @("statistics", "list", "-o", "json")

Write-Host "`n=== P4: polish (supervisor guard / default columns / validation / --file) ==="
# Container has no Supervisor -> addon commands must fail fast with a friendly message
Check "addon no-supervisor" "no Supervisor" @("addon", "list")
# default table columns: state list (table mode) shows the ENTITY header
Check "state list default cols" "ENTITY" @("state", "list")
# energy prefs save via --file (YAML)
$epYaml = Join-Path $tmp "ep.yaml"
Set-Content -Path $epYaml -Value "energy_sources: []`ndevice_consumption: []" -Encoding ascii
Check "energy prefs save --file" "energy_sources" @("energy", "prefs", "save", "--file", $epYaml, "-o", "json")
# write-command validation: empty --data update is rejected locally
$ErrorActionPreference = "Continue"
$vout = (& .\hass-cli.exe registry area create 2>&1 | Out-String)
$ErrorActionPreference = "Stop"
if ($vout -match "data is required") {
    Write-Host "  PASS  empty create rejected" -ForegroundColor Green; $script:pass++
} else {
    Write-Host "  FAIL  empty create rejected" -ForegroundColor Red
    Write-Host ("        got: {0}" -f ($vout.Trim() -replace "\s+", " ")); $script:fail++
}

# cleanup created helpers
foreach ($h in @(@("input_boolean", "hc_flag"), @("counter", "hc_count"), @("input_number", "hc_level"), @("input_select", "hc_mode"))) {
    & .\hass-cli.exe helper $h[0] delete $h[1] 2>&1 | Out-Null
}

Write-Host "`n=== summary ==="
Write-Host ("  {0} passed, {1} failed" -f $script:pass, $script:fail) -ForegroundColor $(if ($script:fail -eq 0) { "Green" } else { "Red" })
Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue

if (-not $KeepRunning) {
    Write-Host "tearing down..."
    docker rm -f $Name | Out-Null
}
if ($script:fail -gt 0) { exit 1 }
