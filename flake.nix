{
  description = "Naisdevice";

  inputs = {
    flake-parts.url = "github:hercules-ci/flake-parts";
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    devshell.url = "github:numtide/devshell";
  };

  outputs =
    inputs@{ self, flake-parts, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      imports = [ inputs.devshell.flakeModule ];
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "aarch64-darwin"
        "x86_64-darwin"
      ];
      perSystem =
        { config, pkgs, ... }:
        {
          packages.default = config.packages.naisdevice;

          packages.naisdevice = pkgs.callPackage ./packaging/nix/naisdevice/package.nix { inherit self; };

          checks.naisdevice = pkgs.callPackage ./packaging/nix/naisdevice/test.nix {
            naisdevice = config.packages.naisdevice;
          };
          devshells.default = {
            packages = with pkgs; [
              go
              gopls
              gotools
              go-tools
              protobuf
              sqlite
              imagemagick
            ];
          };
          formatter = pkgs.nixfmt-rfc-style;
        };
      flake = {
        nixosModules.naisdevice =
          {
            config,
            lib,
            pkgs,

            ...
          }:
          let
            inherit (lib) types mkOption;
            cfg = config.services.naisdevice;
          in
          {
            options.services.naisdevice = {
              enable = lib.mkEnableOption "naisdevice-helper service";
              package = mkOption {
                type = types.package;
                default = self.packages.${pkgs.stdenv.hostPlatform.system}.naisdevice;
                description = lib.mdDoc ''
                  The naisdevice package to use.
                '';
              };
            };

            config = lib.mkIf cfg.enable {
              environment.systemPackages = [ pkgs.wireguard-tools ];
              systemd.services.naisdevice-helper = {
                description = "naisdevice-helper service";
                wantedBy = [ "multi-user.target" ];
                path = [
                  pkgs.wireguard-tools
                  pkgs.iproute2
                ];
                serviceConfig.ExecStart = "${cfg.package}/bin/naisdevice-helper";
                serviceConfig.Restart = "always";
              };
            };
          };
      };
    };
}
