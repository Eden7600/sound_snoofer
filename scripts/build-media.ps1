param([string]$OutputDirectory = 'bin')
$ErrorActionPreference = 'Stop'
$repo = Split-Path $PSScriptRoot -Parent
Push-Location $repo
try {
    $vswhere = Join-Path ${env:ProgramFiles(x86)} 'Microsoft Visual Studio/Installer/vswhere.exe'
    $vs = & $vswhere -latest -products '*' -requires Microsoft.VisualStudio.Component.VC.Tools.x86.x64 -property installationPath
    if (!$vs) { throw 'MSVC and the Windows SDK (C++/WinRT) are required to build media sessions.' }
    $vcvars = Join-Path $vs 'VC/Auxiliary/Build/vcvars64.bat'
    $out = [IO.Path]::GetFullPath((Join-Path $repo $OutputDirectory))
    New-Item -ItemType Directory -Force '.local/media', $out | Out-Null
    $build = Join-Path $repo '.local/media/build.cmd'
    @"
@echo off
call "$vcvars" >nul
if errorlevel 1 exit /b 1
cl /nologo /W4 /WX /EHsc /O2 /MT /LD /std:c++17 /permissive- /bigobj /Fo.local\media\ /Fe"$out\snoofer-media.dll" internal\mediasessions\native\media.cpp /link windowsapp.lib /IMPLIB:.local\media\media.lib
"@ | Set-Content -LiteralPath $build
    & $env:ComSpec /d /c $build
    if ($LASTEXITCODE -ne 0) { throw 'Native media sessions build failed; do not replace an in-use DLL.' }
} finally { Pop-Location }
