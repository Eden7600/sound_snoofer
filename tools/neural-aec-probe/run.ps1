$ErrorActionPreference = 'Stop'
$repo = Split-Path (Split-Path $PSScriptRoot -Parent) -Parent
Push-Location $repo
try {
    $out = Join-Path $repo '.local/neural-aec-probe'
    New-Item -ItemType Directory -Force $out | Out-Null
    $vswhere = Join-Path ${env:ProgramFiles(x86)} 'Microsoft Visual Studio/Installer/vswhere.exe'
    $vs = & $vswhere -latest -products '*' -requires Microsoft.VisualStudio.Component.VC.Tools.x86.x64 -property installationPath
    $vcvars = Join-Path $vs 'VC/Auxiliary/Build/vcvars64.bat'
    @"
@echo off
call "$vcvars" >nul
if errorlevel 1 exit /b 1
cl /nologo /W4 /WX /O2 /EHsc /std:c++20 /Iinternal\voicemeeter\callback /Fo"$out\probe.obj" /Fe"$out\probe.exe" tools\neural-aec-probe\probe.cpp winmm.lib
"@ | Set-Content "$out/build.cmd"
    & $env:ComSpec /d /c "$out/build.cmd"
    if ($LASTEXITCODE) { throw 'Neural probe compilation failed' }
    # GGML's CPU dispatch searches the executable and current directories.
    Push-Location bin
    try {
        foreach ($model in 'localvqe-v1.4-aec-200K-f32.gguf','localvqe-v1.3-4.8M-f32.gguf') {
            & "$out/probe.exe" ([IO.Path]::GetFullPath("$repo/bin/snoofer-neural-aec.dll")) ([IO.Path]::GetFullPath("$repo/bin/models/$model")) "$repo/.local/neural-aec/LocalVQE/ggml/tests/fixtures/regression_input.f32"
            if ($LASTEXITCODE) { throw "Neural probe failed: $model" }
        }
        & "$out/probe.exe" ([IO.Path]::GetFullPath("$repo/bin/snoofer-neural-aec.dll")) "$repo/bin/models/localvqe-v1.3-4.8M-f32.gguf" "$repo/.local/neural-aec/LocalVQE/ggml/tests/fixtures/regression_input.f32" --fullband
        if ($LASTEXITCODE) { throw "Full-band neural probe failed" }
    } finally { Pop-Location }
} finally { Pop-Location }

