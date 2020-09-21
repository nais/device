$deviceagent = Get-Process "device-agent" -ErrorAction SilentlyContinue
if ($deviceagent) {
  # try gracefully first
  $deviceagent.CloseMainWindow()
  # kill after five seconds
  Sleep 5
  if (!$deviceagent.HasExited) {
    $deviceagent | Stop-Process -Force
  }
}
Remove-Variable deviceagent
if (Get-Service "naisdevice-agent-helper" -ErrorAction SilentlyContinue) {
    net stop naisdevice-agent-helper
    sc.exe delete naisdevice-agent-helper
}
if (Test-Path "c:\naisdevice") {
    Remove-Item -LiteralPath "c:\naisdevice" -Force -Recurse
}
