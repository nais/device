final: prev:
let
  goVersion = "1.25.1";
  newerGoVersion = prev.go.overrideAttrs (old: {
    inherit goVersion;
    src = prev.fetchurl {
      url = "https://go.dev/dl/go${goVersion}.src.tar.gz";
      hash = "sha256-0BDBCc7pTYDv5oHqtGvepJGskGv0ZYPDLp8NuwvRpZQ=";
    };
  });
  nixpkgsVersion = prev.go.version;
  newVersionNotInNixpkgs = -1 == builtins.compareVersions nixpkgsVersion goVersion;
in
{
  go = if newVersionNotInNixpkgs then newerGoVersion else prev.go;
  buildGoModule = prev.buildGoModule.override { go = final.go; };
}
