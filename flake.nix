{
  description = "aidir development environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "aiw";
          version = "dev";
          src = ./.;
          vendorHash = null;
        };

        devShells.default = pkgs.mkShell {
          packages = [
            pkgs.sqlite
            pkgs.go
            pkgs.goreleaser
            pkgs.gotools # goimports, godoc, etc.
            pkgs.gh
            pkgs.fzf
            pkgs.zellij
            pkgs.golangci-lint
          ];
        };
      }
    );
}
