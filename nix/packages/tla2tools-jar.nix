{ pkgs }:
pkgs.runCommand "tla2tools-jar" { } ''
  mkdir -p "$out"
  cp ${pkgs.tlaplus}/share/java/tla2tools.jar "$out/tla2tools.jar"
  sha256sum "$out/tla2tools.jar" | cut -d' ' -f1 > "$out/tla2tools.jar.sha256"
''
