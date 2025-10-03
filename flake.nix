{
  description = "naisdevice";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

    flake-parts.url = "github:hercules-ci/flake-parts";
    flake-root.url = "github:srid/flake-root";

    devshell.url = "github:numtide/devshell";
    devshell.inputs.nixpkgs.follows = "nixpkgs";

    nixos-generators.url = "github:nix-community/nixos-generators";
    nixos-generators.inputs.nixpkgs.follows = "nixpkgs";
  };

  outputs = inputs @ {
    self,
    nixpkgs,
    flake-parts,
    nixos-generators,
    ...
  }:
    flake-parts.lib.mkFlake {inherit inputs;} {
      imports = [
        inputs.flake-root.flakeModule
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
        _module.args.pkgs = import nixpkgs {
          inherit system;
          overlays = [
            (import ./packaging/nix/naisdevice/overlay.nix) # Allows referencing new versions of go before landed in channel
          ];
        };
        packages.default = config.packages.naisdevice;
        packages.naisdevice = pkgs.callPackage ./packaging/nix/naisdevice/package.nix {inherit self;};
        packages.apiserver = pkgs.callPackage ./packaging/nix/controlplane/package.nix {
          inherit self;
          subPackage = "apiserver";
        };
        checks.naisdevice = pkgs.callPackage ./packaging/nix/naisdevice/test.nix {
          inherit (config.packages) naisdevice;
        };
        devshells.default = {
          packages = with pkgs; [
            gcc # needed for sqlite3-go
            gnumake
            go
            gopls
            graphviz
            imagemagick
            protobuf
            sqlite-interactive # -interactive gives readline / ncurses
          ];
        };
      };
      flake = {
        gce = let
          system = "x86_64-linux";
        in
          nixos-generators.nixosGenerate {
            inherit system;
            modules = [
              ./packaging/nix/controlplane/module.nix
            ];
            specialArgs = {
              package = self.packages.${system}.apiserver;
            };
            format = "gce";
          };
        nixosModules = rec {
          default = naisdevice;
          naisdevice = {
            pkgs,
            lib,
            config,
            ...
          }:
            import ./packaging/nix/naisdevice/module.nix {inherit self lib pkgs config;};
        };
      };
    };
}
