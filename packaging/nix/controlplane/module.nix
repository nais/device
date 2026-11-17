{
  lib,
  package,
  ...
}: {
  system.stateVersion = lib.version;
  systemd.services.apiserver = {
    description = "apiserver service";
    after = ["network.target"];
    serviceConfig.ExecStart = "${package}/bin/apiserver";
    serviceConfig.Restart = "always";
  };
}
