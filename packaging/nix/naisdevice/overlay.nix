let
  goVersion = "1.22.5";
  goSha256 = "sha256-rJxyPyJJaa7mJLw0/TTJ4T8qIS11xxyAfeZEu0bhEvY=";
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
