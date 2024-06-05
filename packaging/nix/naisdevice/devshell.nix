{pkgs, ...}: {
  shell = pkgs.mkShell {
    buildInputs = with pkgs; [
      go
      gopls
      gotools
      go-tools
      protobuf
      sqlite
      imagemagick
    ];
  };
}
