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
  migranaTest = makeTest' {
    name = "roundtrip migrations";

    nodes.oldGrafana = { config, ... }: {
      networking.firewall.allowedTCPPorts = [ 3500 ];
      systemd.services.grafana = {
        after = [ "network-interfaces.target" ];
        wants = [ "network-interfaces.target" ];
      };

      environment.etc = { dashboards.source = ./dashboards; };
      services.loki.enable = true;
      services.loki.configFile = pkgs.writeText "loki.yml" ''
        ingester:
          chunk_target_size: 5242880
        auth_enabled: false
        server:
          http_listen_port: 3100
          grpc_listen_port: 9096

        common:
          ring:
            instance_addr: 127.0.0.1
            kvstore:
              store: inmemory
          replication_factor: 1
          path_prefix: /tmp/loki

        schema_config:
          configs:
          - from: 2020-05-19
            store: boltdb-shipper
            object_store: filesystem
            schema: v11
            index:
              prefix: index_
              period: 24h
        analytics:
          reporting_enabled: false
      '';

      services.prometheus.enable = true;
      services.grafana = {
        package = pkgs.grafana9;
        enable = true;
        provision = {
          enable = true;
          datasources = {
            settings = {
              datasources = [
                {
                  name = "Prometheus";
                  type = "prometheus";
                  access = "proxy";
                  url = "http://127.0.0.1:6969";
                  isDefault = true;
                }
                {
                  name = "Loki";
                  type = "loki";
                  access = "proxy";
                  url = "http://127.0.0.1:127.0.0.1";
                }
              ];
            };
          };
          dashboards = {
            settings = {
              providers = [{
                name = "My Dashboards";
                options.path = "/etc/dashboards";
              }];
            };
          };
        };

        settings = {
          server = {
            http_port = 3500;
            http_addr = "";
            protocol = "http";
          };
        };
      };
    };

    nodes.newGrafana = { config, ... }: {
      networking.firewall.allowedTCPPorts = [ 3000 ];
      systemd.services.grafana = {
        after = [ "network-interfaces.target" ];
        wants = [ "network-interfaces.target" ];
      };
      services.grafana = {
        enable = true;
        settings = {
          server = {
            http_port = 3000;
            http_addr = "";
            protocol = "http";
          };
        };
      };
    };

    nodes.client = { ... }: {
      imports = [ ];
      environment.systemPackages = with pkgs; [
        curl
        jq
        inetutils
        migrana
        tree
      ];
    };

    testScript = "";
  };
}
