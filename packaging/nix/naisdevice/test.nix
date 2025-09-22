{
  naisdevice,
  runCommand,
}:
runCommand "test-naisdevice-binaries-stdout-help-text" {inherit naisdevice;} ''
  (
    set -x
    for bin in "agent" "systray" "helper"; do
      set +e # TODO: `pflag: help requested` returns returncode 2
      help_output="$(${naisdevice}/bin/naisdevice-''${bin} --help)"
      set -e # TODO: `pflag: help requested` returns returncode 2

      # Check that string is set to a value that is not empty
      [[ -n "$help_output" ]] && [[ "$help_output" = "pflag: help requested" ]]
    done
  )
  touch $out
''
