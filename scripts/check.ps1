param([switch]$Fuzz, [switch]$GUI)
$ErrorActionPreference = "Stop"
Push-Location (Split-Path $PSScriptRoot -Parent)
$previousCache = $env:GOCACHE
try {
    $go = Get-Command go -ErrorAction SilentlyContinue
    if ($go) { $goPath = $go.Source } else { $goPath = "$env:ProgramFiles\Go\bin\go.exe" }
    $env:GOCACHE = Join-Path (Get-Location) ".local/go-build"

    & $goPath test ./... -timeout 30s
    if ($LASTEXITCODE -ne 0) { throw "Tests failed" }
    node --test app/web/model.test.mjs
    if ($LASTEXITCODE -ne 0) { throw "GUI model tests failed" }
    & $goPath vet ./...
    if ($LASTEXITCODE -ne 0) { throw "Vet failed" }
    node node_modules/@fission-ai/openspec/bin/openspec.js validate --all --strict --no-interactive
    if ($LASTEXITCODE -ne 0) { throw "OpenSpec validation failed" }
    $cgo = & $goPath env CGO_ENABLED
    if ($LASTEXITCODE -ne 0) { throw "Cannot inspect race toolchain" }
    if ($cgo -eq '1') {
        & $goPath test -race ./... -timeout 60s
        if ($LASTEXITCODE -ne 0) { throw "Race check failed" }
    } else { Write-Output "SKIPPED: race detector requires an enabled, supported CGO toolchain" }
    if ($Fuzz) {
        & $goPath test ./internal/config -run '^$' -fuzz FuzzDecode -fuzztime 3s -parallel 2
        if ($LASTEXITCODE -ne 0) { throw "Config fuzz check failed" }
        & $goPath test ./internal/streamdeck -run '^$' -fuzz FuzzHIDDecode -fuzztime 3s -parallel 2
        if ($LASTEXITCODE -ne 0) { throw "HID fuzz check failed" }
    }
    Write-Output "SKIPPED: native probes, GUI smoke, and audible/hardware acceptance are separate opt-in checks"
    & ./scripts/build.ps1
    if ($GUI) {
        node scripts/check-gui.cjs
        if ($LASTEXITCODE -ne 0) { throw "Browser GUI checks failed" }
        node scripts/check-desktop.cjs
        if ($LASTEXITCODE -ne 0) { throw "Native GUI checks failed" }
    }
} finally {
    $env:GOCACHE = $previousCache
    Pop-Location
}
