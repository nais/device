final: prev:
let
  goVersion = "1.26.4";
  newerGoVersion = prev.go_latest.overrideAttrs (old: {
    inherit goVersion;
    src = prev.fetchurl {
      url = "https://go.dev/dl/go${goVersion}.src.tar.gz";
      hash = "sha256-T2aKMvv8ETLmqIH7lowvHa2mMUkqM5IRc1+7JVpCYC0=";
    };
  });
  nixpkgsVersion = prev.go_latest.version;
  newVersionNotInNixpkgs = -1 == builtins.compareVersions nixpkgsVersion goVersion;
in
{
  go_latest = if newVersionNotInNixpkgs then newerGoVersion else prev.go_latest;
  buildGoLatestModule = prev.buildGoLatestModule.override { go = final.go_latest; };
}
