{
  pkgs,
  version,
  rev,
  ...
}:
pkgs.buildGoModule {
  pname = "naisdevice";
  subPackages = [
    "cmd/naisdevice-helper"
    "cmd/naisdevice-systray"
    "cmd/naisdevice-agent"
  ];
  inherit version;
  src = ../../../.;
  vendorHash = "sha256-Sul8Bre6uvS9OSKa2Hqinlz51kvts6ZM76SAzttJ4tw=";

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
