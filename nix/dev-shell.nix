{ goToolchain, pkgs }:

pkgs.mkShell {
  packages = with pkgs; [
    goToolchain
    gopls
    gotools
    golangci-lint
    nodejs_24
    nodePackages.typescript
    just
    git
    jq
    ripgrep
    fd
    curl
  ];

  shellHook = ''
    if [ -t 1 ] && [ -n "''${PS1:-}" ]; then
      echo "Entering dev shell"
      just --list
    fi
  '';
}
