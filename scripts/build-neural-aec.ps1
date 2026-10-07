param([string]$OutputDirectory = 'bin')
$ErrorActionPreference = 'Stop'
$repo = Split-Path $PSScriptRoot -Parent
Push-Location $repo
try {
    $cache = Join-Path $repo '.local/neural-aec'
    $out = [IO.Path]::GetFullPath((Join-Path $repo $OutputDirectory))
    New-Item -ItemType Directory -Force $cache, "$out/models", "$out/licenses/localvqe" | Out-Null
    function Fetch([string]$url, [string]$path, [string]$sha) {
        if (!(Test-Path $path) -or (Get-FileHash $path).Hash -ne $sha) {
            Invoke-WebRequest $url -OutFile "$path.download"
            if ((Get-FileHash "$path.download").Hash -ne $sha) { throw "Hash mismatch: $url" }
            Move-Item -LiteralPath "$path.download" -Destination $path -Force
        }
    }
    $cmake = Join-Path $cache 'cmake-3.31.6-windows-x86_64/bin/cmake.exe'
    if (!(Test-Path $cmake)) {
        Fetch 'https://github.com/Kitware/CMake/releases/download/v3.31.6/cmake-3.31.6-windows-x86_64.zip' "$cache/cmake.zip" 'd163cd3ab4959b0a53fa8988f2ddbd2e6c501658201e6a154386bad9dbe4f836'
        Expand-Archive "$cache/cmake.zip" -DestinationPath $cache -Force
    }
    $src = Join-Path $cache 'LocalVQE'
    $revision = 'f53063c9eb2a85f96479867d1dd911dc3bf6319b'
    if (!(Test-Path "$src/.git")) {
        git clone --no-checkout https://github.com/localai-org/LocalVQE.git $src
        if ($LASTEXITCODE) { throw 'LocalVQE clone failed' }
        git -C $src checkout --detach $revision
        if ($LASTEXITCODE) { throw 'LocalVQE checkout failed' }
        git -C $src submodule update --init --recursive
        if ($LASTEXITCODE) { throw 'GGML checkout failed' }
    }
    if ((git -C $src rev-parse HEAD) -ne $revision) { throw 'Unexpected LocalVQE revision' }
    if ((git -C "$src/ggml/vendor/ggml" rev-parse HEAD) -ne 'c044a8eeae2591faa0950c8b5e514cbc4bbfc4ca') { throw 'Unexpected GGML revision' }
    # Upstream spells this GCC-only flag unconditionally; use the MSVC equivalent.
    $cmakeFile = "$src/ggml/CMakeLists.txt"
    $content = [IO.File]::ReadAllText($cmakeFile)
    if ($content.Contains('"-ffp-contract=off"')) {
        [IO.File]::WriteAllText($cmakeFile, $content.Replace('"-ffp-contract=off"', '"/fp:strict"'))
    }
    & $cmake -S "$src/ggml" -B "$cache/build" -G 'Visual Studio 17 2022' -A x64 -DLOCALVQE_BUILD_SHARED=ON -DGGML_OPENMP=OFF
    if ($LASTEXITCODE) { throw 'LocalVQE configure failed' }
    & $cmake --build "$cache/build" --config Release --target localvqe_shared -j 8
    if ($LASTEXITCODE) { throw 'LocalVQE build failed' }
    Copy-Item "$cache/build/bin/Release/*.dll" $out -Force
    Copy-Item "$src/LICENSE" "$out/licenses/localvqe/LICENSE" -Force
    Copy-Item "$src/ggml/vendor/ggml/LICENSE" "$out/licenses/localvqe/GGML-LICENSE" -Force
    $models = @{
        'localvqe-v1.4-aec-200K-f32.gguf' = 'b6e43138588a83bfe903ab5e143b4020b91c1e1629f5a575ac5855ff0003c731'
        'localvqe-v1.3-4.8M-f32.gguf' = 'c4f7912485c32cfc206c536f2f050b52513f2f613fdbc616391f6b26ab1d51ec'
    }
    foreach ($name in $models.Keys) { Fetch "https://huggingface.co/LocalAI-io/LocalVQE/resolve/main/$name" "$out/models/$name" $models[$name] }
    $vswhere = Join-Path ${env:ProgramFiles(x86)} 'Microsoft Visual Studio/Installer/vswhere.exe'
    $vs = & $vswhere -latest -products '*' -requires Microsoft.VisualStudio.Component.VC.Tools.x86.x64 -property installationPath
    if (!$vs) { throw 'MSVC and Windows SDK required' }
    $vcvars = Join-Path $vs 'VC/Auxiliary/Build/vcvars64.bat'
    $lines = @('@echo off', "call `"$vcvars`" >nul", 'if errorlevel 1 exit /b 1')
    $lines += "cl /nologo /c /O2 /MT /EHsc /std:c++20 /utf-8 /W4 /WX /DNOMINMAX /DWEBRTC_WIN /D_WIN32 /DWEBRTC_LIBRARY_IMPL /D_WINSOCKAPI_ /DNDEBUG /external:W0 /external:Ithird_party\webrtc-audio-processing\webrtc /external:Ithird_party\abseil-cpp /Iinternal\voicemeeter\callback /I`"$src\ggml`" /Fo`"$cache\neural.obj`" internal\aec\native\neural.cpp || exit /b 1"
    $lines += "link /nologo /DLL /OUT:`"$out\snoofer-neural-aec.dll`" /IMPLIB:`"$cache\snoofer-neural-aec.lib`" `"$cache\neural.obj`" `"$cache\build\Release\localvqe.lib`" .local\aec\webrtc-apm.lib winmm.lib || exit /b 1"
    Set-Content "$cache/build-shim.cmd" $lines
    & $env:ComSpec /d /c "$cache/build-shim.cmd"
    if ($LASTEXITCODE) { throw 'Neural AEC bridge build failed' }
} finally { Pop-Location }
