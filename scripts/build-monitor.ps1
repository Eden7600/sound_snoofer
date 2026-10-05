param([string]$OutputDirectory = 'bin')
$ErrorActionPreference = 'Stop'
$repo = Split-Path $PSScriptRoot -Parent
Push-Location $repo
try {
    # Reuse the probe build and bit-preserving callback self-test before shipping it.
    & ./tools/callback-probe/run.ps1
    $out = [IO.Path]::GetFullPath((Join-Path $repo $OutputDirectory))
    New-Item -ItemType Directory -Force $out | Out-Null
    $vswhere = Join-Path ${env:ProgramFiles(x86)} 'Microsoft Visual Studio/Installer/vswhere.exe'
    $vs = & $vswhere -latest -products '*' -requires Microsoft.VisualStudio.Component.VC.Tools.x86.x64 -property installationPath
    if (!$vs) { throw 'MSVC and Windows SDK are required to build the audio monitor.' }
    $vcvars = Join-Path $vs 'VC/Auxiliary/Build/vcvars64.bat'
    $build = Join-Path $repo '.local/stall-probe/build-monitor.cmd'
    @"
@echo off
call "$vcvars" >nul
if errorlevel 1 exit /b 1
cl /nologo /W4 /WX /O2 /MT /LD /Fo.local\stall-probe\monitor.obj /Fe"$out\snoofer-audio-monitor.dll" internal\voicemeeter\callback\monitor.c /link /IMPLIB:.local\stall-probe\monitor.lib
"@ | Set-Content -LiteralPath $build
    & $env:ComSpec /d /c $build
    if ($LASTEXITCODE -ne 0) { throw 'Native monitor build failed; do not replace an in-use DLL.' }
} finally { Pop-Location }
