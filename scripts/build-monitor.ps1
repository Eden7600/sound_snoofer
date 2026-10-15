param([string]$OutputDirectory = 'bin')
$ErrorActionPreference = 'Stop'
$repo = Split-Path $PSScriptRoot -Parent
. (Join-Path $repo 'scripts/native-build.ps1')
Push-Location $repo
try {
    # Reuse the probe build and bit-preserving callback self-test before shipping it.
    & ./tools/callback-probe/run.ps1
    $out = [IO.Path]::GetFullPath((Join-Path $repo $OutputDirectory))
    New-Item -ItemType Directory -Force $out | Out-Null
    $build = Join-Path $repo '.local/stall-probe/build-monitor.cmd'
    $commands = @"
cl /nologo /W4 /WX /O2 /MT /LD /Fo.local\stall-probe\monitor.obj /Fe"$out\snoofer-audio-monitor.dll" internal\voicemeeter\callback\monitor.c /link /IMPLIB:.local\stall-probe\monitor.lib
"@
    Invoke-MsvcBuild -CommandFile $build -Commands $commands -FailureMessage 'Native monitor build failed; do not replace an in-use DLL.'
} finally { Pop-Location }
