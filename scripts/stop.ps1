param([int]$ProcessId = 0, [ValidateRange(1,60)][int]$TimeoutSeconds = 15)
$ErrorActionPreference = 'Stop'
$exe = [IO.Path]::GetFullPath((Join-Path (Split-Path $PSScriptRoot -Parent) 'bin/snoofer.exe'))
$running = @(Get-CimInstance Win32_Process -Filter "Name='snoofer.exe'" | Where-Object {
    $_.ExecutablePath -eq $exe
})
if ($ProcessId -ne 0) {
    # An unrelated parent PID must not authorize stopping its Snoofer child.
    if (!($running | Where-Object { $_.ProcessId -eq $ProcessId })) { return }
    $running = @($running | Where-Object { $_.ProcessId -eq $ProcessId -or $_.ParentProcessId -eq $ProcessId })
}
if (!$running.Count) { return }
if (-not ('SnooferStopNative' -as [type])) {
    Add-Type -TypeDefinition @'
using System;
using System.ComponentModel;
using System.Runtime.InteropServices;
using System.Text;
public static class SnooferStopNative {
    delegate bool WindowCallback(IntPtr window, IntPtr data);
    [DllImport("user32.dll")] static extern bool EnumWindows(WindowCallback callback, IntPtr data);
    [DllImport("user32.dll")] static extern uint GetWindowThreadProcessId(IntPtr window, out uint process);
    [DllImport("user32.dll", CharSet=CharSet.Unicode)] static extern int GetClassName(IntPtr window, StringBuilder name, int size);
    [DllImport("user32.dll", SetLastError=true)] static extern bool PostMessage(IntPtr window, uint message, IntPtr w, IntPtr l);
    public static bool CloseTray(int process) {
        IntPtr target = IntPtr.Zero;
        EnumWindows((window, data) => {
            uint owner;
            GetWindowThreadProcessId(window, out owner);
            if (owner != process) return true;
            var name = new StringBuilder(64);
            GetClassName(window, name, name.Capacity);
            if (name.ToString() != "SystrayClass") return true;
            target = window;
            return false;
        }, IntPtr.Zero);
        if (target == IntPtr.Zero) return false;
        if (!PostMessage(target, 0x0010, IntPtr.Zero, IntPtr.Zero))
            throw new Win32Exception(Marshal.GetLastWin32Error());
        return true;
    }
}
'@
}
$targets = @()
$deadline = [DateTime]::UtcNow.AddSeconds($TimeoutSeconds)
try {
    foreach ($entry in $running) {
        $process = Get-Process -Id $entry.ProcessId -ErrorAction SilentlyContinue
        if (!$process) { continue }
        # Hold the handle so exit/PID reuse cannot redirect the wait.
        $null = $process.Handle
        $targets += $process
        if ($entry.CommandLine -match '\b__controls\b') { continue }
        while (!$process.HasExited -and ![SnooferStopNative]::CloseTray($process.Id)) {
            if ([DateTime]::UtcNow -ge $deadline) {
                throw "No responsive Snoofer tray for PID $($process.Id). No process was killed."
            }
            Start-Sleep -Milliseconds 100
        }
    }
    foreach ($process in $targets) {
        $remaining = [Math]::Max(0, [int]($deadline - [DateTime]::UtcNow).TotalMilliseconds)
        if (!$process.WaitForExit($remaining)) {
            throw "Snoofer PID $($process.Id) did not finish cleanup. Build aborted; no process was killed."
        }
        if ($process.ExitCode -ne 0) {
            throw "Snoofer PID $($process.Id) exited with code $($process.ExitCode); cleanup was not confirmed."
        }
        Write-Output "Snoofer PID $($process.Id) exited normally."
    }
} finally {
    foreach ($process in $targets) { $process.Dispose() }
}
