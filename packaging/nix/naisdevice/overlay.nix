let
  goVersion = "1.22.4";
  goSha256 = "sha256-/tcgZ45yinyjC6jR3tHKr+J9FgKPqwIyuLqOIgCPt4Q=";
in
  _final: prev: {
    go = prev.go.overrideAttrs (_old: {
      version = goVersion;
      src = prev.fetchurl {
        url = "https://go.dev/dl/go${goVersion}.src.tar.gz";
        hash = goSha256;
      };
    });
  }
