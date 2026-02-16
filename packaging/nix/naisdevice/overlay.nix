final: prev: let
  goVersion = "1.26.0";
  newerGoVersion = prev.go.overrideAttrs (old: {
    inherit goVersion;
    src = prev.fetchurl {
      url = "https://go.dev/dl/go${goVersion}.src.tar.gz";
      hash = "sha256-yRMqih9r0qpKrR10uCMdlSdJUEg6SVBlfubFbm6Bd5A=";
    };
  });
  nixpkgsVersion = prev.go.version;
  newVersionNotInNixpkgs = -1 == builtins.compareVersions nixpkgsVersion goVersion;
in {
  go =
    if newVersionNotInNixpkgs
    then newerGoVersion
    else prev.go;
  buildGoModule = prev.buildGoModule.override {go = final.go;};
}
