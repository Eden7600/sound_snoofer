param([string]$Output = "bin/sound-snoofer.exe")
$ErrorActionPreference = "Stop"
Push-Location (Split-Path $PSScriptRoot -Parent)
try {
    $go = Get-Command go -ErrorAction SilentlyContinue
    if ($go) { $goPath = $go.Source } else { $goPath = "$env:ProgramFiles\Go\bin\go.exe" }
    & $goPath build -trimpath -ldflags "-H=windowsgui" -o $Output ./cmd/sound-snoofer
    if ($LASTEXITCODE -ne 0) { throw "Build failed. If the executable is running, use -Output with a replacement filename." }
} finally { Pop-Location }
