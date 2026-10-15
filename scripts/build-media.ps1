param([string]$OutputDirectory = 'bin')
$ErrorActionPreference = 'Stop'
$repo = Split-Path $PSScriptRoot -Parent
. (Join-Path $repo 'scripts/native-build.ps1')
Push-Location $repo
try {
    $out = [IO.Path]::GetFullPath((Join-Path $repo $OutputDirectory))
    New-Item -ItemType Directory -Force '.local/media', $out | Out-Null
    $build = Join-Path $repo '.local/media/build.cmd'
    $commands = @"
cl /nologo /W4 /WX /EHsc /O2 /MT /std:c++17 /permissive- /bigobj /Fo.local\media\ /Fe.local\media\async-wait-test.exe internal\mediasessions\native\async_wait_test.cpp /link windowsapp.lib
if errorlevel 1 exit /b 1
.local\media\async-wait-test.exe
if errorlevel 1 exit /b 1
cl /nologo /W4 /WX /EHsc /O2 /MT /LD /std:c++17 /permissive- /bigobj /Fo.local\media\ /Fe"$out\snoofer-media.dll" internal\mediasessions\native\media.cpp /link windowsapp.lib /IMPLIB:.local\media\media.lib
"@
    Invoke-MsvcBuild -CommandFile $build -Commands $commands -FailureMessage 'Native media sessions build or self-test failed; do not replace an in-use DLL.'
} finally { Pop-Location }
