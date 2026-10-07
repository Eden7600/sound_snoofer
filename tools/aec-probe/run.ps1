param([string]$Dll = 'bin/snoofer-aec.dll')
$ErrorActionPreference = 'Stop'
$repo = Split-Path (Split-Path $PSScriptRoot -Parent) -Parent
Push-Location $repo
try {
    if (!(Test-Path $Dll)) { throw "$Dll not found; build it with scripts/build-aec.ps1 first." }
    $out = Join-Path $repo '.local/aec-probe'
    New-Item -ItemType Directory -Force $out | Out-Null
    $vswhere = Join-Path ${env:ProgramFiles(x86)} 'Microsoft Visual Studio/Installer/vswhere.exe'
    if (!(Test-Path $vswhere)) { throw 'MSVC Build Tools with a Windows SDK are required.' }
    $vs = & $vswhere -latest -products '*' -requires Microsoft.VisualStudio.Component.VC.Tools.x86.x64 -property installationPath
    if (!$vs) { throw 'No installed MSVC x64 toolchain found.' }
    $vcvars = Join-Path $vs 'VC/Auxiliary/Build/vcvars64.bat'
    $build = Join-Path $out 'build.cmd'
    @"
@echo off
call "$vcvars" >nul
if errorlevel 1 exit /b 1
cl /nologo /W4 /WX /O2 /std:c11 /Iinternal\voicemeeter\callback /Fo.local\aec-probe\probe.obj /Fe.local\aec-probe\aec-probe.exe tools\aec-probe\probe.c
"@ | Set-Content -LiteralPath $build
    & $env:ComSpec /d /c $build
    if ($LASTEXITCODE -ne 0) { throw 'Probe compilation failed.' }
    & (Join-Path $out 'aec-probe.exe') ([IO.Path]::GetFullPath($Dll))
    if ($LASTEXITCODE -ne 0) { throw 'Echo cancellation probe failed.' }
} finally {
    Pop-Location
}
