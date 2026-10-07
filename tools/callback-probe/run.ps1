param([switch]$Observe, [switch]$Paired)
$ErrorActionPreference = 'Stop'
$repo = Split-Path (Split-Path $PSScriptRoot -Parent) -Parent
Push-Location $repo
try {
    $out = Join-Path $repo '.local/stall-probe'
    New-Item -ItemType Directory -Force $out | Out-Null
    $vswhere = Join-Path ${env:ProgramFiles(x86)} 'Microsoft Visual Studio/Installer/vswhere.exe'
    if (!(Test-Path $vswhere)) { throw 'MSVC Build Tools with a Windows SDK are required.' }
    $vs = & $vswhere -latest -products '*' -requires Microsoft.VisualStudio.Component.VC.Tools.x86.x64 -property installationPath
    if (!$vs) { throw 'No installed MSVC x64 toolchain found.' }
    $vcvars = Join-Path $vs 'VC/Auxiliary/Build/vcvars64.bat'
    $build = Join-Path $out 'build-callback.cmd'
    @"
@echo off
call "$vcvars" >nul
if errorlevel 1 exit /b 1
cl /nologo /W4 /WX /O2 /std:c11 /Fo.local\stall-probe\callback-probe.obj /Fe.local\stall-probe\callback-probe.exe tools\callback-probe\probe.c
"@ | Set-Content -LiteralPath $build
    & $env:ComSpec /d /c $build
    if ($LASTEXITCODE -ne 0) { throw 'Probe compilation failed. Check MSVC and Windows SDK installation.' }
    $exe = Join-Path $out 'callback-probe.exe'
    & $exe --self-test
    if ($LASTEXITCODE -ne 0) { throw 'Callback probe self-test failed; live capture refused.' }
    if ($Observe -or $Paired) {
        $uninstall = Get-ItemProperty 'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\VB:Voicemeeter {17359A74-1236-5467}' -ErrorAction SilentlyContinue
        # Same installation discovery source as the Go adapter.
        if (!$uninstall -or !$uninstall.UninstallString) { throw 'Installed Voicemeeter path not found.' }
        $uninstaller = $uninstall.UninstallString.Trim()
        if ($uninstaller -match '^"([^"]+)"') { $uninstaller = $Matches[1] }
        if (![IO.Path]::IsPathRooted($uninstaller) -or [IO.Path]::GetExtension($uninstaller) -ne '.exe') {
            throw 'Invalid Voicemeeter installation path.'
        }
        $dll = Join-Path (Split-Path $uninstaller -Parent) 'VoicemeeterRemote64.dll'
        if (!(Test-Path $dll)) { throw "Installed Remote64 DLL not found at $dll" }
        $log = Join-Path $out ('callback-' + (Get-Date -Format 'yyyyMMdd-HHmmss-fff') + '.log')
        "Capture start UTC: $([DateTime]::UtcNow.ToString('o'))" | Set-Content $log
        $mode = if ($Paired) { '--observe-paired' } else { '--observe' }
        & $exe $mode $dll 2>&1 | Tee-Object -FilePath $log -Append
        if ($LASTEXITCODE -ne 0) { throw "Probe failed; inspect $log" }
        Write-Host "Capture: $log"
    }
} finally {
    Pop-Location
}
