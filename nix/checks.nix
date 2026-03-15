{
  devShell,
  lib,
  pkgs,
  releaseMetadata,
  releaseArtifacts,
  self,
  system,
}:

let
  declaredReleaseBinaries = pkgs.writeText "runecode-release-binaries-check.txt" (
    lib.concatStringsSep "\n" releaseMetadata.binaries + "\n"
  );
in
{
  dev-shell = devShell;

  release-binaries =
    pkgs.runCommand "release-binaries-check"
      {
        nativeBuildInputs = [
          pkgs.coreutils
          pkgs.diffutils
        ];
      }
      ''
        actual="$TMPDIR/actual-release-binaries"
        declared="$TMPDIR/declared-release-binaries"

        for dir in ${self}/cmd/*; do
          [ -d "$dir" ] || continue
          basename "$dir"
        done | sort > "$actual"

        sort ${declaredReleaseBinaries} > "$declared"

        diff -u "$declared" "$actual"
        touch "$out"
      '';

  nix-format =
    pkgs.runCommand "nix-format-check"
      {
        nativeBuildInputs = [
          pkgs.fd
          pkgs.nixfmt-rfc-style
        ];
      }
      ''
        files=("${self}/flake.nix")
        while IFS= read -r file; do
          files+=("$file")
        done < <(${pkgs.fd}/bin/fd --extension nix --type f . ${self}/nix)

        nixfmt --check "''${files[@]}"
        touch "$out"
      '';
}
// lib.optionalAttrs (system == "x86_64-linux") {
  # Release artifacts are checked on the same Linux architecture used by the
  # canonical tag-driven release workflow.
  release-artifacts = releaseArtifacts;
}
