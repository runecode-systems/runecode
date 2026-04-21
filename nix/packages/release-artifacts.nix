{
  goToolchain,
  lib,
  pkgs,
  releaseMetadata,
  self,
}:

let
  renderTemplate =
    template: replacements:
    lib.foldlAttrs (
      rendered: name: value:
      lib.replaceStrings [ "@${name}@" ] [ (toString value) ] rendered
    ) (builtins.readFile template) replacements;

  releaseSource = lib.cleanSourceWith {
    src = self;
    filter =
      path: _type:
      let
        root = toString self;
        pathString = toString path;
        relativePath = if pathString == root then "." else lib.removePrefix "${root}/" pathString;
        keepPrefixes = [
          "cmd"
          "internal"
          "third_party"
          "tools/releasebuilder"
        ];
        matchesPrefix =
          prefix:
          relativePath == prefix
          || lib.hasPrefix "${prefix}/" relativePath
          || lib.hasPrefix "${relativePath}/" prefix;
      in
      relativePath == "."
      || lib.elem relativePath [
        "go.mod"
        "go.sum"
        "LICENSE"
        "NOTICE"
        "README.md"
      ]
      || lib.any matchesPrefix keepPrefixes;
  };

  binariesFile = pkgs.writeText "runecode-release-binaries.txt" (
    lib.concatStringsSep "\n" releaseMetadata.binaries + "\n"
  );

  targetsFile = pkgs.writeText "runecode-release-targets.txt" (
    lib.concatMapStringsSep "\n" (
      target: "${target.goos} ${target.goarch} ${target.archive}"
    ) releaseMetadata.targets
    + "\n"
  );

  buildScript = pkgs.writeText "build-release-artifacts.sh" (
    renderTemplate ../scripts/build-release-artifacts.sh {
      packageName = releaseMetadata.packageName;
      tag = releaseMetadata.tag;
      version = releaseMetadata.version;
      binariesFile = binariesFile;
      targetsFile = targetsFile;
      coreutils = pkgs.coreutils;
      gnutar = pkgs.gnutar;
      gzip = pkgs.gzip;
    }
  );
in
pkgs.buildGoModule {
  pname = "${releaseMetadata.packageName}-release-artifacts";
  version = releaseMetadata.version;
  src = releaseSource;
  go = goToolchain;
  # Refresh explicitly with `just refresh-release-vendor-hash`.
  vendorHash = "sha256-/tHE8xO+WET4nqUVL3dqPscxlN129vvQa+x7hoTX+pU=";
  # The workflow runs `just ci` before building this packaging-focused derivation.
  doCheck = false;
  strictDeps = true;

  nativeBuildInputs = [
    pkgs.bash
    pkgs.coreutils
    pkgs.gnutar
    pkgs.gzip
  ];

  buildPhase = ''
    runHook preBuild
    bash ${buildScript}
    runHook postBuild
  '';

  installPhase = ''
    runHook preInstall

    mkdir -p "$out"
    cp -R release/dist "$out/dist"
    cp -R release/payload "$out/payload"

    runHook postInstall
  '';
}
