param([string]$Tags = "")
$ErrorActionPreference = "Stop"
Push-Location (Split-Path $PSScriptRoot -Parent)
$previousCache = $env:GOCACHE
try {
    # Check before compiling the companion or touching either deployed binary.
    foreach ($name in @("snoofer.exe", "snoofer-audio-monitor.dll", "snoofer-soundboard.dll")) {
        $path = Join-Path (Get-Location) "bin/$name"
        if (Test-Path -LiteralPath $path) {
            try {
                $probe = [IO.File]::Open($path, [IO.FileMode]::Open, [IO.FileAccess]::ReadWrite, [IO.FileShare]::None)
                $probe.Dispose()
            } catch {
                throw "Cannot replace bin/$name. Exit Snoofer from the tray, then run scripts/build.ps1 again. No alternate output will be created."
            }
        }
    }
    $go = Get-Command go -ErrorAction SilentlyContinue
    if ($go) { $goPath = $go.Source } else { $goPath = "$env:ProgramFiles\Go\bin\go.exe" }
    $env:GOCACHE = Join-Path (Get-Location) ".local/go-build"
    New-Item -ItemType Directory -Force bin | Out-Null
    $selected = $Tags -split '[, ]+'
    if ($selected -notcontains "core" -and $selected -notcontains "no_audio") {
        & ./scripts/build-monitor.ps1 -OutputDirectory bin
        if ($selected -notcontains "no_soundboard") { & ./scripts/build-soundboard.ps1 }
    }
    $arguments = @("build", "-trimpath", "-ldflags", "-H=windowsgui", "-o", "bin/snoofer.exe")
    if ($Tags) { $arguments += @("-tags", $Tags) }
    & $goPath @arguments ./cmd/snoofer
    if ($LASTEXITCODE -ne 0) { throw "Build failed. The only app output is bin/snoofer.exe." }
    Write-Output "Built: bin/snoofer.exe"
} finally {
    $env:GOCACHE = $previousCache
    Pop-Location
}
