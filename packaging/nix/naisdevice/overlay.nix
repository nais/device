final: prev: let
  goVersion = "1.26.2";
  newerGoVersion = prev.go_latest.overrideAttrs (old: {
    inherit goVersion;
    src = prev.fetchurl {
      url = "https://go.dev/dl/go${goVersion}.src.tar.gz";
      hash = "sha256-LpHrtpR6lulDb7KzkmqIAu/mOm03Xf/sT4Kqnb1v1Ds=";
    };
  });
  nixpkgsVersion = prev.go_latest.version;
  newVersionNotInNixpkgs = -1 == builtins.compareVersions nixpkgsVersion goVersion;
in {
  go_latest =
    if newVersionNotInNixpkgs
    then newerGoVersion
    else prev.go_latest;
  buildGoLatestModule = prev.buildGoLatestModule.override {go = final.go_latest;};
}
