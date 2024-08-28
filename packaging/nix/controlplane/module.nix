{
  self,
  config,
  lib,
  pkgs,
  system,
  ...
}: let
  cfg = config.services.apiserver;
in {
  options.services.apiserver = {
    enable = lib.mkEnableOption "naisdevice-apiserver service";
    package = lib.mkOption {
      type = lib.types.package;
      default = self.packages.${system}.apiserver;
      description = lib.mdDoc ''
        The apiserver package to use.
      '';
    };
  };

  config = lib.mkIf cfg.enable {
    systemd.services.apiserver = {
      description = "apiserver service";
      after = ["network.target"];
      path = [
        pkgs.wireguard-tools
        pkgs.iproute2
      ];
      serviceConfig.ExecStart = "${cfg.package}/bin/apiserver";
      serviceConfig.Restart = "always";
    };
  };
}
