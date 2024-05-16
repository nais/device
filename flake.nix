{
  description = "A simple Go package";

  # Nixpkgs / NixOS version to use.
  inputs.nixpkgs.url = "nixpkgs/nixos-unstable";

  outputs =
    { self, nixpkgs }:
    let
      # to work with older version of flakes
      lastModifiedDate = self.lastModifiedDate or self.lastModified or "19700101";

      # Generate a user-friendly version number.
      version = builtins.substring 0 8 lastModifiedDate;

      # System types to support.
      supportedSystems = [ "x86_64-linux" ]; # "x86_64-darwin" "aarch64-linux" "aarch64-darwin"];

      # Helper function to generate an attrset '{ x86_64-linux = f "x86_64-linux"; ... }'.
      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;

      # Nixpkgs instantiated for supported system types.
      nixpkgsFor = forAllSystems (system: import nixpkgs { inherit system overlays; });

      overlays = [
        goOverlay
        naisdeviceOverlay
      ];

      goVersion = "1.22.3";
      goOverlay = final: prev: {
        go = prev.go.overrideAttrs (old: {
          version = goVersion;
          src = prev.fetchurl {
            url = "https://go.dev/dl/go${goVersion}.src.tar.gz";
            hash = "sha256-gGSO80+QMZPXKlnA3/AZ9fmK4MmqE63gsOy/+ZGnb2g=";
          };
        });
      };

      buildNaisdevice =
        pkgs: vendorHash:
        pkgs.buildGoModule {
          pname = "naisdevice";
          subPackages = [
            "cmd/naisdevice-helper"
            "cmd/naisdevice-systray"
            "cmd/naisdevice-agent"
          ];
          inherit version;
          src = ./.;
          vendorHash = vendorHash;

          meta = with pkgs.lib; {
            description = "naisdevice - next gen vpn";
            homepage = "https://github.com/nais/device";
            license = licenses.mit;
          };
        };
      naisdeviceOverlay = final: prev: {
        naisdevice = buildNaisdevice prev.pkgs "sha256-+Wgx4/usjAivatYC4jcwjpssGS8U22nimcvVmLfsvfA=";
      };
    in
    {
      package = nixpkgsFor.x86_64-linux.naisdevice;
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
              default = nixpkgsFor.x86_64-linux.naisdevice;
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

      devShells = forAllSystems (
        system:
        let
          pkgs = nixpkgsFor.${system};
        in
        {
          default = pkgs.mkShell {
            buildInputs = with pkgs; [
              go
              gopls
              gotools
              go-tools
              protobuf
              sqlite
              imagemagick
            ];
          };
        }
      );

      formatter.x86_64-linux = nixpkgs.legacyPackages.x86_64-linux.nixfmt-rfc-style;
    };
}
