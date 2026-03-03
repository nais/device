{
  naisdevice,
  runCommand,
}:
runCommand "test-naisdevice-binaries-stdout-help-text" {inherit naisdevice;} ''
  (
    set -x
    for bin in "agent" "systray" "helper"; do
      binary="${naisdevice}/bin/naisdevice-''${bin}"

      help_output="$(''${binary} --help 2>&1)"
      # Check that string is set to a value that is not empty
      [[ -n "$help_output" ]] && [[ "$help_output" == *"Usage of ''${binary}:"* ]]
    done
  )
  touch $out
''
