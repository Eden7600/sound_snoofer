param([switch]$Observe, [switch]$Paired)
$ErrorActionPreference = 'Stop'
$repo = Split-Path (Split-Path $PSScriptRoot -Parent) -Parent
. (Join-Path $repo 'scripts/native-build.ps1')
Push-Location $repo
try {
    $out = Join-Path $repo '.local/stall-probe'
    New-Item -ItemType Directory -Force $out | Out-Null
    $build = Join-Path $out 'build-callback.cmd'
    $commands = @"
cl /nologo /W4 /WX /O2 /std:c11 /Fo.local\stall-probe\callback-probe.obj /Fe.local\stall-probe\callback-probe.exe tools\callback-probe\probe.c
if errorlevel 1 exit /b 1
cl /nologo /W4 /WX /O2 /std:c11 /Fo.local\stall-probe\stop-test.obj /Fe.local\stall-probe\stop-test.exe tools\callback-probe\stop_test.c
if errorlevel 1 exit /b 1
cl /nologo /W4 /WX /O2 /TC /Fo.local\stall-probe\buffer-c.obj /Fe.local\stall-probe\buffer-c.exe tools\callback-probe\buffer_test.c
if errorlevel 1 exit /b 1
cl /nologo /W4 /WX /O2 /TP /std:c++17 /Fo.local\stall-probe\buffer-cpp.obj /Fe.local\stall-probe\buffer-cpp.exe tools\callback-probe\buffer_test.c
"@
    Invoke-MsvcBuild -CommandFile $build -Commands $commands -FailureMessage 'Probe compilation failed. Check MSVC and Windows SDK installation.'
    $exe = Join-Path $out 'callback-probe.exe'
    & $exe --self-test
    if ($LASTEXITCODE -ne 0) { throw 'Callback probe self-test failed; live capture refused.' }
    & (Join-Path $out 'stop-test.exe')
    if ($LASTEXITCODE -ne 0) { throw 'Callback stop self-test failed; live capture refused.' }
    foreach ($test in 'buffer-c.exe', 'buffer-cpp.exe') {
        & (Join-Path $out $test)
        if ($LASTEXITCODE -ne 0) { throw 'Audio buffer self-test failed; live capture refused.' }
    }
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
