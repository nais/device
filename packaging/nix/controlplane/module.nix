{
  pkgs,
  lib,
  package,
  ...
}: {
  system.stateVersion = lib.version;
  systemd.services.apiserver = {
    description = "apiserver service";
    after = ["network.target"];
    path = [
      pkgs.wireguard-tools
      pkgs.iproute2
    ];
    serviceConfig.ExecStart = "${package}/bin/apiserver";
    serviceConfig.Restart = "always";
  };
}
