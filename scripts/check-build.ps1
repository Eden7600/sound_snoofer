# Exercise the real build scripts with fake tools in a disposable repository.
$ErrorActionPreference = 'Stop'
$sandbox = Join-Path ([IO.Path]::GetTempPath()) ('snoofer-build-check-' + [guid]::NewGuid())
$sandbox = [IO.Path]::GetFullPath($sandbox)
$originalProgramFiles = ${env:ProgramFiles(x86)}
function Assert([bool]$condition, [string]$message) {
    if (!$condition) { throw $message }
}
function Put([string]$path, [string]$content = 'fixture') {
    New-Item -ItemType Directory -Force (Split-Path $path -Parent) | Out-Null
    Set-Content -LiteralPath $path -Value $content
}
function Snapshot([string]$directory) {
    return @(Get-ChildItem -LiteralPath $directory -File -Recurse | Sort-Object FullName | ForEach-Object {
        $_.FullName.Substring($directory.Length) + ':' + (Get-FileHash -LiteralPath $_.FullName).Hash
    }) -join "`n"
}
try {
    & {
        $repo = Join-Path $sandbox 'build'
        New-Item -ItemType Directory -Force "$repo/scripts" | Out-Null
        Copy-Item -LiteralPath "$PSScriptRoot/build.ps1" -Destination "$repo/scripts/build.ps1"
        Put "$repo/scripts/stop.ps1" '$buildCalls.Add("stop")'
        $helper = @'
param([string]$OutputDirectory = 'bin')
$name = [IO.Path]::GetFileNameWithoutExtension($MyInvocation.MyCommand.Name).Substring(6)
$buildCalls.Add($name)
if ($buildFailure -eq $name) { throw "injected $name failure" }
Put "$OutputDirectory/$name.dll" "new $name"
if ($name -eq 'neural-aec') {
    Put "$OutputDirectory/models/model.gguf" 'new model'
    Put "$OutputDirectory/licenses/localvqe/LICENSE" 'new license'
}
'@
        foreach ($name in @('monitor', 'soundboard', 'aec', 'neural-aec', 'media')) {
            Put "$repo/scripts/build-$name.ps1" $helper
        }
        function Get-Command {
            [CmdletBinding()] param([string]$Name)
            Assert ($Name -eq 'go') "Unexpected command lookup: $Name"
            return [pscustomobject]@{ Source = 'FakeGo' }
        }
        function FakeGo {
            $buildCalls.Add('go')
            Assert (($args -join ' ') -like '*-o bin/snoofer.exe*') 'Alternate application output'
            $global:LASTEXITCODE = 1
            if ($buildFailure -eq 'go') { return }
            Put 'bin/snoofer.exe' 'new executable'
            if ($buildFailure -eq 'go-write') { return }
            $global:LASTEXITCODE = 0
        }
        function npm {
            $buildCalls.Add('assets')
            Assert (($args -join ' ') -eq 'run --silent build:gui') 'Missing esbuild/Tailwind build'
            $global:LASTEXITCODE = 1
            if ($buildFailure -eq 'assets') { return }
            Put 'app/web/dist/app.js' 'bundle'
            if ($buildFailure -ne 'missing-css') { Put 'app/web/dist/app.css' 'styles' }
            $global:LASTEXITCODE = 0
        }
        function Copy-Item {
            [CmdletBinding()] param(
                [Parameter(ValueFromPipeline)]$InputObject, [string]$Destination, [switch]$Recurse, [switch]$Force
            )
            process {
                if ($buildFailure -eq 'copy') { throw 'injected copy failure' }
                Microsoft.PowerShell.Management\Copy-Item -LiteralPath $InputObject.FullName -Destination $Destination -Recurse:$Recurse -Force:$Force
            }
        }
        $bin = Join-Path $repo 'bin'
        $buildCalls = [Collections.Generic.List[string]]::new()
        foreach ($buildFailure in @('monitor', 'soundboard', 'aec', 'neural-aec', 'media', 'assets', 'missing-css', 'go', 'go-write', 'copy', '')) {
            Put "$bin/snoofer.exe" 'old executable'
            Put "$bin/monitor.dll" 'old monitor'
            Put "$bin/models/model.gguf" 'old model'
            Put "$bin/licenses/localvqe/LICENSE" 'old license'
            Put "$bin/unrelated.txt" 'keep'
            Put "$repo/.local/stage/stale.dll" 'must not deploy'
            if (Test-Path "$repo/app/web/dist/app.css") { Remove-Item -LiteralPath "$repo/app/web/dist/app.css" }
            $before = Snapshot $bin
            $buildCalls.Clear()
            $output = [Collections.Generic.List[string]]::new()
            $failure = $null
            try { & "$repo/scripts/build.ps1" | ForEach-Object { $output.Add("$_") } } catch { $failure = $_ }
            Assert (($null -ne $failure) -eq ($buildFailure -ne '')) "Unexpected result for failure '$buildFailure'"
            Assert ($buildCalls[0] -eq 'stop') 'Shutdown did not run first'
            Assert (!(Test-Path "$bin/stale.dll")) 'Stale stage was deployed'
            if ($buildFailure -ne '') {
                Assert (!($output -match '^Built:')) 'Failed build reported success'
                if ($buildFailure -in @('go-write', 'copy')) {
                    Assert ($failure.ToString() -like '*incomplete*') 'Deployment failure omitted incomplete-output warning'
                } else { Assert ((Snapshot $bin) -eq $before) "Failure '$buildFailure' changed deployed files" }
            } else {
                Assert (($buildCalls -join ',') -eq 'stop,monitor,soundboard,aec,neural-aec,media,assets,go') 'Build order changed'
                Assert ((Get-Content "$bin/models/model.gguf") -eq 'new model') 'Model was not deployed'
                Assert ((Get-Content "$bin/licenses/localvqe/LICENSE") -eq 'new license') 'License was not deployed'
            }
        }
        foreach ($case in @(
            @{ Tags = 'core'; Calls = 'stop,assets,go' },
            @{ Tags = 'no_audio'; Calls = 'stop,media,assets,go' },
            @{ Tags = 'no_aec,no_soundboard no_nowplaying'; Calls = 'stop,monitor,assets,go' }
        )) {
            $buildCalls.Clear()
            Put "$repo/.local/stage/media.dll" 'stale media'
            $oldMedia = Get-Content "$bin/media.dll"
            & "$repo/scripts/build.ps1" -Tags $case.Tags | Out-Null
            Assert (($buildCalls -join ',') -eq $case.Calls) "Wrong companions for tags '$($case.Tags)'"
            if ($case.Tags -ne 'no_audio') { Assert ((Get-Content "$bin/media.dll") -eq $oldMedia) 'Excluded companion overwritten' }
            Assert ((Get-Content "$bin/unrelated.txt") -eq 'keep') 'Unrelated file overwritten'
        }
    }
    & {
        $repo = Join-Path $sandbox 'neural'
        New-Item -ItemType Directory -Force "$repo/scripts" | Out-Null
        Copy-Item -LiteralPath "$PSScriptRoot/build-neural-aec.ps1" -Destination "$repo/scripts/build-neural-aec.ps1"
        Copy-Item -LiteralPath "$PSScriptRoot/build-soundboard.ps1" -Destination "$repo/scripts/build-soundboard.ps1"
        Copy-Item -LiteralPath "$PSScriptRoot/native-build.ps1" -Destination "$repo/scripts/native-build.ps1"
        ${env:ProgramFiles(x86)} = Join-Path $repo 'Program Files'
        $cache = Join-Path $repo '.local/neural-aec'
        $cmakeDirectory = Join-Path $cache 'cmake-3.31.6-windows-x86_64'
        $cmakePath = Join-Path $cmakeDirectory 'bin/cmake.exe'
        $vswherePath = Join-Path ${env:ProgramFiles(x86)} 'Microsoft Visual Studio/Installer/vswhere.exe'
        Put $vswherePath
        Put "$repo/vs/VC/Auxiliary/Build/vcvars64.bat"
        $revision = 'f53063c9eb2a85f96479867d1dd911dc3bf6319b'
        $ggmlRevision = 'c044a8eeae2591faa0950c8b5e514cbc4bbfc4ca'
        $setup = @{ Head = 'wrong'; GGML = ''; Extracts = 0; Downloads = 0; Clones = 0; Fetches = 0; Checkouts = 0; Submodules = 0; FailExtract = $true }
        function Get-FileHash {
            param([Parameter(Position = 0)][string]$LiteralPath)
            $hash = switch -Wildcard ($LiteralPath) {
                '*cmake.zip*' { 'd163cd3ab4959b0a53fa8988f2ddbd2e6c501658201e6a154386bad9dbe4f836' }
                '*v1.4*' { 'b6e43138588a83bfe903ab5e143b4020b91c1e1629f5a575ac5855ff0003c731' }
                '*v1.3*' { 'c4f7912485c32cfc206c536f2f050b52513f2f613fdbc616391f6b26ab1d51ec' }
                default { throw "Unexpected hash: $LiteralPath" }
            }
            return [pscustomobject]@{ Hash = $hash }
        }
        function Invoke-WebRequest {
            param([string]$Uri, [string]$OutFile)
            $setup.Downloads++
            Put $OutFile 'download'
        }
        function Expand-Archive {
            param([string]$Path, [string]$DestinationPath)
            $setup.Extracts++
            Put "$DestinationPath/cmake-3.31.6-windows-x86_64/bin/cmake.exe"
            if ($setup.FailExtract) { throw 'injected interrupted extraction' }
            Put "$DestinationPath/cmake-3.31.6-windows-x86_64/share/complete.txt"
        }
        function git {
            $global:LASTEXITCODE = 0
            if ($args[0] -eq 'clone') { $setup.Clones++; Put "$($args[-1])/.git/config"; return }
            $target = $args[1]
            switch ($args[2]) {
                'rev-parse' {
                    if ($target -like '*/ggml/vendor/ggml') { return $setup.GGML }
                    return $setup.Head
                }
                'fetch' { $setup.Fetches++; return }
                'checkout' {
                    Assert ($args[-1] -eq $revision) 'Wrong LocalVQE revision'
                    $setup.Checkouts++; $setup.Head = $revision
                    Put "$target/ggml/CMakeLists.txt" '"-ffp-contract=off"'
                    Put "$target/LICENSE"
                }
                'submodule' {
                    $setup.Submodules++; $setup.GGML = $ggmlRevision
                    Put "$target/ggml/vendor/ggml/LICENSE"
                }
                default { throw "Unexpected git invocation: $args" }
            }
        }
        Set-Item -LiteralPath "Function:$cmakePath" -Value {
            $global:LASTEXITCODE = 0
            Put "$cache/build/bin/Release/localvqe.dll"
        }
        Set-Item -LiteralPath "Function:$vswherePath" -Value { $global:LASTEXITCODE = 0; return "$repo/vs" }
        $nativeCommands = [Collections.Generic.List[string]]::new()
        Set-Item -LiteralPath "Function:$env:ComSpec" -Value {
            $nativeCommands.Add((Get-Content -LiteralPath $args[-1] -Raw))
            $global:LASTEXITCODE = 0
        }
        Put $cmakePath 'partial old extraction'
        Put "$cache/LocalVQE/.git/config"
        Put "$repo/bin/models/localvqe-v1.4-aec-200K-f32.gguf" 'previous verified model'
        $failed = $false
        try { & "$repo/scripts/build-neural-aec.ps1" -OutputDirectory '.local/stage' } catch {
            Assert ($_.ToString() -like '*interrupted extraction*') "Unexpected setup failure: $_"
            $failed = $true
        }
        Assert $failed 'Interrupted extraction unexpectedly succeeded'
        Assert (!(Test-Path "$cmakeDirectory/.complete")) 'Partial extraction marked complete'
        $setup.FailExtract = $false
        & "$repo/scripts/build-neural-aec.ps1" -OutputDirectory '.local/stage'
        Assert (Test-Path "$cmakeDirectory/.complete") 'Completed extraction marker missing'
        Assert (Test-Path "$cmakeDirectory/share/complete.txt") 'Old partial extraction reused'
        Assert ($setup.Extracts -eq 2) 'Interrupted extraction was not retried'
        Assert ($setup.Fetches -eq 1 -and $setup.Checkouts -eq 1 -and $setup.Submodules -eq 1) 'Wrong/missing revisions were not repaired'
        Assert ((Get-Content "$cache/LocalVQE/ggml/CMakeLists.txt") -eq '"/fp:strict"') 'MSVC patch lost'
        Assert ($setup.Downloads -eq 2) 'Expected only archive and one uncached model download'
        $setup.GGML = ''
        & "$repo/scripts/build-neural-aec.ps1" -OutputDirectory '.local/second-stage'
        Assert ($setup.Submodules -eq 2) 'Missing GGML checkout was not repaired'
        Assert ($setup.Downloads -eq 2 -and $setup.Extracts -eq 2) 'Verified downloads or extraction were not reused'
        Assert (Test-Path "$repo/.local/second-stage/models/localvqe-v1.4-aec-200K-f32.gguf") 'Cached model did not reach fresh stage'
        Assert (Test-Path "$repo/.local/second-stage/licenses/localvqe/GGML-LICENSE") 'License did not reach fresh stage'
        $gitDirectory = [IO.Path]::GetFullPath((Join-Path $cache 'LocalVQE/.git'))
        Assert ($gitDirectory.StartsWith($sandbox + [IO.Path]::DirectorySeparatorChar)) 'Git fixture outside sandbox'
        Remove-Item -LiteralPath $gitDirectory -Recurse -Force
        $setup.Head = ''; $setup.GGML = ''
        & "$repo/scripts/build-neural-aec.ps1" -OutputDirectory '.local/second-stage'
        Assert ($setup.Clones -eq 1 -and $setup.Submodules -eq 3) 'Incomplete clone was not repaired'
        & "$repo/scripts/build-soundboard.ps1" -OutputDirectory '.local/soundboard-stage'
        $soundboardOutput = [IO.Path]::GetFullPath((Join-Path $repo '.local/soundboard-stage/snoofer-soundboard.dll'))
        Assert ($nativeCommands[-1].Contains('/Fe"' + $soundboardOutput + '"')) 'Soundboard did not use the requested output directory'
    }
    & {
        $repo = Join-Path $sandbox 'native setup with spaces'
        ${env:ProgramFiles(x86)} = Join-Path $repo 'Program Files'
        $vswherePath = Join-Path ${env:ProgramFiles(x86)} 'Microsoft Visual Studio/Installer/vswhere.exe'
        $vs = Join-Path $repo 'Visual Studio'
        $vcvars = Join-Path $vs 'VC/Auxiliary/Build/vcvars64.bat'
        $build = Join-Path $repo 'output with spaces/build.cmd'
        $marker = Join-Path $repo 'output with spaces/compiled.txt'
        Put $vswherePath
        Put $vcvars "@echo off`r`nexit /b 0"
        New-Item -ItemType Directory -Force (Split-Path $build -Parent) | Out-Null
        Set-Item -LiteralPath "Function:$vswherePath" -Value { $global:LASTEXITCODE = 0; return $vs }
        . "$PSScriptRoot/native-build.ps1"
        function ExpectNativeFailure([string[]]$commands, [string]$expected) {
            $failure = $null
            try { Invoke-MsvcBuild -CommandFile $build -Commands $commands -FailureMessage 'Fixture build failed.' } catch { $failure = $_ }
            Assert ($null -ne $failure -and $failure.ToString().Contains($expected)) "Unexpected native result: $failure"
        }
        $body = "echo compiled>`"$marker`""
        Invoke-MsvcBuild -CommandFile $build -Commands $body -FailureMessage 'Fixture build failed.'
        Assert (Test-Path -LiteralPath $marker) 'Paths with spaces did not execute the command body'
        Remove-Item -LiteralPath $marker
        Put $vcvars "@echo off`r`nexit /b 1"
        ExpectNativeFailure $body 'Fixture build failed.'
        Assert (!(Test-Path -LiteralPath $marker)) 'Failed vcvars setup ran the command body'
        Put $vcvars "@echo off`r`nexit /b 0"
        ExpectNativeFailure @('exit /b 7', $body) 'Fixture build failed.'
        Assert (!(Test-Path -LiteralPath $marker)) 'Failed command continued the command body'
        Set-Item -LiteralPath "Function:$vswherePath" -Value { $global:LASTEXITCODE = 1; return $vs }
        ExpectNativeFailure $body 'No installed MSVC x64 toolchain'
        Set-Item -LiteralPath "Function:$vswherePath" -Value { $global:LASTEXITCODE = 0; return $vs }
        Remove-Item -LiteralPath $vcvars
        ExpectNativeFailure $body 'MSVC environment setup is missing'
        Remove-Item -LiteralPath $vswherePath
        ExpectNativeFailure $body 'MSVC Build Tools discovery is missing'
    }

    Write-Output 'Build staging and dependency recovery checks passed (fake tools; no build or network).'
} finally {
    ${env:ProgramFiles(x86)} = $originalProgramFiles
    $tempRoot = [IO.Path]::GetFullPath([IO.Path]::GetTempPath())
    if (!$sandbox.StartsWith($tempRoot, [StringComparison]::OrdinalIgnoreCase) -or
        [IO.Path]::GetFileName($sandbox) -notlike 'snoofer-build-check-*') { throw 'Refusing cleanup outside the test sandbox.' }
    if (Test-Path -LiteralPath $sandbox) { Remove-Item -LiteralPath $sandbox -Recurse -Force }
}
