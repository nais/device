{
  naisdevice,
  runCommand,
}:
runCommand "test-naisdevice-agent" {inherit naisdevice;} ''
  (
    set -x
    [[ "" == "$(${naisdevice}/bin/naisdevice-agent --help)" ]]
    [[ "" == "$(${naisdevice}/bin/naisdevice-systray --help)" ]]
    [[ "" == "$(${naisdevice}/bin/naisdevice-helper --help)" ]]
  )
  touch $out
''
