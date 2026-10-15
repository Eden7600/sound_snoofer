param([string]$Tags = "")
$ErrorActionPreference = "Stop"
$repo = Split-Path $PSScriptRoot -Parent
Push-Location $repo
$previousCache = $env:GOCACHE
try {
    & ./scripts/stop.ps1
    # Check before compiling the companion or touching either deployed binary.
    $companions = @(Get-ChildItem bin -Filter *.dll -ErrorAction SilentlyContinue | ForEach-Object Name)
    foreach ($name in ($companions + @("snoofer.exe", "snoofer-audio-monitor.dll", "snoofer-soundboard.dll", "snoofer-media.dll", "snoofer-aec.dll"))) {
        $path = Join-Path (Get-Location) "bin/$name"
        if (Test-Path -LiteralPath $path) {
            try {
                $probe = [IO.File]::Open($path, [IO.FileMode]::Open, [IO.FileAccess]::ReadWrite, [IO.FileShare]::None)
                $probe.Dispose()
            } catch {
                throw "Cannot replace bin/$name. Graceful shutdown did not release the file; inspect Snoofer before retrying. No alternate output will be created."
            }
        }
    }
    $go = Get-Command go -ErrorAction SilentlyContinue
    if ($go) { $goPath = $go.Source } else { $goPath = "$env:ProgramFiles\Go\bin\go.exe" }
    $env:GOCACHE = Join-Path (Get-Location) ".local/go-build"
    New-Item -ItemType Directory -Force bin | Out-Null
    $stage = [IO.Path]::GetFullPath((Join-Path $repo '.local/stage'))
    if (!$stage.StartsWith($repo + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase)) {
        throw 'Build stage must stay inside the repository.'
    }
    if (Test-Path -LiteralPath $stage) { Remove-Item -LiteralPath $stage -Recurse -Force }
    New-Item -ItemType Directory -Path $stage | Out-Null
    $selected = $Tags -split '[, ]+'
    if ($selected -notcontains "core" -and $selected -notcontains "no_audio") {
        & ./scripts/build-monitor.ps1 -OutputDirectory '.local/stage'
        if ($selected -notcontains "no_soundboard") { & ./scripts/build-soundboard.ps1 -OutputDirectory '.local/stage' }
        if ($selected -notcontains "no_aec") {
            & ./scripts/build-aec.ps1 -OutputDirectory '.local/stage'
            & ./scripts/build-neural-aec.ps1 -OutputDirectory '.local/stage'
        }
    }
    if ($selected -notcontains "core" -and $selected -notcontains "no_nowplaying") { & ./scripts/build-media.ps1 -OutputDirectory '.local/stage' }
    # Go embeds the generated JavaScript bundle and Tailwind stylesheet.
    & npm run --silent build:gui
    if ($LASTEXITCODE -ne 0) { throw "GUI asset generation failed; run npm install first." }
    foreach ($assetPath in @("app/web/dist/app.js", "app/web/dist/app.css")) {
        $asset = Get-Item -LiteralPath $assetPath -ErrorAction SilentlyContinue
        if (-not $asset -or $asset.Length -eq 0) { throw "$assetPath is missing or empty; the GUI cannot be packaged." }
    }
    $arguments = @("build", "-trimpath", "-ldflags", "-H=windowsgui", "-o", "bin/snoofer.exe")
    $arguments += @("-tags", ("production " + $Tags).Trim())
    & $goPath @arguments ./cmd/snoofer
    if ($LASTEXITCODE -ne 0) {
        throw 'Go build failed; staged companions were not deployed. If writing bin/snoofer.exe failed, deployed output may be incomplete. Rebuild successfully before launching.'
    }
    # shortcut: deployment copies are not transactional; add rollback if I/O failures require automatic recovery.
    try {
        Get-ChildItem -LiteralPath $stage | Copy-Item -Destination (Join-Path $repo 'bin') -Recurse -Force
    } catch {
        throw "Companion deployment failed; bin/ may be incomplete. Rebuild successfully before launching. $($_.Exception.Message)"
    }
    Write-Output "Built: bin/snoofer.exe"
} finally {
    $env:GOCACHE = $previousCache
    Pop-Location
}
