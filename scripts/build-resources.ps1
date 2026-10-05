$ErrorActionPreference = "Stop"
Push-Location (Split-Path $PSScriptRoot -Parent)
try {
    $go = Get-Command go -ErrorAction SilentlyContinue
    if ($go) { $goPath = $go.Source } else { $goPath = "$env:ProgramFiles\Go\bin\go.exe" }
    & $goPath run github.com/akavel/rsrc@v0.10.2 -arch amd64 -ico app/tray.ico -o cmd/snoofer/resource_windows_amd64.syso
    if ($LASTEXITCODE -ne 0) { throw "Windows icon resource generation failed." }
} finally { Pop-Location }
