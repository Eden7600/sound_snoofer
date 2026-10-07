$ErrorActionPreference='Stop'
$repo=Split-Path (Split-Path $PSScriptRoot -Parent) -Parent
Push-Location $repo
try {
    New-Item -ItemType Directory -Force .local/fullband-aec-probe | Out-Null
    $vswhere=Join-Path ${env:ProgramFiles(x86)} 'Microsoft Visual Studio/Installer/vswhere.exe'
    $vs=& $vswhere -latest -products '*' -requires Microsoft.VisualStudio.Component.VC.Tools.x86.x64 -property installationPath
    $vcvars=Join-Path $vs 'VC/Auxiliary/Build/vcvars64.bat'
    @"
@echo off
call "$vcvars" >nul
if errorlevel 1 exit /b 1
cl /nologo /W4 /WX /O2 /MT /EHsc /std:c++20 /DNOMINMAX /DWEBRTC_WIN /D_WIN32 /DWEBRTC_LIBRARY_IMPL /D_WINSOCKAPI_ /external:W0 /external:Ithird_party\webrtc-audio-processing\webrtc /external:Ithird_party\abseil-cpp /Fo.local\fullband-aec-probe\probe.obj /Fe.local\fullband-aec-probe\probe.exe tools\fullband-aec-probe\probe.cpp .local\aec\webrtc-apm.lib winmm.lib
"@ | Set-Content .local/fullband-aec-probe/build.cmd
    & $env:ComSpec /d /c (Join-Path $repo '.local/fullband-aec-probe/build.cmd')
    if ($LASTEXITCODE) {throw 'Full-band probe compilation failed'}
    & .local/fullband-aec-probe/probe.exe
    if ($LASTEXITCODE) {throw 'Full-band probe failed'}
} finally {Pop-Location}
