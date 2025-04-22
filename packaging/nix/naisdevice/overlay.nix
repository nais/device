let
  goVersion = "1.24.2";
  goSha256 = "sha256-NpMBYqk99BfZC9IsbhTa/0cFuqwrAkGO3aZxzfqc0H8=";
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
