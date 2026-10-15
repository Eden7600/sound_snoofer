# Shared MSVC setup; callers own their working directory, commands and outputs.
function Invoke-MsvcBuild {
    param(
        [Parameter(Mandatory)][string]$CommandFile,
        [Parameter(Mandatory)][string[]]$Commands,
        [Parameter(Mandatory)][string]$FailureMessage
    )
    $vswhere = Join-Path ${env:ProgramFiles(x86)} 'Microsoft Visual Studio/Installer/vswhere.exe'
    if (!(Test-Path -LiteralPath $vswhere)) { throw "MSVC Build Tools discovery is missing. $FailureMessage" }
    $vs = & $vswhere -latest -products '*' -requires Microsoft.VisualStudio.Component.VC.Tools.x86.x64 -property installationPath
    if ($LASTEXITCODE -ne 0 -or !$vs) { throw "No installed MSVC x64 toolchain found. $FailureMessage" }
    $vcvars = Join-Path $vs 'VC/Auxiliary/Build/vcvars64.bat'
    if (!(Test-Path -LiteralPath $vcvars)) { throw "MSVC environment setup is missing: $vcvars. $FailureMessage" }
    $lines = @('@echo off', "call `"$vcvars`" >nul", 'if errorlevel 1 exit /b 1') + $Commands
    Set-Content -LiteralPath $CommandFile -Value $lines
    & $env:ComSpec /d /c $CommandFile
    if ($LASTEXITCODE -ne 0) { throw $FailureMessage }
}
