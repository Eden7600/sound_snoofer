param([string]$OutputDirectory = 'bin')
$ErrorActionPreference = 'Stop'
$repo = Split-Path $PSScriptRoot -Parent
Push-Location $repo
try {
    $vswhere = Join-Path ${env:ProgramFiles(x86)} 'Microsoft Visual Studio/Installer/vswhere.exe'
    $vs = & $vswhere -latest -products '*' -requires Microsoft.VisualStudio.Component.VC.Tools.x86.x64 -property installationPath
    if (!$vs) { throw 'MSVC and the Windows SDK are required to build camera controls.' }
    $vcvars = Join-Path $vs 'VC/Auxiliary/Build/vcvars64.bat'
    $out = [IO.Path]::GetFullPath((Join-Path $repo $OutputDirectory))
    New-Item -ItemType Directory -Force '.local/camera', $out | Out-Null
    $build = Join-Path $repo '.local/camera/build.cmd'
    @"
@echo off
call "$vcvars" >nul
if errorlevel 1 exit /b 1
cl /nologo /W4 /WX /EHsc /O2 /MT /LD /std:c++17 /permissive- /Fo.local\camera\ /Fe"$out\snoofer-camera.dll" internal\camera\native\camera.cpp /link strmiids.lib ole32.lib oleaut32.lib /IMPLIB:.local\camera\camera.lib
"@ | Set-Content -LiteralPath $build
    & $env:ComSpec /d /c $build
    if ($LASTEXITCODE -ne 0) { throw 'Native camera build failed; do not replace an in-use DLL.' }
} finally { Pop-Location }
