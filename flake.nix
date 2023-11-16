{
  description = "Naisdevice";

  inputs = {
    nixpkgs.url = # Pick a some commit
      "github:NixOS/nixpkgs";
    flake-utils.url = "github:numtide/flake-utils";
    gitignore = {
      url = "github:hercules-ci/gitignore.nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs = { self, nixpkgs, flake-utils, gitignore }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};

        naisdevice = pkgs.buildGoModule {
          name = "naisdevice";
          nativeBuildInputs = with pkgs; [ protobuf3_20 ];
          subPackages = [
            "./cmd/apiserver"
            "./cmd/device-agent"
            "./cmd/gateway-agent"
            "./cmd/helper"
          ];
          src = gitignore.lib.gitignoreSource ./.;
          vendorHash = "sha256-zjtLAt2H6PhWb2YQNaIEvxY0Zii7pcA0TU91C3nCKXM=";
          # "sha256-h2x22TJkOrRzkU8TAV6OUJTSxIPWGyccDzKeacj43B4=";
          # nixpkgs.lib.fakeSha256;
        };

      in {
        defaultPackage = naisdevice;
        devShell = pkgs.mkShell {
          packages = with pkgs; [ go_1_21 golangci-lint gopls gotools ];
        };
      });
}
