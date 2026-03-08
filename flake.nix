{
  description = "RuneCode dev environment (Nix >= 2.18)";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-24.11";
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
        pkgs = import nixpkgs { inherit system; };
        devPackages = with pkgs; [
          go
          gopls
          gotools
          golangci-lint
          nodejs_22
          nodePackages.typescript
          just
          git
          jq
          ripgrep
          fd
          curl
        ];

        devShell = pkgs.mkShell {
          packages = devPackages;
          shellHook = ''
            if [ -t 1 ] && [ -n "''${PS1:-}" ]; then
              echo "Entering dev shell"
              just --list
            fi
          '';
        };
      in
      {
        formatter = pkgs.nixfmt-rfc-style;

        devShells.default = devShell;

        checks = {
          dev-shell = devShell;

          nix-format =
            pkgs.runCommand "nix-format-check"
              {
                nativeBuildInputs = [ pkgs.nixfmt-rfc-style ];
              }
              ''
                nixfmt --check ${self}/flake.nix
                touch "$out"
              '';
        };
      }
    );
}
