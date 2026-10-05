param([string]$Output = "bin/snoofer.exe", [string]$Tags = "")
$ErrorActionPreference = "Stop"
Push-Location (Split-Path $PSScriptRoot -Parent)
try {
    $go = Get-Command go -ErrorAction SilentlyContinue
    if ($go) { $goPath = $go.Source } else { $goPath = "$env:ProgramFiles\Go\bin\go.exe" }
    $env:GOCACHE = Join-Path (Get-Location) ".local/go-build"
    $selected = $Tags -split '[, ]+'
    if ($selected -notcontains "core" -and $selected -notcontains "no_audio") {
        & ./scripts/build-monitor.ps1 -OutputDirectory (Split-Path $Output -Parent)
    }
    & $goPath build -trimpath -tags $Tags -ldflags "-H=windowsgui" -o $Output ./cmd/snoofer
    if ($LASTEXITCODE -ne 0) { throw "Build failed. If the executable is running, choose another repository-local output directory." }
} finally { Pop-Location }
