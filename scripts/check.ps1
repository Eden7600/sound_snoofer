param([switch]$Fuzz, [switch]$GUI)
$ErrorActionPreference = "Stop"
Push-Location (Split-Path $PSScriptRoot -Parent)
$previousCache = $env:GOCACHE
try {
    & ./scripts/check-build.ps1
    $go = Get-Command go -ErrorAction SilentlyContinue
    if ($go) { $goPath = $go.Source } else { $goPath = "$env:ProgramFiles\Go\bin\go.exe" }
    $env:GOCACHE = Join-Path (Get-Location) ".local/go-build"

    & $goPath test ./... -timeout 30s
    if ($LASTEXITCODE -ne 0) { throw "Tests failed" }
    & npm run --silent typecheck
    if ($LASTEXITCODE -ne 0) { throw "GUI type-check failed" }
    & npm run --silent lint:gui
    if ($LASTEXITCODE -ne 0) { throw "GUI lint failed" }
    & npm run --silent format:gui
    if ($LASTEXITCODE -ne 0) { throw "GUI format check failed" }
    & npm run --silent test:model
    if ($LASTEXITCODE -ne 0) { throw "GUI model tests failed" }
    # Browser GUI suite: installed Chrome, never a browser download.
    npx playwright test -c app/web/test
    if ($LASTEXITCODE -ne 0) { throw "Browser GUI tests failed" }
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
    Write-Output "SKIPPED: native probes, native GUI smoke, and audible/hardware acceptance are separate opt-in checks"
    & ./scripts/build.ps1
    if ($GUI) {
        node scripts/check-desktop.cjs
        if ($LASTEXITCODE -ne 0) { throw "Native GUI checks failed" }
    }
} finally {
    $env:GOCACHE = $previousCache
    Pop-Location
}
