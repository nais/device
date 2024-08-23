{
  pkgs,
  self,
  ...
}: let
  version = builtins.substring 0 8 (self.lastModifiedDate or self.lastModified or "19700101");
  rev = self.rev or "dirty";
in
  pkgs.buildGoModule {
    pname = "naisdevice-controlplane";
    subPackages = [
      "cmd/apiserver"
      "cmd/gateway-agent"
      "cmd/enroller"
      "cmd/auth-server"
    ];
    inherit version;
    src = self;
    vendorHash = "sha256-2iJEbCSv0q6liCBWxruLC8aOavOoiFXVrAlbczRJbyA=";

    ldflags = [
      "-X github.com/nais/device/internal/version.Revision=${rev}"
      "-X github.com/nais/device/internal/version.Version=${version}"
      "-X github.com/nais/device/internal/otel.endpointURL=https://collector-internet.nav.cloud.nais.io"
    ];

    meta = with pkgs.lib; {
      description = "naisdevice - next gen vpn";
      homepage = "https://github.com/nais/device";
      license = licenses.mit;
    };
  }
