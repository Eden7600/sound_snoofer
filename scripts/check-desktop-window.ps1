param([int]$ProcessId, [switch]$CloseOnly)
$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName UIAutomationClient
Add-Type -AssemblyName UIAutomationTypes
$window = $null
for ($attempt=0; $attempt -lt 40; $attempt++) {
    $process = Get-Process -Id $ProcessId
    if ($process.MainWindowHandle -ne 0) {
        $window = [System.Windows.Automation.AutomationElement]::FromHandle($process.MainWindowHandle)
        break
    }
    Start-Sleep -Milliseconds 200
}
if (!$window) { throw 'No native controls window' }
if ($CloseOnly) {
    $window.GetCurrentPattern([System.Windows.Automation.WindowPattern]::Pattern).Close()
    Write-Output 'PASS: native window close'
    exit 0
}
function Find-Control([string]$Name) {
    $condition = [System.Windows.Automation.AndCondition]::new(
        [System.Windows.Automation.PropertyCondition]::new([System.Windows.Automation.AutomationElement]::NameProperty,$Name),
        [System.Windows.Automation.PropertyCondition]::new([System.Windows.Automation.AutomationElement]::ControlTypeProperty,[System.Windows.Automation.ControlType]::Button))
    for ($attempt=0; $attempt -lt 30; $attempt++) {
        $element = $window.FindFirst([System.Windows.Automation.TreeScope]::Descendants,$condition)
        if (!$element) {
            $buttons = $window.FindAll([System.Windows.Automation.TreeScope]::Descendants,[System.Windows.Automation.PropertyCondition]::new([System.Windows.Automation.AutomationElement]::ControlTypeProperty,[System.Windows.Automation.ControlType]::Button))
            $element = $buttons | Where-Object { $_.Current.Name.EndsWith(' ' + $Name) } | Select-Object -First 1
        }
        if ($element) { return $element }
        Start-Sleep -Milliseconds 200
    }
    $names=$window.FindAll([System.Windows.Automation.TreeScope]::Descendants,[System.Windows.Automation.Condition]::TrueCondition) | ForEach-Object { $_.Current.Name }
    throw "Missing accessible control: $Name; found: $($names -join ', ')"
}
$lights = Find-Control 'Lights'
$invoke = $lights.GetCurrentPattern([System.Windows.Automation.InvokePattern]::Pattern)
$invoke.Invoke()
$sync = Find-Control 'Start sync'
$toggle = $sync.GetCurrentPattern([System.Windows.Automation.TogglePattern]::Pattern)
$toggle.Toggle()
Write-Output 'PASS: native window and accessible Lights / Start sync controls'

