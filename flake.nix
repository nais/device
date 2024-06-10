{
  description = "naisdevice";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-parts.url = "github:hercules-ci/flake-parts";
    flake-root.url = "github:srid/flake-root";
    treefmt-nix.url = "github:numtide/treefmt-nix";
    devshell.url = "github:numtide/devshell";
  };

  outputs = inputs @ {
    self,
    flake-parts,
    ...
  }:
    flake-parts.lib.mkFlake {inherit inputs;} {
      imports = [
        inputs.flake-root.flakeModule
        inputs.treefmt-nix.flakeModule
        inputs.devshell.flakeModule
      ];
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "aarch64-darwin"
        "x86_64-darwin"
      ];
      perSystem = {
        config,
        pkgs,
        system,
        ...
      }: {
        _module.args.pkgs = import inputs.nixpkgs {
          inherit system;
          overlays = [
            (import ./packaging/nix/naisdevice/overlay.nix)
          ];
        };
        packages.default = config.packages.naisdevice;

        packages.naisdevice = pkgs.callPackage ./packaging/nix/naisdevice/package.nix {inherit self;};

        checks.naisdevice = pkgs.callPackage ./packaging/nix/naisdevice/test.nix {
          inherit (config.packages) naisdevice;
        };
        devshells.default = {
          packages =
            (with pkgs; [
              gcc # needed for sqlite3-go
              gnumake
              go
              go-tools
              gopls
              gotools
              imagemagick
              protobuf
              sqlite
            ])
            ++ [config.treefmt.build.wrapper];
        };
        treefmt.config = {
          inherit (config.flake-root) projectRootFile;
          package = pkgs.treefmt;

          programs = {
            alejandra.enable = true;
            deadnix.enable = true;
            gofumpt.enable = true;
            prettier.enable = true;
            statix.enable = true;
          };
        };
      };
      flake = {
        nixosModules.naisdevice = {
          config,
          lib,
          pkgs,
          ...
        }: let
          inherit (lib) types mkOption;
          cfg = config.services.naisdevice;
        in {
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
            environment.systemPackages = [pkgs.wireguard-tools];
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
        };
      };
    };
}
