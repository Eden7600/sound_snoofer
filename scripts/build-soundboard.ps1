param([string]$OutputDirectory = 'bin')
$ErrorActionPreference = 'Stop'
$repo = Split-Path $PSScriptRoot -Parent
. (Join-Path $repo 'scripts/native-build.ps1')
Push-Location $repo
try {
    $out = [IO.Path]::GetFullPath((Join-Path $repo $OutputDirectory))
    New-Item -ItemType Directory -Force '.local/soundboard', $out | Out-Null
    $build = Join-Path $repo '.local/soundboard/build.cmd'
    $commands = @"
cl /nologo /W4 /WX /EHsc /O2 /MT /LD /Fo.local\soundboard\ /Fe"$out\snoofer-soundboard.dll" internal\voicemeeter\player\player.cpp internal\voicemeeter\player\normalize.cpp /link ole32.lib oleaut32.lib strmiids.lib mfplat.lib mfreadwrite.lib mfuuid.lib /IMPLIB:.local\soundboard\player.lib
"@
    Invoke-MsvcBuild -CommandFile $build -Commands $commands -FailureMessage 'Native soundboard build failed.'
} finally { Pop-Location }

