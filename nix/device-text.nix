{ flake ? builtins.getFlake (toString ./.)
, pkgs ? flake.inputs.nixpkgs.legacyPackages.${builtins.currentSystem}
, makeTest ?
  pkgs.callPackage (flake.inputs.nixpkgs + "/nixos/tests/make-test-python.nix")
, migrana ? flake.defaultPackage.${builtins.currentSystem} }:

let
  makeTest' = test:
    makeTest test {
      inherit pkgs;
      inherit (pkgs) system;
    };
in {
  naisDeviceTest = makeTest' {
    name = "nais-device test";
    nodes.apiserver = { config, ... }: {
      networking.firewall.allowedTCPPorts = [ ];
    };

    nodes.client = { ... }: {
      imports = [ ];
      environment.systemPackages = with pkgs; [ ];
    };

    testScript = "";
  };
}
