{
  self,
  config,
  lib,
  pkgs,
  ...
}: let
  cfg = config.services.naisdevice;
in {
  options.services.naisdevice = {
    enable = lib.mkEnableOption "naisdevice-helper service";
    package = lib.mkOption {
      type = lib.types.package;
      default = self.packages.${pkgs.stdenv.hostPlatform.system}.naisdevice;
      description = lib.mdDoc ''
        The naisdevice package to use.
      '';
    };
  };

  config = lib.mkIf cfg.enable {
    systemd.services.naisdevice-helper = {
      description = "naisdevice-helper service";
      wantedBy = ["multi-user.target"];
      path = [
        pkgs.wireguard-tools
        pkgs.iproute2
      ];
      serviceConfig.ExecStart = "${cfg.package}/bin/naisdevice-helper";
      serviceConfig.Restart = "always";
    };
  };
}
