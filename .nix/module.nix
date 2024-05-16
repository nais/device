naisdevice:
{
  config,
  lib,
  pkgs,
  ...
}:
let
  inherit (lib) types mkOption;
  cfg = config.services.naisdevice;
  pkg = cfg.package;
in
{
  options.services.naisdevice = {
    enable = lib.mkEnableOption "naisdevice-helper service";
    package = mkOption {
      type = types.package;
      default = pkgs.naisdevice;
      description = lib.mdDoc ''
        The naisdevice package to use.
      '';
    };
  };

  config = lib.mkIf cfg.enable {
    systemd.services.naisdevice-helper = {
      description = "naisdevice-helper service";
      wantedBy = [ "multi-user.target" ];
      serviceConfig.ExecStart = "${pkg}/bin/helper";
      serviceConfig.Restart = "Always";
    };
  };
}
