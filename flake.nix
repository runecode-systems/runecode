{
  description = "RuneCode dev environment and canonical release builder (Nix >= 2.18)";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.11";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    let
      releaseMetadata = import ./nix/release/metadata.nix;
    in
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs { inherit system; };
        lib = pkgs.lib;
        goToolchain = pkgs.go_1_25 or (throw "nixpkgs must provide go_1_25 for RuneCode release builds");

        releaseArtifacts = import ./nix/packages/release-artifacts.nix {
          inherit
            goToolchain
            lib
            pkgs
            self
            ;
          releaseMetadata = releaseMetadata;
        };

        devShell = import ./nix/dev-shell.nix {
          inherit goToolchain pkgs;
        };

        checks = import ./nix/checks.nix {
          inherit
            devShell
            lib
            pkgs
            releaseMetadata
            releaseArtifacts
            self
            system
            ;
        };
      in
      {
        formatter = pkgs.nixfmt-rfc-style;

        devShells.default = devShell;

        packages = {
          release-artifacts = releaseArtifacts;
        };

        inherit checks;
      }
    )
    // {
      lib.release = releaseMetadata;
    };
}
