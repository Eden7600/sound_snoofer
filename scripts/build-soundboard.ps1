$ErrorActionPreference = 'Stop'
$repo = Split-Path $PSScriptRoot -Parent
Push-Location $repo
try {
    $vswhere = Join-Path ${env:ProgramFiles(x86)} 'Microsoft Visual Studio/Installer/vswhere.exe'
    $vs = & $vswhere -latest -products '*' -requires Microsoft.VisualStudio.Component.VC.Tools.x86.x64 -property installationPath
    if (!$vs) { throw 'MSVC and Windows SDK are required to build soundboard.' }
    $vcvars = Join-Path $vs 'VC/Auxiliary/Build/vcvars64.bat'
    New-Item -ItemType Directory -Force '.local/soundboard', 'bin' | Out-Null
    $build = Join-Path $repo '.local/soundboard/build.cmd'
    @"
@echo off
call "$vcvars" >nul
if errorlevel 1 exit /b 1
cl /nologo /W4 /WX /EHsc /O2 /MT /LD /Fo.local\soundboard\ /Febin\snoofer-soundboard.dll internal\voicemeeter\player\player.cpp internal\voicemeeter\player\normalize.cpp /link ole32.lib oleaut32.lib strmiids.lib mfplat.lib mfreadwrite.lib mfuuid.lib /IMPLIB:.local\soundboard\player.lib
"@ | Set-Content -LiteralPath $build
    & $env:ComSpec /d /c $build
    if ($LASTEXITCODE -ne 0) { throw 'Native soundboard build failed.' }
} finally { Pop-Location }

