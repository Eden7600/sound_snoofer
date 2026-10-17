# Exercise shutdown selection without enumerating processes or calling user32.
$ErrorActionPreference = 'Stop'
if ('SnooferStopNative' -as [type]) { throw 'Run this isolated check in a fresh PowerShell process.' }
Add-Type -TypeDefinition @'
using System;
using System.Collections.Generic;
public static class SnooferStopNative {
    public static readonly List<int> Closed = new List<int>();
    public static Func<int, bool> OnClose;
    public static bool CloseTray(int process) {
        Closed.Add(process);
        return OnClose(process);
    }
}
'@

function Assert([bool]$condition, [string]$message) {
    if (!$condition) { throw $message }
}
function New-FakeProcess([int]$id) {
    $process = [pscustomobject]@{
        Id = $id; Handle = [IntPtr]1; HasExited = $false; ExitCode = 0
        Waited = $false; Disposed = $false
    }
    $process | Add-Member -MemberType ScriptMethod -Name WaitForExit -Value {
        param([int]$milliseconds)
        $this.Waited = $true
        return $this.HasExited
    }
    $process | Add-Member -MemberType ScriptMethod -Name Dispose -Value { $this.Disposed = $true }
    return $process
}
function Get-CimInstance {
    [CmdletBinding()] param([string]$ClassName, [string]$Filter)
    Assert ($ClassName -eq 'Win32_Process' -and $Filter -eq "Name='snoofer.exe'") 'Unexpected process query'
    return $fakeEntries
}
function Get-Process {
    [CmdletBinding()] param([int]$Id)
    Assert ($fakeProcesses.ContainsKey($Id)) 'Unexpected process handle lookup'
    return $fakeProcesses[$Id]
}

$stopScript = Join-Path $PSScriptRoot 'stop.ps1'
$exe = [IO.Path]::GetFullPath((Join-Path (Split-Path $PSScriptRoot -Parent) 'bin/snoofer.exe'))
foreach ($order in @('child-first', 'host-first')) {
    $fakeProcesses = @{ 10 = (New-FakeProcess 10); 11 = (New-FakeProcess 11) }
    # The ordinary host's config name must not be mistaken for a worker command.
    $parent = [pscustomobject]@{
        ProcessId = 10; ParentProcessId = 1; ExecutablePath = $exe
        CommandLine = '"' + $exe + '" --config "C:\profiles\vrclipboard worker"'
    }
    $child = [pscustomobject]@{
        ProcessId = 11; ParentProcessId = 10; ExecutablePath = $exe
        CommandLine = '"' + $exe + '" --config "C:\profiles\exact profile.json" vrclipboard worker'
    }
    $fakeEntries = if ($order -eq 'child-first') { @($child, $parent) } else { @($parent, $child) }
    [SnooferStopNative]::Closed.Clear()
    [SnooferStopNative]::OnClose = {
        param([int]$id)
        if ($id -ne 10) { return $false }
        foreach ($process in $fakeProcesses.Values) { $process.HasExited = $true }
        return $true
    }
    & $stopScript -TimeoutSeconds 1
    Assert (([SnooferStopNative]::Closed -join ',') -eq '10') "$order tried to close a child tray"
    foreach ($process in $fakeProcesses.Values) {
        Assert $process.Waited "$order did not wait for PID $($process.Id)"
        Assert $process.Disposed "$order leaked PID $($process.Id)'s held handle"
    }
}
Write-Output 'PASS: child-first and host-first shutdown closes only the parent and joins both processes.'
